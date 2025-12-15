package locations

import (
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// LocationRenderer renders DataSync locations.
type LocationRenderer struct {
	render.BaseRenderer
}

// NewLocationRenderer creates a new LocationRenderer.
func NewLocationRenderer() render.Renderer {
	return &LocationRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "datasync",
			Resource: "locations",
			Cols: []render.Column{
				{Name: "LOCATION ID", Width: 35, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "LOCATION URI", Width: 60, Getter: getLocationUri},
			},
		},
	}
}

func getLocationUri(r dao.Resource) string {
	loc, ok := r.(*LocationResource)
	if !ok {
		return ""
	}
	return loc.LocationUri()
}

// RenderDetail renders the detail view for a location.
func (r *LocationRenderer) RenderDetail(resource dao.Resource) string {
	loc, ok := resource.(*LocationResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("DataSync Location", loc.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Location ID", loc.GetID())
	d.Field("ARN", loc.GetARN())
	d.Field("Location URI", loc.LocationUri())

	return d.String()
}

// RenderSummary renders summary fields for a location.
func (r *LocationRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	loc, ok := resource.(*LocationResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Location ID", Value: loc.GetID()},
		{Label: "Location URI", Value: loc.LocationUri()},
	}
}
