package view

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/clawscli/claws/internal/ui"
)

type ModalStyle int

const (
	ModalStyleNormal ModalStyle = iota
	ModalStyleWarning
	ModalStyleDanger
)

const (
	// modalBoxPadding: border (1*2) + padding (2*2) = 6
	modalBoxPadding   = 6
	modalScreenMargin = 10
	modalDefaultWidth = 60
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
		modalWidth = min(lipgloss.Width(content)+modalBoxPadding, width-modalScreenMargin)
	}
	boxStyle = boxStyle.Width(modalWidth)

	box := boxStyle.Render(content)

	dimmedBg := dimBackground(bg, width, height)
	return placeOverlay(box, dimmedBg, width, height)
}

func dimBackground(bg string, width, height int) string {
	dimStyle := lipgloss.NewStyle().Faint(true)
	lines := strings.Split(bg, "\n")

	for i, line := range lines {
		lines[i] = dimStyle.Render(line)
	}

	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}

	return strings.Join(lines, "\n")
}

func placeOverlay(fg, bg string, width, height int) string {
	fgLines := strings.Split(fg, "\n")
	bgLines := strings.Split(bg, "\n")

	fgWidth := lipgloss.Width(fg)
	fgHeight := len(fgLines)

	startX := (width - fgWidth) / 2
	startY := (height - fgHeight) / 2

	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}

	for len(bgLines) < height {
		bgLines = append(bgLines, strings.Repeat(" ", width))
	}

	for i, fgLine := range fgLines {
		bgY := startY + i
		if bgY >= len(bgLines) {
			break
		}
		bgLines[bgY] = overlayLine(fgLine, bgLines[bgY], startX)
	}

	return strings.Join(bgLines, "\n")
}

func overlayLine(fgLine, bgLine string, x int) string {
	bgWidth := ansi.StringWidth(bgLine)
	fgWidth := ansi.StringWidth(fgLine)

	if bgWidth < x+fgWidth {
		bgLine += strings.Repeat(" ", x+fgWidth-bgWidth)
	}

	left := ansi.Cut(bgLine, 0, x)
	right := ansi.Cut(bgLine, x+fgWidth, ansi.StringWidth(bgLine))

	return left + fgLine + right
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
		modalWidth = min(modalDefaultWidth, width-modalScreenMargin)
	}
	contentWidth := modalWidth - modalBoxPadding
	contentHeight := height - 10
	return m.Content.SetSize(contentWidth, contentHeight)
}
