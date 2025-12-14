package versions

import (
	"bytes"
	"encoding/json"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

type VersionRenderer struct {
	render.BaseRenderer
}

func NewVersionRenderer() render.Renderer {
	return &VersionRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "bedrock-agentcore",
			Resource: "versions",
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
					Name:  "VERSION",
					Width: 10,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*VersionResource); ok {
							return v.Version()
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "STATUS",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*VersionResource); ok {
							return v.Status()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "UPDATED",
					Width: 10,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*VersionResource); ok {
							if v.LastUpdatedAt() != nil {
								return render.FormatAge(*v.LastUpdatedAt())
							}
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "CREATED",
					Width: 10,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*VersionResource); ok {
							if v.CreatedAt() != nil {
								return render.FormatAge(*v.CreatedAt())
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

func (r *VersionRenderer) RenderDetail(resource dao.Resource) string {
	v, ok := resource.(*VersionResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()
	d.Title("Agent Runtime Version", v.GetName())

	d.Section("Basic Information")
	d.Field("ID", v.GetID())
	d.Field("Name", v.RuntimeName())
	d.Field("Version", v.Version())
	d.Field("ARN", v.GetARN())
	d.Field("Status", v.Status())

	d.Section("Description")
	if desc := v.Description(); desc != "" {
		d.Line(desc)
	} else {
		d.Line("-")
	}

	d.Section("Timestamps")
	if v.CreatedAt() != nil {
		d.Field("Created", render.FormatAge(*v.CreatedAt()))
	}
	if v.LastUpdatedAt() != nil {
		d.Field("Updated", render.FormatAge(*v.LastUpdatedAt()))
	}

	// Show full JSON at bottom
	d.Section("Full Details")
	if v.DetailItem != nil {
		d.Line(prettyJSON(v.DetailItem))
	} else {
		d.Line(prettyJSON(v.Item))
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

func (r *VersionRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	v, ok := resource.(*VersionResource)
	if !ok {
		return nil
	}

	return []render.SummaryField{
		{Label: "Name", Value: v.RuntimeName()},
		{Label: "Version", Value: v.Version()},
		{Label: "Status", Value: v.Status()},
	}
}
