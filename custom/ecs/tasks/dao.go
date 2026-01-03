package tasks

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
	"github.com/clawscli/claws/internal/log"
)

// TaskDAO provides data access for ECS tasks
type TaskDAO struct {
	dao.BaseDAO
	client *ecs.Client
}

// NewTaskDAO creates a new TaskDAO
func NewTaskDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &TaskDAO{
		BaseDAO: dao.NewBaseDAO("ecs", "tasks"),
		client:  ecs.NewFromConfig(cfg),
	}, nil
}

func (d *TaskDAO) List(ctx context.Context) ([]dao.Resource, error) {
	clusterName := dao.GetFilterFromContext(ctx, "ClusterName")
	if clusterName == "" {
		// List tasks from all clusters
		return d.listAllTasks(ctx)
	}

	return d.listTasksInCluster(ctx, clusterName)
}

func (d *TaskDAO) listAllTasks(ctx context.Context) ([]dao.Resource, error) {
	// First get all clusters
	clusterArns, err := appaws.Paginate(ctx, func(token *string) ([]string, *string, error) {
		output, err := d.client.ListClusters(ctx, &ecs.ListClustersInput{NextToken: token})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list clusters")
		}
		return output.ClusterArns, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, 0, len(clusterArns))
	for _, clusterArn := range clusterArns {
		clusterTasks, err := d.listTasksInCluster(ctx, clusterArn)
		if err != nil {
			log.Warn("failed to list tasks in cluster", "cluster", clusterArn, "error", err)
			continue
		}
		resources = append(resources, clusterTasks...)
	}

	return resources, nil
}

func (d *TaskDAO) listTasksInCluster(ctx context.Context, cluster string) ([]dao.Resource, error) {
	serviceName := dao.GetFilterFromContext(ctx, "ServiceName")

	taskArns, err := appaws.Paginate(ctx, func(token *string) ([]string, *string, error) {
		input := &ecs.ListTasksInput{
			Cluster:   &cluster,
			NextToken: token,
		}
		if serviceName != "" {
			input.ServiceName = &serviceName
		}
		output, err := d.client.ListTasks(ctx, input)
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list tasks")
		}
		return output.TaskArns, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	if len(taskArns) == 0 {
		return nil, nil
	}

	// Describe tasks in batches of 100 (API limit)
	resources := make([]dao.Resource, 0, len(taskArns))
	for i := 0; i < len(taskArns); i += 100 {
		end := i + 100
		if end > len(taskArns) {
			end = len(taskArns)
		}

		descInput := &ecs.DescribeTasksInput{
			Cluster: &cluster,
			Tasks:   taskArns[i:end],
		}

		descOutput, err := d.client.DescribeTasks(ctx, descInput)
		if err != nil {
			log.Warn("failed to describe tasks", "cluster", cluster, "error", err)
			continue
		}

		for _, task := range descOutput.Tasks {
			resources = append(resources, NewTaskResource(task))
		}
	}

	return resources, nil
}

func (d *TaskDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	clusterName := dao.GetFilterFromContext(ctx, "ClusterName")
	if clusterName == "" {
		return nil, fmt.Errorf("cluster name filter required")
	}

	input := &ecs.DescribeTasksInput{
		Cluster: &clusterName,
		Tasks:   []string{id},
	}

	output, err := d.client.DescribeTasks(ctx, input)
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe task %s", id)
	}

	if len(output.Tasks) == 0 {
		return nil, fmt.Errorf("task not found: %s", id)
	}

	return NewTaskResource(output.Tasks[0]), nil
}

func (d *TaskDAO) Delete(ctx context.Context, id string) error {
	clusterName := dao.GetFilterFromContext(ctx, "ClusterName")
	if clusterName == "" {
		return fmt.Errorf("cluster name filter required")
	}

	input := &ecs.StopTaskInput{
		Cluster: &clusterName,
		Task:    &id,
		Reason:  appaws.StringPtr("Stopped via claws"),
	}

	_, err := d.client.StopTask(ctx, input)
	if err != nil {
		return apperrors.Wrapf(err, "stop task %s", id)
	}

	return nil
}

// TaskResource wraps an ECS task
type TaskResource struct {
	dao.BaseResource
	Item types.Task
}

// NewTaskResource creates a new TaskResource
func NewTaskResource(task types.Task) *TaskResource {
	// Extract task ID from ARN
	taskArn := appaws.Str(task.TaskArn)
	taskID := taskArn
	if parts := splitArn(taskArn); len(parts) > 0 {
		taskID = parts[len(parts)-1]
	}

	return &TaskResource{
		BaseResource: dao.BaseResource{
			ID:   taskID,
			Name: taskID,
			ARN:  taskArn,
			Data: task,
		},
		Item: task,
	}
}

func splitArn(arn string) []string {
	var parts []string
	current := ""
	for _, c := range arn {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
			}
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

// LastStatus returns the last known status
func (r *TaskResource) LastStatus() string {
	return appaws.Str(r.Item.LastStatus)
}

// DesiredStatus returns the desired status
func (r *TaskResource) DesiredStatus() string {
	return appaws.Str(r.Item.DesiredStatus)
}

// LaunchType returns the launch type
func (r *TaskResource) LaunchType() string {
	return string(r.Item.LaunchType)
}

// TaskDefinitionArn returns the task definition ARN
func (r *TaskResource) TaskDefinitionArn() string {
	return appaws.Str(r.Item.TaskDefinitionArn)
}

// CPU returns the CPU units
func (r *TaskResource) CPU() string {
	return appaws.Str(r.Item.Cpu)
}

// Memory returns the memory
func (r *TaskResource) Memory() string {
	return appaws.Str(r.Item.Memory)
}

// Containers returns the containers
func (r *TaskResource) Containers() []types.Container {
	return r.Item.Containers
}

// StartedAt returns when the task started
func (r *TaskResource) StartedAt() string {
	if r.Item.StartedAt != nil {
		return r.Item.StartedAt.Format("2006-01-02 15:04:05")
	}
	return ""
}

// StoppedReason returns why the task stopped
func (r *TaskResource) StoppedReason() string {
	return appaws.Str(r.Item.StoppedReason)
}

// HealthStatus returns the health status
func (r *TaskResource) HealthStatus() string {
	return string(r.Item.HealthStatus)
}

// ClusterArn returns the cluster ARN
func (r *TaskResource) ClusterArn() string {
	return appaws.Str(r.Item.ClusterArn)
}

// Group returns the task group
func (r *TaskResource) Group() string {
	return appaws.Str(r.Item.Group)
}

// FirstContainerName returns the name of the first container (for ECS Exec)
func (r *TaskResource) FirstContainerName() string {
	if len(r.Item.Containers) > 0 && r.Item.Containers[0].Name != nil {
		return *r.Item.Containers[0].Name
	}
	return ""
}

// EnableExecuteCommand returns whether execute command is enabled for this task
func (r *TaskResource) EnableExecuteCommand() bool {
	return r.Item.EnableExecuteCommand
}
