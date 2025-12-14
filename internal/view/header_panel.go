package view

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/render"
	"github.com/clawscli/claws/internal/ui"
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

// Render renders the header panel
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

	// Build content
	var lines []string
	lines = append(lines, contextLine)

	if len(summaryFields) == 0 {
		lines = append(lines, s.dim.Render("No resource selected"))
	} else {
		// Add separator
		sepWidth := h.width - 6
		if sepWidth < 20 {
			sepWidth = 60
		}
		lines = append(lines, s.separator.Render(strings.Repeat("─", sepWidth)))

		// Render fields in rows (3 fields per row for better readability)
		fieldsPerRow := 3
		var currentRow []string

		for i, field := range summaryFields {
			// Format field with appropriate styling
			var styledValue string
			if field.Style.GetForeground() != (lipgloss.NoColor{}) {
				styledValue = field.Style.Render(field.Value)
			} else {
				styledValue = s.value.Render(field.Value)
			}
			part := s.label.Render(field.Label+": ") + styledValue
			currentRow = append(currentRow, part)

			// Check if we should start a new row
			if len(currentRow) >= fieldsPerRow || i == len(summaryFields)-1 {
				lines = append(lines, strings.Join(currentRow, s.dim.Render("  │  ")))
				currentRow = nil
			}
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
