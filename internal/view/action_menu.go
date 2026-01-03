package view

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
	"github.com/clawscli/claws/internal/log"
	navmsg "github.com/clawscli/claws/internal/msg"
	"github.com/clawscli/claws/internal/ui"
)

// ActionMenu displays available actions for a resource
// actionMenuStyles holds cached lipgloss styles for performance
type actionMenuStyles struct {
	title     lipgloss.Style
	item      lipgloss.Style
	selected  lipgloss.Style
	shortcut  lipgloss.Style
	box       lipgloss.Style
	dangerBox lipgloss.Style
	yes       lipgloss.Style
	no        lipgloss.Style
	bold      lipgloss.Style
	input     lipgloss.Style
}

func newActionMenuStyles() actionMenuStyles {
	t := ui.Current()
	return actionMenuStyles{
		title:     ui.TitleStyle(),
		item:      lipgloss.NewStyle().PaddingLeft(2),
		selected:  ui.SelectedStyle().PaddingLeft(2),
		shortcut:  ui.SecondaryStyle(),
		box:       ui.BoxStyle().MarginTop(1),
		dangerBox: ui.BoxStyle().BorderForeground(t.Danger).MarginTop(1),
		yes:       ui.BoldSuccessStyle(),
		no:        ui.BoldDangerStyle(),
		bold:      lipgloss.NewStyle().Bold(true),
		input:     ui.InputStyle(),
	}
}

type dangerousState struct {
	active bool
	input  string
	token  string
}

type ActionMenu struct {
	ctx            context.Context
	resource       dao.Resource
	service        string
	resType        string
	actions        []action.Action
	cursor         int
	result         *action.ActionResult
	confirming     bool
	confirmIdx     int
	lastExecAction *action.Action
	styles         actionMenuStyles
	dangerous      dangerousState
}

// NewActionMenu creates a new ActionMenu
func NewActionMenu(ctx context.Context, resource dao.Resource, service, resType string) *ActionMenu {
	actions := action.Global.Get(service, resType)

	filtered := make([]action.Action, 0, len(actions))
	readOnly := config.Global().ReadOnly()
	for _, act := range actions {
		if act.Filter != nil && !act.Filter(resource) {
			continue
		}
		if readOnly && !action.IsAllowedInReadOnly(act) {
			continue
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
	case navmsg.ProfilesChangedMsg, navmsg.RegionChangedMsg:
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
		if !m.confirming && !m.dangerous.active {
			if idx := m.getActionAtPosition(msg.Y); idx >= 0 && idx != m.cursor {
				m.cursor = idx
			}
		}
		return m, nil

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft && !m.confirming && !m.dangerous.active {
			if idx := m.getActionAtPosition(msg.Y); idx >= 0 {
				m.cursor = idx
				return m.handleActionConfirm(m.actions[idx], idx)
			}
		}
		return m, nil

	case tea.KeyPressMsg:
		if m.dangerous.active {
			switch msg.String() {
			case "enter":
				if action.ConfirmMatches(m.dangerous.token, m.dangerous.input) {
					m.dangerous.active = false
					m.dangerous.input = ""
					m.dangerous.token = ""
					if m.confirmIdx < len(m.actions) {
						return m.executeAction(m.actions[m.confirmIdx])
					}
				}
				return m, nil
			case "esc":
				m.dangerous.active = false
				m.dangerous.input = ""
				m.dangerous.token = ""
				return m, nil
			default:
				if msg.Code == tea.KeyBackspace || msg.String() == "backspace" {
					if len(m.dangerous.input) > 0 {
						m.dangerous.input = m.dangerous.input[:len(m.dangerous.input)-1]
					}
					return m, nil
				}
				if len(msg.String()) == 1 {
					m.dangerous.input += msg.String()
				}
				return m, nil
			}
		}

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
				return m.handleActionConfirm(act, m.cursor)
			}
		default:
			log.Debug("action menu key pressed", "key", msg.String(), "actionsCount", len(m.actions))
			for i, act := range m.actions {
				if msg.String() == act.Shortcut {
					log.Debug("shortcut matched", "shortcut", act.Shortcut, "action", act.Name)
					m.cursor = i
					return m.handleActionConfirm(act, i)
				}
			}
		}
	}
	return m, nil
}

func (m *ActionMenu) handleActionConfirm(act action.Action, idx int) (tea.Model, tea.Cmd) {
	switch act.Confirm {
	case action.ConfirmDangerous:
		m.dangerous.active = true
		m.dangerous.input = ""
		m.confirmIdx = idx
		m.dangerous.token = m.getConfirmToken(act)
		return m, nil
	case action.ConfirmSimple:
		m.confirming = true
		m.confirmIdx = idx
		return m, nil
	default:
		return m.executeAction(act)
	}
}

func (m *ActionMenu) getConfirmToken(act action.Action) string {
	if act.ConfirmToken != nil {
		return act.ConfirmToken(m.resource)
	}
	return m.resource.GetID()
}

func (m *ActionMenu) executeAction(act action.Action) (tea.Model, tea.Cmd) {
	if act.Type == action.ActionTypeExec {
		m.lastExecAction = &act
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
			Region:     aws.GetRegionFromContext(m.ctx),
			SkipAWSEnv: act.SkipAWSEnv,
		}
		return m, tea.Exec(exec, func(err error) tea.Msg {
			if err != nil {
				return execResultMsg{success: false, err: err}
			}
			return execResultMsg{success: true, message: "Session ended"}
		})
	}

	result := action.ExecuteWithDAO(m.ctx, act, m.resource, m.service, m.resType)
	m.result = &result
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

	if m.dangerous.active && m.confirmIdx < len(m.actions) {
		act := m.actions[m.confirmIdx]
		out += "\n"
		out += m.renderDangerousConfirm(act)
	} else if m.confirming && m.confirmIdx < len(m.actions) {
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
		} else if m.result.ErrorKind != apperrors.Unknown {
			out += ui.DangerStyle().Render(fmt.Sprintf("[%s] %v", m.result.ErrorKind, m.result.Error))
		} else {
			out += ui.DangerStyle().Render(fmt.Sprintf("Error: %v", m.result.Error))
		}
	}

	if !m.confirming && !m.dangerous.active {
		out += "\n\n" + ui.DimStyle().Render("Press shortcut key or Enter to execute, Esc to cancel")
	}

	return out
}

func (m *ActionMenu) renderDangerousConfirm(act action.Action) string {
	s := m.styles
	t := ui.Current()

	dangerTitle := ui.BoldDangerStyle().Render("⚠ DANGER")
	content := dangerTitle + "\n\n"
	content += fmt.Sprintf("You are about to %s:\n", s.no.Render(act.Name))
	content += s.bold.Render(m.dangerous.token) + "\n\n"

	suffix := action.ConfirmSuffix(m.dangerous.token)
	if len(suffix) < len(m.dangerous.token) {
		content += fmt.Sprintf("Type last %d chars: ...%s\n", len(suffix), suffix)
	} else {
		content += "Type to confirm:\n"
	}

	inputStyle := s.input
	matched := action.ConfirmMatches(m.dangerous.token, m.dangerous.input)
	if matched {
		inputStyle = inputStyle.BorderForeground(t.Success)
	} else if len(m.dangerous.input) > 0 && strings.HasPrefix(suffix, m.dangerous.input) {
		inputStyle = inputStyle.BorderForeground(t.Warning)
	}
	content += inputStyle.Render(m.dangerous.input+"▌") + "\n\n"
	content += ui.DimStyle().Render("Press Enter to confirm, Esc to cancel")

	return s.dangerBox.Render(content)
}

func (m *ActionMenu) View() tea.View {
	return tea.NewView(m.ViewString())
}

func (m *ActionMenu) getActionAtPosition(y int) int {
	actionMenuHeaderHeight := 3
	idx := y - actionMenuHeaderHeight
	if idx >= 0 && idx < len(m.actions) {
		return idx
	}
	return -1
}

// SetSize implements View
func (m *ActionMenu) SetSize(_, _ int) tea.Cmd {
	return nil
}

func (m *ActionMenu) StatusLine() string {
	if m.dangerous.active {
		suffix := action.ConfirmSuffix(m.dangerous.token)
		if m.dangerous.input != "" && !strings.HasPrefix(suffix, m.dangerous.input) {
			return "Token does not match"
		}
		if len(suffix) < len(m.dangerous.token) {
			return fmt.Sprintf("Type last %d chars to confirm", len(suffix))
		}
		return "Type resource ID to confirm"
	}
	if m.confirming {
		return "Confirm: Y/N"
	}
	return fmt.Sprintf("Actions for %s • Enter to execute • Esc to cancel", m.resource.GetID())
}

func (m *ActionMenu) HasActiveInput() bool {
	return m.dangerous.active
}
