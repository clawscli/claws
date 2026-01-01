package flows

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// FlowRenderer renders Bedrock Flow resources
// Ensure FlowRenderer implements render.Navigator
var _ render.Navigator = (*FlowRenderer)(nil)

type FlowRenderer struct {
	render.BaseRenderer
}

// NewFlowRenderer creates a new FlowRenderer
func NewFlowRenderer() render.Renderer {
	return &FlowRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "bedrock-agent",
			Resource: "flows",
			Cols: []render.Column{
				{Name: "NAME", Width: 30, Getter: func(r dao.Resource) string { return r.GetName() }},
				{Name: "STATUS", Width: 14, Getter: getFlowStatus},
				{Name: "VERSION", Width: 10, Getter: getFlowVersion},
				{Name: "DESCRIPTION", Width: 30, Getter: getFlowDescription},
				{Name: "UPDATED", Width: 12, Getter: getFlowAge},
			},
		},
	}
}

func getFlowStatus(r dao.Resource) string {
	if flow, ok := r.(*FlowResource); ok {
		return flow.Status()
	}
	return ""
}

func getFlowVersion(r dao.Resource) string {
	if flow, ok := r.(*FlowResource); ok {
		return flow.Version()
	}
	return ""
}

func getFlowDescription(r dao.Resource) string {
	if flow, ok := r.(*FlowResource); ok {
		desc := flow.Description()
		if len(desc) > 30 {
			return desc[:27] + "..."
		}
		return desc
	}
	return ""
}

func getFlowAge(r dao.Resource) string {
	if flow, ok := r.(*FlowResource); ok {
		if updated := flow.UpdatedAt(); updated != nil {
			return render.FormatAge(*updated)
		}
	}
	return "-"
}

// RenderDetail renders detailed flow information
func (r *FlowRenderer) RenderDetail(resource dao.Resource) string {
	flow, ok := resource.(*FlowResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Bedrock Flow", flow.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", flow.GetName())
	d.Field("ID", flow.GetID())
	d.Field("Status", flow.Status())
	d.Field("Version", flow.Version())

	if arn := flow.GetARN(); arn != "" {
		d.Field("ARN", arn)
	}

	if desc := flow.Description(); desc != "" {
		d.Field("Description", desc)
	}

	// Configuration
	d.Section("Configuration")
	if roleArn := flow.ExecutionRoleArn(); roleArn != "" {
		d.Field("Execution Role ARN", roleArn)
	}
	if nodeCount := flow.NodeCount(); nodeCount > 0 {
		d.Field("Nodes", fmt.Sprintf("%d", nodeCount))
	}
	if connCount := flow.ConnectionCount(); connCount > 0 {
		d.Field("Connections", fmt.Sprintf("%d", connCount))
	}

	// Timestamps
	d.Section("Timestamps")
	if created := flow.CreatedAt(); created != nil {
		d.Field("Created", created.Format("2006-01-02 15:04:05"))
	}
	if updated := flow.UpdatedAt(); updated != nil {
		d.Field("Updated", updated.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *FlowRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	flow, ok := resource.(*FlowResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: flow.GetName()},
		{Label: "ID", Value: flow.GetID()},
		{Label: "Status", Value: flow.Status()},
		{Label: "Version", Value: flow.Version()},
	}

	if arn := flow.GetARN(); arn != "" {
		fields = append(fields, render.SummaryField{Label: "ARN", Value: arn})
	}

	if desc := flow.Description(); desc != "" {
		fields = append(fields, render.SummaryField{Label: "Description", Value: desc})
	}

	if created := flow.CreatedAt(); created != nil {
		fields = append(fields, render.SummaryField{Label: "Created", Value: fmt.Sprintf("%s ago", render.FormatAge(*created))})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *FlowRenderer) Navigations(resource dao.Resource) []render.Navigation {
	return nil
}
