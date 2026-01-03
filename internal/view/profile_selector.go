package view

import (
	"fmt"
	"io"
	"os/exec"
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

	loginResult *loginResultMsg
	typeStyle   lipgloss.Style
	regionStyle lipgloss.Style
}

func NewProfileSelector() *ProfileSelector {
	initialSelected := make([]string, 0)
	for _, sel := range config.Global().Selections() {
		initialSelected = append(initialSelected, sel.ID())
	}

	p := &ProfileSelector{
		selector:    NewMultiSelector[profileItem]("Select Profiles", initialSelected),
		profileInfo: make(map[string]aws.ProfileInfo),
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

	case loginResultMsg:
		p.loginResult = &msg
		if msg.success {
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

	if _, err := exec.LookPath("aws"); err != nil {
		p.loginResult = &loginResultMsg{
			profileID: profile.id,
			success:   false,
			err:       fmt.Errorf("aws CLI not found in PATH"),
		}
		p.updateExtraHeight()
		return p, nil
	}

	profileID := profile.id
	return p, tea.Exec(&ssoLoginCmd{profileName: profileID}, func(err error) tea.Msg {
		if err != nil {
			return loginResultMsg{profileID: profileID, success: false, err: err}
		}
		return loginResultMsg{profileID: profileID, success: true}
	})
}

type ssoLoginCmd struct {
	profileName string
	stdin       io.Reader
	stdout      io.Writer
	stderr      io.Writer
}

func (s *ssoLoginCmd) Run() error {
	cmd := exec.Command("aws", "sso", "login", "--profile", s.profileName)
	cmd.Stdin = s.stdin
	cmd.Stdout = s.stdout
	cmd.Stderr = s.stderr
	return cmd.Run()
}

func (s *ssoLoginCmd) SetStdin(r io.Reader)  { s.stdin = r }
func (s *ssoLoginCmd) SetStdout(w io.Writer) { s.stdout = w }
func (s *ssoLoginCmd) SetStderr(w io.Writer) { s.stderr = w }

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

	if _, err := exec.LookPath("aws"); err != nil {
		p.loginResult = &loginResultMsg{
			profileID:      profile.id,
			success:        false,
			err:            fmt.Errorf("aws CLI not found in PATH"),
			isConsoleLogin: true,
		}
		p.updateExtraHeight()
		return p, nil
	}

	profileID := profile.id
	execCmd := &action.SimpleExec{
		Command:    "aws login --remote --profile " + profileID,
		ActionName: action.ActionNameLogin,
		SkipAWSEnv: true,
	}
	return p, tea.Exec(execCmd, func(err error) tea.Msg {
		if err != nil {
			return loginResultMsg{profileID: profileID, success: false, err: err, isConsoleLogin: true}
		}
		sel := config.NamedProfile(profileID)
		config.Global().SetSelection(sel)
		return loginResultMsg{profileID: profileID, success: true, isConsoleLogin: true}
	})
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
			content += ui.SuccessStyle().Render(loginType + " login successful")
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
