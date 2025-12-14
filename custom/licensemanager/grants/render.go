package grants

import (
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// GrantRenderer renders License Manager grants.
type GrantRenderer struct {
	render.BaseRenderer
}

// NewGrantRenderer creates a new GrantRenderer.
func NewGrantRenderer() render.Renderer {
	return &GrantRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "license-manager",
			Resource: "grants",
			Cols: []render.Column{
				{Name: "NAME", Width: 30, Getter: getName},
				{Name: "GRANTEE", Width: 40, Getter: getGrantee},
				{Name: "STATUS", Width: 15, Getter: getStatus},
			},
		},
	}
}

func getName(r dao.Resource) string {
	grant, ok := r.(*GrantResource)
	if !ok {
		return ""
	}
	return grant.Name()
}

func getGrantee(r dao.Resource) string {
	grant, ok := r.(*GrantResource)
	if !ok {
		return ""
	}
	return grant.GranteePrincipal()
}

func getStatus(r dao.Resource) string {
	grant, ok := r.(*GrantResource)
	if !ok {
		return ""
	}
	return grant.Status()
}

// RenderDetail renders the detail view for a grant.
func (r *GrantRenderer) RenderDetail(resource dao.Resource) string {
	grant, ok := resource.(*GrantResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("License Grant", grant.Name())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", grant.Name())
	d.Field("ARN", grant.GetARN())
	d.Field("Status", grant.Status())

	// Parties
	d.Section("Parties")
	d.Field("Grantee Principal", grant.GranteePrincipal())
	if grant.ParentArn() != "" {
		d.Field("Parent ARN", grant.ParentArn())
	}

	return d.String()
}

// RenderSummary renders summary fields for a grant.
func (r *GrantRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	grant, ok := resource.(*GrantResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Name", Value: grant.Name()},
		{Label: "Status", Value: grant.Status()},
	}
}
