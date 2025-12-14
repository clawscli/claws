package ous

import (
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// OURenderer renders Organizations OUs.
// Ensure OURenderer implements render.Navigator
var _ render.Navigator = (*OURenderer)(nil)

type OURenderer struct {
	render.BaseRenderer
}

// NewOURenderer creates a new OURenderer.
func NewOURenderer() render.Renderer {
	return &OURenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "organizations",
			Resource: "ous",
			Cols: []render.Column{
				{Name: "OU ID", Width: 25, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "NAME", Width: 40, Getter: getName},
			},
		},
	}
}

func getName(r dao.Resource) string {
	ou, ok := r.(*OUResource)
	if !ok {
		return ""
	}
	return ou.Name()
}

// RenderDetail renders the detail view for an OU.
func (r *OURenderer) RenderDetail(resource dao.Resource) string {
	ou, ok := resource.(*OUResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Organizations OU", ou.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("OU ID", ou.GetID())
	d.Field("Name", ou.Name())
	d.Field("ARN", ou.GetARN())

	return d.String()
}

// RenderSummary renders summary fields for an OU.
func (r *OURenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	ou, ok := resource.(*OUResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "OU ID", Value: ou.GetID()},
		{Label: "Name", Value: ou.Name()},
	}
}

// Navigations returns available navigations from an OU.
func (r *OURenderer) Navigations(resource dao.Resource) []render.Navigation {
	ou, ok := resource.(*OUResource)
	if !ok {
		return nil
	}
	return []render.Navigation{
		{
			Key:         "o",
			Label:       "Child OUs",
			Service:     "organizations",
			Resource:    "ous",
			FilterField: "ParentId",
			FilterValue: ou.GetID(),
		},
	}
}
