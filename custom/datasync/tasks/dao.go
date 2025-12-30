package tasks

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/datasync"
	"github.com/aws/aws-sdk-go-v2/service/datasync/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// TaskDAO provides data access for DataSync tasks.
type TaskDAO struct {
	dao.BaseDAO
	client *datasync.Client
}

// NewTaskDAO creates a new TaskDAO.
func NewTaskDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new datasync/tasks dao: %w", err)
	}
	return &TaskDAO{
		BaseDAO: dao.NewBaseDAO("datasync", "tasks"),
		client:  datasync.NewFromConfig(cfg),
	}, nil
}

// List returns all DataSync tasks.
func (d *TaskDAO) List(ctx context.Context) ([]dao.Resource, error) {
	tasks, err := appaws.Paginate(ctx, func(token *string) ([]types.TaskListEntry, *string, error) {
		output, err := d.client.ListTasks(ctx, &datasync.ListTasksInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("list datasync tasks: %w", err)
		}
		return output.Tasks, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(tasks))
	for i, task := range tasks {
		resources[i] = NewTaskResource(task)
	}
	return resources, nil
}

// Get returns a specific task.
func (d *TaskDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	// id could be ARN or task ID
	taskArn := id
	if !strings.HasPrefix(id, "arn:") {
		// List and find by ID
		resources, err := d.List(ctx)
		if err != nil {
			return nil, err
		}
		for _, r := range resources {
			if r.GetID() == id {
				return r, nil
			}
		}
		return nil, fmt.Errorf("task not found: %s", id)
	}

	output, err := d.client.DescribeTask(ctx, &datasync.DescribeTaskInput{
		TaskArn: &taskArn,
	})
	if err != nil {
		return nil, fmt.Errorf("describe datasync task: %w", err)
	}

	schedule := ""
	if output.Schedule != nil && output.Schedule.ScheduleExpression != nil {
		schedule = *output.Schedule.ScheduleExpression
	}
	taskMode := ""
	if output.TaskMode != "" {
		taskMode = string(output.TaskMode)
	}

	return &TaskResource{
		BaseResource: dao.BaseResource{
			ID:  extractTaskID(appaws.Str(output.TaskArn)),
			ARN: appaws.Str(output.TaskArn),
		},
		Task: &types.TaskListEntry{
			TaskArn: output.TaskArn,
			Name:    output.Name,
			Status:  output.Status,
		},
		SourceLocationArn:     appaws.Str(output.SourceLocationArn),
		DestLocationArn:       appaws.Str(output.DestinationLocationArn),
		CloudWatchLogGroupArn: appaws.Str(output.CloudWatchLogGroupArn),
		CurrentExecutionArn:   appaws.Str(output.CurrentTaskExecutionArn),
		Schedule:              schedule,
		ErrorCode:             appaws.Str(output.ErrorCode),
		ErrorDetail:           appaws.Str(output.ErrorDetail),
		CreationTime:          output.CreationTime,
		Options:               output.Options,
		TaskMode:              taskMode,
	}, nil
}

// Delete deletes a DataSync task.
func (d *TaskDAO) Delete(ctx context.Context, id string) error {
	// Get the task to find the ARN
	resource, err := d.Get(ctx, id)
	if err != nil {
		return err
	}
	taskArn := resource.GetARN()

	_, err = d.client.DeleteTask(ctx, &datasync.DeleteTaskInput{
		TaskArn: &taskArn,
	})
	if err != nil {
		return fmt.Errorf("delete datasync task: %w", err)
	}
	return nil
}

// TaskResource wraps a DataSync task.
type TaskResource struct {
	dao.BaseResource
	Task                  *types.TaskListEntry
	SourceLocationArn     string
	DestLocationArn       string
	CloudWatchLogGroupArn string
	CurrentExecutionArn   string
	Schedule              string
	ErrorCode             string
	ErrorDetail           string
	CreationTime          *time.Time
	Options               *types.Options
	TaskMode              string
}

// NewTaskResource creates a new TaskResource.
func NewTaskResource(task types.TaskListEntry) *TaskResource {
	arn := appaws.Str(task.TaskArn)
	return &TaskResource{
		BaseResource: dao.BaseResource{
			ID:  extractTaskID(arn),
			ARN: arn,
		},
		Task: &task,
	}
}

// extractTaskID extracts the task ID from an ARN.
func extractTaskID(arn string) string {
	// Format: arn:aws:datasync:region:account:task/task-xxx
	if idx := strings.LastIndex(arn, "/"); idx != -1 {
		return arn[idx+1:]
	}
	return arn
}

// Name returns the task name.
func (r *TaskResource) Name() string {
	if r.Task != nil && r.Task.Name != nil {
		return *r.Task.Name
	}
	return ""
}

// Status returns the task status.
func (r *TaskResource) Status() string {
	if r.Task != nil {
		return string(r.Task.Status)
	}
	return ""
}

// TaskArn returns the task ARN.
func (r *TaskResource) TaskArn() string {
	return r.ARN
}

// GetCloudWatchLogGroupArn returns the CloudWatch log group ARN.
func (r *TaskResource) GetCloudWatchLogGroupArn() string {
	return r.CloudWatchLogGroupArn
}

// GetCurrentExecutionArn returns the current execution ARN.
func (r *TaskResource) GetCurrentExecutionArn() string {
	return r.CurrentExecutionArn
}

// GetSchedule returns the task schedule.
func (r *TaskResource) GetSchedule() string {
	return r.Schedule
}

// GetErrorCode returns the error code.
func (r *TaskResource) GetErrorCode() string {
	return r.ErrorCode
}

// GetErrorDetail returns the error detail.
func (r *TaskResource) GetErrorDetail() string {
	return r.ErrorDetail
}

// GetCreationTime returns the creation time.
func (r *TaskResource) GetCreationTime() *time.Time {
	return r.CreationTime
}

// GetOptions returns the task options.
func (r *TaskResource) GetOptions() *types.Options {
	return r.Options
}

// GetTaskMode returns the task mode.
func (r *TaskResource) GetTaskMode() string {
	return r.TaskMode
}
