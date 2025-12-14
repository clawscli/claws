package steps

import (
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// StepRenderer renders EMR steps.
type StepRenderer struct {
	render.BaseRenderer
}

// NewStepRenderer creates a new StepRenderer.
func NewStepRenderer() render.Renderer {
	return &StepRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "emr",
			Resource: "steps",
			Cols: []render.Column{
				{Name: "STEP ID", Width: 20, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "NAME", Width: 40, Getter: getName},
				{Name: "STATE", Width: 15, Getter: getState},
			},
		},
	}
}

func getName(r dao.Resource) string {
	step, ok := r.(*StepResource)
	if !ok {
		return ""
	}
	return step.Name()
}

func getState(r dao.Resource) string {
	step, ok := r.(*StepResource)
	if !ok {
		return ""
	}
	return step.State()
}

// RenderDetail renders the detail view for a step.
func (r *StepRenderer) RenderDetail(resource dao.Resource) string {
	step, ok := resource.(*StepResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("EMR Step", step.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Step ID", step.GetID())
	d.Field("Name", step.Name())
	d.Field("State", step.State())

	if step.ActionOnFailure != "" {
		d.Field("Action On Failure", step.ActionOnFailure)
	}

	if step.StateReason() != "" {
		d.Field("State Reason", step.StateReason())
	}

	return d.String()
}

// RenderSummary renders summary fields for a step.
func (r *StepRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	step, ok := resource.(*StepResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Step ID", Value: step.GetID()},
		{Label: "Name", Value: step.Name()},
		{Label: "State", Value: step.State()},
	}
}
