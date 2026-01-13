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
	return headerPanelStyles{
		panel:     ui.BoxStyle(),
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

// renderProfileAccountLine renders line 1: Profile(AccountID) format
func (h *HeaderPanel) renderProfileAccountLine() string {
	cfg := config.Global()
	s := h.styles

	var profileWithAccount string
	if cfg.IsMultiProfile() {
		selections := cfg.Selections()
		profileWithAccount = formatProfilesWithAccounts(selections, cfg.AccountIDs(), ui.DangerStyle())
	} else {
		name := cfg.Selection().DisplayName()
		accID := cmp.Or(cfg.AccountID(), "-")

		var accDisplay string
		if accID == "-" || accID == "" {
			accDisplay = ui.DangerStyle().Render("(-)")
		} else if len(accID) > 12 {
			accDisplay = "(" + accID + ")"
		} else {
			accDisplay = "(" + accID + ")"
		}

		profileWithAccount = name + " " + accDisplay
	}

	return s.label.Render("Profile: ") + s.value.Render(profileWithAccount)
}

// renderRegionServiceLine renders line 2: Region on left, Service›Type right-aligned
func (h *HeaderPanel) renderRegionServiceLine(service, resourceType string) string {
	cfg := config.Global()
	s := h.styles

	regions := cfg.Regions()
	regionDisplay := cmp.Or(strings.Join(regions, ", "), "-")
	leftPart := s.label.Render("Region: ") + s.value.Render(regionDisplay)

	if service == "" {
		return leftPart
	}

	displayName := registry.Global.GetDisplayName(service)
	rightPart := s.accent.Render(displayName) +
		s.dim.Render(" › ") +
		s.accent.Render(resourceType)

	leftWidth := lipgloss.Width(leftPart)
	rightWidth := lipgloss.Width(rightPart)
	availableWidth := h.width - 6
	if availableWidth < 40 {
		availableWidth = 60
	}

	padding := max(2, availableWidth-leftWidth-rightWidth)

	return leftPart + strings.Repeat(" ", padding) + rightPart
}

// formatProfilesWithAccounts formats profiles with account IDs in the format: name1 (acc1), name2 (acc2)
// Connection failures are shown as name (-) in red
func formatProfilesWithAccounts(selections []config.ProfileSelection, accountIDs map[string]string, dangerStyle lipgloss.Style) string {
	const maxShow = 2
	parts := make([]string, 0, len(selections))

	for i, sel := range selections {
		if i >= maxShow {
			remaining := len(selections) - maxShow
			parts = append(parts, "(+"+strconv.Itoa(remaining)+")")
			break
		}

		name := sel.DisplayName()
		accID := accountIDs[sel.ID()]

		// Truncate long account IDs to first 3 digits + "..."
		var accDisplay string
		if accID == "" || accID == "-" {
			// Connection failure - show red (-)
			accDisplay = dangerStyle.Render("(-)")
		} else if len(accID) > 6 {
			accDisplay = "(" + accID[:3] + "...)"
		} else {
			accDisplay = "(" + accID + ")"
		}

		parts = append(parts, name+" "+accDisplay)
	}

	return strings.Join(parts, ", ")
}

// SetWidth sets the panel width
func (h *HeaderPanel) SetWidth(width int) {
	h.width = width
}

func (h *HeaderPanel) ReloadStyles() {
	h.styles = newHeaderPanelStyles()
}

// Height returns the number of lines the rendered header will take
func (h *HeaderPanel) Height(rendered string) int {
	return strings.Count(rendered, "\n") + 1
}

// RenderHome renders a simple header box for the home page (no service/resource info)
func (h *HeaderPanel) RenderHome() string {
	if config.Global().CompactHeader() {
		return h.RenderCompact("", "")
	}

	s := h.styles

	lines := make([]string, headerFixedLines)
	lines[0] = h.renderProfileAccountLine()
	lines[1] = h.renderRegionServiceLine("", "")

	sepWidth := h.width - 6
	if sepWidth < 20 {
		sepWidth = 60
	}
	lines[2] = s.separator.Render(strings.Repeat("─", sepWidth))

	lines[3] = ""
	lines[4] = ""

	content := strings.Join(lines, "\n")

	panelStyle := h.styles.panel
	if h.width > 4 {
		panelStyle = panelStyle.Width(h.width - 2)
	}

	return panelStyle.Render(content)
}

// RenderCompact renders a single-line compact header
// Format: profile(acc) │ region │ Service › Type
func (h *HeaderPanel) RenderCompact(service, resourceType string) string {
	cfg := config.Global()
	s := h.styles

	var profilePart string
	if cfg.IsMultiProfile() {
		selections := cfg.Selections()
		profilePart = formatProfilesWithAccounts(selections, cfg.AccountIDs(), ui.DangerStyle())
	} else {
		name := cfg.Selection().DisplayName()
		accID := cmp.Or(cfg.AccountID(), "-")

		var accDisplay string
		if accID == "-" || accID == "" {
			accDisplay = ui.DangerStyle().Render("(-)")
		} else if len(accID) >= 3 {
			accDisplay = "(" + accID[:3] + "...)"
		} else {
			accDisplay = "(" + accID + ")"
		}

		profilePart = TruncateString(name, 20) + " " + accDisplay
	}

	regions := cfg.Regions()
	regionDisplay := cmp.Or(strings.Join(regions, ", "), "-")
	regionPart := TruncateString(regionDisplay, 30)

	var servicePart string
	if service != "" {
		displayName := registry.Global.GetDisplayName(service)
		servicePart = displayName + " › " + resourceType
	}

	separator := " │ "
	var parts []string
	parts = append(parts, profilePart)
	parts = append(parts, regionPart)
	if servicePart != "" {
		parts = append(parts, servicePart)
	}

	content := strings.Join(parts, separator)

	availableWidth := h.width - 6
	if availableWidth < 40 {
		availableWidth = 60
	}
	content = TruncateString(content, availableWidth)

	panelStyle := s.panel
	if h.width > 4 {
		panelStyle = panelStyle.Width(h.width - 2)
	}

	return panelStyle.Render(content)
}

// Render renders the header panel with fixed height
// service: current service name (e.g., "ec2")
// resourceType: current resource type (e.g., "instances")
// summaryFields: fields from renderer.RenderSummary()
func (h *HeaderPanel) Render(service, resourceType string, summaryFields []render.SummaryField) string {
	if config.Global().CompactHeader() {
		return h.RenderCompact(service, resourceType)
	}

	s := h.styles

	lines := make([]string, headerFixedLines)

	lines[0] = h.renderProfileAccountLine()
	lines[1] = h.renderRegionServiceLine(service, resourceType)

	sepWidth := h.width - 6
	if sepWidth < 20 {
		sepWidth = 60
	}
	lines[2] = s.separator.Render(strings.Repeat("─", sepWidth))

	if len(summaryFields) == 0 {
		lines[3] = s.dim.Render("No resource selected")
		lines[4] = ""
	} else {
		availableWidth := h.width - 6
		if availableWidth < 40 {
			availableWidth = 60
		}

		separator := s.dim.Render("  │  ")
		sepWidth := lipgloss.Width(separator)

		maxRows := 2
		rowIndex := 0
		currentLineWidth := 0
		var currentRow []string

		for _, field := range summaryFields {
			if rowIndex >= maxRows {
				break
			}

			truncatedValue := TruncateString(field.Value, maxFieldValueWidth)

			var styledValue string
			if field.Style.GetForeground() != (lipgloss.NoColor{}) {
				styledValue = field.Style.Render(truncatedValue)
			} else {
				styledValue = s.value.Render(truncatedValue)
			}
			part := s.label.Render(field.Label+": ") + styledValue
			partWidth := lipgloss.Width(part)

			if len(currentRow) > 0 {
				if currentLineWidth+sepWidth+partWidth > availableWidth {
					lines[3+rowIndex] = strings.Join(currentRow, separator)
					currentRow = []string{part}
					currentLineWidth = partWidth
					rowIndex++
					if rowIndex >= maxRows {
						break
					}
				} else {
					currentRow = append(currentRow, part)
					currentLineWidth += sepWidth + partWidth
				}
			} else {
				currentRow = []string{part}
				currentLineWidth = partWidth
			}
		}

		if len(currentRow) > 0 && rowIndex < maxRows {
			lines[3+rowIndex] = strings.Join(currentRow, separator)
			rowIndex++
		}

		for i := 3 + rowIndex; i < headerFixedLines; i++ {
			lines[i] = ""
		}
	}

	content := strings.Join(lines, "\n")

	panelStyle := s.panel
	if h.width > 4 {
		panelStyle = panelStyle.Width(h.width - 2)
	}

	return panelStyle.Render(content)
}
