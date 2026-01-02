package taskexecutions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/datasync"
	"github.com/aws/aws-sdk-go-v2/service/datasync/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// TaskExecutionDAO provides data access for DataSync task executions.
type TaskExecutionDAO struct {
	dao.BaseDAO
	client *datasync.Client
}

// NewTaskExecutionDAO creates a new TaskExecutionDAO.
func NewTaskExecutionDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new datasync/task-executions dao")
	}
	return &TaskExecutionDAO{
		BaseDAO: dao.NewBaseDAO("datasync", "task-executions"),
		client:  datasync.NewFromConfig(cfg),
	}, nil
}

// List returns task executions for the specified task.
func (d *TaskExecutionDAO) List(ctx context.Context) ([]dao.Resource, error) {
	taskArn := dao.GetFilterFromContext(ctx, "TaskArn")
	if taskArn == "" {
		return nil, fmt.Errorf("task ARN filter required")
	}

	executions, err := appaws.Paginate(ctx, func(token *string) ([]types.TaskExecutionListEntry, *string, error) {
		output, err := d.client.ListTaskExecutions(ctx, &datasync.ListTaskExecutionsInput{
			TaskArn:   &taskArn,
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list datasync task executions")
		}
		return output.TaskExecutions, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(executions))
	for i, exec := range executions {
		resources[i] = NewTaskExecutionResource(exec)
	}
	return resources, nil
}

// Get returns a specific task execution.
func (d *TaskExecutionDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	// id is the execution ARN
	output, err := d.client.DescribeTaskExecution(ctx, &datasync.DescribeTaskExecutionInput{
		TaskExecutionArn: &id,
	})
	if err != nil {
		return nil, apperrors.Wrap(err, "describe datasync task execution")
	}

	return &TaskExecutionResource{
		BaseResource: dao.BaseResource{
			ID:  extractExecutionID(id),
			ARN: id,
		},
		Execution: &types.TaskExecutionListEntry{
			TaskExecutionArn: output.TaskExecutionArn,
			Status:           output.Status,
		},
		BytesWritten:     output.BytesWritten,
		BytesTransferred: output.BytesTransferred,
		FilesTransferred: output.FilesTransferred,
		EstimatedFiles:   output.EstimatedFilesToTransfer,
		EstimatedBytes:   output.EstimatedBytesToTransfer,
		StartTime:        output.StartTime,
		FilesDeleted:     output.FilesDeleted,
		FilesSkipped:     output.FilesSkipped,
		FilesVerified:    output.FilesVerified,
		BytesCompressed:  output.BytesCompressed,
		Result:           output.Result,
	}, nil
}

// Delete is not supported for task executions.
func (d *TaskExecutionDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.CancelTaskExecution(ctx, &datasync.CancelTaskExecutionInput{
		TaskExecutionArn: &id,
	})
	if err != nil {
		return apperrors.Wrap(err, "cancel datasync task execution")
	}
	return nil
}

// TaskExecutionResource wraps a DataSync task execution.
type TaskExecutionResource struct {
	dao.BaseResource
	Execution        *types.TaskExecutionListEntry
	BytesWritten     int64
	BytesTransferred int64
	FilesTransferred int64
	EstimatedFiles   int64
	EstimatedBytes   int64
	StartTime        *time.Time
	FilesDeleted     int64
	FilesSkipped     int64
	FilesVerified    int64
	BytesCompressed  int64
	Result           *types.TaskExecutionResultDetail
}

// NewTaskExecutionResource creates a new TaskExecutionResource.
func NewTaskExecutionResource(exec types.TaskExecutionListEntry) *TaskExecutionResource {
	arn := appaws.Str(exec.TaskExecutionArn)
	return &TaskExecutionResource{
		BaseResource: dao.BaseResource{
			ID:  extractExecutionID(arn),
			ARN: arn,
		},
		Execution: &exec,
	}
}

// extractExecutionID extracts the execution ID from an ARN.
func extractExecutionID(arn string) string {
	// Format: arn:aws:datasync:region:account:task/task-xxx/execution/exec-xxx
	if idx := strings.LastIndex(arn, "/"); idx != -1 {
		return arn[idx+1:]
	}
	return arn
}

// Status returns the execution status.
func (r *TaskExecutionResource) Status() string {
	if r.Execution != nil {
		return string(r.Execution.Status)
	}
	return ""
}

// GetStartTime returns the start time.
func (r *TaskExecutionResource) GetStartTime() *time.Time {
	return r.StartTime
}

// GetFilesDeleted returns the files deleted count.
func (r *TaskExecutionResource) GetFilesDeleted() int64 {
	return r.FilesDeleted
}

// GetFilesSkipped returns the files skipped count.
func (r *TaskExecutionResource) GetFilesSkipped() int64 {
	return r.FilesSkipped
}

// GetFilesVerified returns the files verified count.
func (r *TaskExecutionResource) GetFilesVerified() int64 {
	return r.FilesVerified
}

// GetBytesCompressed returns the bytes compressed.
func (r *TaskExecutionResource) GetBytesCompressed() int64 {
	return r.BytesCompressed
}

// GetResult returns the execution result.
func (r *TaskExecutionResource) GetResult() *types.TaskExecutionResultDetail {
	return r.Result
}
