package app

import (
	"context"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/clipboard"
	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/log"
	navmsg "github.com/clawscli/claws/internal/msg"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/ui"
	"github.com/clawscli/claws/internal/view"
)

type clearErrorMsg struct{}

type clearFlashMsg struct{}

const flashDuration = 2 * time.Second

// awsContextReadyMsg is sent when AWS context initialization completes
type awsContextReadyMsg struct {
	err error
}

// profileRefreshDoneMsg is sent when async profile refresh completes
type profileRefreshDoneMsg struct {
	refreshID  uint64
	region     string
	accountIDs map[string]string
	err        error
}

// App is the main application model
// appStyles holds cached lipgloss styles for performance
type appStyles struct {
	status       lipgloss.Style
	readOnly     lipgloss.Style
	warningTitle lipgloss.Style
	warningItem  lipgloss.Style
	warningDim   lipgloss.Style
	warningBox   lipgloss.Style
}

func newAppStyles(width int) appStyles {
	t := ui.Current()
	return appStyles{
		status:       ui.TableHeaderStyle().Padding(0, 1).Width(width),
		readOnly:     lipgloss.NewStyle().Background(t.Warning).Foreground(lipgloss.Color("#000000")).Bold(true).Padding(0, 1),
		warningTitle: ui.BoldPendingStyle().MarginBottom(1),
		warningItem:  ui.WarningStyle(),
		warningDim:   ui.DimStyle().MarginTop(1),
		warningBox:   ui.BoxStyle().BorderForeground(t.Pending).Padding(1, 2),
	}
}

type App struct {
	ctx      context.Context
	registry *registry.Registry
	width    int
	height   int

	currentView view.View
	viewStack   []view.View

	commandInput *view.CommandInput
	commandMode  bool

	help help.Model
	keys keyMap

	err error

	showWarnings  bool
	warningsReady bool

	awsInitializing     bool
	profileRefreshID    uint64
	profileRefreshing   bool
	profileRefreshError error

	modal         *view.Modal
	modalStack    []*view.Modal
	modalRenderer *view.ModalRenderer

	clipboardFlash   string
	clipboardWarning bool

	styles appStyles
}

func New(ctx context.Context, reg *registry.Registry) *App {
	return &App{
		ctx:           ctx,
		registry:      reg,
		commandInput:  view.NewCommandInput(ctx, reg),
		help:          help.New(),
		keys:          defaultKeyMap(),
		modalRenderer: view.NewModalRenderer(),
		styles:        newAppStyles(0),
	}
}

// Init implements tea.Model
func (a *App) Init() tea.Cmd {
	// Start with the dashboard view immediately (no blocking on AWS calls)
	a.currentView = view.NewDashboardView(a.ctx, a.registry)
	a.awsInitializing = true

	// Initialize AWS context in background (region detection, account ID fetch)
	// Use timeout to avoid indefinite hang on network issues
	initAWSCmd := func() tea.Msg {
		ctx, cancel := context.WithTimeout(a.ctx, config.File().AWSInitTimeout())
		defer cancel()
		err := aws.InitContext(ctx)
		return awsContextReadyMsg{err: err}
	}

	return tea.Batch(a.currentView.Init(), initAWSCmd)
}

// Update implements tea.Model
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if a.showWarnings && a.warningsReady {
		if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
			if keyMsg.Code == tea.KeyEnter || keyMsg.String() == "space" || keyMsg.String() == "q" {
				a.showWarnings = false
				return a, nil
			}
			return a, nil
		}
	}

	if a.modal != nil {
		return a.handleModalUpdate(msg)
	}

	// Handle command mode first
	if a.commandMode {
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
			cmd, nav := a.commandInput.Update(msg)
			if !a.commandInput.IsActive() {
				a.commandMode = false
			}
			if nav != nil {
				// Navigate to the command result
				if nav.ClearStack {
					// Go home - clear the stack
					a.viewStack = nil
				} else if a.currentView != nil {
					a.viewStack = append(a.viewStack, a.currentView)
				}
				a.currentView = nav.View
				cmds := []tea.Cmd{
					cmd,
					a.currentView.Init(),
					a.currentView.SetSize(a.width, a.height-2),
				}
				return a, tea.Batch(cmds...)
			}
			return a, cmd
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.help.SetWidth(msg.Width)
		a.commandInput.SetWidth(msg.Width)
		// Update cached styles with new width
		a.styles = newAppStyles(msg.Width)
		// Mark warnings ready after first WindowSizeMsg (terminal initialized).
		// Safe to set unconditionally - only affects dismissal when showWarnings is true.
		a.warningsReady = true
		if a.currentView != nil {
			return a, a.currentView.SetSize(msg.Width, msg.Height-2)
		}
		return a, nil

	case tea.MouseClickMsg:
		// Mouse back button navigates back (same as esc/backspace)
		if msg.Button == tea.MouseBackward && len(a.viewStack) > 0 {
			a.currentView = a.viewStack[len(a.viewStack)-1]
			a.viewStack = a.viewStack[:len(a.viewStack)-1]
			return a, a.currentView.SetSize(a.width, a.height-2)
		}

	case tea.KeyPressMsg:
		// Handle back navigation (esc or backspace)
		isBack := view.IsEscKey(msg) || msg.Code == tea.KeyBackspace

		if isBack {
			// If current view has active input, let it handle esc first
			if ic, ok := a.currentView.(view.InputCapture); ok && ic.HasActiveInput() {
				model, cmd := a.currentView.Update(msg)
				if v, ok := model.(view.View); ok {
					a.currentView = v
				}
				return a, cmd
			}
			// Otherwise, go back
			if len(a.viewStack) > 0 {
				a.currentView = a.viewStack[len(a.viewStack)-1]
				a.viewStack = a.viewStack[:len(a.viewStack)-1]
				return a, a.currentView.SetSize(a.width, a.height-2)
			}
			return a, nil
		}

		switch {
		case key.Matches(msg, a.keys.Quit):
			switch a.currentView.(type) {
			case *view.DetailView, *view.DiffView, *view.LogView:
				if len(a.viewStack) > 0 {
					a.currentView = a.viewStack[len(a.viewStack)-1]
					a.viewStack = a.viewStack[:len(a.viewStack)-1]
					return a, a.currentView.SetSize(a.width, a.height-2)
				}
			}
			return a, tea.Quit

		case key.Matches(msg, a.keys.Help):
			helpView := view.NewHelpView()
			a.modal = &view.Modal{Content: helpView, Width: view.ModalWidthHelp}
			return a, a.modal.SetSize(a.width, a.height)

		case key.Matches(msg, a.keys.Command):
			a.commandMode = true
			// Set completion providers if current view is a ResourceBrowser
			if rb, ok := a.currentView.(*view.ResourceBrowser); ok {
				a.commandInput.SetTagProvider(rb)
				a.commandInput.SetDiffProvider(rb)
			} else {
				a.commandInput.SetTagProvider(nil)
				a.commandInput.SetDiffProvider(nil)
			}
			return a, a.commandInput.Activate()

		case key.Matches(msg, a.keys.Region):
			regionSelector := view.NewRegionSelector(a.ctx)
			a.modal = &view.Modal{Content: regionSelector, Width: view.ModalWidthRegion}
			return a, tea.Batch(
				regionSelector.Init(),
				a.modal.SetSize(a.width, a.height),
			)

		case key.Matches(msg, a.keys.Profile):
			profileSelector := view.NewProfileSelector()
			a.modal = &view.Modal{Content: profileSelector, Width: view.ModalWidthProfile}
			return a, tea.Batch(
				profileSelector.Init(),
				a.modal.SetSize(a.width, a.height),
			)
		}

	case view.ShowModalMsg:
		return a.showModal(msg.Modal)

	case view.NavigateMsg:
		return a.handleNavigate(msg)

	case view.ErrorMsg:
		log.Error("application error", "error", msg.Err)
		a.err = msg.Err
		// Auto-clear transient errors after 3 seconds
		return a, tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
			return clearErrorMsg{}
		})

	case clearErrorMsg:
		a.err = nil
		return a, nil

	case clipboard.CopiedMsg:
		a.clipboardFlash = "Copied " + msg.Label
		a.clipboardWarning = false
		return a, tea.Tick(flashDuration, func(t time.Time) tea.Msg {
			return clearFlashMsg{}
		})

	case clipboard.NoARNMsg:
		a.clipboardFlash = "No ARN available"
		a.clipboardWarning = true
		return a, tea.Tick(flashDuration, func(t time.Time) tea.Msg {
			return clearFlashMsg{}
		})

	case clearFlashMsg:
		a.clipboardFlash = ""
		return a, nil

	case awsContextReadyMsg:
		a.awsInitializing = false
		if msg.err != nil {
			errStr := msg.err.Error()
			// IMDS errors are expected on non-EC2 environments - log only, no warning
			if strings.Contains(errStr, "ec2imds") {
				log.Debug("IMDS region detection failed (expected on non-EC2)", "error", msg.err)
			} else {
				log.Debug("AWS context initialization failed", "error", msg.err)
				config.Global().AddWarning("AWS init failed: " + errStr)
				a.showWarnings = true
			}
		}
		return a, nil

	case profileRefreshDoneMsg:
		if msg.refreshID != a.profileRefreshID {
			log.Debug("ignoring stale profile refresh", "got", msg.refreshID, "want", a.profileRefreshID)
			return a, nil
		}
		a.profileRefreshing = false
		a.profileRefreshError = msg.err
		if msg.err != nil {
			log.Warn("profile refresh failed", "error", msg.err)
			return a, nil
		}
		if msg.region != "" {
			config.Global().AddRegion(msg.region)
		}
		if len(msg.accountIDs) > 0 {
			for profileID, accountID := range msg.accountIDs {
				config.Global().SetAccountIDForProfile(profileID, accountID)
			}
		}
		return a, nil

	case navmsg.RegionChangedMsg:
		return a.handleRegionChanged(msg)

	case navmsg.ProfilesChangedMsg:
		return a.handleProfilesChanged(msg)

	case view.SortMsg:
		// Delegate sort command to current view
		if a.currentView != nil {
			model, cmd := a.currentView.Update(msg)
			if v, ok := model.(view.View); ok {
				a.currentView = v
			}
			return a, cmd
		}
		return a, nil
	}

	// Delegate to current view
	if a.currentView != nil {
		model, cmd := a.currentView.Update(msg)
		if v, ok := model.(view.View); ok {
			a.currentView = v
		}
		return a, cmd
	}

	return a, nil
}

// newAltScreenView creates a View with AltScreen and mouse support enabled
func newAltScreenView(content string) tea.View {
	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeAllMotion // AllMotion for hover tracking
	return v
}

func (a *App) View() tea.View {
	if a.showWarnings {
		return newAltScreenView(a.renderWarnings())
	}

	var content string
	if a.currentView != nil {
		content = a.currentView.ViewString()
	}

	if a.commandMode {
		cmdView := a.commandInput.View()
		return newAltScreenView(content + "\n" + cmdView)
	}

	var statusContent string
	if a.err != nil {
		statusContent = ui.DangerStyle().Render("Error: " + a.err.Error())
	} else if a.clipboardFlash != "" {
		if a.clipboardWarning {
			statusContent = ui.WarningStyle().Render("⚠ " + a.clipboardFlash)
		} else {
			statusContent = ui.SuccessStyle().Render("✓ " + a.clipboardFlash)
		}
	} else if a.currentView != nil {
		statusContent = a.currentView.StatusLine()
	}

	if config.Global().ReadOnly() {
		roIndicator := a.styles.readOnly.Render("READ-ONLY")
		statusContent = roIndicator + " " + statusContent
	}

	if a.awsInitializing {
		statusContent = ui.DimStyle().Render("AWS initializing...") + " • " + statusContent
	}

	if a.profileRefreshError != nil {
		statusContent = ui.WarningStyle().Render("⚠ Profile error") + " • " + statusContent
	} else if a.profileRefreshing {
		statusContent = ui.DimStyle().Render("Refreshing profile...") + " • " + statusContent
	}

	status := a.styles.status.Render(statusContent)
	mainView := content + "\n" + status

	if a.modal != nil {
		return newAltScreenView(a.modalRenderer.Render(a.modal, mainView, a.width, a.height))
	}

	return newAltScreenView(mainView)
}

// renderWarnings renders the startup warnings modal
func (a *App) renderWarnings() string {
	warnings := config.Global().Warnings()
	s := a.styles

	var content string
	content += s.warningTitle.Render("⚠ Startup Warnings") + "\n\n"

	for _, w := range warnings {
		content += s.warningItem.Render("• "+w) + "\n"
	}

	content += "\n" + s.warningDim.Render("Press Enter, Space, or q to continue...")

	boxStyle := s.warningBox.Width(a.width - 10)
	box := boxStyle.Render(content)

	// Center the box
	return lipgloss.Place(
		a.width,
		a.height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}

func (a *App) handleModalUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case view.HideModalMsg:
		return a.popModal()

	case view.ShowModalMsg:
		return a.showModal(msg.Modal)

	case view.NavigateMsg:
		a.clearModalState()
		return a.handleNavigate(msg)

	case navmsg.RegionChangedMsg:
		a.clearModalState()
		return a.handleRegionChanged(msg)

	case navmsg.ProfilesChangedMsg:
		a.clearModalState()
		return a.handleProfilesChanged(msg)

	case tea.KeyPressMsg:
		if view.IsEscKey(msg) || msg.Code == tea.KeyBackspace || msg.String() == "q" {
			if ic, ok := a.modal.Content.(view.InputCapture); ok && ic.HasActiveInput() {
				break
			}
			return a.popModal()
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.styles = newAppStyles(msg.Width)
		var viewCmd tea.Cmd
		if a.currentView != nil {
			viewCmd = a.currentView.SetSize(msg.Width, msg.Height-2)
		}
		modalCmd := a.modal.SetSize(msg.Width, msg.Height)
		return a, tea.Batch(viewCmd, modalCmd)
	}

	modal, cmd := a.modal.Update(msg)
	a.modal = modal
	return a, cmd
}

func (a *App) popModal() (tea.Model, tea.Cmd) {
	if len(a.modalStack) > 0 {
		a.modal = a.modalStack[len(a.modalStack)-1]
		a.modalStack = a.modalStack[:len(a.modalStack)-1]
		return a, a.modal.SetSize(a.width, a.height)
	}
	a.modal = nil
	return a, nil
}

func (a *App) clearModalState() {
	a.modal = nil
	a.modalStack = nil
}

func (a *App) showModal(modal *view.Modal) (tea.Model, tea.Cmd) {
	if a.modal != nil {
		a.modalStack = append(a.modalStack, a.modal)
	}
	a.modal = modal
	return a, a.modal.SetSize(a.width, a.height)
}

func (a *App) handleNavigate(msg view.NavigateMsg) (tea.Model, tea.Cmd) {
	log.Debug("navigating", "clearStack", msg.ClearStack, "stackDepth", len(a.viewStack))
	if msg.ClearStack {
		a.viewStack = nil
	} else if a.currentView != nil {
		a.viewStack = append(a.viewStack, a.currentView)
	}
	a.currentView = msg.View
	return a, tea.Batch(
		a.currentView.Init(),
		a.currentView.SetSize(a.width, a.height-2),
	)
}

func (a *App) handleRegionChanged(msg navmsg.RegionChangedMsg) (tea.Model, tea.Cmd) {
	log.Info("regions changed", "regions", msg.Regions)
	if config.File().PersistenceEnabled() {
		_, existingProfile := config.File().GetStartup()
		config.File().SetStartup(msg.Regions, existingProfile)
		if err := config.File().Save(); err != nil {
			log.Warn("failed to persist config", "error", err)
		}
	}
	return a.popToRefreshableView()
}

func (a *App) handleProfilesChanged(msg navmsg.ProfilesChangedMsg) (tea.Model, tea.Cmd) {
	log.Info("profiles changed", "count", len(msg.Selections))
	if config.File().PersistenceEnabled() {
		profile := ""
		if len(msg.Selections) > 0 && msg.Selections[0].IsNamedProfile() {
			profile = msg.Selections[0].ProfileName
		}
		existingRegions := config.Global().Regions()
		config.File().SetStartup(existingRegions, profile)
		if err := config.File().Save(); err != nil {
			log.Warn("failed to persist config", "error", err)
		}
	}
	a.profileRefreshID++
	a.profileRefreshing = true
	a.profileRefreshError = nil
	refreshID := a.profileRefreshID
	refreshCmd := func() tea.Msg {
		ctx, cancel := context.WithTimeout(a.ctx, config.File().AWSInitTimeout())
		defer cancel()
		region, accountIDs, err := aws.RefreshContextData(ctx)
		return profileRefreshDoneMsg{
			refreshID:  refreshID,
			region:     region,
			accountIDs: accountIDs,
			err:        err,
		}
	}

	cmds := []tea.Cmd{refreshCmd}

	for len(a.viewStack) > 0 {
		a.currentView = a.viewStack[len(a.viewStack)-1]
		a.viewStack = a.viewStack[:len(a.viewStack)-1]

		if r, ok := a.currentView.(view.Refreshable); ok && r.CanRefresh() {
			cmds = append(cmds,
				a.currentView.SetSize(a.width, a.height-2),
				func() tea.Msg { return view.RefreshMsg{} },
			)
			return a, tea.Batch(cmds...)
		}
	}
	a.currentView = view.NewDashboardView(a.ctx, a.registry)
	cmds = append(cmds,
		a.currentView.Init(),
		a.currentView.SetSize(a.width, a.height-2),
	)
	return a, tea.Batch(cmds...)
}

func (a *App) popToRefreshableView() (tea.Model, tea.Cmd) {
	for len(a.viewStack) > 0 {
		a.currentView = a.viewStack[len(a.viewStack)-1]
		a.viewStack = a.viewStack[:len(a.viewStack)-1]
		if r, ok := a.currentView.(view.Refreshable); ok && r.CanRefresh() {
			return a, tea.Batch(
				a.currentView.SetSize(a.width, a.height-2),
				func() tea.Msg { return view.RefreshMsg{} },
			)
		}
	}
	a.currentView = view.NewDashboardView(a.ctx, a.registry)
	return a, tea.Batch(
		a.currentView.Init(),
		a.currentView.SetSize(a.width, a.height-2),
	)
}

type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Enter   key.Binding
	Back    key.Binding
	Filter  key.Binding
	Command key.Binding
	Region  key.Binding
	Profile key.Binding
	Help    key.Binding
	Quit    key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		Command: key.NewBinding(
			key.WithKeys(":"),
			key.WithHelp(":", "command"),
		),
		Region: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "region"),
		),
		Profile: key.NewBinding(
			key.WithKeys("P"),
			key.WithHelp("P", "profile"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// ShortHelp returns short help
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Command, k.Help, k.Quit}
}

// FullHelp returns full help
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter, k.Back},
		{k.Filter, k.Command, k.Help, k.Quit},
	}
}
