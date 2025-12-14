package findings

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// FindingRenderer renders GuardDuty findings
// Ensure FindingRenderer implements render.Navigator
var _ render.Navigator = (*FindingRenderer)(nil)

type FindingRenderer struct {
	render.BaseRenderer
}

// NewFindingRenderer creates a new FindingRenderer
func NewFindingRenderer() *FindingRenderer {
	return &FindingRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "guardduty",
			Resource: "findings",
			Cols: []render.Column{
				{Name: "SEVERITY", Width: 10, Getter: getSeverity},
				{Name: "TYPE", Width: 35, Getter: getType},
				{Name: "TITLE", Width: 40, Getter: getTitle},
				{Name: "RESOURCE", Width: 15, Getter: getResourceType},
				{Name: "AGE", Width: 12, Getter: getAge},
			},
		},
	}
}

func getSeverity(r dao.Resource) string {
	if f, ok := r.(*FindingResource); ok {
		return fmt.Sprintf("%.1f %s", f.Severity(), f.SeverityLabel())
	}
	return ""
}

func getType(r dao.Resource) string {
	if f, ok := r.(*FindingResource); ok {
		return f.Type()
	}
	return ""
}

func getTitle(r dao.Resource) string {
	if f, ok := r.(*FindingResource); ok {
		return f.TitleShort()
	}
	return ""
}

func getResourceType(r dao.Resource) string {
	if f, ok := r.(*FindingResource); ok {
		return f.ResourceType()
	}
	return ""
}

func getAge(r dao.Resource) string {
	if f, ok := r.(*FindingResource); ok {
		if t := f.CreatedAtTime(); t != nil {
			return render.FormatAge(*t)
		}
	}
	return "-"
}

// RenderDetail renders detailed finding information
func (r *FindingRenderer) RenderDetail(resource dao.Resource) string {
	finding, ok := resource.(*FindingResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("GuardDuty Finding", finding.TitleShort())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Finding ID", finding.FindingId())
	d.Field("Title", finding.Title())
	if arn := finding.GetARN(); arn != "" {
		d.Field("ARN", arn)
	}
	d.Field("Severity", fmt.Sprintf("%.1f (%s)", finding.Severity(), finding.SeverityLabel()))
	d.Field("Type", finding.Type())

	// Location
	d.Section("Location")
	d.Field("Region", finding.Region())
	d.Field("Account ID", finding.AccountId())

	// Affected Resource
	if resType := finding.ResourceType(); resType != "" {
		d.Section("Affected Resource")
		d.Field("Resource Type", resType)
	}

	// Description
	if desc := finding.Description(); desc != "" {
		d.Section("Description")
		d.Field("", desc)
	}

	// Timestamps
	d.Section("Timestamps")
	if created := finding.CreatedAt(); created != "" {
		d.Field("Created", created)
	}
	if updated := finding.UpdatedAt(); updated != "" {
		d.Field("Updated", updated)
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *FindingRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	finding, ok := resource.(*FindingResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Finding ID", Value: finding.FindingId()},
		{Label: "Severity", Value: fmt.Sprintf("%.1f (%s)", finding.Severity(), finding.SeverityLabel())},
		{Label: "Type", Value: finding.Type()},
	}

	if resType := finding.ResourceType(); resType != "" {
		fields = append(fields, render.SummaryField{Label: "Resource Type", Value: resType})
	}

	fields = append(fields, render.SummaryField{Label: "Region", Value: finding.Region()})

	if created := finding.CreatedAt(); created != "" {
		fields = append(fields, render.SummaryField{Label: "Created", Value: created})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *FindingRenderer) Navigations(resource dao.Resource) []render.Navigation {
	return nil
}
