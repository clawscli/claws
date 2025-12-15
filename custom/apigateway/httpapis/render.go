package httpapis

import (
	"fmt"
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// Ensure HttpAPIRenderer implements render.Navigator
var _ render.Navigator = (*HttpAPIRenderer)(nil)

// HttpAPIRenderer renders API Gateway HTTP/WebSocket APIs
type HttpAPIRenderer struct {
	render.BaseRenderer
}

// NewHttpAPIRenderer creates a new HttpAPIRenderer
func NewHttpAPIRenderer() render.Renderer {
	return &HttpAPIRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "apigateway",
			Resource: "http-apis",
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
					Name:  "PROTOCOL",
					Width: 10,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*HttpAPIResource); ok {
							return rr.ProtocolType()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "ENDPOINT",
					Width: 50,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*HttpAPIResource); ok {
							return rr.ApiEndpoint()
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "CORS",
					Width: 6,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*HttpAPIResource); ok {
							if rr.HasCors() {
								return "Yes"
							}
							return "No"
						}
						return ""
					},
					Priority: 4,
				},
				{
					Name:  "CREATED",
					Width: 20,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*HttpAPIResource); ok {
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

// RenderDetail renders detailed HTTP API information
func (r *HttpAPIRenderer) RenderDetail(resource dao.Resource) string {
	rr, ok := resource.(*HttpAPIResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("API Gateway HTTP/WebSocket API", rr.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", rr.GetName())
	d.Field("API ID", rr.GetID())
	d.Field("Protocol", rr.ProtocolType())
	if rr.Description() != "" {
		d.Field("Description", rr.Description())
	}
	if rr.Version() != "" {
		d.Field("Version", rr.Version())
	}
	d.Field("Created", rr.CreatedDate().Format("2006-01-02 15:04:05 MST"))

	// Endpoint
	d.Section("Endpoint")
	d.Field("API Endpoint", rr.ApiEndpoint())
	d.Field("Default Endpoint Disabled", fmt.Sprintf("%v", rr.DisableExecuteApiEndpoint()))

	// Routing
	if rr.RouteSelectionExpression() != "" {
		d.Section("Routing")
		d.Field("Route Selection Expression", rr.RouteSelectionExpression())
	}

	// CORS Configuration
	if rr.HasCors() {
		d.Section("CORS Configuration")
		if len(rr.CorsAllowOrigins()) > 0 {
			d.Field("Allow Origins", strings.Join(rr.CorsAllowOrigins(), ", "))
		}
		if len(rr.CorsAllowMethods()) > 0 {
			d.Field("Allow Methods", strings.Join(rr.CorsAllowMethods(), ", "))
		}
		if len(rr.CorsAllowHeaders()) > 0 {
			d.Field("Allow Headers", strings.Join(rr.CorsAllowHeaders(), ", "))
		}
	}

	// Management
	d.Section("Management")
	d.Field("API Gateway Managed", fmt.Sprintf("%v", rr.ApiGatewayManaged()))

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
func (r *HttpAPIRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	rr, ok := resource.(*HttpAPIResource)
	if !ok {
		return nil
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: rr.GetName()},
		{Label: "API ID", Value: rr.GetID()},
		{Label: "Protocol", Value: rr.ProtocolType()},
	}

	endpoint := rr.ApiEndpoint()
	if len(endpoint) > 60 {
		endpoint = endpoint[:57] + "..."
	}
	fields = append(fields, render.SummaryField{Label: "Endpoint", Value: endpoint})

	if rr.HasCors() {
		fields = append(fields, render.SummaryField{Label: "CORS", Value: "Enabled"})
	}

	return fields
}

// Navigations returns navigation shortcuts for HTTP APIs
func (r *HttpAPIRenderer) Navigations(resource dao.Resource) []render.Navigation {
	rr, ok := resource.(*HttpAPIResource)
	if !ok {
		return nil
	}

	return []render.Navigation{
		{
			Key: "s", Label: "Stages", Service: "apigateway", Resource: "stages-v2",
			FilterField: "ApiId", FilterValue: rr.GetID(),
		},
	}
}
