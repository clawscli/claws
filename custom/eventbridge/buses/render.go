package buses

import (
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// Ensure BusRenderer implements render.Navigator
var _ render.Navigator = (*BusRenderer)(nil)

// BusRenderer renders EventBridge event buses with custom columns
type BusRenderer struct {
	render.BaseRenderer
}

// NewBusRenderer creates a new BusRenderer
func NewBusRenderer() render.Renderer {
	return &BusRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "eventbridge",
			Resource: "buses",
			Cols: []render.Column{
				{
					Name:  "NAME",
					Width: 40,
					Getter: func(r dao.Resource) string {
						return r.GetName()
					},
					Priority: 0,
				},
				{
					Name:  "ARN",
					Width: 80,
					Getter: func(r dao.Resource) string {
						if br, ok := r.(*BusResource); ok {
							return br.ARN()
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "DEFAULT",
					Width: 8,
					Getter: func(r dao.Resource) string {
						if br, ok := r.(*BusResource); ok {
							if br.IsDefault() {
								return "Yes"
							}
							return "No"
						}
						return ""
					},
					Priority: 2,
				},
			},
		},
	}
}

// RenderDetail renders detailed bus information
func (r *BusRenderer) RenderDetail(resource dao.Resource) string {
	br, ok := resource.(*BusResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("EventBridge Event Bus", br.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", br.GetName())
	d.Field("ARN", br.ARN())
	if br.IsDefault() {
		d.Field("Type", "Default")
	} else {
		d.Field("Type", "Custom")
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *BusRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	br, ok := resource.(*BusResource)
	if !ok {
		return nil
	}

	busType := "Custom"
	if br.IsDefault() {
		busType = "Default"
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: br.GetName()},
		{Label: "ARN", Value: br.ARN()},
		{Label: "Type", Value: busType},
	}

	return fields
}

// Navigations returns navigation shortcuts for event buses
func (r *BusRenderer) Navigations(resource dao.Resource) []render.Navigation {
	br, ok := resource.(*BusResource)
	if !ok {
		return nil
	}

	var navs []render.Navigation

	// Rules navigation
	navs = append(navs, render.Navigation{
		Key: "r", Label: "Rules", Service: "eventbridge", Resource: "rules",
		FilterField: "EventBusName", FilterValue: br.GetName(),
	})

	return navs
}
