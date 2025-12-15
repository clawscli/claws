package view

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/render"
	"github.com/clawscli/claws/internal/ui"
	"github.com/mattn/go-runewidth"
)

const (
	// headerFixedLines is the fixed number of content lines in the header panel
	// 1: context line, 1: separator, 3: summary field rows
	headerFixedLines = 5
	// maxFieldValueWidth is the maximum width for a single field value before truncation
	maxFieldValueWidth = 30
)

// truncateValue truncates a string to maxWidth, adding "…" if truncated
func truncateValue(s string, maxWidth int) string {
	if runewidth.StringWidth(s) <= maxWidth {
		return s
	}
	// Truncate to fit maxWidth-1 to leave room for ellipsis
	truncated := runewidth.Truncate(s, maxWidth-1, "")
	return truncated + "…"
}

// HeaderPanel renders the fixed header panel at the top of resource views
// headerPanelStyles holds cached lipgloss styles for performance
type headerPanelStyles struct {
	panel     lipgloss.Style
	label     lipgloss.Style
	value     lipgloss.Style
	accent    lipgloss.Style
	dim       lipgloss.Style
	separator lipgloss.Style
}

func newHeaderPanelStyles() headerPanelStyles {
	t := ui.Current()
	return headerPanelStyles{
		panel:     lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(t.Border).Padding(0, 1),
		label:     lipgloss.NewStyle().Foreground(t.TextDim),
		value:     lipgloss.NewStyle().Foreground(t.Text),
		accent:    lipgloss.NewStyle().Foreground(t.Accent).Bold(true),
		dim:       lipgloss.NewStyle().Foreground(t.TextMuted),
		separator: lipgloss.NewStyle().Foreground(t.Border),
	}
}

type HeaderPanel struct {
	width  int
	styles headerPanelStyles
}

// NewHeaderPanel creates a new HeaderPanel
func NewHeaderPanel() *HeaderPanel {
	return &HeaderPanel{
		styles: newHeaderPanelStyles(),
	}
}

// RenderContextLine renders the AWS account/region context line
// Can be used standalone by other views
func RenderContextLine(service, resourceType string) string {
	cfg := config.Global()
	s := newHeaderPanelStyles()

	profile := cfg.Profile()
	if profile == "" {
		profile = "default"
	}
	accountID := cfg.AccountID()
	if accountID == "" {
		accountID = "-"
	}
	region := cfg.Region()
	if region == "" {
		region = "-"
	}

	return s.label.Render("Profile: ") + s.value.Render(profile) +
		s.dim.Render("  │  ") +
		s.label.Render("Account: ") + s.value.Render(accountID) +
		s.dim.Render("  │  ") +
		s.label.Render("Region: ") + s.value.Render(region) +
		s.dim.Render("  │  ") +
		s.accent.Render(strings.ToUpper(service)) +
		s.dim.Render(" › ") +
		s.accent.Render(resourceType)
}

// SetWidth sets the panel width
func (h *HeaderPanel) SetWidth(width int) {
	h.width = width
}

// Height returns the number of lines the rendered header will take
func (h *HeaderPanel) Height(rendered string) int {
	return strings.Count(rendered, "\n") + 1
}

// RenderHome renders a simple header box for the home page (no service/resource info)
func (h *HeaderPanel) RenderHome() string {
	cfg := config.Global()
	s := h.styles

	profile := cfg.Profile()
	if profile == "" {
		profile = "default"
	}
	accountID := cfg.AccountID()
	if accountID == "" {
		accountID = "-"
	}
	region := cfg.Region()
	if region == "" {
		region = "-"
	}

	contextLine := s.label.Render("Profile: ") + s.value.Render(profile) +
		s.dim.Render("  │  ") +
		s.label.Render("Account: ") + s.value.Render(accountID) +
		s.dim.Render("  │  ") +
		s.label.Render("Region: ") + s.value.Render(region)

	panelStyle := s.panel
	if h.width > 4 {
		panelStyle = panelStyle.Width(h.width - 2)
	}

	return panelStyle.Render(contextLine)
}

// Render renders the header panel with fixed height
// service: current service name (e.g., "ec2")
// resourceType: current resource type (e.g., "instances")
// summaryFields: fields from renderer.RenderSummary()
func (h *HeaderPanel) Render(service, resourceType string, summaryFields []render.SummaryField) string {
	cfg := config.Global()
	s := h.styles

	// Line 1: AWS Context + Current location
	profile := cfg.Profile()
	if profile == "" {
		profile = "default"
	}
	accountID := cfg.AccountID()
	if accountID == "" {
		accountID = "-"
	}
	region := cfg.Region()
	if region == "" {
		region = "-"
	}

	contextLine := s.label.Render("Profile: ") + s.value.Render(profile) +
		s.dim.Render("  │  ") +
		s.label.Render("Account: ") + s.value.Render(accountID) +
		s.dim.Render("  │  ") +
		s.label.Render("Region: ") + s.value.Render(region) +
		s.dim.Render("  │  ") +
		s.accent.Render(strings.ToUpper(service)) +
		s.dim.Render(" › ") +
		s.accent.Render(resourceType)

	// Build content lines (fixed to headerFixedLines)
	lines := make([]string, headerFixedLines)
	lines[0] = contextLine

	// Line 2: Separator
	sepWidth := h.width - 6
	if sepWidth < 20 {
		sepWidth = 60
	}
	lines[1] = s.separator.Render(strings.Repeat("─", sepWidth))

	if len(summaryFields) == 0 {
		// No resource selected - show placeholder on line 3, empty line 4
		lines[2] = s.dim.Render("No resource selected")
		lines[3] = ""
	} else {
		// Render fields in rows (3 fields per row), max 3 rows
		fieldsPerRow := 3
		maxRows := 3
		var currentRow []string
		rowIndex := 0

		for i, field := range summaryFields {
			if rowIndex >= maxRows {
				break // Only show first 2 rows of fields
			}

			// Truncate long values to prevent line wrapping
			truncatedValue := truncateValue(field.Value, maxFieldValueWidth)

			// Format field with appropriate styling
			var styledValue string
			if field.Style.GetForeground() != (lipgloss.NoColor{}) {
				styledValue = field.Style.Render(truncatedValue)
			} else {
				styledValue = s.value.Render(truncatedValue)
			}
			part := s.label.Render(field.Label+": ") + styledValue
			currentRow = append(currentRow, part)

			// Check if we should start a new row
			if len(currentRow) >= fieldsPerRow || i == len(summaryFields)-1 {
				lines[2+rowIndex] = strings.Join(currentRow, s.dim.Render("  │  "))
				currentRow = nil
				rowIndex++
			}
		}

		// Fill remaining lines with empty strings
		for i := 2 + rowIndex; i < headerFixedLines; i++ {
			lines[i] = ""
		}
	}

	// Combine lines
	content := strings.Join(lines, "\n")

	// Apply panel style with width
	panelStyle := s.panel
	if h.width > 4 {
		panelStyle = panelStyle.Width(h.width - 2)
	}

	return panelStyle.Render(content)
}
