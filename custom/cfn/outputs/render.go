package outputs

import (
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// OutputRenderer renders CloudFormation stack outputs
type OutputRenderer struct {
	render.BaseRenderer
}

// NewOutputRenderer creates a new OutputRenderer
func NewOutputRenderer() render.Renderer {
	return &OutputRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "cloudformation",
			Resource: "outputs",
			Cols: []render.Column{
				{
					Name:  "KEY",
					Width: 35,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*OutputResource); ok {
							return v.OutputKey()
						}
						return ""
					},
					Priority: 0,
				},
				{
					Name:  "VALUE",
					Width: 50,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*OutputResource); ok {
							return v.OutputValue()
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "EXPORT",
					Width: 30,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*OutputResource); ok {
							return v.ExportName()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "DESCRIPTION",
					Width: 50,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*OutputResource); ok {
							return v.Description()
						}
						return ""
					},
					Priority: 3,
				},
			},
		},
	}
}

// RenderDetail renders detailed output information
func (r *OutputRenderer) RenderDetail(resource dao.Resource) string {
	v, ok := resource.(*OutputResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Stack Output", v.OutputKey())

	d.Section("Output")
	d.Field("Key", v.OutputKey())
	d.Field("Value", v.OutputValue())

	if export := v.ExportName(); export != "" {
		d.Section("Export")
		d.Field("Export Name", export)
	}

	if desc := v.Description(); desc != "" {
		d.Section("Description")
		d.DimIndent(desc)
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *OutputRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	v, ok := resource.(*OutputResource)
	if !ok {
		return nil
	}

	fields := []render.SummaryField{
		{Label: "Key", Value: v.OutputKey()},
		{Label: "Value", Value: v.OutputValue()},
	}

	if export := v.ExportName(); export != "" {
		fields = append(fields, render.SummaryField{Label: "Export Name", Value: export})
	}

	return fields
}
