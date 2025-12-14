package quotas

import (
	"fmt"
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// Ensure QuotaRenderer implements render.Navigator
var _ render.Navigator = (*QuotaRenderer)(nil)

// QuotaRenderer renders Service Quotas quotas
type QuotaRenderer struct {
	render.BaseRenderer
}

// NewQuotaRenderer creates a new QuotaRenderer
func NewQuotaRenderer() render.Renderer {
	return &QuotaRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "service-quotas",
			Resource: "quotas",
			Cols: []render.Column{
				{
					Name:  "QUOTA NAME",
					Width: 50,
					Getter: func(r dao.Resource) string {
						if qr, ok := r.(*QuotaResource); ok {
							return qr.QuotaName()
						}
						return ""
					},
					Priority: 0,
				},
				{
					Name:  "VALUE",
					Width: 15,
					Getter: func(r dao.Resource) string {
						if qr, ok := r.(*QuotaResource); ok {
							return formatValue(qr.Value(), qr.Unit())
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "ADJUSTABLE",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if qr, ok := r.(*QuotaResource); ok {
							if qr.Adjustable() {
								return "Yes"
							}
							return "No"
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "GLOBAL",
					Width: 8,
					Getter: func(r dao.Resource) string {
						if qr, ok := r.(*QuotaResource); ok {
							if qr.GlobalQuota() {
								return "Yes"
							}
							return "-"
						}
						return ""
					},
					Priority: 3,
				},
			},
		},
	}
}

// formatValue formats a quota value with its unit
func formatValue(value float64, unit string) string {
	// Skip "None" or empty units
	if unit == "None" || unit == "" {
		unit = ""
	}

	// Format large numbers nicely
	if value >= 1000000 {
		if unit != "" {
			return fmt.Sprintf("%.1fM %s", value/1000000, unit)
		}
		return fmt.Sprintf("%.1fM", value/1000000)
	}
	if value >= 1000 {
		if unit != "" {
			return fmt.Sprintf("%.1fK %s", value/1000, unit)
		}
		return fmt.Sprintf("%.1fK", value/1000)
	}
	// Check if it's a whole number
	if value == float64(int64(value)) {
		if unit != "" {
			return fmt.Sprintf("%d %s", int64(value), unit)
		}
		return fmt.Sprintf("%d", int64(value))
	}
	if unit != "" {
		return fmt.Sprintf("%.2f %s", value, unit)
	}
	return fmt.Sprintf("%.2f", value)
}

// RenderDetail renders detailed quota information
func (r *QuotaRenderer) RenderDetail(resource dao.Resource) string {
	qr, ok := resource.(*QuotaResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Service Quota", qr.QuotaName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Quota Code", qr.QuotaCode())
	d.Field("Quota Name", qr.QuotaName())
	d.Field("ARN", qr.GetARN())

	// Service
	d.Section("Service")
	d.Field("Service Code", qr.ServiceCode())
	d.Field("Service Name", qr.ServiceName())

	// Value
	d.Section("Quota Value")
	d.Field("Value", formatValue(qr.Value(), qr.Unit()))
	d.Field("Unit", qr.Unit())

	// Properties
	d.Section("Properties")
	adjustable := "No"
	if qr.Adjustable() {
		adjustable = "Yes (can request increase)"
	}
	d.Field("Adjustable", adjustable)

	global := "No (regional)"
	if qr.GlobalQuota() {
		global = "Yes (applies to all regions)"
	}
	d.Field("Global Quota", global)

	// Description
	if desc := qr.Description(); desc != "" {
		d.Section("Description")
		// Word wrap long descriptions
		wrapped := wordWrap(desc, 70)
		for _, line := range wrapped {
			d.Field("", line)
		}
	}

	// Show CLI command for quota increase if adjustable
	if qr.Adjustable() {
		d.Section("Request Quota Increase")
		d.Line("aws service-quotas request-service-quota-increase \\")
		d.Line(fmt.Sprintf("  --service-code %s \\", qr.ServiceCode()))
		d.Line(fmt.Sprintf("  --quota-code %s \\", qr.QuotaCode()))
		d.Line("  --desired-value <NEW_VALUE>")
	}

	return d.String()
}

// wordWrap wraps text at the specified width
func wordWrap(text string, width int) []string {
	var lines []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return lines
	}

	currentLine := words[0]
	for _, word := range words[1:] {
		if len(currentLine)+1+len(word) <= width {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}
	lines = append(lines, currentLine)
	return lines
}

// RenderSummary returns summary fields for the header panel
func (r *QuotaRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	qr, ok := resource.(*QuotaResource)
	if !ok {
		return nil
	}

	fields := []render.SummaryField{
		{Label: "Quota Code", Value: qr.QuotaCode()},
		{Label: "Quota Name", Value: qr.QuotaName()},
		{Label: "Service", Value: qr.ServiceName()},
		{Label: "Value", Value: formatValue(qr.Value(), qr.Unit())},
	}

	if qr.Adjustable() {
		fields = append(fields, render.SummaryField{Label: "Adjustable", Value: "Yes"})
	}

	if qr.GlobalQuota() {
		fields = append(fields, render.SummaryField{Label: "Scope", Value: "Global"})
	} else {
		fields = append(fields, render.SummaryField{Label: "Scope", Value: "Regional"})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *QuotaRenderer) Navigations(resource dao.Resource) []render.Navigation {
	qr, ok := resource.(*QuotaResource)
	if !ok {
		return nil
	}

	return []render.Navigation{
		{
			Key:         "s",
			Label:       "Services",
			Service:     "service-quotas",
			Resource:    "services",
			FilterField: "ServiceCode",
			FilterValue: qr.ServiceCode(),
		},
	}
}
