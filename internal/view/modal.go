package view

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/clawscli/claws/internal/ui"
)

type ModalStyle int

const (
	ModalStyleNormal ModalStyle = iota
	ModalStyleWarning
	ModalStyleDanger
)

type Modal struct {
	Content View
	Style   ModalStyle
	Width   int
	Height  int
}

type ShowModalMsg struct {
	Modal *Modal
}

type HideModalMsg struct{}

type modalStyles struct {
	box     lipgloss.Style
	warning lipgloss.Style
	danger  lipgloss.Style
}

func newModalStyles() modalStyles {
	t := ui.Current()
	return modalStyles{
		box: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Border).
			Padding(1, 2),
		warning: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Warning).
			Padding(1, 2),
		danger: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Danger).
			Padding(1, 2),
	}
}

type ModalRenderer struct {
	styles modalStyles
}

func NewModalRenderer() *ModalRenderer {
	return &ModalRenderer{
		styles: newModalStyles(),
	}
}

func (r *ModalRenderer) Render(modal *Modal, bg string, width, height int) string {
	if modal == nil || modal.Content == nil {
		return bg
	}

	content := modal.Content.ViewString()

	var boxStyle lipgloss.Style
	switch modal.Style {
	case ModalStyleWarning:
		boxStyle = r.styles.warning
	case ModalStyleDanger:
		boxStyle = r.styles.danger
	default:
		boxStyle = r.styles.box
	}

	modalWidth := modal.Width
	if modalWidth == 0 {
		modalWidth = min(lipgloss.Width(content)+6, width-10)
	}
	boxStyle = boxStyle.Width(modalWidth)

	box := boxStyle.Render(content)

	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}

func (m *Modal) Update(msg tea.Msg) (*Modal, tea.Cmd) {
	if m.Content == nil {
		return m, nil
	}
	model, cmd := m.Content.Update(msg)
	if v, ok := model.(View); ok {
		m.Content = v
	}
	return m, cmd
}

func (m *Modal) SetSize(width, height int) tea.Cmd {
	if m.Content == nil {
		return nil
	}
	modalWidth := m.Width
	if modalWidth == 0 {
		modalWidth = min(60, width-10)
	}
	contentWidth := modalWidth - 6
	contentHeight := height - 10
	return m.Content.SetSize(contentWidth, contentHeight)
}
