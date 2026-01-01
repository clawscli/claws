package computeenvironments

import (
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// ComputeEnvironmentRenderer renders Batch compute environments.
type ComputeEnvironmentRenderer struct {
	render.BaseRenderer
}

// NewComputeEnvironmentRenderer creates a new ComputeEnvironmentRenderer.
func NewComputeEnvironmentRenderer() render.Renderer {
	return &ComputeEnvironmentRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "batch",
			Resource: "compute-environments",
			Cols: []render.Column{
				{Name: "NAME", Width: 40, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "TYPE", Width: 12, Getter: getType},
				{Name: "STATE", Width: 12, Getter: getState},
				{Name: "STATUS", Width: 12, Getter: getStatus},
			},
		},
	}
}

func getType(r dao.Resource) string {
	env, ok := r.(*ComputeEnvironmentResource)
	if !ok {
		return ""
	}
	return env.Type()
}

func getState(r dao.Resource) string {
	env, ok := r.(*ComputeEnvironmentResource)
	if !ok {
		return ""
	}
	return env.State()
}

func getStatus(r dao.Resource) string {
	env, ok := r.(*ComputeEnvironmentResource)
	if !ok {
		return ""
	}
	return env.Status()
}

// RenderDetail renders the detail view for a compute environment.
func (r *ComputeEnvironmentRenderer) RenderDetail(resource dao.Resource) string {
	env, ok := resource.(*ComputeEnvironmentResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Batch Compute Environment", env.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", env.GetID())
	d.Field("ARN", env.GetARN())
	d.Field("Type", env.Type())
	d.Field("State", env.State())
	d.Field("Status", env.Status())

	// IAM
	if env.ServiceRole() != "" {
		d.Section("IAM")
		d.Field("Service Role", env.ServiceRole())
	}

	return d.String()
}

// RenderSummary renders summary fields for a compute environment.
func (r *ComputeEnvironmentRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	env, ok := resource.(*ComputeEnvironmentResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Name", Value: env.GetID()},
		{Label: "Type", Value: env.Type()},
		{Label: "State", Value: env.State()},
		{Label: "Status", Value: env.Status()},
	}
}
