package view

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	added     lipgloss.Style
	removed   lipgloss.Style
	unchanged lipgloss.Style
	separator lipgloss.Style
}

func newDiffViewStyles() diffViewStyles {
	t := ui.Current()
	return diffViewStyles{
		title:     lipgloss.NewStyle().Bold(true).Foreground(t.Primary),
		header:    lipgloss.NewStyle().Bold(true).Foreground(t.Secondary),
		added:     lipgloss.NewStyle().Foreground(t.Success),
		removed:   lipgloss.NewStyle().Foreground(t.Danger),
		unchanged: lipgloss.NewStyle().Foreground(t.TextDim),
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
	case tea.KeyMsg:
		if IsEscKey(msg) || msg.String() == "q" {
			return d, nil // Let app handle back navigation
		}
	}

	var cmd tea.Cmd
	d.viewport, cmd = d.viewport.Update(msg)
	return d, cmd
}

// View implements tea.Model
func (d *DiffView) View() string {
	if !d.ready {
		return "Loading..."
	}

	return d.viewport.View()
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
		d.viewport = viewport.New(width, viewportHeight)
		d.ready = true
	} else {
		d.viewport.Width = width
		d.viewport.Height = viewportHeight
	}

	content := d.renderDiff()
	d.viewport.SetContent(content)

	return nil
}

// StatusLine implements View
func (d *DiffView) StatusLine() string {
	return "↑/↓:scroll • esc:back"
}

// renderDiff generates the side-by-side diff view
func (d *DiffView) renderDiff() string {
	s := d.styles
	var out strings.Builder

	// Header
	out.WriteString(s.title.Render("Diff: "+d.resourceType) + "\n")
	out.WriteString(s.header.Render("← "+d.left.GetName()) + "  " + s.separator.Render("│") + "  " + s.header.Render(d.right.GetName()+" →") + "\n")
	out.WriteString(strings.Repeat("─", d.width) + "\n\n")

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

	// Compute diff and render side-by-side
	diff := computeLineDiff(leftLines, rightLines)
	for _, entry := range diff {
		leftLine := truncateOrPad(entry.Left, colWidth)
		rightLine := truncateOrPad(entry.Right, colWidth)

		switch entry.Type {
		case diffEqual:
			out.WriteString(s.unchanged.Render(leftLine))
			out.WriteString(s.separator.Render(" │ "))
			out.WriteString(s.unchanged.Render(rightLine))
		case diffRemoved:
			out.WriteString(s.removed.Render(leftLine))
			out.WriteString(s.separator.Render(" │ "))
			out.WriteString(strings.Repeat(" ", colWidth))
		case diffAdded:
			out.WriteString(strings.Repeat(" ", colWidth))
			out.WriteString(s.separator.Render(" │ "))
			out.WriteString(s.added.Render(rightLine))
		case diffChanged:
			out.WriteString(s.removed.Render(leftLine))
			out.WriteString(s.separator.Render(" │ "))
			out.WriteString(s.added.Render(rightLine))
		}
		out.WriteString("\n")
	}

	return out.String()
}

// diffType represents the type of difference
type diffType int

const (
	diffEqual diffType = iota
	diffAdded
	diffRemoved
	diffChanged
)

// diffEntry represents a single diff line
type diffEntry struct {
	Type  diffType
	Left  string
	Right string
}

// computeLineDiff computes line-by-line diff between two sets of lines
// This is a simple implementation; Phase 2 will use jd for JSON blocks
func computeLineDiff(left, right []string) []diffEntry {
	// Use LCS (Longest Common Subsequence) algorithm for proper diff
	// For now, use a simple line-by-line comparison
	result := []diffEntry{}

	// Build a map of right lines for quick lookup
	rightMap := make(map[string][]int)
	for i, line := range right {
		rightMap[line] = append(rightMap[line], i)
	}

	// Simple diff: compare line by line
	maxLen := len(left)
	if len(right) > maxLen {
		maxLen = len(right)
	}

	for i := 0; i < maxLen; i++ {
		var leftLine, rightLine string
		hasLeft := i < len(left)
		hasRight := i < len(right)

		if hasLeft {
			leftLine = left[i]
		}
		if hasRight {
			rightLine = right[i]
		}

		if hasLeft && hasRight {
			if leftLine == rightLine {
				result = append(result, diffEntry{Type: diffEqual, Left: leftLine, Right: rightLine})
			} else {
				result = append(result, diffEntry{Type: diffChanged, Left: leftLine, Right: rightLine})
			}
		} else if hasLeft {
			result = append(result, diffEntry{Type: diffRemoved, Left: leftLine})
		} else {
			result = append(result, diffEntry{Type: diffAdded, Right: rightLine})
		}
	}

	return result
}

// truncateOrPad ensures a string is exactly the specified width
func truncateOrPad(s string, width int) string {
	// Remove ANSI escape codes for length calculation
	plainLen := lipgloss.Width(s)

	if plainLen > width {
		// Truncate with ellipsis
		runes := []rune(s)
		if len(runes) > width-1 {
			return string(runes[:width-1]) + "…"
		}
		return s
	}

	// Pad with spaces
	return s + strings.Repeat(" ", width-plainLen)
}
