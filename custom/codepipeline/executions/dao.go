package executions

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/codepipeline"
	"github.com/aws/aws-sdk-go-v2/service/codepipeline/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// ExecutionDAO provides data access for CodePipeline executions
type ExecutionDAO struct {
	dao.BaseDAO
	client *codepipeline.Client
}

// NewExecutionDAO creates a new ExecutionDAO
func NewExecutionDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new codepipeline/executions dao: %w", err)
	}
	return &ExecutionDAO{
		BaseDAO: dao.NewBaseDAO("codepipeline", "executions"),
		client:  codepipeline.NewFromConfig(cfg),
	}, nil
}

// List returns executions (first page only for backwards compatibility).
// For paginated access, use ListPage instead.
func (d *ExecutionDAO) List(ctx context.Context) ([]dao.Resource, error) {
	resources, _, err := d.ListPage(ctx, 50, "")
	return resources, err
}

// ListPage returns a page of CodePipeline executions.
// Implements dao.PaginatedDAO interface.
func (d *ExecutionDAO) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
	// Get pipeline name from filter context
	pipelineName := dao.GetFilterFromContext(ctx, "PipelineName")
	if pipelineName == "" {
		return nil, "", fmt.Errorf("pipeline name filter required")
	}

	maxResults := int32(pageSize)
	if maxResults > 100 {
		maxResults = 100 // AWS API max
	}

	input := &codepipeline.ListPipelineExecutionsInput{
		PipelineName: &pipelineName,
		MaxResults:   &maxResults,
	}
	if pageToken != "" {
		input.NextToken = &pageToken
	}

	output, err := d.client.ListPipelineExecutions(ctx, input)
	if err != nil {
		return nil, "", fmt.Errorf("list pipeline executions: %w", err)
	}

	resources := make([]dao.Resource, 0, len(output.PipelineExecutionSummaries))
	for _, exec := range output.PipelineExecutionSummaries {
		resources = append(resources, NewExecutionResource(exec, pipelineName))
	}

	nextToken := ""
	if output.NextToken != nil {
		nextToken = *output.NextToken
	}

	return resources, nextToken, nil
}

// Get returns a specific execution
func (d *ExecutionDAO) Get(ctx context.Context, executionId string) (dao.Resource, error) {
	pipelineName := dao.GetFilterFromContext(ctx, "PipelineName")
	if pipelineName == "" {
		return nil, fmt.Errorf("pipeline name filter required")
	}

	input := &codepipeline.GetPipelineExecutionInput{
		PipelineName:        &pipelineName,
		PipelineExecutionId: &executionId,
	}

	output, err := d.client.GetPipelineExecution(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("get execution %s: %w", executionId, err)
	}

	return NewExecutionResourceFromDetail(output.PipelineExecution, pipelineName), nil
}

// Delete stops a pipeline execution
func (d *ExecutionDAO) Delete(ctx context.Context, executionId string) error {
	pipelineName := dao.GetFilterFromContext(ctx, "PipelineName")
	if pipelineName == "" {
		return fmt.Errorf("pipeline name filter required")
	}

	_, err := d.client.StopPipelineExecution(ctx, &codepipeline.StopPipelineExecutionInput{
		PipelineName:        &pipelineName,
		PipelineExecutionId: &executionId,
		Abandon:             true,
		Reason:              appaws.StringPtr("Stopped via claws"),
	})
	if err != nil {
		return fmt.Errorf("stop execution %s: %w", executionId, err)
	}
	return nil
}

// Supports returns supported operations
func (d *ExecutionDAO) Supports(op dao.Operation) bool {
	switch op {
	case dao.OpList, dao.OpGet, dao.OpDelete:
		return true
	default:
		return false
	}
}

// ExecutionResource represents a CodePipeline execution
type ExecutionResource struct {
	dao.BaseResource
	Summary      *types.PipelineExecutionSummary
	Detail       *types.PipelineExecution
	PipelineName string
}

// NewExecutionResource creates a new ExecutionResource from summary
func NewExecutionResource(summary types.PipelineExecutionSummary, pipelineName string) *ExecutionResource {
	id := appaws.Str(summary.PipelineExecutionId)

	return &ExecutionResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: id,
			ARN:  "",
			Tags: make(map[string]string),
			Data: summary,
		},
		Summary:      &summary,
		PipelineName: pipelineName,
	}
}

// NewExecutionResourceFromDetail creates a new ExecutionResource from detail
func NewExecutionResourceFromDetail(detail *types.PipelineExecution, pipelineName string) *ExecutionResource {
	id := appaws.Str(detail.PipelineExecutionId)

	return &ExecutionResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: id,
			ARN:  "",
			Tags: make(map[string]string),
			Data: detail,
		},
		Detail:       detail,
		PipelineName: pipelineName,
	}
}

// ExecutionId returns the execution ID
func (r *ExecutionResource) ExecutionId() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.PipelineExecutionId)
	}
	if r.Detail != nil {
		return appaws.Str(r.Detail.PipelineExecutionId)
	}
	return ""
}

// Status returns the execution status
func (r *ExecutionResource) Status() string {
	if r.Summary != nil {
		return string(r.Summary.Status)
	}
	if r.Detail != nil {
		return string(r.Detail.Status)
	}
	return ""
}

// Trigger returns what triggered the execution
func (r *ExecutionResource) Trigger() string {
	if r.Summary != nil && r.Summary.Trigger != nil {
		return string(r.Summary.Trigger.TriggerType)
	}
	if r.Detail != nil && r.Detail.Trigger != nil {
		return string(r.Detail.Trigger.TriggerType)
	}
	return ""
}

// TriggerDetail returns detailed trigger info
func (r *ExecutionResource) TriggerDetail() string {
	if r.Summary != nil && r.Summary.Trigger != nil {
		return appaws.Str(r.Summary.Trigger.TriggerDetail)
	}
	if r.Detail != nil && r.Detail.Trigger != nil {
		return appaws.Str(r.Detail.Trigger.TriggerDetail)
	}
	return ""
}

// SourceRevisions returns source revisions
func (r *ExecutionResource) SourceRevisions() []types.SourceRevision {
	if r.Summary != nil {
		return r.Summary.SourceRevisions
	}
	return nil
}

// StartTime returns the start time
func (r *ExecutionResource) StartTime() string {
	if r.Summary != nil && r.Summary.StartTime != nil {
		return r.Summary.StartTime.Format("2006-01-02 15:04:05")
	}
	return ""
}

// StartTimeT returns the start time as time.Time
func (r *ExecutionResource) StartTimeT() *time.Time {
	if r.Summary != nil {
		return r.Summary.StartTime
	}
	return nil
}

// LastUpdateTime returns the last update time
func (r *ExecutionResource) LastUpdateTime() string {
	if r.Summary != nil && r.Summary.LastUpdateTime != nil {
		return r.Summary.LastUpdateTime.Format("2006-01-02 15:04:05")
	}
	return ""
}

// StopTime returns the stop time
func (r *ExecutionResource) StopTime() string {
	if r.Summary != nil && r.Summary.StopTrigger != nil && r.Summary.StopTrigger.Reason != nil {
		return *r.Summary.StopTrigger.Reason
	}
	return ""
}

// ExecutionMode returns the execution mode
func (r *ExecutionResource) ExecutionMode() string {
	if r.Summary != nil {
		return string(r.Summary.ExecutionMode)
	}
	return ""
}

// PipelineVersion returns the pipeline version
func (r *ExecutionResource) PipelineVersion() int32 {
	if r.Detail != nil && r.Detail.PipelineVersion != nil {
		return *r.Detail.PipelineVersion
	}
	return 0
}
