package roots

import (
	"fmt"
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// RootRenderer renders Organizations roots.
// Ensure RootRenderer implements render.Navigator
var _ render.Navigator = (*RootRenderer)(nil)

type RootRenderer struct {
	render.BaseRenderer
}

// NewRootRenderer creates a new RootRenderer.
func NewRootRenderer() render.Renderer {
	return &RootRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "organizations",
			Resource: "roots",
			Cols: []render.Column{
				{Name: "ROOT ID", Width: 20, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "NAME", Width: 20, Getter: getName},
				{Name: "POLICY TYPES", Width: 50, Getter: getPolicyTypes},
			},
		},
	}
}

func getName(r dao.Resource) string {
	root, ok := r.(*RootResource)
	if !ok {
		return ""
	}
	return root.Name()
}

func getPolicyTypes(r dao.Resource) string {
	root, ok := r.(*RootResource)
	if !ok {
		return ""
	}
	pts := root.PolicyTypes()
	if len(pts) == 0 {
		return ""
	}
	var types []string
	for _, pt := range pts {
		types = append(types, string(pt.Type))
	}
	return strings.Join(types, ", ")
}

// RenderDetail renders the detail view for a root.
func (r *RootRenderer) RenderDetail(resource dao.Resource) string {
	root, ok := resource.(*RootResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Organizations Root", root.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Root ID", root.GetID())
	d.Field("Name", root.Name())
	d.Field("ARN", root.GetARN())

	// Policy Types
	pts := root.PolicyTypes()
	if len(pts) > 0 {
		d.Section("Policy Types")
		for i, pt := range pts {
			label := fmt.Sprintf("Policy %d", i+1)
			d.Field(label, fmt.Sprintf("%s (%s)", pt.Type, pt.Status))
		}
	}

	return d.String()
}

// RenderSummary renders summary fields for a root.
func (r *RootRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	root, ok := resource.(*RootResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Root ID", Value: root.GetID()},
		{Label: "Name", Value: root.Name()},
	}
}

// Navigations returns available navigations from a root.
func (r *RootRenderer) Navigations(resource dao.Resource) []render.Navigation {
	root, ok := resource.(*RootResource)
	if !ok {
		return nil
	}
	return []render.Navigation{
		{
			Key:         "o",
			Label:       "OUs",
			Service:     "organizations",
			Resource:    "ous",
			FilterField: "ParentId",
			FilterValue: root.GetID(),
		},
	}
}
