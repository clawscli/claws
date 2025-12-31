package view

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"gopkg.in/ini.v1"

	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/log"
	navmsg "github.com/clawscli/claws/internal/msg"
	"github.com/clawscli/claws/internal/ui"
)

type profileItem struct {
	id      string
	display string
	isSSO   bool
}

func (p profileItem) GetID() string    { return p.id }
func (p profileItem) GetLabel() string { return p.display }
func (p profileItem) IsSSO() bool      { return p.isSSO }

type ProfileSelector struct {
	ctx      context.Context
	selector *MultiSelector[profileItem]
	profiles []profileItem

	ssoResult *ssoResultMsg
	ssoStyle  lipgloss.Style
}

func NewProfileSelector(ctx context.Context) *ProfileSelector {
	initialSelected := make([]string, 0)
	for _, sel := range config.Global().Selections() {
		initialSelected = append(initialSelected, sel.ID())
	}

	p := &ProfileSelector{
		ctx:      ctx,
		selector: NewMultiSelector[profileItem]("Select Profiles", initialSelected),
		ssoStyle: lipgloss.NewStyle().Foreground(ui.Current().Secondary),
	}

	p.selector.SetRenderExtra(func(item profileItem) string {
		if item.isSSO {
			return p.ssoStyle.Render("[SSO]")
		}
		return ""
	})

	return p
}

func (p *ProfileSelector) Init() tea.Cmd {
	return p.loadProfiles
}

type profilesLoadedMsg struct {
	profiles []profileItem
}

type ssoResultMsg struct {
	profileID string
	success   bool
	err       error
}

func (p *ProfileSelector) loadProfiles() tea.Msg {
	profiles := []profileItem{
		{id: config.ProfileIDSDKDefault, display: config.SDKDefault().DisplayName()},
		{id: config.ProfileIDEnvOnly, display: config.EnvOnly().DisplayName()},
	}

	loaded, err := loadProfilesWithSSO()
	if err != nil {
		log.Error("failed to load profiles", "error", err)
	}
	profiles = append(profiles, loaded...)

	return profilesLoadedMsg{profiles: profiles}
}

func loadProfilesWithSSO() ([]profileItem, error) {
	type profileData struct {
		name  string
		isSSO bool
	}
	profileMap := make(map[string]*profileData)

	configPath := os.Getenv("AWS_CONFIG_FILE")
	if configPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		configPath = filepath.Join(homeDir, ".aws", "config")
	}

	cfg, err := ini.Load(configPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Debug("failed to parse aws config", "path", configPath, "error", err)
	}
	if err == nil {
		for _, section := range cfg.Sections() {
			name := section.Name()
			if name == "DEFAULT" {
				continue
			}

			var profileName string
			if after, found := strings.CutPrefix(name, "profile "); found {
				profileName = after
			} else if name == "default" {
				profileName = "default"
			} else {
				continue
			}

			isSSO := section.Key("sso_start_url").String() != "" ||
				section.Key("sso_session").String() != ""

			profileMap[profileName] = &profileData{name: profileName, isSSO: isSSO}
		}
	}

	credPath := os.Getenv("AWS_SHARED_CREDENTIALS_FILE")
	if credPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		credPath = filepath.Join(homeDir, ".aws", "credentials")
	}

	creds, err := ini.Load(credPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Debug("failed to parse aws credentials", "path", credPath, "error", err)
	}
	if err == nil {
		for _, section := range creds.Sections() {
			name := section.Name()
			if name == "DEFAULT" {
				continue
			}
			if _, exists := profileMap[name]; !exists {
				profileMap[name] = &profileData{name: name, isSSO: false}
			}
		}
	}

	names := make([]string, 0, len(profileMap))
	for name := range profileMap {
		names = append(names, name)
	}
	sort.Strings(names)

	items := make([]profileItem, 0, len(names))
	for _, name := range names {
		data := profileMap[name]
		items = append(items, profileItem{
			id:      name,
			display: name,
			isSSO:   data.isSSO,
		})
	}
	return items, nil
}

func (p *ProfileSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case profilesLoadedMsg:
		p.profiles = msg.profiles
		p.selector.SetItems(p.profiles)
		return p, nil

	case ssoResultMsg:
		p.ssoResult = &msg
		if msg.success {
			p.selector.Selected()[msg.profileID] = true
			p.selector.ClearResult()
		}
		p.updateExtraHeight()
		return p, nil

	case consoleLoginResultMsg:
		p.ssoResult = &ssoResultMsg{profileID: msg.profileID, success: msg.success, err: msg.err}
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
				p.ssoResult = nil
				p.updateExtraHeight()
			case "c":
				p.ssoResult = nil
				p.updateExtraHeight()
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
	if p.ssoResult != nil {
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
		p.ssoResult = &ssoResultMsg{
			profileID: profile.id,
			success:   false,
			err:       errors.New("not an SSO profile"),
		}
		p.updateExtraHeight()
		return p, nil
	}

	if config.Global().ReadOnly() && !action.IsExecAllowedInReadOnly(action.ActionNameSSOLogin) {
		p.ssoResult = &ssoResultMsg{
			profileID: profile.id,
			success:   false,
			err:       errors.New("SSO login denied in read-only mode"),
		}
		p.updateExtraHeight()
		return p, nil
	}

	if _, err := exec.LookPath("aws"); err != nil {
		p.ssoResult = &ssoResultMsg{
			profileID: profile.id,
			success:   false,
			err:       errors.New("aws cli not found in PATH"),
		}
		p.updateExtraHeight()
		return p, nil
	}

	profileID := profile.id
	return p, tea.Exec(&ssoLoginCmd{profileName: profileID}, func(err error) tea.Msg {
		if err != nil {
			return ssoResultMsg{profileID: profileID, success: false, err: err}
		}
		return ssoResultMsg{profileID: profileID, success: true}
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

type consoleLoginResultMsg struct {
	profileID string
	success   bool
	err       error
}

func (p *ProfileSelector) consoleLoginCurrentProfile() (tea.Model, tea.Cmd) {
	profile, ok := p.selector.CurrentItem()
	if !ok {
		return p, nil
	}

	if profile.id == config.ProfileIDSDKDefault || profile.id == config.ProfileIDEnvOnly {
		p.ssoResult = &ssoResultMsg{
			profileID: profile.id,
			success:   false,
			err:       errors.New("console login requires a named profile"),
		}
		p.updateExtraHeight()
		return p, nil
	}

	if config.Global().ReadOnly() && !action.IsExecAllowedInReadOnly(action.ActionNameLogin) {
		p.ssoResult = &ssoResultMsg{
			profileID: profile.id,
			success:   false,
			err:       errors.New("console login denied in read-only mode"),
		}
		p.updateExtraHeight()
		return p, nil
	}

	if _, err := exec.LookPath("aws"); err != nil {
		p.ssoResult = &ssoResultMsg{
			profileID: profile.id,
			success:   false,
			err:       errors.New("aws cli not found in PATH"),
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
			return consoleLoginResultMsg{profileID: profileID, success: false, err: err}
		}
		sel := config.NamedProfile(profileID)
		config.Global().SetSelection(sel)
		return consoleLoginResultMsg{profileID: profileID, success: true}
	})
}

func (p *ProfileSelector) ViewString() string {
	content := p.selector.ViewString()

	if p.ssoResult != nil {
		content += "\n"
		if p.ssoResult.success {
			content += ui.SuccessStyle().Render("SSO login successful")
		} else {
			content += ui.DangerStyle().Render("SSO login failed: " + p.ssoResult.err.Error())
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

	return "Space:toggle • Enter:apply" + loginHints + " • " + strings.Repeat("●", count) + " selected"
}

func (p *ProfileSelector) HasActiveInput() bool {
	return p.selector.FilterActive()
}
