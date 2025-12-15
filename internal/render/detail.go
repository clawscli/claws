package render

import (
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/ui"
)

// DetailStyles contains common styles for detail views
// cachedDetailStyles holds the cached default styles
var cachedDetailStyles *DetailStyles

type DetailStyles struct {
	Title   lipgloss.Style
	Section lipgloss.Style
	Label   lipgloss.Style
	Value   lipgloss.Style
	Dim     lipgloss.Style
	Success lipgloss.Style
}

// DefaultDetailStyles returns the default styles for detail views
func DefaultDetailStyles() DetailStyles {
	if cachedDetailStyles != nil {
		return *cachedDetailStyles
	}
	t := ui.Current()
	styles := DetailStyles{
		Title:   lipgloss.NewStyle().Bold(true).Foreground(t.Primary),
		Section: lipgloss.NewStyle().Bold(true).Foreground(t.Secondary).MarginTop(1),
		Label:   lipgloss.NewStyle().Foreground(t.TextDim).Width(20),
		Value:   lipgloss.NewStyle().Foreground(t.Text),
		Dim:     lipgloss.NewStyle().Foreground(t.TextDim),
		Success: lipgloss.NewStyle().Foreground(t.Success),
	}
	cachedDetailStyles = &styles
	return styles
}

// DetailBuilder helps construct detail views with consistent styling
type DetailBuilder struct {
	styles DetailStyles
	sb     strings.Builder
}

// NewDetailBuilder creates a new DetailBuilder with default styles
func NewDetailBuilder() *DetailBuilder {
	return &DetailBuilder{
		styles: DefaultDetailStyles(),
	}
}

// Title adds a title line
func (d *DetailBuilder) Title(resourceType, name string) *DetailBuilder {
	d.sb.WriteString(d.styles.Title.Render(resourceType+": "+name) + "\n\n")
	return d
}

// Section adds a section header
func (d *DetailBuilder) Section(name string) *DetailBuilder {
	d.sb.WriteString("\n" + d.styles.Section.Render(name) + "\n")
	return d
}

// Field adds a label: value line
func (d *DetailBuilder) Field(label, value string) *DetailBuilder {
	d.sb.WriteString(d.styles.Label.Render(label+":") + d.styles.Value.Render(value) + "\n")
	return d
}

// FieldStyled adds a label: value line with custom value styling
func (d *DetailBuilder) FieldStyled(label, value string, style lipgloss.Style) *DetailBuilder {
	d.sb.WriteString(d.styles.Label.Render(label+":") + style.Render(value) + "\n")
	return d
}

// FieldIf adds a field only if the pointer is not nil
func (d *DetailBuilder) FieldIf(label string, ptr *string) *DetailBuilder {
	if ptr != nil && *ptr != "" {
		d.Field(label, *ptr)
	}
	return d
}

// Line adds a raw line
func (d *DetailBuilder) Line(text string) *DetailBuilder {
	d.sb.WriteString(text + "\n")
	return d
}

// Dim adds a dimmed text line
func (d *DetailBuilder) Dim(text string) *DetailBuilder {
	d.sb.WriteString(d.styles.Dim.Render(text) + "\n")
	return d
}

// DimIndent adds a dimmed indented text line
func (d *DetailBuilder) DimIndent(text string) *DetailBuilder {
	d.sb.WriteString("  " + d.styles.Dim.Render(text) + "\n")
	return d
}

// Tag adds a single tag line (key: value format)
func (d *DetailBuilder) Tag(key, value string) *DetailBuilder {
	d.sb.WriteString("  " + d.styles.Dim.Render(key+":") + " " + d.styles.Value.Render(value) + "\n")
	return d
}

// Tags renders a "Tags" section with all tags from a map.
// Keys are sorted alphabetically for consistent display.
// Tags are hidden in demo mode to avoid exposing sensitive information.
func (d *DetailBuilder) Tags(tags map[string]string) *DetailBuilder {
	if len(tags) == 0 || config.Global().DemoMode() {
		return d
	}
	d.Section("Tags")
	// Sort keys for consistent display
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, key := range keys {
		d.Tag(key, tags[key])
	}
	return d
}

// Styles returns the styles for custom rendering
func (d *DetailBuilder) Styles() DetailStyles {
	return d.styles
}

// String returns the built detail view
func (d *DetailBuilder) String() string {
	return d.sb.String()
}
