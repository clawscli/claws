package app

import (
	"context"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/log"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/ui"
	"github.com/clawscli/claws/internal/view"
)

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
		status:       lipgloss.NewStyle().Background(t.TableHeader).Foreground(t.TableHeaderText).Padding(0, 1).Width(width),
		readOnly:     lipgloss.NewStyle().Background(t.Warning).Foreground(lipgloss.Color("#000000")).Bold(true).Padding(0, 1),
		warningTitle: lipgloss.NewStyle().Bold(true).Foreground(t.Pending).MarginBottom(1),
		warningItem:  lipgloss.NewStyle().Foreground(t.Warning),
		warningDim:   lipgloss.NewStyle().Foreground(t.TextDim).MarginTop(1),
		warningBox:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(t.Pending).Padding(1, 2),
	}
}

type App struct {
	ctx      context.Context
	registry *registry.Registry
	width    int
	height   int

	// Current view
	currentView view.View
	viewStack   []view.View

	// Command mode
	commandInput *view.CommandInput
	commandMode  bool

	// UI components
	help help.Model
	keys keyMap

	// Status
	err error

	// Startup warnings
	showWarnings  bool
	warningsReady bool // true after first render, to ignore initial terminal responses

	// Cached styles
	styles appStyles
}

// New creates a new App instance
func New(ctx context.Context, reg *registry.Registry) *App {
	return &App{
		ctx:          ctx,
		registry:     reg,
		commandInput: view.NewCommandInput(ctx, reg),
		help:         help.New(),
		keys:         defaultKeyMap(),
		styles:       newAppStyles(0),
	}
}

// Init implements tea.Model
func (a *App) Init() tea.Cmd {
	// Initialize config (detect region from IMDS)
	_ = config.Global().Init(a.ctx)

	// Show warnings if any
	if len(config.Global().Warnings()) > 0 {
		a.showWarnings = true
	}

	// Start with the service browser view
	a.currentView = view.NewServiceBrowser(a.ctx, a.registry)
	return a.currentView.Init()
}

// Update implements tea.Model
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Dismiss warnings on Enter or Space only
	if a.showWarnings && a.warningsReady {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if keyMsg.Type == tea.KeyEnter || keyMsg.String() == " " {
				a.showWarnings = false
				return a, nil
			}
			// Ignore other keys while showing warnings
			return a, nil
		}
	}

	// Handle command mode first
	if a.commandMode {
		switch msg := msg.(type) {
		case tea.KeyMsg:
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
		a.help.Width = msg.Width
		a.commandInput.SetWidth(msg.Width)
		// Update cached styles with new width
		a.styles = newAppStyles(msg.Width)
		// Mark warnings as ready to be dismissed after first window size (terminal init complete)
		if a.showWarnings && !a.warningsReady {
			a.warningsReady = true
		}
		if a.currentView != nil {
			return a, a.currentView.SetSize(msg.Width, msg.Height-2)
		}
		return a, nil

	case tea.KeyMsg:
		// Handle back navigation (esc or backspace)
		// Check for ESC key in various forms (KeyEsc, KeyEscape, or raw ESC byte as KeyRunes)
		isEsc := msg.String() == "esc" || msg.Type == tea.KeyEsc || msg.Type == tea.KeyEscape ||
			(msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && msg.Runes[0] == 27)
		isBack := isEsc || msg.Type == tea.KeyBackspace

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
			return a, tea.Quit

		case key.Matches(msg, a.keys.Help):
			// Show full help view
			helpView := view.NewHelpView()
			if a.currentView != nil {
				a.viewStack = append(a.viewStack, a.currentView)
			}
			a.currentView = helpView
			return a, a.currentView.SetSize(a.width, a.height-2)

		case key.Matches(msg, a.keys.Command):
			a.commandMode = true
			// Set tag completion provider if current view is a ResourceBrowser
			if rb, ok := a.currentView.(*view.ResourceBrowser); ok {
				a.commandInput.SetTagProvider(rb)
			} else {
				a.commandInput.SetTagProvider(nil)
			}
			return a, a.commandInput.Activate()

		case key.Matches(msg, a.keys.Region):
			regionSelector := view.NewRegionSelector(a.ctx)
			if a.currentView != nil {
				a.viewStack = append(a.viewStack, a.currentView)
			}
			a.currentView = regionSelector
			return a, tea.Batch(
				a.currentView.Init(),
				a.currentView.SetSize(a.width, a.height-2),
			)

		case key.Matches(msg, a.keys.Profile):
			profileSelector := view.NewProfileSelector()
			if a.currentView != nil {
				a.viewStack = append(a.viewStack, a.currentView)
			}
			a.currentView = profileSelector
			return a, tea.Batch(
				a.currentView.Init(),
				a.currentView.SetSize(a.width, a.height-2),
			)
		}

	case view.NavigateMsg:
		log.Debug("navigating", "clearStack", msg.ClearStack, "stackDepth", len(a.viewStack))
		// Push current view to stack (unless ClearStack is set)
		if msg.ClearStack {
			a.viewStack = nil
		} else if a.currentView != nil {
			a.viewStack = append(a.viewStack, a.currentView)
		}
		a.currentView = msg.View
		cmds := []tea.Cmd{
			a.currentView.Init(),
			a.currentView.SetSize(a.width, a.height-2),
		}
		return a, tea.Batch(cmds...)

	case view.ErrorMsg:
		log.Error("application error", "error", msg.Err)
		a.err = msg.Err
		return a, nil

	case view.RegionChangedMsg:
		log.Info("region changed", "region", msg.Region)
		// Pop views until we find a refreshable one (ResourceBrowser or ServiceBrowser)
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
		// Fallback to service browser if no refreshable view found
		a.currentView = view.NewServiceBrowser(a.ctx, a.registry)
		return a, tea.Batch(
			a.currentView.Init(),
			a.currentView.SetSize(a.width, a.height-2),
		)

	case view.ProfileChangedMsg:
		log.Info("profile changed", "profile", msg.Profile)
		// Pop views until we find a refreshable one (ResourceBrowser or ServiceBrowser)
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
		// Fallback to service browser if no refreshable view found
		a.currentView = view.NewServiceBrowser(a.ctx, a.registry)
		return a, tea.Batch(
			a.currentView.Init(),
			a.currentView.SetSize(a.width, a.height-2),
		)

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

// View implements tea.Model
func (a *App) View() string {
	// Show warnings modal if active
	if a.showWarnings {
		return a.renderWarnings()
	}

	var content string
	if a.currentView != nil {
		content = a.currentView.View()
	}

	// Command input (replaces status bar when active)
	if a.commandMode {
		cmdView := a.commandInput.View()
		return content + "\n" + cmdView
	}

	// Status bar (use cached style)
	var statusContent string
	if a.err != nil {
		statusContent = ui.DangerStyle().Render("Error: " + a.err.Error())
	} else if a.currentView != nil {
		statusContent = a.currentView.StatusLine()
	}

	// Add read-only indicator (use cached style)
	if config.Global().ReadOnly() {
		roIndicator := a.styles.readOnly.Render("READ-ONLY")
		statusContent = roIndicator + " " + statusContent
	}

	status := a.styles.status.Render(statusContent)

	return content + "\n" + status
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

	content += "\n" + s.warningDim.Render("Press Enter or Space to continue...")

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

// keyMap defines the key bindings
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
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
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
