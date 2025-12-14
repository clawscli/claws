package groups

import (
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// GroupRenderer renders IAM Groups
type GroupRenderer struct {
	render.BaseRenderer
}

// NewGroupRenderer creates a new GroupRenderer
func NewGroupRenderer() render.Renderer {
	return &GroupRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "iam",
			Resource: "groups",
			Cols: []render.Column{
				{
					Name:  "NAME",
					Width: 35,
					Getter: func(r dao.Resource) string {
						return r.GetName()
					},
					Priority: 0,
				},
				{
					Name:  "PATH",
					Width: 20,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*GroupResource); ok {
							return v.Path()
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "GROUP ID",
					Width: 22,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*GroupResource); ok {
							return v.GroupId()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "CREATED",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*GroupResource); ok {
							if v.Item.CreateDate != nil {
								return render.FormatAge(*v.Item.CreateDate)
							}
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "ARN",
					Width: 60,
					Getter: func(r dao.Resource) string {
						if v, ok := r.(*GroupResource); ok {
							return v.Arn()
						}
						return ""
					},
					Priority: 4,
				},
			},
		},
	}
}

// RenderDetail renders detailed group information
func (r *GroupRenderer) RenderDetail(resource dao.Resource) string {
	v, ok := resource.(*GroupResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("IAM Group", v.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Group Name", v.GetName())
	d.Field("Group ID", v.GroupId())
	d.Field("Path", v.Path())
	d.Field("ARN", v.Arn())

	// Timestamps
	d.Section("Timestamps")
	if v.Item.CreateDate != nil {
		d.Field("Created", v.Item.CreateDate.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *GroupRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	v, ok := resource.(*GroupResource)
	if !ok {
		return nil
	}

	fields := []render.SummaryField{
		{Label: "Group Name", Value: v.GetName()},
		{Label: "Group ID", Value: v.GroupId()},
		{Label: "Path", Value: v.Path()},
	}

	fields = append(fields, render.SummaryField{Label: "ARN", Value: v.Arn()})

	if v.Item.CreateDate != nil {
		fields = append(fields, render.SummaryField{
			Label: "Created",
			Value: v.Item.CreateDate.Format("2006-01-02 15:04"),
		})
	}

	return fields
}
