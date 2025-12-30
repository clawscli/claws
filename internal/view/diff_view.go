package view

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
	"github.com/clawscli/claws/internal/ui"
)

// DiffView displays side-by-side comparison of two resources
type DiffView struct {
	ctx          context.Context
	left         dao.Resource
	right        dao.Resource
	renderer     render.Renderer
	service      string
	resourceType string
	viewport     viewport.Model
	ready        bool
	width        int
	height       int
	styles       diffViewStyles
}

type diffViewStyles struct {
	title     lipgloss.Style
	header    lipgloss.Style
	content   lipgloss.Style
	separator lipgloss.Style
}

func newDiffViewStyles() diffViewStyles {
	t := ui.Current()
	return diffViewStyles{
		title:     lipgloss.NewStyle().Bold(true).Foreground(t.Primary),
		header:    lipgloss.NewStyle().Bold(true).Foreground(t.Secondary),
		content:   lipgloss.NewStyle().Foreground(t.Text),
		separator: lipgloss.NewStyle().Foreground(t.TableBorder),
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
		if IsEscKey(msg) || msg.String() == "q" {
			return d, nil // Let app handle back navigation
		}
	}

	var cmd tea.Cmd
	d.viewport, cmd = d.viewport.Update(msg)
	return d, cmd
}

// ViewString returns the view content as a string
func (d *DiffView) ViewString() string {
	if !d.ready {
		return "Loading..."
	}

	return d.viewport.View()
}

// View implements tea.Model
func (d *DiffView) View() tea.View {
	return tea.NewView(d.ViewString())
}

// SetSize implements View
func (d *DiffView) SetSize(width, height int) tea.Cmd {
	d.width = width
	d.height = height

	// Reserve space for header
	headerHeight := 3
	viewportHeight := height - headerHeight
	if viewportHeight < 5 {
		viewportHeight = 5
	}

	if !d.ready {
		d.viewport = viewport.New(viewport.WithWidth(width), viewport.WithHeight(viewportHeight))
		d.ready = true
	} else {
		d.viewport.SetWidth(width)
		d.viewport.SetHeight(viewportHeight)
	}

	content := d.renderSideBySide()
	d.viewport.SetContent(content)

	return nil
}

// StatusLine implements View
func (d *DiffView) StatusLine() string {
	return d.left.GetName() + " vs " + d.right.GetName() + " • ↑/↓:scroll • esc:back"
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
		leftDetail = d.renderer.RenderDetail(d.left)
		rightDetail = d.renderer.RenderDetail(d.right)
	}

	// Split into lines
	leftLines := strings.Split(leftDetail, "\n")
	rightLines := strings.Split(rightDetail, "\n")

	// Calculate column width (half of available width minus separator)
	colWidth := (d.width - 3) / 2

	// Column headers
	leftHeader := truncateOrPad("◀ "+d.left.GetName(), colWidth)
	rightHeader := truncateOrPad(d.right.GetName()+" ▶", colWidth)
	out.WriteString(s.header.Render(leftHeader))
	out.WriteString(s.separator.Render(" │ "))
	out.WriteString(s.header.Render(rightHeader))
	out.WriteString("\n")
	out.WriteString(strings.Repeat("─", colWidth))
	out.WriteString("─┼─")
	out.WriteString(strings.Repeat("─", colWidth))
	out.WriteString("\n")

	// Render side by side
	maxLines := len(leftLines)
	if len(rightLines) > maxLines {
		maxLines = len(rightLines)
	}

	for i := 0; i < maxLines; i++ {
		leftLine := ""
		rightLine := ""

		if i < len(leftLines) {
			leftLine = leftLines[i]
		}
		if i < len(rightLines) {
			rightLine = rightLines[i]
		}

		out.WriteString(truncateOrPad(leftLine, colWidth))
		out.WriteString(s.separator.Render(" │ "))
		out.WriteString(truncateOrPad(rightLine, colWidth))
		out.WriteString("\n")
	}

	return out.String()
}

// truncateOrPad ensures a string is exactly the specified width
func truncateOrPad(s string, width int) string {
	if width <= 0 {
		return ""
	}

	// Use lipgloss.Width for proper ANSI-aware width calculation
	plainLen := lipgloss.Width(s)

	if plainLen > width {
		// Use ansi.Truncate for proper ANSI-aware truncation
		return ansi.Truncate(s, width, "…")
	}

	// Pad with spaces
	return s + strings.Repeat(" ", width-plainLen)
}
