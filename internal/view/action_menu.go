package view

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/ui"
)

// ActionMenu displays available actions for a resource
// actionMenuStyles holds cached lipgloss styles for performance
type actionMenuStyles struct {
	title     lipgloss.Style
	item      lipgloss.Style
	selected  lipgloss.Style
	shortcut  lipgloss.Style
	dangerous lipgloss.Style
	box       lipgloss.Style
	boxDanger lipgloss.Style
	yes       lipgloss.Style
	no        lipgloss.Style
	bold      lipgloss.Style
}

func newActionMenuStyles() actionMenuStyles {
	t := ui.Current()
	return actionMenuStyles{
		title:     lipgloss.NewStyle().Bold(true).Foreground(t.Primary).MarginBottom(1),
		item:      lipgloss.NewStyle().PaddingLeft(2),
		selected:  lipgloss.NewStyle().PaddingLeft(2).Background(t.Selection).Foreground(t.SelectionText),
		shortcut:  lipgloss.NewStyle().Foreground(t.Secondary),
		dangerous: lipgloss.NewStyle().Foreground(t.Danger),
		box:       lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(t.Border).Padding(0, 1).MarginTop(1),
		boxDanger: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(t.Danger).Padding(0, 1).MarginTop(1),
		yes:       lipgloss.NewStyle().Bold(true).Foreground(t.Success),
		no:        lipgloss.NewStyle().Bold(true).Foreground(t.Danger),
		bold:      lipgloss.NewStyle().Bold(true),
	}
}

type ActionMenu struct {
	ctx        context.Context
	resource   dao.Resource
	service    string
	resType    string
	actions    []action.Action
	cursor     int
	width      int
	height     int
	result     *action.ActionResult
	confirming bool
	confirmIdx int
	styles     actionMenuStyles
}

// NewActionMenu creates a new ActionMenu
func NewActionMenu(ctx context.Context, resource dao.Resource, service, resType string) *ActionMenu {
	actions := action.Global.Get(service, resType)

	// Filter out dangerous actions in read-only mode
	if config.Global().ReadOnly() {
		filtered := make([]action.Action, 0, len(actions))
		for _, act := range actions {
			if !act.Dangerous && act.Type != action.ActionTypeAPI {
				// In read-only mode, only allow non-dangerous, non-API actions (like view, exec for SSH)
				filtered = append(filtered, act)
			}
		}
		actions = filtered
	}

	return &ActionMenu{
		ctx:      ctx,
		resource: resource,
		service:  service,
		resType:  resType,
		actions:  actions,
		styles:   newActionMenuStyles(),
	}
}

// Init implements tea.Model
func (m *ActionMenu) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m *ActionMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case execResultMsg:
		// Handle exec action result
		m.result = &action.ActionResult{
			Success: msg.success,
			Message: msg.message,
			Error:   msg.err,
		}
		return m, nil

	case tea.KeyMsg:
		// Handle confirmation mode
		if m.confirming {
			switch msg.String() {
			case "y", "Y":
				m.confirming = false
				if m.confirmIdx < len(m.actions) {
					act := m.actions[m.confirmIdx]
					return m.executeAction(act)
				}
				return m, nil
			case "n", "N", "esc":
				m.confirming = false
				return m, nil
			}
			return m, nil
		}

		switch msg.String() {
		// Don't intercept esc/q - let the app handle back navigation
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.actions)-1 {
				m.cursor++
			}
		case "enter":
			if m.cursor < len(m.actions) {
				act := m.actions[m.cursor]
				if act.Confirm || act.Dangerous {
					m.confirming = true
					m.confirmIdx = m.cursor
					return m, nil
				}
				return m.executeAction(act)
			}
		default:
			// Check if key matches a shortcut
			for i, act := range m.actions {
				if msg.String() == act.Shortcut {
					if act.Confirm || act.Dangerous {
						m.confirming = true
						m.confirmIdx = i
						m.cursor = i
						return m, nil
					}
					return m.executeAction(act)
				}
			}
		}
	}
	return m, nil
}

// executeAction executes the given action, handling exec-type actions specially
func (m *ActionMenu) executeAction(act action.Action) (tea.Model, tea.Cmd) {
	if act.Type == action.ActionTypeExec {
		// For exec actions, use tea.Exec to suspend bubbletea
		execCmd, err := action.ExpandVariables(act.Command, m.resource)
		if err != nil {
			return m, func() tea.Msg {
				return execResultMsg{success: false, err: err}
			}
		}
		exec := &action.ExecWithHeader{
			Command:  execCmd,
			Resource: m.resource,
			Service:  m.service,
			ResType:  m.resType,
		}
		return m, tea.Exec(exec, func(err error) tea.Msg {
			if err != nil {
				return execResultMsg{success: false, err: err}
			}
			return execResultMsg{success: true, message: "Session ended"}
		})
	}

	// For other actions, execute directly
	result := action.ExecuteWithDAO(m.ctx, act, m.resource, m.service, m.resType)
	m.result = &result
	return m, nil
}

// execResultMsg is sent when an exec action completes
type execResultMsg struct {
	success bool
	message string
	err     error
}

// View implements tea.Model
func (m *ActionMenu) View() string {
	s := m.styles

	var out string
	out += s.title.Render(fmt.Sprintf("Actions for %s", m.resource.GetName())) + "\n\n"

	if len(m.actions) == 0 {
		out += ui.DimStyle().Render("No actions available")
		return out
	}

	for i, act := range m.actions {
		style := s.item
		if i == m.cursor {
			style = s.selected
		}

		shortcut := s.shortcut.Render(fmt.Sprintf("[%s]", act.Shortcut))
		name := act.Name
		if act.Dangerous {
			name = s.dangerous.Render(name + " ⚠")
		}

		out += style.Render(fmt.Sprintf("%s %s", shortcut, name)) + "\n"
	}

	// Show confirmation dialog if confirming
	if m.confirming && m.confirmIdx < len(m.actions) {
		act := m.actions[m.confirmIdx]
		out += "\n"

		boxStyle := s.box
		if act.Dangerous {
			boxStyle = s.boxDanger
		}

		confirmTitle := "Confirm Action"
		if act.Dangerous {
			confirmTitle = "⚠ DANGEROUS ACTION"
		}

		confirmContent := s.bold.Render(confirmTitle) + "\n"
		confirmContent += fmt.Sprintf("Execute '%s' on %s?\n\n", act.Name, m.resource.GetID())
		confirmContent += "Press " + s.yes.Render("[Y]") + " to confirm or " + s.no.Render("[N]") + " to cancel"

		out += boxStyle.Render(confirmContent)
	} else if m.result != nil {
		out += "\n"
		if m.result.Success {
			out += ui.SuccessStyle().Render(m.result.Message)
		} else {
			out += ui.DangerStyle().Render(fmt.Sprintf("Error: %v", m.result.Error))
		}
	}

	if !m.confirming {
		out += "\n\n" + ui.DimStyle().Render("Press shortcut key or Enter to execute, Esc to cancel")
	}

	return out
}

// SetSize implements View
func (m *ActionMenu) SetSize(width, height int) tea.Cmd {
	m.width = width
	m.height = height
	return nil
}

// StatusLine implements View
func (m *ActionMenu) StatusLine() string {
	if m.confirming {
		return "Confirm: Y/N"
	}
	return fmt.Sprintf("Actions for %s • Enter to execute • Esc to cancel", m.resource.GetID())
}
