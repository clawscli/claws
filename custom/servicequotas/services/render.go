package services

import (
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// Ensure ServiceRenderer implements render.Navigator
var _ render.Navigator = (*ServiceRenderer)(nil)

// ServiceRenderer renders Service Quotas services
type ServiceRenderer struct {
	render.BaseRenderer
}

// NewServiceRenderer creates a new ServiceRenderer
func NewServiceRenderer() render.Renderer {
	return &ServiceRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "service-quotas",
			Resource: "services",
			Cols: []render.Column{
				{
					Name:  "SERVICE CODE",
					Width: 30,
					Getter: func(r dao.Resource) string {
						if sr, ok := r.(*ServiceResource); ok {
							return sr.ServiceCode()
						}
						return ""
					},
					Priority: 0,
				},
				{
					Name:  "SERVICE NAME",
					Width: 50,
					Getter: func(r dao.Resource) string {
						if sr, ok := r.(*ServiceResource); ok {
							return sr.ServiceName()
						}
						return ""
					},
					Priority: 1,
				},
			},
		},
	}
}

// RenderDetail renders detailed service information
func (r *ServiceRenderer) RenderDetail(resource dao.Resource) string {
	sr, ok := resource.(*ServiceResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Service Quotas", sr.ServiceName())

	d.Section("Basic Information")
	d.Field("Service Code", sr.ServiceCode())
	d.Field("Service Name", sr.ServiceName())

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *ServiceRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	sr, ok := resource.(*ServiceResource)
	if !ok {
		return nil
	}

	return []render.SummaryField{
		{Label: "Service Code", Value: sr.ServiceCode()},
		{Label: "Service Name", Value: sr.ServiceName()},
	}
}

// Navigations returns navigation shortcuts
func (r *ServiceRenderer) Navigations(resource dao.Resource) []render.Navigation {
	sr, ok := resource.(*ServiceResource)
	if !ok {
		return nil
	}

	return []render.Navigation{
		{
			Key:         "q",
			Label:       "Quotas",
			Service:     "service-quotas",
			Resource:    "quotas",
			FilterField: "ServiceCode",
			FilterValue: sr.ServiceCode(),
		},
	}
}
