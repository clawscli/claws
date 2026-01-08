package view

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/clawscli/claws/internal/ai"
	"github.com/clawscli/claws/internal/ui"
)

const ModalWidthSessionHistory = 50

type sessionHistoryStyles struct {
	title    lipgloss.Style
	item     lipgloss.Style
	selected lipgloss.Style
	hint     lipgloss.Style
	current  lipgloss.Style
}

func newSessionHistoryStyles() sessionHistoryStyles {
	return sessionHistoryStyles{
		title:    ui.TableHeaderStyle().Padding(0, 1),
		item:     ui.TextStyle().PaddingLeft(2),
		selected: ui.SelectedStyle().PaddingLeft(2),
		hint:     ui.DimStyle(),
		current:  ui.AccentStyle(),
	}
}

type SessionSelectedMsg struct {
	Session *ai.Session
}

type NewSessionMsg struct{}

type CloseHistoryMsg struct{}

type SessionHistory struct {
	sessions  []ai.Session
	currentID string
	cursor    int
	styles    sessionHistoryStyles
	width     int
	height    int
}

func NewSessionHistory(sessions []ai.Session, currentID string) *SessionHistory {
	return &SessionHistory{
		sessions:  sessions,
		currentID: currentID,
		cursor:    0,
		styles:    newSessionHistoryStyles(),
	}
}

func (s *SessionHistory) Init() tea.Cmd {
	return nil
}

func (s *SessionHistory) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			if s.cursor > 0 {
				s.cursor--
			}
			return s, nil
		case "down", "j":
			if s.cursor < len(s.sessions)-1 {
				s.cursor++
			}
			return s, nil
		case "enter":
			if s.cursor >= 0 && s.cursor < len(s.sessions) {
				return s, func() tea.Msg {
					return SessionSelectedMsg{Session: &s.sessions[s.cursor]}
				}
			}
			return s, nil
		case "n":
			return s, func() tea.Msg {
				return NewSessionMsg{}
			}
		case "esc", "q", "ctrl+c", "ctrl+h":
			return s, func() tea.Msg {
				return CloseHistoryMsg{}
			}
		}
	}
	return s, nil
}

func (s *SessionHistory) View() tea.View {
	return tea.NewView(s.ViewString())
}

func (s *SessionHistory) ViewString() string {
	var b strings.Builder

	b.WriteString(s.styles.title.Render("Chat History"))
	b.WriteString("\n\n")

	if len(s.sessions) == 0 {
		b.WriteString(s.styles.hint.Render("  No saved sessions"))
		b.WriteString("\n")
	} else {
		for i, sess := range s.sessions {
			style := s.styles.item
			prefix := "  "
			if i == s.cursor {
				style = s.styles.selected
				prefix = "> "
			}

			dateStr := sess.UpdatedAt.Format("2006-01-02 15:04")
			msgCount := len(sess.Messages)
			line := fmt.Sprintf("%s%s  (%d msgs)", prefix, dateStr, msgCount)

			if sess.ID == s.currentID {
				line += " " + s.styles.current.Render("*")
			}

			b.WriteString(style.Render(line))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(s.styles.hint.Render("j/k:select  enter:load  n:new  esc:close"))

	return b.String()
}

func (s *SessionHistory) SetSize(width, height int) tea.Cmd {
	s.width = width
	s.height = height
	return nil
}

func (s *SessionHistory) StatusLine() string {
	return ""
}
