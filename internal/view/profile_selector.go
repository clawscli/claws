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

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
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
	ID          string
	DisplayName string
	IsSSO       bool
}

type profileSelectorStyles struct {
	title        lipgloss.Style
	item         lipgloss.Style
	itemSelected lipgloss.Style
	itemChecked  lipgloss.Style
	itemSSO      lipgloss.Style
	filter       lipgloss.Style
}

func newProfileSelectorStyles() profileSelectorStyles {
	t := ui.Current()
	return profileSelectorStyles{
		title:        lipgloss.NewStyle().Background(t.TableHeader).Foreground(t.TableHeaderText).Padding(0, 1),
		item:         lipgloss.NewStyle().PaddingLeft(2),
		itemSelected: lipgloss.NewStyle().PaddingLeft(2).Background(t.Selection).Foreground(t.SelectionText),
		itemChecked:  lipgloss.NewStyle().PaddingLeft(2).Foreground(t.Success),
		itemSSO:      lipgloss.NewStyle().Foreground(t.Secondary),
		filter:       lipgloss.NewStyle().Foreground(t.Accent),
	}
}

type ProfileSelector struct {
	ctx      context.Context
	profiles []profileItem
	cursor   int
	width    int
	height   int

	selected map[string]bool

	viewport viewport.Model
	ready    bool

	filterInput  textinput.Model
	filterActive bool
	filterText   string
	filtered     []profileItem

	styles profileSelectorStyles

	ssoResult *ssoResultMsg
}

func NewProfileSelector(ctx context.Context) *ProfileSelector {
	ti := textinput.New()
	ti.Placeholder = FilterPlaceholder
	ti.Prompt = "/"
	ti.CharLimit = 50

	selected := make(map[string]bool)
	for _, sel := range config.Global().Selections() {
		selected[sel.ID()] = true
	}

	return &ProfileSelector{
		ctx:         ctx,
		selected:    selected,
		filterInput: ti,
		styles:      newProfileSelectorStyles(),
	}
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
		{ID: config.ProfileIDSDKDefault, DisplayName: config.SDKDefault().DisplayName()},
		{ID: config.ProfileIDEnvOnly, DisplayName: config.EnvOnly().DisplayName()},
	}

	loaded, err := loadProfilesWithSSO()
	if err != nil {
		log.Debug("failed to load profiles", "error", err)
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
			if strings.HasPrefix(name, "profile ") {
				profileName = strings.TrimPrefix(name, "profile ")
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
			ID:          name,
			DisplayName: name,
			IsSSO:       data.isSSO,
		})
	}
	return items, nil
}

func (p *ProfileSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case profilesLoadedMsg:
		p.profiles = msg.profiles
		p.applyFilter()
		p.clampCursor()
		for i, profile := range p.filtered {
			if p.selected[profile.ID] {
				p.cursor = i
				break
			}
		}
		p.updateViewport()
		return p, nil

	case ssoResultMsg:
		p.ssoResult = &msg
		if msg.success {
			p.selected[msg.profileID] = true
			p.updateViewport()
		}
		return p, nil

	case consoleLoginResultMsg:
		p.ssoResult = &ssoResultMsg{profileID: msg.profileID, success: msg.success, err: msg.err}
		if msg.success {
			p.selected[msg.profileID] = true
			p.updateViewport()
		}
		return p, nil

	case tea.MouseWheelMsg:
		var cmd tea.Cmd
		p.viewport, cmd = p.viewport.Update(msg)
		return p, cmd

	case tea.MouseMotionMsg:
		if idx := p.getItemAtPosition(msg.Y); idx >= 0 && idx != p.cursor {
			p.cursor = idx
			p.updateViewport()
		}
		return p, nil

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			if idx := p.getItemAtPosition(msg.Y); idx >= 0 {
				p.cursor = idx
				p.toggleCurrent()
				p.updateViewport()
			}
		}
		return p, nil

	case tea.KeyPressMsg:
		if p.filterActive {
			switch msg.String() {
			case "esc":
				p.filterActive = false
				p.filterInput.Blur()
				return p, nil
			case "enter":
				p.filterActive = false
				p.filterInput.Blur()
				p.filterText = p.filterInput.Value()
				p.applyFilter()
				p.clampCursor()
				p.updateViewport()
				return p, nil
			default:
				var cmd tea.Cmd
				p.filterInput, cmd = p.filterInput.Update(msg)
				p.filterText = p.filterInput.Value()
				p.applyFilter()
				p.clampCursor()
				p.updateViewport()
				return p, cmd
			}
		}

		switch msg.String() {
		case "/":
			p.filterActive = true
			p.filterInput.Focus()
			return p, textinput.Blink
		case "c":
			p.filterText = ""
			p.filterInput.SetValue("")
			p.ssoResult = nil
			p.applyFilter()
			p.clampCursor()
			p.updateViewport()
			return p, nil
		case "up", "k":
			if p.cursor > 0 {
				p.cursor--
				p.ssoResult = nil
				p.updateViewport()
			}
			return p, nil
		case "down", "j":
			if p.cursor < len(p.filtered)-1 {
				p.cursor++
				p.ssoResult = nil
				p.updateViewport()
			}
			return p, nil
		case "space":
			p.toggleCurrent()
			p.updateViewport()
			return p, nil
		case "a":
			for _, profile := range p.filtered {
				p.selected[profile.ID] = true
			}
			p.updateViewport()
			return p, nil
		case "n":
			for _, profile := range p.filtered {
				delete(p.selected, profile.ID)
			}
			p.updateViewport()
			return p, nil
		case "enter":
			return p.applySelection()
		case "l":
			return p.ssoLoginCurrentProfile()
		case "L":
			return p.consoleLoginCurrentProfile()
		}
	}

	var cmd tea.Cmd
	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
}

func (p *ProfileSelector) toggleCurrent() {
	if p.cursor >= 0 && p.cursor < len(p.filtered) {
		profile := p.filtered[p.cursor]
		if p.selected[profile.ID] {
			delete(p.selected, profile.ID)
		} else {
			p.selected[profile.ID] = true
		}
	}
}

func (p *ProfileSelector) applySelection() (tea.Model, tea.Cmd) {
	var selections []config.ProfileSelection
	for _, profile := range p.profiles {
		if p.selected[profile.ID] {
			selections = append(selections, config.ProfileSelectionFromID(profile.ID))
		}
	}
	if len(selections) == 0 {
		return p, nil
	}
	config.Global().SetSelections(selections)
	return p, func() tea.Msg {
		return navmsg.ProfilesChangedMsg{Selections: selections}
	}
}

func (p *ProfileSelector) ssoLoginCurrentProfile() (tea.Model, tea.Cmd) {
	if p.cursor < 0 || p.cursor >= len(p.filtered) {
		return p, nil
	}
	profile := p.filtered[p.cursor]
	if !profile.IsSSO {
		p.ssoResult = &ssoResultMsg{
			profileID: profile.ID,
			success:   false,
			err:       errors.New("not an SSO profile"),
		}
		return p, nil
	}

	if config.Global().ReadOnly() && !action.IsExecAllowedInReadOnly(action.ActionNameSSOLogin) {
		p.ssoResult = &ssoResultMsg{
			profileID: profile.ID,
			success:   false,
			err:       errors.New("SSO login denied in read-only mode"),
		}
		return p, nil
	}

	profileID := profile.ID
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
	if p.cursor < 0 || p.cursor >= len(p.filtered) {
		return p, nil
	}
	profile := p.filtered[p.cursor]

	if profile.ID == config.ProfileIDSDKDefault || profile.ID == config.ProfileIDEnvOnly {
		p.ssoResult = &ssoResultMsg{
			profileID: profile.ID,
			success:   false,
			err:       errors.New("console login requires a named profile"),
		}
		return p, nil
	}

	if config.Global().ReadOnly() && !action.IsExecAllowedInReadOnly(action.ActionNameLogin) {
		p.ssoResult = &ssoResultMsg{
			profileID: profile.ID,
			success:   false,
			err:       errors.New("console login denied in read-only mode"),
		}
		return p, nil
	}

	profileID := profile.ID
	exec := &action.SimpleExec{
		Command:    "aws login --remote --profile " + profileID,
		ActionName: action.ActionNameLogin,
		SkipAWSEnv: true,
	}
	return p, tea.Exec(exec, func(err error) tea.Msg {
		if err != nil {
			return consoleLoginResultMsg{profileID: profileID, success: false, err: err}
		}
		sel := config.NamedProfile(profileID)
		config.Global().SetSelection(sel)
		return consoleLoginResultMsg{profileID: profileID, success: true}
	})
}

func (p *ProfileSelector) applyFilter() {
	if p.filterText == "" {
		p.filtered = p.profiles
		return
	}

	filter := strings.ToLower(p.filterText)
	p.filtered = nil
	for _, profile := range p.profiles {
		if strings.Contains(strings.ToLower(profile.DisplayName), filter) {
			p.filtered = append(p.filtered, profile)
		}
	}
}

func (p *ProfileSelector) clampCursor() {
	if len(p.filtered) == 0 {
		p.cursor = -1
	} else if p.cursor >= len(p.filtered) {
		p.cursor = len(p.filtered) - 1
	} else if p.cursor < 0 {
		p.cursor = 0
	}
}

func (p *ProfileSelector) updateViewport() {
	if !p.ready {
		return
	}
	p.viewport.SetContent(p.renderContent())

	if p.cursor >= 0 {
		viewportHeight := p.viewport.Height()
		if viewportHeight > 0 {
			if p.cursor < p.viewport.YOffset() {
				p.viewport.SetYOffset(p.cursor)
			} else if p.cursor >= p.viewport.YOffset()+viewportHeight {
				p.viewport.SetYOffset(p.cursor - viewportHeight + 1)
			}
		}
	}
}

func (p *ProfileSelector) renderContent() string {
	var b strings.Builder

	for i, profile := range p.filtered {
		style := p.styles.item
		isChecked := p.selected[profile.ID]

		if i == p.cursor {
			style = p.styles.itemSelected
		} else if isChecked {
			style = p.styles.itemChecked
		}

		checkbox := "☐ "
		if isChecked {
			checkbox = "☑ "
		}

		line := checkbox + profile.DisplayName
		if profile.IsSSO {
			line += " " + p.styles.itemSSO.Render("[SSO]")
		}

		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	return b.String()
}

func (p *ProfileSelector) getItemAtPosition(y int) int {
	if !p.ready {
		return -1
	}
	headerHeight := 1
	if p.filterActive || p.filterText != "" {
		headerHeight++
	}

	contentY := y - headerHeight + p.viewport.YOffset()
	if contentY >= 0 && contentY < len(p.filtered) {
		return contentY
	}
	return -1
}

func (p *ProfileSelector) ViewString() string {
	s := p.styles

	title := s.title.Render("Select Profiles")

	var filterView string
	if p.filterActive {
		filterView = p.styles.filter.Render(p.filterInput.View()) + "\n"
	} else if p.filterText != "" {
		filterView = p.styles.filter.Render("filter: "+p.filterText) + "\n"
	}

	if !p.ready {
		return title + "\n" + filterView + "Loading..."
	}

	content := title + "\n" + filterView + p.viewport.View()

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
	p.width = width
	p.height = height

	viewportHeight := height - 2
	if p.filterActive || p.filterText != "" {
		viewportHeight--
	}
	if p.ssoResult != nil {
		viewportHeight--
	}

	if !p.ready {
		p.viewport = viewport.New(viewport.WithWidth(width), viewport.WithHeight(viewportHeight))
		p.ready = true
	} else {
		p.viewport.SetWidth(width)
		p.viewport.SetHeight(viewportHeight)
	}
	p.updateViewport()
	return nil
}

func (p *ProfileSelector) StatusLine() string {
	count := len(p.selected)
	if p.filterActive {
		return "Type to filter • Enter confirm • Esc cancel"
	}

	var loginHints string
	if p.cursor >= 0 && p.cursor < len(p.filtered) {
		profile := p.filtered[p.cursor]
		if profile.IsSSO {
			loginHints = " • l:SSO"
		}
		if profile.ID != config.ProfileIDSDKDefault && profile.ID != config.ProfileIDEnvOnly {
			loginHints += " • L:console"
		}
	}

	return "Space:toggle • Enter:apply" + loginHints + " • " + strings.Repeat("●", count) + " selected"
}

func (p *ProfileSelector) HasActiveInput() bool {
	return p.filterActive
}
