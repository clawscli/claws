package view

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/clawscli/claws/internal/ui"
)

const (
	modalBoxPadding     = 6 // border (1*2) + padding (2*2)
	modalScreenMargin   = 10
	modalDefaultWidth   = 60
	modalContentOffsetX = 3
	modalContentOffsetY = 2

	// Modal widths for specific views
	ModalWidthHelp          = 70
	ModalWidthRegion        = 45
	ModalWidthProfile       = 55
	ModalWidthProfileDetail = 65
	ModalWidthActionMenu    = 60
	ModalWidthSettings      = 75
	ModalWidthChat          = 80
)

type Modal struct {
	Content      View
	Width        int
	screenWidth  int
	screenHeight int
}

type ShowModalMsg struct {
	Modal *Modal
}

type HideModalMsg struct{}

type modalStyles struct {
	box lipgloss.Style
}

func newModalStyles() modalStyles {
	return modalStyles{
		box: ui.BoxStyle().Padding(1, 2),
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

func (r *ModalRenderer) ReloadStyles() {
	r.styles = newModalStyles()
}

func (r *ModalRenderer) Render(modal *Modal, bg string, width, height int) string {
	if modal == nil || modal.Content == nil {
		return bg
	}

	content := modal.Content.ViewString()
	boxStyle := r.styles.box

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
	faintStyle := ui.FaintStyle()
	lines := strings.Split(bg, "\n")

	for i, line := range lines {
		lines[i] = faintStyle.Render(line)
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

	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		startX, startY := m.getModalPosition()
		msg.X -= startX + modalContentOffsetX
		msg.Y -= startY + modalContentOffsetY
		model, cmd := m.Content.Update(msg)
		if v, ok := model.(View); ok {
			m.Content = v
		}
		return m, cmd
	case tea.MouseMotionMsg:
		startX, startY := m.getModalPosition()
		msg.X -= startX + modalContentOffsetX
		msg.Y -= startY + modalContentOffsetY
		model, cmd := m.Content.Update(msg)
		if v, ok := model.(View); ok {
			m.Content = v
		}
		return m, cmd
	}

	model, cmd := m.Content.Update(msg)
	if v, ok := model.(View); ok {
		m.Content = v
	}
	return m, cmd
}

func (m *Modal) getModalPosition() (x, y int) {
	if m.Content == nil || m.screenWidth == 0 || m.screenHeight == 0 {
		return 0, 0
	}

	content := m.Content.ViewString()
	modalWidth := m.Width
	if modalWidth == 0 {
		modalWidth = min(lipgloss.Width(content)+modalBoxPadding, m.screenWidth-modalScreenMargin)
	}

	modalVerticalChrome := 4
	fgHeight := strings.Count(content, "\n") + 1 + modalVerticalChrome
	fgWidth := modalWidth + 2

	startX := (m.screenWidth - fgWidth) / 2
	startY := (m.screenHeight - fgHeight) / 2

	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}

	return startX, startY
}

func (m *Modal) SetSize(width, height int) tea.Cmd {
	if m.Content == nil {
		return nil
	}
	m.screenWidth = width
	m.screenHeight = height
	modalWidth := m.Width
	if modalWidth == 0 {
		modalWidth = min(modalDefaultWidth, width-modalScreenMargin)
	}
	contentWidth := modalWidth - modalBoxPadding
	contentHeight := height - 10
	return m.Content.SetSize(contentWidth, contentHeight)
}
