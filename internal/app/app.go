package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/clawscli/claws/internal/ai"
	"github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/clipboard"
	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
	"github.com/clawscli/claws/internal/log"
	navmsg "github.com/clawscli/claws/internal/msg"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/ui"
	"github.com/clawscli/claws/internal/view"
)

type clearErrorMsg struct{}

type clearFlashMsg struct{}

// StartupPath specifies the initial view to show when the app starts.
type StartupPath struct {
	Service      string
	ResourceType string
	ResourceID   string
}

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

type startupResourceMsg struct {
	resource dao.Resource
	err      error
}

type noOpMsg struct{}

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
		readOnly:     ui.ReadOnlyBadgeStyle(),
		warningTitle: ui.BoldPendingStyle().MarginBottom(1),
		warningItem:  ui.WarningStyle(),
		warningDim:   ui.DimStyle().MarginTop(1),
		warningBox:   ui.BoxStyle().BorderForeground(t.Pending).Padding(1, 2),
	}
}

type App struct {
	ctx         context.Context
	registry    *registry.Registry
	startupPath *StartupPath
	width       int
	height      int

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

func New(ctx context.Context, reg *registry.Registry, startupPath *StartupPath) *App {
	return &App{
		ctx:           ctx,
		registry:      reg,
		startupPath:   startupPath,
		commandInput:  view.NewCommandInput(ctx, reg),
		help:          help.New(),
		keys:          defaultKeyMap(),
		modalRenderer: view.NewModalRenderer(),
		styles:        newAppStyles(0),
	}
}

// Init implements tea.Model
func (a *App) Init() tea.Cmd {
	a.awsInitializing = true

	if a.startupPath != nil {
		// CLI `-s` option takes precedence
		viewName := a.startupPath.Service
		if a.startupPath.ResourceType != "" {
			viewName = fmt.Sprintf("%s/%s", a.startupPath.Service, a.startupPath.ResourceType)
		}
		a.currentView = a.resolveStartupView(viewName)
	} else {
		// Check config startup.view
		startupView := config.File().GetStartupView()
		a.currentView = a.resolveStartupView(startupView)
	}

	initAWSCmd := func() tea.Msg {
		ctx, cancel := context.WithTimeout(a.ctx, config.File().AWSInitTimeout())
		defer cancel()
		err := aws.InitContext(ctx)
		return awsContextReadyMsg{err: err}
	}

	cmds := []tea.Cmd{a.currentView.Init(), initAWSCmd}

	if a.startupPath != nil && a.startupPath.ResourceID != "" {
		cmds = append(cmds, a.fetchStartupResource)
	}

	return tea.Batch(cmds...)
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
				a.pushOrClearStack(nav.ClearStack)
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
		// Update cached styles with new width
		a.styles = newAppStyles(msg.Width)
		// Mark warnings ready after first WindowSizeMsg (terminal initialized).
		// Safe to set unconditionally - only affects dismissal when showWarnings is true.
		a.warningsReady = true
		if a.currentView != nil {
			return a, a.currentView.SetSize(msg.Width, msg.Height-2)
		}
		return a, nil

	case view.ThemeChangedMsg:
		a.styles = newAppStyles(a.width)
		a.modalRenderer.ReloadStyles()
		a.commandInput.ReloadStyles()
		if a.currentView != nil {
			a.currentView.Update(msg)
		}
		for _, v := range a.viewStack {
			v.Update(msg)
		}
		return a, nil

	case view.CompactHeaderChangedMsg:
		if a.currentView != nil {
			a.currentView.Update(msg)
		}
		for _, v := range a.viewStack {
			v.Update(msg)
		}
		return a, nil

	case view.ThemeChangeMsg:
		theme := ui.GetPreset(msg.Name)
		if theme == nil {
			a.err = fmt.Errorf("unknown theme: %s (available: %v)", msg.Name, ui.AvailableThemes())
			return a, nil
		}
		ui.SetTheme(theme)
		a.clipboardFlash = "Theme: " + msg.Name
		a.clipboardWarning = false
		if config.File().PersistenceEnabled() {
			if err := config.File().SaveTheme(msg.Name); err != nil {
				log.Warn("failed to persist theme", "error", err)
				a.clipboardFlash = "Theme: " + msg.Name + " (save failed)"
				a.clipboardWarning = true
			}
		}
		return a, tea.Batch(
			func() tea.Msg { return view.ThemeChangedMsg{} },
			tea.Tick(flashDuration, func(t time.Time) tea.Msg { return clearFlashMsg{} }),
		)

	case view.PersistenceChangeMsg:
		if err := config.File().SavePersistence(msg.Enabled); err != nil {
			a.err = fmt.Errorf("failed to save autosave setting: %w", err)
			return a, nil
		}
		if msg.Enabled {
			a.clipboardFlash = "Autosave enabled"
		} else {
			a.clipboardFlash = "Autosave disabled"
		}
		a.clipboardWarning = false
		return a, tea.Tick(flashDuration, func(t time.Time) tea.Msg {
			return clearFlashMsg{}
		})

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseBackward {
			if cmd := a.navigateBack(); cmd != nil {
				return a, cmd
			}
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
			if cmd := a.navigateBack(); cmd != nil {
				return a, cmd
			}
			return a, nil
		}

		switch {
		case key.Matches(msg, a.keys.Quit):
			switch a.currentView.(type) {
			case *view.DetailView, *view.DiffView, *view.LogView:
				if cmd := a.navigateBack(); cmd != nil {
					return a, cmd
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

		case key.Matches(msg, a.keys.AI):
			aiCtx := a.buildAIContext()
			chatOverlay := view.NewChatOverlay(a.ctx, a.registry, aiCtx)
			a.modal = &view.Modal{Content: chatOverlay, Width: view.ModalWidthChat}
			return a, tea.Batch(
				chatOverlay.Init(),
				a.modal.SetSize(a.width, a.height),
			)

		case key.Matches(msg, a.keys.CompactHeader):
			compact := !config.Global().CompactHeader()
			config.Global().SetCompactHeader(compact)
			if config.File().PersistenceEnabled() {
				if err := config.File().SaveCompactHeader(compact); err != nil {
					log.Warn("failed to persist compact header", "error", err)
				}
			}
			return a, func() tea.Msg { return view.CompactHeaderChangedMsg{} }
		}

	case view.ShowModalMsg:
		return a.showModal(msg.Modal)

	case view.NavigateMsg:
		return a.handleNavigate(msg)

	case view.ClearHistoryMsg:
		log.Debug("clearing navigation history", "stackDepth", len(a.viewStack))
		a.viewStack = nil
		return a, nil

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
		if config.File().PersistenceEnabled() {
			if err := config.File().SaveRegions(config.Global().Regions()); err != nil {
				log.Warn("failed to persist regions", "error", err)
			}
		}
		if len(msg.accountIDs) > 0 {
			for profileID, accountID := range msg.accountIDs {
				config.Global().SetAccountIDForProfile(profileID, accountID)
			}
		}
		return a, nil

	case startupResourceMsg:
		if a.startupPath == nil {
			return a, nil
		}
		if msg.err != nil || msg.resource == nil {
			if msg.err != nil {
				log.Warn("startup resource fetch failed", "error", msg.err, "id", a.startupPath.ResourceID)
			}
			a.clipboardFlash = "Resource not found: " + a.startupPath.ResourceID
			a.clipboardWarning = true
			return a, tea.Tick(flashDuration, func(t time.Time) tea.Msg {
				return clearFlashMsg{}
			})
		}
		renderer, err := a.registry.GetRenderer(a.startupPath.Service, a.startupPath.ResourceType)
		if err != nil {
			log.Warn("failed to get renderer for startup resource", "error", err)
			return a, nil
		}
		// DAO is optional - DetailView handles nil gracefully (just disables refresh).
		// Unlike renderer which is required for display, DAO only enables refresh functionality.
		d, err := a.registry.GetDAO(a.ctx, a.startupPath.Service, a.startupPath.ResourceType)
		if err != nil {
			log.Warn("failed to get DAO for startup resource", "error", err)
		}
		detailView := view.NewDetailView(a.ctx, msg.resource, renderer, a.startupPath.Service, a.startupPath.ResourceType, a.registry, d)
		a.viewStack = append(a.viewStack, a.currentView)
		a.currentView = detailView
		return a, tea.Batch(detailView.Init(), detailView.SetSize(a.width, a.height-2))

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

	var statusContent string
	if a.commandMode {
		statusContent = a.commandInput.View() + ui.DimStyle().Render(" • Esc:cancel Enter:run Tab:complete")
	} else {
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
	}

	status := a.styles.status.Render(statusContent)

	// Fix content height to keep status line at bottom regardless of content size.
	contentHeight := a.height - 1
	if contentHeight < 1 {
		contentHeight = 1
	}
	paddedContent := ui.NoStyle().Height(contentHeight).Render(content)
	mainView := paddedContent + "\n" + status

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
		if view.IsEscKey(msg) || msg.Code == tea.KeyBackspace || msg.String() == "q" || msg.String() == "ctrl+c" {
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
	a.pushOrClearStack(msg.ClearStack)
	a.currentView = msg.View
	return a, tea.Batch(
		a.currentView.Init(),
		a.currentView.SetSize(a.width, a.height-2),
	)
}

// popView pops the top view from the view stack.
// Returns nil if the stack is empty.
func (a *App) popView() view.View {
	if len(a.viewStack) == 0 {
		return nil
	}
	v := a.viewStack[len(a.viewStack)-1]
	a.viewStack = a.viewStack[:len(a.viewStack)-1]
	return v
}

// navigateBack pops from the view stack and sets it as the current view.
// Calls Init() to ensure the view is properly reinitialized (important for stateful views).
// Returns nil if the stack is empty (no-op).
func (a *App) navigateBack() tea.Cmd {
	v := a.popView()
	if v == nil {
		return nil
	}
	a.currentView = v
	log.Debug("navigating back", "view", a.currentView.StatusLine(), "stackDepth", len(a.viewStack))
	return tea.Batch(
		a.currentView.Init(),
		a.currentView.SetSize(a.width, a.height-2),
	)
}

// pushOrClearStack either clears the view stack (for home navigation) or
// pushes the current view onto the stack (for drill-down navigation).
// Enforces max stack size from config.
func (a *App) pushOrClearStack(clearStack bool) {
	if clearStack {
		a.viewStack = nil
	} else if a.currentView != nil {
		a.viewStack = append(a.viewStack, a.currentView)

		// Enforce max stack size
		maxSize := config.File().MaxStackSize()
		if len(a.viewStack) > maxSize {
			// Remove oldest entries
			a.viewStack = a.viewStack[len(a.viewStack)-maxSize:]
		}
	}
}

func (a *App) fetchStartupResource() tea.Msg {
	if a.startupPath == nil || a.startupPath.ResourceID == "" {
		return noOpMsg{}
	}

	d, err := a.registry.GetDAO(a.ctx, a.startupPath.Service, a.startupPath.ResourceType)
	if err != nil {
		return startupResourceMsg{err: apperrors.Wrap(err, "get DAO for startup resource")}
	}

	resource, err := d.Get(a.ctx, a.startupPath.ResourceID)
	return startupResourceMsg{resource: resource, err: apperrors.Wrap(err, "fetch startup resource")}
}

func (a *App) handleRegionChanged(msg navmsg.RegionChangedMsg) (tea.Model, tea.Cmd) {
	log.Info("regions changed", "regions", msg.Regions)
	if config.File().PersistenceEnabled() {
		if err := config.File().SaveRegions(msg.Regions); err != nil {
			log.Warn("failed to persist regions", "error", err)
		}
	}
	return a.refreshCurrentView()
}

func (a *App) handleProfilesChanged(msg navmsg.ProfilesChangedMsg) (tea.Model, tea.Cmd) {
	log.Info("profiles changed", "count", len(msg.Selections))
	if config.File().PersistenceEnabled() {
		profileIDs := make([]string, len(msg.Selections))
		for i, sel := range msg.Selections {
			profileIDs[i] = sel.ID()
		}
		if err := config.File().SaveProfiles(profileIDs); err != nil {
			log.Warn("failed to persist profiles", "error", err)
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

	_, viewCmd := a.refreshCurrentView()
	return a, tea.Batch(refreshCmd, viewCmd)
}

// refreshCurrentView triggers a refresh on the current view if it's refreshable.
// Unlike the previous popToRefreshableView(), this stays on the current view instead of
// popping the stack to find a refreshable ancestor. This provides better UX by keeping
// the user's context (e.g., staying on ResourceBrowser after profile/region change).
func (a *App) refreshCurrentView() (tea.Model, tea.Cmd) {
	if a.currentView == nil {
		return a, nil
	}
	cmds := []tea.Cmd{a.currentView.SetSize(a.width, a.height-2)}
	r, canRefresh := a.currentView.(view.Refreshable)
	if canRefresh && r.CanRefresh() {
		cmds = append(cmds, func() tea.Msg { return view.RefreshMsg{} })
	}
	log.Debug("refreshing current view", "view", a.currentView.StatusLine(), "refreshable", canRefresh && r.CanRefresh())
	return a, tea.Batch(cmds...)
}

type keyMap struct {
	Up            key.Binding
	Down          key.Binding
	Enter         key.Binding
	Back          key.Binding
	Filter        key.Binding
	Command       key.Binding
	Region        key.Binding
	Profile       key.Binding
	AI            key.Binding
	CompactHeader key.Binding
	Help          key.Binding
	Quit          key.Binding
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
		AI: key.NewBinding(
			key.WithKeys("A"),
			key.WithHelp("A", "ai chat"),
		),
		CompactHeader: key.NewBinding(
			key.WithKeys("ctrl+e"),
			key.WithHelp("ctrl+e", "compact header"),
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

// resolveStartupView resolves the startup view from config.
// Returns ServiceBrowser by default if config is empty or invalid.
func (a *App) resolveStartupView(viewName string) view.View {
	switch viewName {
	case "dashboard":
		return view.NewDashboardView(a.ctx, a.registry)
	case "services", "":
		// Default to ServiceBrowser
		return view.NewServiceBrowser(a.ctx, a.registry)
	default:
		// Try to parse as AWS service/resource (e.g., "ec2", "rds/snapshots")
		service, resourceType, err := a.registry.ParseServiceResource(viewName)
		if err != nil {
			// Fallback to ServiceBrowser on error
			return view.NewServiceBrowser(a.ctx, a.registry)
		}
		return view.NewResourceBrowserWithType(a.ctx, a.registry, service, resourceType)
	}
}

func (a *App) buildAIContext() *ai.Context {
	regions := config.Global().Regions()
	selections := config.Global().Selections()
	var profiles []string
	for _, sel := range selections {
		if id := sel.ID(); id != "" {
			profiles = append(profiles, id)
		}
	}

	switch v := a.currentView.(type) {
	case *view.ResourceBrowser:
		return &ai.Context{
			Mode:          ai.ContextModeList,
			Service:       v.Service(),
			ResourceType:  v.ResourceType(),
			ResourceCount: v.ResourceCount(),
			FilterText:    v.FilterText(),
			Toggles:       v.ToggleStates(),
			UserRegions:   regions,
			UserProfiles:  profiles,
		}

	case *view.DiffView:
		return &ai.Context{
			Mode:         ai.ContextModeDiff,
			Service:      v.Service(),
			ResourceType: v.ResourceType(),
			DiffLeft:     buildResourceRef(v.Left()),
			DiffRight:    buildResourceRef(v.Right()),
			UserRegions:  regions,
			UserProfiles: profiles,
		}

	case *view.DetailView:
		r := v.Resource()
		if r != nil {
			unwrapped := dao.UnwrapResource(r)
			resourceRegion := dao.GetResourceRegion(r)
			log.Debug("buildAIContext DetailView", "service", v.Service(), "resourceType", v.ResourceType(),
				"id", unwrapped.GetID(), "resourceRegion", resourceRegion, "regions", regions)
			ctx := &ai.Context{
				Mode:            ai.ContextModeSingle,
				Service:         v.Service(),
				ResourceType:    v.ResourceType(),
				ResourceID:      unwrapped.GetID(),
				ResourceName:    unwrapped.GetName(),
				ResourceRegion:  resourceRegion,
				ResourceProfile: dao.GetResourceProfile(r),
				UserRegions:     regions,
				UserProfiles:    profiles,
			}
			if v.Service() == "lambda" && v.ResourceType() == "functions" {
				ctx.LogGroup = "/aws/lambda/" + unwrapped.GetName()
			}
			if clusterArn := dao.GetResourceClusterArn(r); clusterArn != "" {
				ctx.Cluster = aws.ExtractResourceName(clusterArn)
			}
			return ctx
		}

	case *view.LogView:
		return &ai.Context{
			LogGroup:     v.LogGroupName(),
			UserRegions:  regions,
			UserProfiles: profiles,
		}
	}
	return &ai.Context{UserRegions: regions, UserProfiles: profiles}
}

func buildResourceRef(r dao.Resource) *ai.ResourceRef {
	unwrapped := dao.UnwrapResource(r)
	ref := &ai.ResourceRef{
		ID:      unwrapped.GetID(),
		Name:    unwrapped.GetName(),
		Region:  dao.GetResourceRegion(r),
		Profile: dao.GetResourceProfile(r),
	}
	if clusterArn := dao.GetResourceClusterArn(r); clusterArn != "" {
		ref.Cluster = aws.ExtractResourceName(clusterArn)
	}
	return ref
}
