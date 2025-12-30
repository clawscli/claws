package view

import (
	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"

	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/metrics"
	"github.com/clawscli/claws/internal/render"
	"github.com/clawscli/claws/internal/ui"
)

// buildTable constructs the table component with columns, rows, and styling.
// It handles mark indicator, region column (multi-region), and metrics column.
func (r *ResourceBrowser) buildTable() {
	if r.renderer == nil {
		return
	}

	currentCursor := r.table.Cursor()
	cols := r.renderer.Columns()

	const markColWidth = 2
	const regionColWidth = 14
	metricsColWidth := metrics.ColumnWidth

	effectiveMetricsEnabled := r.metricsEnabled && r.getMetricSpec() != nil
	isMultiRegion := config.Global().IsMultiRegion()

	numCols := len(cols) + 1
	if isMultiRegion {
		numCols++
	}
	if effectiveMetricsEnabled {
		numCols++
	}
	tableCols := make([]table.Column, numCols)
	tableCols[0] = table.Column{Title: " ", Width: markColWidth}

	totalColWidth := markColWidth
	for _, col := range cols {
		totalColWidth += col.Width
	}
	if isMultiRegion {
		totalColWidth += regionColWidth
	}
	if effectiveMetricsEnabled {
		totalColWidth += metricsColWidth
	}

	extraWidth := r.width - totalColWidth
	if extraWidth < 0 {
		extraWidth = 0
	}

	colIdx := 1
	for i, col := range cols {
		title := col.Name + r.getSortIndicator(i)
		width := col.Width
		if i == len(cols)-1 && !isMultiRegion && !effectiveMetricsEnabled {
			width += extraWidth
		}
		tableCols[colIdx] = table.Column{
			Title: title,
			Width: width,
		}
		colIdx++
	}

	if isMultiRegion {
		width := regionColWidth
		if !effectiveMetricsEnabled {
			width += extraWidth
		}
		tableCols[colIdx] = table.Column{
			Title: "REGION",
			Width: width,
		}
		colIdx++
	}

	if effectiveMetricsEnabled {
		spec := r.getMetricSpec()
		header := "METRICS"
		if spec != nil {
			header = spec.ColumnHeader
		}
		tableCols[colIdx] = table.Column{
			Title: header,
			Width: metricsColWidth + extraWidth,
		}
	}

	rows := make([]table.Row, len(r.filtered))
	for i, res := range r.filtered {
		row := r.renderer.RenderRow(dao.UnwrapResource(res), cols)
		markIndicator := "  "
		if r.markedResource != nil && r.markedResource.GetID() == res.GetID() {
			markIndicator = "◆ "
		}
		fullRow := make(table.Row, numCols)
		fullRow[0] = markIndicator
		copy(fullRow[1:], row)

		rowIdx := len(cols) + 1
		if isMultiRegion {
			fullRow[rowIdx] = dao.GetResourceRegion(res)
			rowIdx++
		}
		if effectiveMetricsEnabled && r.metricsData != nil {
			unit := ""
			if r.metricsData.Spec != nil {
				unit = r.metricsData.Spec.Unit
			}
			fullRow[rowIdx] = metrics.RenderSparkline(r.metricsData.Get(res.GetID()), unit)
		} else if effectiveMetricsEnabled {
			fullRow[rowIdx] = metrics.RenderSparkline(nil, "")
		}
		rows[i] = fullRow
	}

	// Calculate header height dynamically
	var summaryFields []render.SummaryField
	if len(r.filtered) > 0 && currentCursor >= 0 && currentCursor < len(r.filtered) {
		summaryFields = r.renderer.RenderSummary(dao.UnwrapResource(r.filtered[currentCursor]))
	}
	headerStr := r.headerPanel.Render(r.service, r.resourceType, summaryFields)
	headerHeight := r.headerPanel.Height(headerStr)

	// height - header - tabs(1)
	tableHeight := r.height - headerHeight - 1
	if tableHeight < 5 {
		tableHeight = 5
	}

	t := table.New(
		table.WithColumns(tableCols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(tableHeight),
		table.WithWidth(r.width),
	)

	th := ui.Current()
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(th.TableBorder).
		BorderBottom(true).
		Bold(true).
		Foreground(th.TableHeaderText).
		Background(th.TableHeader)
	s.Selected = s.Selected.
		Foreground(th.SelectionText).
		Background(th.Selection).
		Bold(false)
	// Note: Not setting s.Cell foreground - let Selected style take precedence
	t.SetStyles(s)

	// Restore cursor position (clamped to valid range)
	if len(rows) > 0 {
		if currentCursor >= len(rows) {
			currentCursor = len(rows) - 1
		}
		if currentCursor < 0 {
			currentCursor = 0
		}
		t.SetCursor(currentCursor)
	}

	r.table = t
}
