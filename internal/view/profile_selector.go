package view

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/log"
	navmsg "github.com/clawscli/claws/internal/msg"
	"github.com/clawscli/claws/internal/ui"
)

type profileItem struct {
	id          string
	display     string
	isSSO       bool
	profileType string
	region      string
}

func (p profileItem) GetID() string    { return p.id }
func (p profileItem) GetLabel() string { return p.display }

type ProfileSelector struct {
	selector    *MultiSelector[profileItem]
	profiles    []profileItem
	profileInfo map[string]aws.ProfileInfo
	ssoLogin    ssoLoginRunner

	loginResult *loginResultMsg
	typeStyle   lipgloss.Style
	regionStyle lipgloss.Style
}

type ssoLoginRunner func(context.Context, aws.ProfileInfo, io.Writer) (aws.SSOLoginResult, error)

func NewProfileSelector() *ProfileSelector {
	initialSelected := make([]string, 0)
	for _, sel := range config.Global().Selections() {
		initialSelected = append(initialSelected, sel.ID())
	}

	p := &ProfileSelector{
		selector:    NewMultiSelector[profileItem]("Select Profiles", initialSelected),
		profileInfo: make(map[string]aws.ProfileInfo),
		ssoLogin:    aws.RunSSOLogin,
		typeStyle:   ui.DimStyle(),
		regionStyle: ui.DimStyle(),
	}

	p.selector.SetRenderExtra(func(item profileItem) string {
		var parts []string
		if item.profileType != "" {
			parts = append(parts, p.typeStyle.Render("["+item.profileType+"]"))
		}
		if item.region != "" {
			parts = append(parts, p.regionStyle.Render(item.region))
		}
		return strings.Join(parts, " ")
	})

	return p
}

func (p *ProfileSelector) Init() tea.Cmd {
	return p.loadProfiles
}

type profilesLoadedMsg struct {
	profiles []profileItem
	infoMap  map[string]aws.ProfileInfo
}

type loginResultMsg struct {
	profileID      string
	success        bool
	err            error
	isConsoleLogin bool
	message        string
}

func (p *ProfileSelector) loadProfiles() tea.Msg {
	profiles := []profileItem{
		{id: config.ProfileIDSDKDefault, display: config.SDKDefault().DisplayName(), profileType: "Default"},
		{id: config.ProfileIDEnvOnly, display: config.EnvOnly().DisplayName(), profileType: "Env/IMDS"},
	}
	infoMap := make(map[string]aws.ProfileInfo)

	loaded, err := aws.LoadProfiles()
	if err != nil {
		log.Error("failed to load profiles", "error", err)
	}
	for _, info := range loaded {
		profiles = append(profiles, profileItem{
			id:          info.Name,
			display:     info.Name,
			isSSO:       info.IsSSO,
			profileType: info.ProfileType,
			region:      info.Region,
		})
		infoMap[info.Name] = info
	}

	return profilesLoadedMsg{profiles: profiles, infoMap: infoMap}
}

func (p *ProfileSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case profilesLoadedMsg:
		p.profiles = msg.profiles
		p.profileInfo = msg.infoMap
		p.selector.SetItems(p.profiles)
		return p, nil
	case ThemeChangedMsg:
		p.selector.ReloadStyles()
		return p, nil

	case loginResultMsg:
		p.loginResult = &msg
		if msg.success {
			if msg.isConsoleLogin {
				selected := p.selector.Selected()
				for id := range selected {
					delete(selected, id)
				}
				selected[msg.profileID] = true

				sel := config.NamedProfile(msg.profileID)
				selections := []config.ProfileSelection{sel}
				config.Global().SetSelections(selections)
				p.selector.ClearResult()
				p.updateExtraHeight()
				return p, func() tea.Msg {
					return navmsg.ProfilesChangedMsg{Selections: selections}
				}
			}
			p.selector.Selected()[msg.profileID] = true
			p.selector.ClearResult()
		}
		p.updateExtraHeight()
		return p, nil

	case tea.KeyPressMsg:
		if !p.selector.FilterActive() {
			switch msg.String() {
			case "up", "k", "down", "j":
				p.loginResult = nil
				p.updateExtraHeight()
			case "c":
				p.loginResult = nil
				p.updateExtraHeight()
			case "d":
				return p.toggleDetail()
			case "l":
				return p.ssoLoginCurrentProfile()
			case "L":
				return p.consoleLoginCurrentProfile()

			}
		}
	}

	cmd, result := p.selector.HandleUpdate(msg)
	if result == KeyApply {
		return p.applySelection()
	}
	return p, cmd
}

func (p *ProfileSelector) updateExtraHeight() {
	if p.loginResult != nil {
		p.selector.SetExtraHeight(1)
	} else {
		p.selector.SetExtraHeight(0)
	}
}

func (p *ProfileSelector) applySelection() (tea.Model, tea.Cmd) {
	selected := p.selector.SelectedItems()
	if len(selected) == 0 {
		return p, nil
	}

	selections := make([]config.ProfileSelection, len(selected))
	for i, item := range selected {
		selections[i] = config.ProfileSelectionFromID(item.id)
	}

	config.Global().SetSelections(selections)
	return p, func() tea.Msg {
		return navmsg.ProfilesChangedMsg{Selections: selections}
	}
}

func (p *ProfileSelector) ssoLoginCurrentProfile() (tea.Model, tea.Cmd) {
	profile, ok := p.selector.CurrentItem()
	if !ok {
		return p, nil
	}

	if !profile.isSSO {
		p.loginResult = &loginResultMsg{
			profileID: profile.id,
			success:   false,
			err:       fmt.Errorf("profile %q is not SSO", profile.id),
		}
		p.updateExtraHeight()
		return p, nil
	}

	if config.Global().ReadOnly() && !action.IsExecAllowedInReadOnly(action.ActionNameSSOLogin) {
		p.loginResult = &loginResultMsg{
			profileID: profile.id,
			success:   false,
			err:       fmt.Errorf("SSO login denied: read-only mode"),
		}
		p.updateExtraHeight()
		return p, nil
	}

	profileInfo, ok := p.profileInfo[profile.id]
	if !ok {
		p.loginResult = &loginResultMsg{
			profileID: profile.id,
			success:   false,
			err:       fmt.Errorf("SSO profile metadata not loaded for %q", profile.id),
		}
		p.updateExtraHeight()
		return p, nil
	}

	profileID := profile.id
	execCmd := &ssoLoginExec{
		profile: profileInfo,
		run:     p.ssoLogin,
	}
	return p, tea.Exec(execCmd, func(err error) tea.Msg {
		if err != nil {
			return loginResultMsg{profileID: profileID, success: false, err: err}
		}
		return loginResultMsg{profileID: profileID, success: true, message: execCmd.result.Message}
	})
}

type ssoLoginExec struct {
	profile aws.ProfileInfo
	run     ssoLoginRunner
	result  aws.SSOLoginResult

	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

func (e *ssoLoginExec) SetStdin(r io.Reader)  { e.stdin = r }
func (e *ssoLoginExec) SetStdout(w io.Writer) { e.stdout = w }
func (e *ssoLoginExec) SetStderr(w io.Writer) { e.stderr = w }

func (e *ssoLoginExec) Run() error {
	stdout := e.stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	result, err := e.run(context.Background(), e.profile, stdout)
	if err != nil {
		return err
	}
	e.result = result
	return nil
}

func (p *ProfileSelector) consoleLoginCurrentProfile() (tea.Model, tea.Cmd) {
	profile, ok := p.selector.CurrentItem()
	if !ok {
		return p, nil
	}

	if profile.id == config.ProfileIDSDKDefault || profile.id == config.ProfileIDEnvOnly {
		p.loginResult = &loginResultMsg{
			profileID:      profile.id,
			success:        false,
			err:            fmt.Errorf("console login requires named profile, got %q", profile.id),
			isConsoleLogin: true,
		}
		p.updateExtraHeight()
		return p, nil
	}

	if config.Global().ReadOnly() && !action.IsExecAllowedInReadOnly(action.ActionNameLogin) {
		p.loginResult = &loginResultMsg{
			profileID:      profile.id,
			success:        false,
			err:            fmt.Errorf("console login denied: read-only mode"),
			isConsoleLogin: true,
		}
		p.updateExtraHeight()
		return p, nil
	}

	if _, err := action.ResolveExecutable("aws"); err != nil {
		p.loginResult = &loginResultMsg{
			profileID:      profile.id,
			success:        false,
			err:            fmt.Errorf("aws CLI not found in PATH: %w", err),
			isConsoleLogin: true,
		}
		p.updateExtraHeight()
		return p, nil
	}

	profileID := profile.id
	execCmd, err := newProfileLoginExec(profileID)
	if err != nil {
		p.loginResult = &loginResultMsg{profileID: profileID, success: false, err: err, isConsoleLogin: true}
		p.updateExtraHeight()
		return p, nil
	}
	return p, tea.Exec(execCmd, func(err error) tea.Msg {
		if err != nil {
			return loginResultMsg{profileID: profileID, success: false, err: err, isConsoleLogin: true}
		}
		return loginResultMsg{profileID: profileID, success: true, isConsoleLogin: true}
	})
}

func newProfileLoginExec(profileID string) (*action.SimpleExec, error) {
	if !config.IsValidProfileName(profileID) {
		return nil, fmt.Errorf("invalid profile name: %s", profileID)
	}
	awsPath, err := action.ResolveExecutable("aws")
	if err != nil {
		return nil, fmt.Errorf("aws CLI not found in PATH: %w", err)
	}
	return &action.SimpleExec{
		Args:       []string{awsPath, "login", "--remote", "--profile", profileID},
		ActionName: action.ActionNameLogin,
		SkipAWSEnv: true,
	}, nil
}

func (p *ProfileSelector) ViewString() string {
	content := p.selector.ViewString()

	if p.loginResult != nil {
		content += "\n"
		loginType := "SSO"
		if p.loginResult.isConsoleLogin {
			loginType = "Console"
		}
		if p.loginResult.success {
			message := p.loginResult.message
			if message == "" {
				message = loginType + " login successful"
			}
			content += ui.SuccessStyle().Render(message)
		} else {
			content += ui.DangerStyle().Render(loginType + " login failed: " + p.loginResult.err.Error())
		}
	}

	return content
}

func (p *ProfileSelector) View() tea.View {
	return tea.NewView(p.ViewString())
}

func (p *ProfileSelector) SetSize(width, height int) tea.Cmd {
	p.updateExtraHeight()
	p.selector.SetSize(width, height)
	return nil
}

func (p *ProfileSelector) StatusLine() string {
	count := p.selector.SelectedCount()
	if p.selector.FilterActive() {
		return "Type to filter • Enter confirm • Esc cancel"
	}

	var loginHints string
	if profile, ok := p.selector.CurrentItem(); ok {
		if profile.isSSO {
			loginHints = " • l:SSO"
		}
		if profile.id != config.ProfileIDSDKDefault && profile.id != config.ProfileIDEnvOnly {
			loginHints += " • L:console"
		}
	}

	return "Space:toggle • d:detail • Enter:apply" + loginHints + " • " + strings.Repeat("●", count) + " selected"
}

func (p *ProfileSelector) HasActiveInput() bool {
	return p.selector.FilterActive()
}

func (p *ProfileSelector) toggleDetail() (tea.Model, tea.Cmd) {
	profile, ok := p.selector.CurrentItem()
	if !ok {
		return p, nil
	}
	info, hasInfo := p.profileInfo[profile.id]
	detailView := NewProfileDetailView(profile, info, hasInfo)
	return p, func() tea.Msg {
		return ShowModalMsg{Modal: &Modal{Content: detailView, Width: ModalWidthProfileDetail}}
	}
}
