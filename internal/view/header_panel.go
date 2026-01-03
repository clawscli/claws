package view

import (
	"cmp"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
	"github.com/clawscli/claws/internal/ui"
)

const (
	// headerFixedLines is the fixed number of content lines in the header panel
	// 1: context line, 1: separator, 3: summary field rows
	headerFixedLines = 5
	// maxFieldValueWidth is the maximum width for a single field value before truncation
	maxFieldValueWidth = 30
)

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
		label:     ui.DimStyle(),
		value:     ui.TextStyle(),
		accent:    ui.HighlightStyle(),
		dim:       ui.MutedStyle(),
		separator: ui.BorderStyle(),
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

func (h *HeaderPanel) renderContextLine(service, resourceType string) string {
	cfg := config.Global()
	s := h.styles

	var profileDisplay, accountDisplay string
	if cfg.IsMultiProfile() {
		selections := cfg.Selections()
		profileDisplay = formatMultiProfiles(selections)
		accountDisplay = formatMultiAccounts(selections, cfg.AccountIDs())
	} else {
		profileDisplay = cfg.Selection().DisplayName()
		accountDisplay = cmp.Or(cfg.AccountID(), "-")
	}

	regions := cfg.Regions()
	regionDisplay := cmp.Or(strings.Join(regions, ", "), "-")

	line := s.label.Render("Profile: ") + s.value.Render(profileDisplay) +
		s.dim.Render("  │  ") +
		s.label.Render("Account: ") + s.value.Render(accountDisplay) +
		s.dim.Render("  │  ") +
		s.label.Render("Region: ") + s.value.Render(regionDisplay)

	if service != "" {
		displayName := registry.Global.GetDisplayName(service)
		line += s.dim.Render("  │  ") +
			s.accent.Render(displayName) +
			s.dim.Render(" › ") +
			s.accent.Render(resourceType)
	}

	return line
}

func formatMultiProfiles(selections []config.ProfileSelection) string {
	const maxShow = 2
	if len(selections) <= maxShow {
		names := make([]string, len(selections))
		for i, sel := range selections {
			names[i] = sel.DisplayName()
		}
		return strings.Join(names, ", ")
	}
	names := make([]string, maxShow)
	for i := range maxShow {
		names[i] = selections[i].DisplayName()
	}
	return strings.Join(names, ", ") + " (+" + strconv.Itoa(len(selections)-maxShow) + ")"
}

func formatMultiAccounts(selections []config.ProfileSelection, accountIDs map[string]string) string {
	const maxShow = 2
	accounts := make([]string, 0, len(selections))
	for _, sel := range selections {
		if acc := accountIDs[sel.ID()]; acc != "" {
			accounts = append(accounts, acc)
		}
	}
	if len(accounts) == 0 {
		return "-"
	}
	if len(accounts) <= maxShow {
		return strings.Join(accounts, ", ")
	}
	return strings.Join(accounts[:maxShow], ", ") + " (+" + strconv.Itoa(len(accounts)-maxShow) + ")"
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
	contextLine := h.renderContextLine("", "")

	panelStyle := h.styles.panel
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
	s := h.styles

	// Build content lines (fixed to headerFixedLines)
	lines := make([]string, headerFixedLines)
	lines[0] = h.renderContextLine(service, resourceType)

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
				break // Only show first 3 rows of fields
			}

			// Truncate long values to prevent line wrapping
			truncatedValue := TruncateString(field.Value, maxFieldValueWidth)

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
