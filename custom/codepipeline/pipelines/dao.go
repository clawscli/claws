package pipelines

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/codepipeline"
	"github.com/aws/aws-sdk-go-v2/service/codepipeline/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// PipelineDAO provides data access for CodePipeline pipelines
type PipelineDAO struct {
	dao.BaseDAO
	client *codepipeline.Client
}

// NewPipelineDAO creates a new PipelineDAO
func NewPipelineDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new codepipeline/pipelines dao: %w", err)
	}
	return &PipelineDAO{
		BaseDAO: dao.NewBaseDAO("codepipeline", "pipelines"),
		client:  codepipeline.NewFromConfig(cfg),
	}, nil
}

// List returns all CodePipeline pipelines
func (d *PipelineDAO) List(ctx context.Context) ([]dao.Resource, error) {
	pipelines, err := appaws.Paginate(ctx, func(token *string) ([]types.PipelineSummary, *string, error) {
		output, err := d.client.ListPipelines(ctx, &codepipeline.ListPipelinesInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("list pipelines: %w", err)
		}
		return output.Pipelines, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(pipelines))
	for i, summary := range pipelines {
		resources[i] = NewPipelineResourceFromSummary(summary)
	}

	return resources, nil
}

// Get returns a specific CodePipeline pipeline by name
func (d *PipelineDAO) Get(ctx context.Context, name string) (dao.Resource, error) {
	// Get pipeline definition
	pipelineOutput, err := d.client.GetPipeline(ctx, &codepipeline.GetPipelineInput{
		Name: &name,
	})
	if err != nil {
		return nil, fmt.Errorf("get pipeline %s: %w", name, err)
	}

	// Get pipeline state (execution status)
	stateOutput, err := d.client.GetPipelineState(ctx, &codepipeline.GetPipelineStateInput{
		Name: &name,
	})
	if err != nil {
		// State might not be available if pipeline has never run
		return NewPipelineResourceFromDetail(pipelineOutput, nil), nil
	}

	return NewPipelineResourceFromDetail(pipelineOutput, stateOutput), nil
}

// Delete deletes a CodePipeline pipeline
func (d *PipelineDAO) Delete(ctx context.Context, name string) error {
	_, err := d.client.DeletePipeline(ctx, &codepipeline.DeletePipelineInput{
		Name: &name,
	})
	if err != nil {
		return fmt.Errorf("delete pipeline %s: %w", name, err)
	}
	return nil
}

// PipelineResource represents a CodePipeline pipeline
type PipelineResource struct {
	dao.BaseResource
	Summary  *types.PipelineSummary
	Pipeline *types.PipelineDeclaration
	Metadata *types.PipelineMetadata
	State    *codepipeline.GetPipelineStateOutput
}

// NewPipelineResourceFromSummary creates a new PipelineResource from summary
func NewPipelineResourceFromSummary(summary types.PipelineSummary) *PipelineResource {
	name := appaws.Str(summary.Name)

	return &PipelineResource{
		BaseResource: dao.BaseResource{
			ID:   name,
			Name: name,
			ARN:  "", // ARN not in summary
			Tags: make(map[string]string),
			Data: summary,
		},
		Summary: &summary,
	}
}

// NewPipelineResourceFromDetail creates a new PipelineResource from detail
func NewPipelineResourceFromDetail(pipelineOutput *codepipeline.GetPipelineOutput, stateOutput *codepipeline.GetPipelineStateOutput) *PipelineResource {
	name := ""
	var pipeline *types.PipelineDeclaration
	var metadata *types.PipelineMetadata

	if pipelineOutput != nil {
		pipeline = pipelineOutput.Pipeline
		metadata = pipelineOutput.Metadata

		if pipeline != nil {
			name = appaws.Str(pipeline.Name)
		}
	}

	arn := ""
	if metadata != nil {
		arn = appaws.Str(metadata.PipelineArn)
	}

	return &PipelineResource{
		BaseResource: dao.BaseResource{
			ID:   name,
			Name: name,
			ARN:  arn,
			Tags: make(map[string]string),
			Data: pipelineOutput,
		},
		Pipeline: pipeline,
		Metadata: metadata,
		State:    stateOutput,
	}
}

// PipelineName returns the pipeline name
func (r *PipelineResource) PipelineName() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.Name)
	}
	if r.Pipeline != nil {
		return appaws.Str(r.Pipeline.Name)
	}
	return ""
}

// Version returns the pipeline version
func (r *PipelineResource) Version() int32 {
	if r.Summary != nil && r.Summary.Version != nil {
		return *r.Summary.Version
	}
	if r.Pipeline != nil && r.Pipeline.Version != nil {
		return *r.Pipeline.Version
	}
	return 0
}

// PipelineType returns the pipeline type (V1 or V2)
func (r *PipelineResource) PipelineType() string {
	if r.Summary != nil {
		return string(r.Summary.PipelineType)
	}
	if r.Pipeline != nil {
		return string(r.Pipeline.PipelineType)
	}
	return ""
}

// ExecutionMode returns the pipeline execution mode
func (r *PipelineResource) ExecutionMode() string {
	if r.Summary != nil {
		return string(r.Summary.ExecutionMode)
	}
	if r.Pipeline != nil {
		return string(r.Pipeline.ExecutionMode)
	}
	return ""
}

// StageCount returns the number of stages
func (r *PipelineResource) StageCount() int {
	if r.Pipeline != nil {
		return len(r.Pipeline.Stages)
	}
	return 0
}

// Stages returns the list of stage names
func (r *PipelineResource) Stages() []string {
	if r.Pipeline == nil {
		return nil
	}
	stages := make([]string, 0, len(r.Pipeline.Stages))
	for _, stage := range r.Pipeline.Stages {
		if stage.Name != nil {
			stages = append(stages, *stage.Name)
		}
	}
	return stages
}

// LatestExecutionStatus returns the latest execution status
func (r *PipelineResource) LatestExecutionStatus() string {
	if r.State != nil && len(r.State.StageStates) > 0 {
		// Get the status from the pipeline state
		for _, stage := range r.State.StageStates {
			if stage.LatestExecution != nil {
				return string(stage.LatestExecution.Status)
			}
		}
	}
	return ""
}

// CreatedAt returns the creation date
func (r *PipelineResource) CreatedAt() string {
	if r.Summary != nil && r.Summary.Created != nil {
		return r.Summary.Created.Format("2006-01-02 15:04:05")
	}
	if r.Metadata != nil && r.Metadata.Created != nil {
		return r.Metadata.Created.Format("2006-01-02 15:04:05")
	}
	return ""
}

// CreatedAtTime returns the creation date as time.Time
func (r *PipelineResource) CreatedAtTime() *time.Time {
	if r.Summary != nil && r.Summary.Created != nil {
		return r.Summary.Created
	}
	if r.Metadata != nil && r.Metadata.Created != nil {
		return r.Metadata.Created
	}
	return nil
}

// UpdatedAt returns the last updated date
func (r *PipelineResource) UpdatedAt() string {
	if r.Summary != nil && r.Summary.Updated != nil {
		return r.Summary.Updated.Format("2006-01-02 15:04:05")
	}
	if r.Metadata != nil && r.Metadata.Updated != nil {
		return r.Metadata.Updated.Format("2006-01-02 15:04:05")
	}
	return ""
}

// RoleArn returns the pipeline role ARN
func (r *PipelineResource) RoleArn() string {
	if r.Pipeline != nil {
		return appaws.Str(r.Pipeline.RoleArn)
	}
	return ""
}

// ArtifactStore returns the artifact store information
func (r *PipelineResource) ArtifactStore() string {
	if r.Pipeline != nil && r.Pipeline.ArtifactStore != nil {
		return fmt.Sprintf("%s: %s", r.Pipeline.ArtifactStore.Type, appaws.Str(r.Pipeline.ArtifactStore.Location))
	}
	return ""
}

// StageStates returns the stage states from pipeline state
func (r *PipelineResource) StageStates() []types.StageState {
	if r.State != nil {
		return r.State.StageStates
	}
	return nil
}
