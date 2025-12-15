package pipelines

import (
	"fmt"
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// PipelineRenderer renders CodePipeline pipelines
// Ensure PipelineRenderer implements render.Navigator
var _ render.Navigator = (*PipelineRenderer)(nil)

type PipelineRenderer struct {
	render.BaseRenderer
}

// NewPipelineRenderer creates a new PipelineRenderer
func NewPipelineRenderer() *PipelineRenderer {
	return &PipelineRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "codepipeline",
			Resource: "pipelines",
			Cols: []render.Column{
				{Name: "NAME", Width: 30, Getter: getName},
				{Name: "TYPE", Width: 8, Getter: getPipelineType},
				{Name: "VERSION", Width: 8, Getter: getVersion},
				{Name: "EXEC MODE", Width: 12, Getter: getExecMode},
				{Name: "UPDATED", Width: 20, Getter: getUpdated},
			},
		},
	}
}

func getName(r dao.Resource) string {
	if p, ok := r.(*PipelineResource); ok {
		return p.PipelineName()
	}
	return ""
}

func getPipelineType(r dao.Resource) string {
	if p, ok := r.(*PipelineResource); ok {
		return p.PipelineType()
	}
	return ""
}

func getVersion(r dao.Resource) string {
	if p, ok := r.(*PipelineResource); ok {
		if v := p.Version(); v > 0 {
			return fmt.Sprintf("%d", v)
		}
	}
	return "-"
}

func getExecMode(r dao.Resource) string {
	if p, ok := r.(*PipelineResource); ok {
		return p.ExecutionMode()
	}
	return ""
}

func getUpdated(r dao.Resource) string {
	if p, ok := r.(*PipelineResource); ok {
		return p.UpdatedAt()
	}
	return "-"
}

// RenderDetail renders detailed pipeline information
func (r *PipelineRenderer) RenderDetail(resource dao.Resource) string {
	pipeline, ok := resource.(*PipelineResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("CodePipeline Pipeline", pipeline.PipelineName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", pipeline.PipelineName())
	if arn := pipeline.GetARN(); arn != "" {
		d.Field("ARN", arn)
	}
	d.Field("Type", pipeline.PipelineType())
	d.Field("Version", fmt.Sprintf("%d", pipeline.Version()))
	if mode := pipeline.ExecutionMode(); mode != "" {
		d.Field("Execution Mode", mode)
	}

	// Stages
	if stages := pipeline.Stages(); len(stages) > 0 {
		d.Section("Stages")
		d.Field("Count", fmt.Sprintf("%d", len(stages)))
		d.Field("Stages", strings.Join(stages, " → "))
	}

	// Stage States (if available)
	if stageStates := pipeline.StageStates(); len(stageStates) > 0 {
		d.Section("Stage Status")
		for _, ss := range stageStates {
			status := "-"
			if ss.LatestExecution != nil {
				status = string(ss.LatestExecution.Status)
			}
			if ss.StageName != nil {
				d.Field(*ss.StageName, status)
			}
		}
	}

	// Artifact Store
	if store := pipeline.ArtifactStore(); store != "" {
		d.Section("Artifacts")
		d.Field("Artifact Store", store)
	}

	// IAM
	if role := pipeline.RoleArn(); role != "" {
		d.Section("IAM")
		d.Field("Role ARN", role)
	}

	// Timestamps
	d.Section("Timestamps")
	if created := pipeline.CreatedAt(); created != "" {
		d.Field("Created", created)
	}
	if updated := pipeline.UpdatedAt(); updated != "" {
		d.Field("Last Updated", updated)
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *PipelineRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	pipeline, ok := resource.(*PipelineResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: pipeline.PipelineName()},
		{Label: "Type", Value: pipeline.PipelineType()},
		{Label: "Version", Value: fmt.Sprintf("%d", pipeline.Version())},
	}

	if mode := pipeline.ExecutionMode(); mode != "" {
		fields = append(fields, render.SummaryField{Label: "Execution Mode", Value: mode})
	}

	if stages := pipeline.Stages(); len(stages) > 0 {
		fields = append(fields, render.SummaryField{Label: "Stages", Value: fmt.Sprintf("%d", len(stages))})
	}

	if arn := pipeline.GetARN(); arn != "" {
		fields = append(fields, render.SummaryField{Label: "ARN", Value: arn})
	}

	if updated := pipeline.UpdatedAt(); updated != "" {
		fields = append(fields, render.SummaryField{Label: "Updated", Value: updated})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *PipelineRenderer) Navigations(resource dao.Resource) []render.Navigation {
	pipeline, ok := resource.(*PipelineResource)
	if !ok {
		return nil
	}

	return []render.Navigation{
		{
			Key: "e", Label: "Executions", Service: "codepipeline", Resource: "executions",
			FilterField: "PipelineName", FilterValue: pipeline.PipelineName(),
		},
	}
}
