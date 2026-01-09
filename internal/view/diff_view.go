package view

import (
	"context"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
	"github.com/clawscli/claws/internal/ui"
)

type DiffView struct {
	ctx          context.Context
	left         dao.Resource
	right        dao.Resource
	renderer     render.Renderer
	service      string
	resourceType string
	vp           ViewportState
	width        int
	styles       diffViewStyles
}

type diffViewStyles struct {
	title     lipgloss.Style
	header    lipgloss.Style
	separator lipgloss.Style
}

func newDiffViewStyles() diffViewStyles {
	return diffViewStyles{
		title:     ui.TitleStyle(),
		header:    ui.SectionStyle(),
		separator: ui.MutedStyle(),
	}
}

// NewDiffView creates a new DiffView for comparing two resources
func NewDiffView(ctx context.Context, left, right dao.Resource, renderer render.Renderer, service, resourceType string) *DiffView {
	return &DiffView{
		ctx:          ctx,
		left:         left,
		right:        right,
		renderer:     renderer,
		service:      service,
		resourceType: resourceType,
		styles:       newDiffViewStyles(),
	}
}

// Init implements tea.Model
func (d *DiffView) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (d *DiffView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		// Let app handle back navigation (esc/backspace/q handled by app.go)
		if IsEscKey(msg) {
			return d, nil
		}
	case ThemeChangedMsg:
		d.styles = newDiffViewStyles()
		if d.vp.Ready {
			d.vp.Model.SetContent(d.renderSideBySide())
		}
		return d, nil
	}

	var cmd tea.Cmd
	d.vp.Model, cmd = d.vp.Model.Update(msg)
	return d, cmd
}

func (d *DiffView) ViewString() string {
	if !d.vp.Ready {
		return LoadingMessage
	}

	return d.vp.Model.View()
}

// View implements tea.Model
func (d *DiffView) View() tea.View {
	return tea.NewView(d.ViewString())
}

// SetSize implements View
func (d *DiffView) SetSize(width, height int) tea.Cmd {
	d.width = width

	// Reserve space for header
	headerHeight := 3
	viewportHeight := max(height-headerHeight, 5)

	d.vp.SetSize(width, viewportHeight)

	content := d.renderSideBySide()
	d.vp.Model.SetContent(content)

	return nil
}

// StatusLine implements View
func (d *DiffView) StatusLine() string {
	return dao.UnwrapResource(d.left).GetName() + " vs " + dao.UnwrapResource(d.right).GetName() + " • ↑/↓:scroll • q/esc:back"
}

// renderSideBySide generates the side-by-side view
func (d *DiffView) renderSideBySide() string {
	s := d.styles
	var out strings.Builder

	// Header
	out.WriteString(s.title.Render("Compare: "+d.resourceType) + "\n")
	out.WriteString(strings.Repeat("─", d.width) + "\n")

	// Get rendered detail for both resources
	leftDetail := ""
	rightDetail := ""
	if d.renderer != nil {
		leftDetail = d.renderer.RenderDetail(dao.UnwrapResource(d.left))
		rightDetail = d.renderer.RenderDetail(dao.UnwrapResource(d.right))
	}

	// Split into lines
	leftLines := strings.Split(leftDetail, "\n")
	rightLines := strings.Split(rightDetail, "\n")

	// Calculate column width (half of available width minus separator)
	colWidth := (d.width - 3) / 2

	// Column headers
	leftHeader := TruncateOrPadString("◀ "+dao.UnwrapResource(d.left).GetName(), colWidth)
	rightHeader := TruncateOrPadString(dao.UnwrapResource(d.right).GetName()+" ▶", colWidth)
	out.WriteString(s.header.Render(leftHeader))
	out.WriteString(s.separator.Render(" │ "))
	out.WriteString(s.header.Render(rightHeader))
	out.WriteString("\n")
	out.WriteString(strings.Repeat("─", colWidth))
	out.WriteString("─┼─")
	out.WriteString(strings.Repeat("─", colWidth))
	out.WriteString("\n")

	// Render side by side
	maxLines := max(len(leftLines), len(rightLines))

	for i := range maxLines {
		leftLine := ""
		rightLine := ""

		if i < len(leftLines) {
			leftLine = leftLines[i]
		}
		if i < len(rightLines) {
			rightLine = rightLines[i]
		}

		out.WriteString(TruncateOrPadString(leftLine, colWidth))
		out.WriteString(s.separator.Render(" │ "))
		out.WriteString(TruncateOrPadString(rightLine, colWidth))
		out.WriteString("\n")
	}

	return out.String()
}

func (d *DiffView) Left() dao.Resource   { return d.left }
func (d *DiffView) Right() dao.Resource  { return d.right }
func (d *DiffView) Service() string      { return d.service }
func (d *DiffView) ResourceType() string { return d.resourceType }
