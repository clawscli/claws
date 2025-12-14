package findings

import (
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// FindingRenderer renders Access Analyzer findings.
type FindingRenderer struct {
	render.BaseRenderer
}

// NewFindingRenderer creates a new FindingRenderer.
func NewFindingRenderer() render.Renderer {
	return &FindingRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "accessanalyzer",
			Resource: "findings",
			Cols: []render.Column{
				{Name: "RESOURCE TYPE", Width: 20, Getter: getResourceType},
				{Name: "RESOURCE", Width: 45, Getter: getResource},
				{Name: "STATUS", Width: 10, Getter: getStatus},
				{Name: "PUBLIC", Width: 8, Getter: getIsPublic},
				{Name: "UPDATED", Width: 16, Getter: getUpdated},
			},
		},
	}
}

func getResourceType(r dao.Resource) string {
	finding, ok := r.(*FindingResource)
	if !ok {
		return ""
	}
	return finding.ResourceType()
}

func getResource(r dao.Resource) string {
	finding, ok := r.(*FindingResource)
	if !ok {
		return ""
	}
	resource := finding.Resource()
	if len(resource) > 42 {
		return "..." + resource[len(resource)-42:]
	}
	return resource
}

func getStatus(r dao.Resource) string {
	finding, ok := r.(*FindingResource)
	if !ok {
		return ""
	}
	return finding.Status()
}

func getIsPublic(r dao.Resource) string {
	finding, ok := r.(*FindingResource)
	if !ok {
		return ""
	}
	if finding.IsPublic() {
		return "Yes"
	}
	return "No"
}

func getUpdated(r dao.Resource) string {
	finding, ok := r.(*FindingResource)
	if !ok {
		return ""
	}
	if t := finding.UpdatedAt(); t != nil {
		return render.FormatAge(*t)
	}
	return ""
}

// RenderDetail renders the detail view for a finding.
func (r *FindingRenderer) RenderDetail(resource dao.Resource) string {
	finding, ok := resource.(*FindingResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Access Analyzer Finding", finding.FindingId())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Finding ID", finding.FindingId())
	d.Field("Status", finding.Status())
	if finding.IsPublic() {
		d.Field("Public Access", "Yes")
	} else {
		d.Field("Public Access", "No")
	}

	// Resource Info
	d.Section("Resource")
	d.Field("Type", finding.ResourceType())
	d.Field("ARN", finding.Resource())
	if owner := finding.ResourceOwnerAccount(); owner != "" {
		d.Field("Owner Account", owner)
	}

	// Access Details
	if actions := finding.Action(); len(actions) > 0 {
		d.Section("Access Details")
		d.Field("Actions", strings.Join(actions, ", "))
	}

	// Principal
	if principal := finding.Principal(); len(principal) > 0 {
		d.Section("Principal")
		for k, v := range principal {
			d.Field(k, v)
		}
	}

	// Condition
	if condition := finding.Condition(); len(condition) > 0 {
		d.Section("Condition")
		for k, v := range condition {
			d.Field(k, v)
		}
	}

	// Error
	if err := finding.Error(); err != "" {
		d.Section("Error")
		d.Field("Error", err)
	}

	// Sources
	if sources := finding.Sources(); len(sources) > 0 {
		d.Section("Finding Sources")
		for i, src := range sources {
			srcLabel := ""
			if i == 0 {
				srcLabel = "Source Type"
			} else {
				srcLabel = ""
			}
			if srcLabel != "" {
				d.Field(srcLabel, string(src.Type))
			}
			if src.Detail != nil && src.Detail.AccessPointArn != nil {
				d.Field("Access Point", *src.Detail.AccessPointArn)
			}
			if src.Detail != nil && src.Detail.AccessPointAccount != nil {
				d.Field("Access Point Account", *src.Detail.AccessPointAccount)
			}
		}
	}

	// RCP Restriction
	if rcp := finding.ResourceControlPolicyRestriction(); rcp != "" {
		d.Field("RCP Restriction", rcp)
	}

	// Timestamps
	d.Section("Timestamps")
	if t := finding.AnalyzedAt(); t != nil {
		d.Field("Analyzed", t.Format("2006-01-02 15:04:05"))
	}
	if t := finding.CreatedAt(); t != nil {
		d.Field("Created", t.Format("2006-01-02 15:04:05"))
	}
	if t := finding.UpdatedAt(); t != nil {
		d.Field("Updated", t.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary renders summary fields for a finding.
func (r *FindingRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	finding, ok := resource.(*FindingResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	isPublic := "No"
	if finding.IsPublic() {
		isPublic = "Yes"
	}

	return []render.SummaryField{
		{Label: "Finding ID", Value: finding.FindingId()},
		{Label: "Resource Type", Value: finding.ResourceType()},
		{Label: "Resource", Value: finding.Resource()},
		{Label: "Status", Value: finding.Status()},
		{Label: "Public", Value: isPublic},
	}
}
