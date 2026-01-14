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
	// headerFixedLines: 2 profile/region + 1 separator + 2 summary rows
	headerFixedLines     = 5
	maxFieldValueWidth   = 30
	headerPanelPadding   = 6
	minAvailableWidth    = 40
	profileTruncateWidth = 20
	// profileWidthRatio: profile gets 2/3 of remaining width, region gets 1/3 (compact mode)
	profileWidthRatio = 2
	regionWidthRatio  = 3
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

func (h *HeaderPanel) renderProfileAccountLine() string {
	cfg := config.Global()
	s := h.styles

	labelStr := s.label.Render("Profile: ")
	labelWidth := lipgloss.Width(labelStr)
	availableWidth := h.width - headerPanelPadding - labelWidth

	var profileWithAccount string
	if cfg.IsMultiProfile() {
		selections := cfg.Selections()
		profileWithAccount = formatProfilesWithAccounts(selections, cfg.AccountIDs(), s.value, ui.DangerStyle(), availableWidth)
	} else {
		name := cfg.Selection().DisplayName()
		accID := cmp.Or(cfg.AccountID(), "-")
		profileWithAccount = formatSingleProfile(name, accID, s.value, 0)
	}

	return labelStr + profileWithAccount
}

// renderRegionServiceLine renders line 2: Region on left, Service›Type right-aligned
func (h *HeaderPanel) renderRegionServiceLine(service, resourceType string) string {
	cfg := config.Global()
	s := h.styles

	labelStr := s.label.Render("Region: ")
	labelWidth := lipgloss.Width(labelStr)

	availableWidth := max(h.width-headerPanelPadding, minAvailableWidth)

	var rightPart string
	rightWidth := 0
	if service != "" {
		displayName := registry.Global.GetDisplayName(service)
		rightPart = s.accent.Render(displayName) +
			s.dim.Render(" › ") +
			s.accent.Render(resourceType)
		rightWidth = lipgloss.Width(rightPart)
	}

	minPadding := 2
	regionMaxWidth := availableWidth - labelWidth - rightWidth - minPadding
	regionPart := formatRegions(cfg.Regions(), s.value, regionMaxWidth)
	leftPart := labelStr + regionPart

	if service == "" {
		return leftPart
	}

	leftWidth := lipgloss.Width(leftPart)
	padding := max(minPadding, availableWidth-leftWidth-rightWidth)

	return leftPart + strings.Repeat(" ", padding) + rightPart
}

// formatProfilesWithAccounts formats profiles with account IDs, truncating with (+N) suffix when they don't all fit.
// Note: The first profile is always shown regardless of maxWidth to ensure at least one item is visible.
func formatProfilesWithAccounts(selections []config.ProfileSelection, accountIDs map[string]string, valueStyle, dangerStyle lipgloss.Style, maxWidth int) string {
	if len(selections) == 0 {
		return valueStyle.Render("-")
	}

	separator := valueStyle.Render(", ")
	sepWidth := lipgloss.Width(separator)

	if maxWidth <= 0 && len(selections) > 1 {
		first := selections[0]
		name := first.DisplayName()
		accID := accountIDs[first.ID()]

		var firstPart string
		if accID == "" || accID == "-" {
			firstPart = valueStyle.Render(name+" ") + dangerStyle.Render("(-)")
		} else {
			firstPart = valueStyle.Render(name + " (" + accID + ")")
		}

		suffix := valueStyle.Render("(+" + strconv.Itoa(len(selections)-1) + ")")
		return firstPart + separator + suffix
	}

	parts := make([]string, 0, len(selections))
	currentWidth := 0

	for i, sel := range selections {
		name := sel.DisplayName()
		accID := accountIDs[sel.ID()]

		var part string
		if accID == "" || accID == "-" {
			part = valueStyle.Render(name+" ") + dangerStyle.Render("(-)")
		} else {
			part = valueStyle.Render(name + " (" + accID + ")")
		}

		partWidth := lipgloss.Width(part)

		if maxWidth > 0 && len(parts) > 0 {
			// remainingAfter = items AFTER current (not including current)
			remainingAfter := len(selections) - i - 1
			suffixWidth := 0
			if remainingAfter > 0 {
				// +1 because suffix shows total skipped count (current + remaining)
				suffixWidth = lipgloss.Width("(+" + strconv.Itoa(remainingAfter+1) + ")")
			}

			neededWidth := currentWidth + sepWidth + partWidth
			if remainingAfter > 0 {
				neededWidth += sepWidth + suffixWidth
			}

			if neededWidth > maxWidth {
				skipped := len(selections) - i
				parts = append(parts, valueStyle.Render("(+"+strconv.Itoa(skipped)+")"))
				break
			}
		}

		if len(parts) > 0 {
			currentWidth += sepWidth
		}
		parts = append(parts, part)
		currentWidth += partWidth
	}

	return strings.Join(parts, separator)
}

// formatRegions formats regions with (+N) suffix when they don't all fit.
// Note: The first region is always shown regardless of maxWidth to ensure at least one item is visible.
func formatRegions(regions []string, valueStyle lipgloss.Style, maxWidth int) string {
	if len(regions) == 0 {
		return valueStyle.Render("-")
	}

	if len(regions) == 1 {
		return valueStyle.Render(regions[0])
	}

	if maxWidth <= 0 {
		separator := valueStyle.Render(", ")
		return valueStyle.Render(regions[0]) + separator + valueStyle.Render("(+"+strconv.Itoa(len(regions)-1)+")")
	}

	separator := valueStyle.Render(", ")
	sepWidth := lipgloss.Width(separator)
	parts := make([]string, 0, len(regions))
	currentWidth := 0

	for i, region := range regions {
		part := valueStyle.Render(region)
		partWidth := lipgloss.Width(part)

		if len(parts) > 0 {
			// remainingAfter = items AFTER current (not including current)
			remainingAfter := len(regions) - i - 1
			suffixWidth := 0
			if remainingAfter > 0 {
				// +1 because suffix shows total skipped count (current + remaining)
				suffixWidth = lipgloss.Width("(+" + strconv.Itoa(remainingAfter+1) + ")")
			}

			neededWidth := currentWidth + sepWidth + partWidth
			if remainingAfter > 0 {
				neededWidth += sepWidth + suffixWidth
			}

			if neededWidth > maxWidth {
				skipped := len(regions) - i
				parts = append(parts, valueStyle.Render("(+"+strconv.Itoa(skipped)+")"))
				break
			}
		}

		parts = append(parts, part)
		if len(parts) == 1 {
			currentWidth = partWidth
		} else {
			currentWidth += sepWidth + partWidth
		}
	}

	return strings.Join(parts, separator)
}

// formatSingleProfile formats a single profile with account ID
// truncateWidth: 0 = no truncation, >0 = truncate name to this width
func formatSingleProfile(name, accID string, valueStyle lipgloss.Style, truncateWidth int) string {
	if truncateWidth > 0 {
		name = TruncateString(name, truncateWidth)
	}

	if accID == "-" || accID == "" {
		return valueStyle.Render(name+" ") + ui.DangerStyle().Render("(-)")
	}
	return valueStyle.Render(name + " (" + accID + ")")
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

	lines := []string{
		h.renderProfileAccountLine(),
		h.renderRegionServiceLine("", ""),
	}

	content := strings.Join(lines, "\n")

	panelStyle := h.styles.panel
	if h.width > 4 {
		panelStyle = panelStyle.Width(h.width - 2)
	}

	return panelStyle.Render(content)
}

func (h *HeaderPanel) RenderCompact(service, resourceType string) string {
	cfg := config.Global()
	s := h.styles

	separator := s.dim.Render(" │ ")
	sepWidth := lipgloss.Width(separator)

	availableWidth := max(h.width-headerPanelPadding, minAvailableWidth)

	var servicePart string
	serviceWidth := 0
	if service != "" {
		displayName := registry.Global.GetDisplayName(service)
		servicePart = s.accent.Render(displayName) +
			s.dim.Render(" › ") +
			s.accent.Render(resourceType)
		serviceWidth = lipgloss.Width(servicePart)
	}

	numSeparators := 2
	if servicePart != "" {
		numSeparators = 3
	}
	remainingWidth := availableWidth - serviceWidth - (numSeparators-1)*sepWidth
	profileMaxWidth := remainingWidth * profileWidthRatio / regionWidthRatio
	regionMaxWidth := remainingWidth - profileMaxWidth

	var profilePart string
	if cfg.IsMultiProfile() {
		selections := cfg.Selections()
		profilePart = formatProfilesWithAccounts(selections, cfg.AccountIDs(), s.value, ui.DangerStyle(), profileMaxWidth)
	} else {
		name := cfg.Selection().DisplayName()
		accID := cmp.Or(cfg.AccountID(), "-")
		profilePart = formatSingleProfile(name, accID, s.value, profileTruncateWidth)
	}

	regionPart := formatRegions(cfg.Regions(), s.value, regionMaxWidth)

	var parts []string
	parts = append(parts, profilePart)
	parts = append(parts, regionPart)
	if servicePart != "" {
		parts = append(parts, servicePart)
	}

	content := strings.Join(parts, separator)
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

	sepWidth := max(h.width-headerPanelPadding, minAvailableWidth)
	lines[2] = s.separator.Render(strings.Repeat("─", sepWidth))

	if len(summaryFields) == 0 {
		lines[3] = s.dim.Render("No resource selected")
		lines[4] = ""
	} else {
		availableWidth := max(h.width-headerPanelPadding, minAvailableWidth)

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
