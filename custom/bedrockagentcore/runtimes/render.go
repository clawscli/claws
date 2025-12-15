package runtimes

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// RuntimeRenderer renders Bedrock AgentCore Runtimes
// Ensure RuntimeRenderer implements render.Navigator
var _ render.Navigator = (*RuntimeRenderer)(nil)

type RuntimeRenderer struct {
	render.BaseRenderer
}

// NewRuntimeRenderer creates a new RuntimeRenderer
func NewRuntimeRenderer() render.Renderer {
	return &RuntimeRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "bedrock-agentcore",
			Resource: "runtimes",
			Cols: []render.Column{
				{Name: "NAME", Width: 35, Getter: func(r dao.Resource) string { return r.GetName() }},
				{Name: "STATUS", Width: 12, Getter: getStatus},
				{Name: "VERSION", Width: 10, Getter: getVersion},
				{Name: "UPDATED", Width: 12, Getter: getAge},
			},
		},
	}
}

func getStatus(r dao.Resource) string {
	if runtime, ok := r.(*RuntimeResource); ok {
		return runtime.Status()
	}
	return ""
}

func getVersion(r dao.Resource) string {
	if runtime, ok := r.(*RuntimeResource); ok {
		return runtime.Version()
	}
	return ""
}

func getAge(r dao.Resource) string {
	if runtime, ok := r.(*RuntimeResource); ok {
		if updated := runtime.LastUpdatedAt(); updated != nil {
			return render.FormatAge(*updated)
		}
	}
	return "-"
}

// RenderDetail renders detailed runtime information
func (r *RuntimeRenderer) RenderDetail(resource dao.Resource) string {
	runtime, ok := resource.(*RuntimeResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Bedrock AgentCore Runtime", runtime.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", runtime.GetName())
	d.Field("ID", runtime.GetID())
	d.Field("ARN", runtime.GetARN())
	d.Field("Status", runtime.Status())

	if desc := runtime.Description(); desc != "" {
		d.Field("Description", desc)
	}

	if version := runtime.Version(); version != "" {
		d.Field("Version", version)
	}

	// IAM
	if roleArn := runtime.RoleArn(); roleArn != "" {
		d.Section("IAM")
		d.Field("Role ARN", roleArn)
	}

	// Timestamps
	d.Section("Timestamps")
	if created := runtime.CreatedAt(); created != nil {
		d.Field("Created", created.Format("2006-01-02 15:04:05"))
	}
	if updated := runtime.LastUpdatedAt(); updated != nil {
		d.Field("Last Updated", updated.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *RuntimeRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	runtime, ok := resource.(*RuntimeResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: runtime.GetName()},
		{Label: "ID", Value: runtime.GetID()},
		{Label: "ARN", Value: runtime.GetARN()},
		{Label: "Status", Value: runtime.Status()},
	}

	if desc := runtime.Description(); desc != "" {
		fields = append(fields, render.SummaryField{Label: "Description", Value: desc})
	}

	if version := runtime.Version(); version != "" {
		fields = append(fields, render.SummaryField{Label: "Version", Value: version})
	}

	if created := runtime.CreatedAt(); created != nil {
		fields = append(fields, render.SummaryField{Label: "Created", Value: fmt.Sprintf("%s ago", render.FormatAge(*created))})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *RuntimeRenderer) Navigations(resource dao.Resource) []render.Navigation {
	runtime, ok := resource.(*RuntimeResource)
	if !ok {
		return nil
	}

	return []render.Navigation{
		{
			Key:         "e",
			Label:       "Endpoints",
			Service:     "bedrock-agentcore",
			Resource:    "endpoints",
			FilterField: "AgentRuntimeId",
			FilterValue: runtime.GetID(),
		},
		{
			Key:         "v",
			Label:       "Versions",
			Service:     "bedrock-agentcore",
			Resource:    "versions",
			FilterField: "AgentRuntimeId",
			FilterValue: runtime.GetID(),
		},
	}
}
