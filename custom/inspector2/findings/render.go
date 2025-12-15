package findings

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// FindingRenderer renders Inspector2 findings
// Ensure FindingRenderer implements render.Navigator
var _ render.Navigator = (*FindingRenderer)(nil)

type FindingRenderer struct {
	render.BaseRenderer
}

// NewFindingRenderer creates a new FindingRenderer
func NewFindingRenderer() *FindingRenderer {
	return &FindingRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "inspector2",
			Resource: "findings",
			Cols: []render.Column{
				{Name: "SEVERITY", Width: 10, Getter: getSeverity},
				{Name: "TYPE", Width: 15, Getter: getType},
				{Name: "TITLE", Width: 50, Getter: getTitle},
				{Name: "RESOURCE", Width: 30, Getter: getResource},
				{Name: "STATUS", Width: 10, Getter: getStatus},
			},
		},
	}
}

func getSeverity(r dao.Resource) string {
	if f, ok := r.(*FindingResource); ok {
		return f.Severity()
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

func getResource(r dao.Resource) string {
	if f, ok := r.(*FindingResource); ok {
		return f.ResourceId()
	}
	return ""
}

func getStatus(r dao.Resource) string {
	if f, ok := r.(*FindingResource); ok {
		return f.Status()
	}
	return ""
}

// RenderDetail renders detailed finding information
func (r *FindingRenderer) RenderDetail(resource dao.Resource) string {
	finding, ok := resource.(*FindingResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Inspector Finding", finding.TitleShort())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Title", finding.Title())
	d.Field("Finding ARN", finding.FindingArn())
	d.Field("Severity", finding.Severity())
	d.Field("Status", finding.Status())
	d.Field("Type", finding.Type())

	if score := finding.InspectorScore(); score > 0 {
		d.Field("Inspector Score", fmt.Sprintf("%.1f", score))
	}

	// Vulnerability
	if vulnId := finding.VulnerabilityId(); vulnId != "" {
		d.Section("Vulnerability")
		d.Field("Vulnerability ID", vulnId)
		if vendorSeverity := finding.VendorSeverity(); vendorSeverity != "" {
			d.Field("Vendor Severity", vendorSeverity)
		}
	}

	// Affected Resources
	if resources := finding.Resources(); len(resources) > 0 {
		d.Section("Affected Resources")
		for i, res := range resources {
			prefix := fmt.Sprintf("Resource %d", i+1)
			d.Field(prefix+" Type", string(res.Type))
			if res.Id != nil {
				d.Field(prefix+" ID", *res.Id)
			}
			if res.Region != nil {
				d.Field(prefix+" Region", *res.Region)
			}
		}
	}

	// Description
	if desc := finding.Description(); desc != "" {
		d.Section("Description")
		d.Field("", desc)
	}

	// Remediation
	if remediation := finding.Remediation(); remediation != "" {
		d.Section("Remediation")
		d.Field("", remediation)
	}

	// Timestamps
	d.Section("Timestamps")
	if first := finding.FirstObservedAt(); first != "" {
		d.Field("First Observed", first)
	}
	if last := finding.LastObservedAt(); last != "" {
		d.Field("Last Observed", last)
	}
	if updated := finding.UpdatedAt(); updated != "" {
		d.Field("Last Updated", updated)
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
		{Label: "Severity", Value: finding.Severity()},
		{Label: "Status", Value: finding.Status()},
		{Label: "Type", Value: finding.Type()},
	}

	if vulnId := finding.VulnerabilityId(); vulnId != "" {
		fields = append(fields, render.SummaryField{Label: "Vulnerability", Value: vulnId})
	}

	if score := finding.InspectorScore(); score > 0 {
		fields = append(fields, render.SummaryField{Label: "Score", Value: fmt.Sprintf("%.1f", score)})
	}

	if resId := finding.ResourceId(); resId != "" {
		fields = append(fields, render.SummaryField{Label: "Resource", Value: resId})
	}

	if first := finding.FirstObservedAt(); first != "" {
		fields = append(fields, render.SummaryField{Label: "First Observed", Value: first})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *FindingRenderer) Navigations(resource dao.Resource) []render.Navigation {
	return nil
}
