package view

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"

	"github.com/clawscli/claws/internal/ui"
)

// NewTableStyleFunc returns a StyleFunc for lipgloss/table that applies
// consistent styling: header row with TableHeader colors, selected row
// with Selection colors, and normal rows with Text color.
// Pre-computes styles for each column to avoid per-cell allocations.
func NewTableStyleFunc(widths []int, cursor int) func(row, col int) lipgloss.Style {
	th := ui.Current()
	numCols := len(widths)

	headerStyles := make([]lipgloss.Style, numCols)
	selectedStyles := make([]lipgloss.Style, numCols)
	normalStyles := make([]lipgloss.Style, numCols)

	for col, w := range widths {
		base := ui.NoStyle().Width(w)
		if col == 0 {
			base = base.PaddingLeft(1)
		}
		headerStyles[col] = base.Bold(true).Foreground(th.TableHeaderText).Background(th.TableHeader)
		selectedStyles[col] = base.Foreground(th.SelectionText).Background(th.Selection)
		normalStyles[col] = base.Foreground(th.Text)
	}

	return func(row, col int) lipgloss.Style {
		if col >= numCols {
			return ui.NoStyle()
		}
		switch {
		case row == table.HeaderRow:
			return headerStyles[col]
		case row == cursor:
			return selectedStyles[col]
		default:
			return normalStyles[col]
		}
	}
}

// TableBorderStyle returns a style for table borders using the current theme.
func TableBorderStyle() lipgloss.Style {
	return ui.BorderStyle()
}
