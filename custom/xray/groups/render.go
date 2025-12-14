package groups

import (
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// GroupRenderer renders X-Ray groups.
type GroupRenderer struct {
	render.BaseRenderer
}

// NewGroupRenderer creates a new GroupRenderer.
func NewGroupRenderer() render.Renderer {
	return &GroupRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "xray",
			Resource: "groups",
			Cols: []render.Column{
				{Name: "GROUP NAME", Width: 30, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "FILTER EXPRESSION", Width: 50, Getter: getFilterExpression},
				{Name: "INSIGHTS", Width: 10, Getter: getInsights},
			},
		},
	}
}

func getFilterExpression(r dao.Resource) string {
	group, ok := r.(*GroupResource)
	if !ok {
		return ""
	}
	expr := group.FilterExpression()
	if len(expr) > 47 {
		return expr[:47] + "..."
	}
	return expr
}

func getInsights(r dao.Resource) string {
	group, ok := r.(*GroupResource)
	if !ok {
		return ""
	}
	if group.InsightsEnabled {
		return "Enabled"
	}
	return "Disabled"
}

// RenderDetail renders the detail view for an X-Ray group.
func (r *GroupRenderer) RenderDetail(resource dao.Resource) string {
	group, ok := resource.(*GroupResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("X-Ray Group", group.Name())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Group Name", group.Name())
	d.Field("ARN", group.GetARN())

	// Filter
	if filter := group.FilterExpression(); filter != "" {
		d.Section("Filter Configuration")
		d.Field("Filter Expression", filter)
	}

	// Insights
	d.Section("Insights Configuration")
	if group.InsightsEnabled {
		d.Field("Insights", "Enabled")
	} else {
		d.Field("Insights", "Disabled")
	}
	if group.NotificationsArn != "" {
		d.Field("Notifications ARN", group.NotificationsArn)
	}

	return d.String()
}

// RenderSummary renders summary fields for an X-Ray group.
func (r *GroupRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	group, ok := resource.(*GroupResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Group Name", Value: group.Name()},
		{Label: "ARN", Value: group.GetARN()},
	}

	if filter := group.FilterExpression(); filter != "" {
		fields = append(fields, render.SummaryField{Label: "Filter", Value: filter})
	}

	if group.InsightsEnabled {
		fields = append(fields, render.SummaryField{Label: "Insights", Value: "Enabled"})
	} else {
		fields = append(fields, render.SummaryField{Label: "Insights", Value: "Disabled"})
	}

	return fields
}
