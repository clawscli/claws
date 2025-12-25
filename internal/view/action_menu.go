package view

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/log"
	navmsg "github.com/clawscli/claws/internal/msg"
	"github.com/clawscli/claws/internal/ui"
)

// ActionMenu displays available actions for a resource
// actionMenuStyles holds cached lipgloss styles for performance
type actionMenuStyles struct {
	title    lipgloss.Style
	item     lipgloss.Style
	selected lipgloss.Style
	shortcut lipgloss.Style
	box      lipgloss.Style
	yes      lipgloss.Style
	no       lipgloss.Style
	bold     lipgloss.Style
}

func newActionMenuStyles() actionMenuStyles {
	t := ui.Current()
	return actionMenuStyles{
		title:    lipgloss.NewStyle().Bold(true).Foreground(t.Primary).MarginBottom(1),
		item:     lipgloss.NewStyle().PaddingLeft(2),
		selected: lipgloss.NewStyle().PaddingLeft(2).Background(t.Selection).Foreground(t.SelectionText),
		shortcut: lipgloss.NewStyle().Foreground(t.Secondary),
		box:      lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(t.Border).Padding(0, 1).MarginTop(1),
		yes:      lipgloss.NewStyle().Bold(true).Foreground(t.Success),
		no:       lipgloss.NewStyle().Bold(true).Foreground(t.Danger),
		bold:     lipgloss.NewStyle().Bold(true),
	}
}

type ActionMenu struct {
	ctx            context.Context
	resource       dao.Resource
	service        string
	resType        string
	actions        []action.Action
	cursor         int
	width          int
	height         int
	result         *action.ActionResult
	confirming     bool
	confirmIdx     int
	lastExecAction *action.Action // Last executed exec action for PostExecFollowUp
	styles         actionMenuStyles
}

// NewActionMenu creates a new ActionMenu
func NewActionMenu(ctx context.Context, resource dao.Resource, service, resType string) *ActionMenu {
	actions := action.Global.Get(service, resType)

	// Filter actions based on resource and read-only mode
	filtered := make([]action.Action, 0, len(actions))
	readOnly := config.Global().ReadOnly()
	for _, act := range actions {
		// Apply per-action filter
		if act.Filter != nil && !act.Filter(resource) {
			continue
		}
		// Read-only mode filtering:
		// - View actions: always allowed
		// - Exec actions: allowed only if in ReadOnlyExecAllowlist (auth workflows)
		// - API actions: allowed only if in ReadOnlyAllowlist
		if readOnly {
			switch act.Type {
			case action.ActionTypeView:
				// always allowed
			case action.ActionTypeExec:
				if !action.ReadOnlyExecAllowlist[act.Name] {
					continue // deny arbitrary shells (ECS Exec, SSM Session, etc.)
				}
			case action.ActionTypeAPI:
				if !action.ReadOnlyAllowlist[act.Operation] {
					continue
				}
			}
		}
		filtered = append(filtered, act)
	}
	actions = filtered

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
	case navmsg.ProfileChangedMsg, navmsg.RegionChangedMsg:
		// Let app.go handle these navigation messages
		return m, func() tea.Msg { return msg }

	case execResultMsg:
		// Handle exec action result
		m.result = &action.ActionResult{
			Success: msg.success,
			Message: msg.message,
			Error:   msg.err,
		}
		// Generic post-exec follow-up handling
		if msg.success && m.lastExecAction != nil && m.lastExecAction.PostExecFollowUp != nil {
			followUp := m.lastExecAction.PostExecFollowUp(m.resource)
			if followUp != nil {
				log.Debug("post-exec follow-up", "action", m.lastExecAction.Name, "msgType", fmt.Sprintf("%T", followUp))
				return m, func() tea.Msg { return followUp }
			}
		}
		return m, nil

	case tea.MouseMotionMsg:
		// Hover: update cursor
		if !m.confirming {
			if idx := m.getActionAtPosition(msg.Y); idx >= 0 && idx != m.cursor {
				m.cursor = idx
			}
		}
		return m, nil

	case tea.MouseClickMsg:
		// Click: select and execute
		if msg.Button == tea.MouseLeft && !m.confirming {
			if idx := m.getActionAtPosition(msg.Y); idx >= 0 {
				m.cursor = idx
				act := m.actions[idx]
				if act.Confirm != action.ConfirmNone {
					m.confirming = true
					m.confirmIdx = idx
					return m, nil
				}
				return m.executeAction(act)
			}
		}
		return m, nil

	case tea.KeyPressMsg:
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
				if act.Confirm != action.ConfirmNone {
					m.confirming = true
					m.confirmIdx = m.cursor
					return m, nil
				}
				return m.executeAction(act)
			}
		default:
			// Check if key matches a shortcut
			log.Debug("action menu key pressed", "key", msg.String(), "actionsCount", len(m.actions))
			for i, act := range m.actions {
				if msg.String() == act.Shortcut {
					log.Debug("shortcut matched", "shortcut", act.Shortcut, "action", act.Name)
					if act.Confirm != action.ConfirmNone {
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
		// Record action for post-exec follow-up handling
		m.lastExecAction = &act

		// For exec actions, use tea.Exec to suspend bubbletea
		execCmd, err := action.ExpandVariables(act.Command, m.resource)
		if err != nil {
			return m, func() tea.Msg {
				return execResultMsg{success: false, err: err}
			}
		}
		exec := &action.ExecWithHeader{
			Command:    execCmd,
			ActionName: act.Name,
			Resource:   m.resource,
			Service:    m.service,
			ResType:    m.resType,
			SkipAWSEnv: act.SkipAWSEnv,
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

	// If action has a follow-up message, send it
	if result.FollowUpMsg != nil {
		log.Debug("action has follow-up message", "action", act.Name, "msgType", fmt.Sprintf("%T", result.FollowUpMsg))
		return m, func() tea.Msg { return result.FollowUpMsg }
	}
	return m, nil
}

// execResultMsg is sent when an exec action completes
type execResultMsg struct {
	success bool
	message string
	err     error
}

// ViewString returns the view content as a string
func (m *ActionMenu) ViewString() string {
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
		out += style.Render(fmt.Sprintf("%s %s", shortcut, act.Name)) + "\n"
	}

	// Show confirmation dialog if confirming
	if m.confirming && m.confirmIdx < len(m.actions) {
		act := m.actions[m.confirmIdx]
		out += "\n"

		confirmContent := s.bold.Render("Confirm Action") + "\n"
		confirmContent += fmt.Sprintf("Execute '%s' on %s?\n\n", act.Name, m.resource.GetID())
		confirmContent += "Press " + s.yes.Render("[Y]") + " to confirm or " + s.no.Render("[N]") + " to cancel"

		out += s.box.Render(confirmContent)
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

// View implements tea.Model
func (m *ActionMenu) View() tea.View {
	return tea.NewView(m.ViewString())
}

// getActionAtPosition returns the action index at given Y position, or -1 if none
func (m *ActionMenu) getActionAtPosition(y int) int {
	// Layout: title (1) + margin (1) + empty (1) = 3 lines before actions
	headerHeight := 3
	idx := y - headerHeight

	if idx >= 0 && idx < len(m.actions) {
		return idx
	}
	return -1
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
