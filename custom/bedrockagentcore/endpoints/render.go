package endpoints

import (
	"bytes"
	"encoding/json"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

type EndpointRenderer struct {
	render.BaseRenderer
}

func NewEndpointRenderer() render.Renderer {
	return &EndpointRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "bedrock-agentcore",
			Resource: "endpoints",
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
					Name:  "STATUS",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if ep, ok := r.(*EndpointResource); ok {
							return ep.Status()
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "LIVE VERSION",
					Width: 14,
					Getter: func(r dao.Resource) string {
						if ep, ok := r.(*EndpointResource); ok {
							return ep.LiveVersion()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "TARGET VERSION",
					Width: 14,
					Getter: func(r dao.Resource) string {
						if ep, ok := r.(*EndpointResource); ok {
							return ep.TargetVersion()
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "AGE",
					Width: 10,
					Getter: func(r dao.Resource) string {
						if ep, ok := r.(*EndpointResource); ok {
							if ep.CreatedAt() != nil {
								return render.FormatAge(*ep.CreatedAt())
							}
						}
						return ""
					},
					Priority: 4,
				},
			},
		},
	}
}

func (r *EndpointRenderer) RenderDetail(resource dao.Resource) string {
	ep, ok := resource.(*EndpointResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()
	d.Title("Agent Runtime Endpoint", ep.GetName())

	d.Section("Basic Information")
	d.Field("ID", ep.GetID())
	d.Field("Name", ep.GetName())
	d.Field("ARN", ep.GetARN())
	d.Field("Status", ep.Status())

	d.Section("Version")
	d.Field("Live Version", ep.LiveVersion())
	d.Field("Target Version", ep.TargetVersion())

	d.Section("Description")
	if desc := ep.Description(); desc != "" {
		d.Line(desc)
	} else {
		d.Line("-")
	}

	d.Section("Timestamps")
	if ep.CreatedAt() != nil {
		d.Field("Created", render.FormatAge(*ep.CreatedAt()))
	}
	if ep.LastUpdatedAt() != nil {
		d.Field("Updated", render.FormatAge(*ep.LastUpdatedAt()))
	}

	// Show full JSON at bottom
	d.Section("Full Details")
	if ep.DetailItem != nil {
		d.Line(prettyJSON(ep.DetailItem))
	} else {
		d.Line(prettyJSON(ep.Item))
	}

	return d.String()
}

func prettyJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	var out bytes.Buffer
	if err := json.Indent(&out, b, "", "  "); err != nil {
		return string(b)
	}
	return out.String()
}

func (r *EndpointRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	ep, ok := resource.(*EndpointResource)
	if !ok {
		return nil
	}

	return []render.SummaryField{
		{Label: "Name", Value: ep.GetName()},
		{Label: "Status", Value: ep.Status()},
		{Label: "Live Version", Value: ep.LiveVersion()},
	}
}
