package restapis

import (
	"fmt"
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// Ensure RestAPIRenderer implements render.Navigator
var _ render.Navigator = (*RestAPIRenderer)(nil)

// RestAPIRenderer renders API Gateway REST APIs
type RestAPIRenderer struct {
	render.BaseRenderer
}

// NewRestAPIRenderer creates a new RestAPIRenderer
func NewRestAPIRenderer() render.Renderer {
	return &RestAPIRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "apigateway",
			Resource: "rest-apis",
			Cols: []render.Column{
				{
					Name:  "NAME",
					Width: 30,
					Getter: func(r dao.Resource) string {
						return r.GetName()
					},
					Priority: 0,
				},
				{
					Name:  "ID",
					Width: 12,
					Getter: func(r dao.Resource) string {
						return r.GetID()
					},
					Priority: 1,
				},
				{
					Name:  "ENDPOINT",
					Width: 10,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*RestAPIResource); ok {
							return rr.EndpointType()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "API KEY",
					Width: 10,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*RestAPIResource); ok {
							return rr.ApiKeySource()
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "DESCRIPTION",
					Width: 40,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*RestAPIResource); ok {
							desc := rr.Description()
							if len(desc) > 40 {
								return desc[:37] + "..."
							}
							return desc
						}
						return ""
					},
					Priority: 4,
				},
				{
					Name:  "CREATED",
					Width: 20,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*RestAPIResource); ok {
							t := rr.CreatedDate()
							if !t.IsZero() {
								return t.Format("2006-01-02 15:04")
							}
						}
						return ""
					},
					Priority: 5,
				},
			},
		},
	}
}

// RenderDetail renders detailed REST API information
func (r *RestAPIRenderer) RenderDetail(resource dao.Resource) string {
	rr, ok := resource.(*RestAPIResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("API Gateway REST API", rr.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", rr.GetName())
	d.Field("API ID", rr.GetID())
	if rr.Description() != "" {
		d.Field("Description", rr.Description())
	}
	if rr.Version() != "" {
		d.Field("Version", rr.Version())
	}
	d.Field("Created", rr.CreatedDate().Format("2006-01-02 15:04:05 MST"))

	// Endpoint Configuration
	d.Section("Endpoint Configuration")
	d.Field("Endpoint Type", rr.EndpointType())
	d.Field("Default Endpoint Disabled", fmt.Sprintf("%v", rr.DisableExecuteApiEndpoint()))

	// API Settings
	d.Section("API Settings")
	d.Field("API Key Source", rr.ApiKeySource())
	if rr.RootResourceId() != "" {
		d.Field("Root Resource ID", rr.RootResourceId())
	}
	if rr.MinimumCompressionSize() > 0 {
		d.Field("Min Compression Size", fmt.Sprintf("%d bytes", rr.MinimumCompressionSize()))
	}

	// Binary Media Types
	if len(rr.BinaryMediaTypes()) > 0 {
		d.Field("Binary Media Types", strings.Join(rr.BinaryMediaTypes(), ", "))
	}

	// Tags
	if len(rr.GetTags()) > 0 {
		d.Section("Tags")
		for k, v := range rr.GetTags() {
			d.Field(k, v)
		}
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *RestAPIRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	rr, ok := resource.(*RestAPIResource)
	if !ok {
		return nil
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: rr.GetName()},
		{Label: "API ID", Value: rr.GetID()},
		{Label: "Endpoint", Value: rr.EndpointType()},
		{Label: "API Key Source", Value: rr.ApiKeySource()},
	}

	if rr.Description() != "" {
		desc := rr.Description()
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}
		fields = append(fields, render.SummaryField{Label: "Description", Value: desc})
	}

	return fields
}

// Navigations returns navigation shortcuts for REST APIs
func (r *RestAPIRenderer) Navigations(resource dao.Resource) []render.Navigation {
	rr, ok := resource.(*RestAPIResource)
	if !ok {
		return nil
	}

	return []render.Navigation{
		{
			Key: "s", Label: "Stages", Service: "apigateway", Resource: "stages",
			FilterField: "RestApiId", FilterValue: rr.GetID(),
		},
	}
}
