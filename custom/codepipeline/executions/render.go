package executions

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// ExecutionRenderer renders CodePipeline executions
// Ensure ExecutionRenderer implements render.Navigator
var _ render.Navigator = (*ExecutionRenderer)(nil)

type ExecutionRenderer struct {
	render.BaseRenderer
}

// NewExecutionRenderer creates a new ExecutionRenderer
func NewExecutionRenderer() *ExecutionRenderer {
	return &ExecutionRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "codepipeline",
			Resource: "executions",
			Cols: []render.Column{
				{Name: "EXECUTION ID", Width: 40, Getter: getExecutionId},
				{Name: "STATUS", Width: 12, Getter: getStatus},
				{Name: "TRIGGER", Width: 15, Getter: getTrigger},
				{Name: "STARTED", Width: 20, Getter: getStarted},
				{Name: "UPDATED", Width: 20, Getter: getUpdated},
			},
		},
	}
}

func getExecutionId(r dao.Resource) string {
	if e, ok := r.(*ExecutionResource); ok {
		return e.ExecutionId()
	}
	return ""
}

func getStatus(r dao.Resource) string {
	if e, ok := r.(*ExecutionResource); ok {
		return e.Status()
	}
	return ""
}

func getTrigger(r dao.Resource) string {
	if e, ok := r.(*ExecutionResource); ok {
		return e.Trigger()
	}
	return ""
}

func getStarted(r dao.Resource) string {
	if e, ok := r.(*ExecutionResource); ok {
		return e.StartTime()
	}
	return "-"
}

func getUpdated(r dao.Resource) string {
	if e, ok := r.(*ExecutionResource); ok {
		return e.LastUpdateTime()
	}
	return "-"
}

// RenderDetail renders detailed execution information
func (r *ExecutionRenderer) RenderDetail(resource dao.Resource) string {
	exec, ok := resource.(*ExecutionResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Pipeline Execution", exec.ExecutionId())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Execution ID", exec.ExecutionId())
	d.Field("Pipeline", exec.PipelineName)
	d.Field("Status", exec.Status())
	if version := exec.PipelineVersion(); version > 0 {
		d.Field("Pipeline Version", fmt.Sprintf("%d", version))
	}
	if mode := exec.ExecutionMode(); mode != "" {
		d.Field("Execution Mode", mode)
	}

	// Trigger
	d.Section("Trigger")
	d.Field("Type", exec.Trigger())
	if detail := exec.TriggerDetail(); detail != "" {
		d.Field("Detail", detail)
	}

	// Source Revisions
	if revisions := exec.SourceRevisions(); len(revisions) > 0 {
		d.Section("Source Revisions")
		for _, rev := range revisions {
			if rev.ActionName != nil {
				d.Field("Action", *rev.ActionName)
			}
			if rev.RevisionId != nil {
				d.Field("Revision ID", *rev.RevisionId)
			}
			if rev.RevisionSummary != nil {
				d.Field("Summary", *rev.RevisionSummary)
			}
		}
	}

	// Timestamps
	d.Section("Timestamps")
	if started := exec.StartTime(); started != "" {
		d.Field("Started", started)
	}
	if updated := exec.LastUpdateTime(); updated != "" {
		d.Field("Last Updated", updated)
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *ExecutionRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	exec, ok := resource.(*ExecutionResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Execution ID", Value: exec.ExecutionId()},
		{Label: "Pipeline", Value: exec.PipelineName},
		{Label: "Status", Value: exec.Status()},
		{Label: "Trigger", Value: exec.Trigger()},
	}

	if started := exec.StartTime(); started != "" {
		fields = append(fields, render.SummaryField{Label: "Started", Value: started})
	}

	if updated := exec.LastUpdateTime(); updated != "" {
		fields = append(fields, render.SummaryField{Label: "Updated", Value: updated})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *ExecutionRenderer) Navigations(resource dao.Resource) []render.Navigation {
	return nil
}
