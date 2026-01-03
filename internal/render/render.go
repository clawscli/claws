package render

import (
	"fmt"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/ui"
)

// Column defines a table column configuration
type Column struct {
	Name     string
	Width    int
	Getter   func(resource dao.Resource) string
	Style    lipgloss.Style
	Priority int // Lower = more important, shown first when space is limited
}

// SummaryField defines a field in the header summary panel
type SummaryField struct {
	Label string
	Value string
	Style lipgloss.Style // Optional styling for the value
}

// ViewTypeLogView indicates navigation should open a LogView instead of ResourceBrowser
const ViewTypeLogView = "log-view"

// Navigation defines a navigation shortcut to related resources or custom views
type Navigation struct {
	Key            string
	Label          string
	Service        string
	Resource       string
	FilterField    string
	FilterValue    string
	AutoReload     bool
	ReloadInterval time.Duration
	ViewType       string
}

// Renderer defines the interface for rendering resources in table format
type Renderer interface {
	// ServiceName returns the AWS service name
	ServiceName() string

	// ResourceType returns the resource type
	ResourceType() string

	// Columns returns the column definitions for this resource type
	Columns() []Column

	// RenderRow renders a single resource row
	RenderRow(resource dao.Resource, columns []Column) []string

	// RenderDetail renders detailed view of a single resource
	RenderDetail(resource dao.Resource) string

	// RenderSummary returns summary fields for the header panel
	// These are displayed when a resource is selected
	RenderSummary(resource dao.Resource) []SummaryField
}

// Navigator is an optional interface that renderers can implement to provide navigation shortcuts
type Navigator interface {
	// Navigations returns available navigation shortcuts for a resource
	// The resource parameter is used to extract filter values
	Navigations(resource dao.Resource) []Navigation
}

// MetricSpecProvider is an optional interface for renderers that support inline metrics.
type MetricSpecProvider interface {
	MetricSpec() *MetricSpec
}

// MetricSpec defines which CloudWatch metric to fetch for inline display.
type MetricSpec struct {
	Namespace     string
	MetricName    string
	DimensionName string
	Stat          string
	ColumnHeader  string
	Unit          string // Display unit (e.g., "%", "", "ms"). Empty for count-based metrics.
}

// BaseRenderer provides a default implementation
type BaseRenderer struct {
	Service  string
	Resource string
	Cols     []Column
}

func (r *BaseRenderer) ServiceName() string  { return r.Service }
func (r *BaseRenderer) ResourceType() string { return r.Resource }
func (r *BaseRenderer) Columns() []Column    { return r.Cols }

func (r *BaseRenderer) RenderRow(resource dao.Resource, columns []Column) []string {
	row := make([]string, len(columns))
	for i, col := range columns {
		if col.Getter != nil {
			row[i] = col.Getter(resource)
		}
	}
	return row
}

func (r *BaseRenderer) RenderDetail(resource dao.Resource) string {
	return ""
}

func (r *BaseRenderer) RenderSummary(resource dao.Resource) []SummaryField {
	// Default implementation returns ID and Name
	fields := []SummaryField{
		{Label: "ID", Value: resource.GetID()},
	}
	if name := resource.GetName(); name != "" && name != resource.GetID() {
		fields = append(fields, SummaryField{Label: "Name", Value: name})
	}
	return fields
}

// Colorer is a function that applies styling based on value
type Colorer func(value string) lipgloss.Style

// StateColorer returns a colorer for common state values
func StateColorer() Colorer {
	return func(value string) lipgloss.Style {
		switch value {
		case "running", "available", "active", "healthy":
			return ui.SuccessStyle()
		case "in-use", "attached":
			return ui.InfoStyle()
		case "stopped", "stopping", "deleting":
			return ui.WarningStyle()
		case "terminated", "failed", "error", "unhealthy", "deleted":
			return ui.DangerStyle()
		case "pending", "starting", "creating":
			return ui.PendingStyle()
		default:
			return lipgloss.NewStyle()
		}
	}
}

// Factory creates Renderer instances
type Factory func() Renderer

// FormatAge formats a time.Time as a human-readable age string
func FormatAge(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	d := time.Since(t)

	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	days := int(d.Hours() / 24)
	if days < 30 {
		return fmt.Sprintf("%dd", days)
	}
	if days < 365 {
		return fmt.Sprintf("%dmo", days/30)
	}
	return fmt.Sprintf("%dy", days/365)
}

// FormatDuration formats a duration as a human-readable string
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		if secs > 0 {
			return fmt.Sprintf("%dm%ds", mins, secs)
		}
		return fmt.Sprintf("%dm", mins)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	if mins > 0 {
		return fmt.Sprintf("%dh%dm", hours, mins)
	}
	return fmt.Sprintf("%dh", hours)
}

// Style is an alias for lipgloss.Style for convenience
type Style = lipgloss.Style

// DefaultStyle returns a default unstyled style
func DefaultStyle() lipgloss.Style {
	return lipgloss.NewStyle()
}

// FormatTags formats tags for table display
// It shows the most important tags first (Name is excluded since it's usually shown separately)
func FormatTags(tags map[string]string, maxLen int) string {
	if len(tags) == 0 {
		return ""
	}

	// Priority tags to show first
	priority := []string{"Environment", "Env", "Project", "Team", "Owner", "Application", "App"}
	prioritySet := make(map[string]struct{}, len(priority))
	for _, key := range priority {
		prioritySet[key] = struct{}{}
	}

	var parts []string

	// Add priority tags first
	for _, key := range priority {
		if val, ok := tags[key]; ok {
			parts = append(parts, key+"="+val)
		}
	}

	// Add remaining tags (excluding Name which is usually shown separately)
	for k, v := range tags {
		if k == "Name" {
			continue
		}
		// Skip if already added from priority (O(1) lookup)
		if _, isPriority := prioritySet[k]; !isPriority {
			parts = append(parts, k+"="+v)
		}
	}

	result := ""
	for i, part := range parts {
		if i > 0 {
			result += ", "
		}
		if len(result)+len(part) > maxLen-3 {
			result += "..."
			break
		}
		result += part
	}
	return result
}

// TagsColumn returns a Column definition for displaying tags
func TagsColumn(width int, priority int) Column {
	return Column{
		Name:  "TAGS",
		Width: width,
		Getter: func(r dao.Resource) string {
			return FormatTags(r.GetTags(), width)
		},
		Priority: priority,
	}
}

// FormatSize formats bytes as a human-readable size string
func FormatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.1f TiB", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.1f GiB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1f MiB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1f KiB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
