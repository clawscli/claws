package jobqueues

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/batch"
	"github.com/aws/aws-sdk-go-v2/service/batch/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// JobQueueDAO provides data access for Batch job queues.
type JobQueueDAO struct {
	dao.BaseDAO
	client *batch.Client
}

// NewJobQueueDAO creates a new JobQueueDAO.
func NewJobQueueDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new batch/job-queues dao")
	}
	return &JobQueueDAO{
		BaseDAO: dao.NewBaseDAO("batch", "job-queues"),
		client:  batch.NewFromConfig(cfg),
	}, nil
}

// List returns all Batch job queues.
func (d *JobQueueDAO) List(ctx context.Context) ([]dao.Resource, error) {
	queues, err := appaws.Paginate(ctx, func(token *string) ([]types.JobQueueDetail, *string, error) {
		output, err := d.client.DescribeJobQueues(ctx, &batch.DescribeJobQueuesInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "describe batch job queues")
		}
		return output.JobQueues, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(queues))
	for i, queue := range queues {
		resources[i] = NewJobQueueResource(queue)
	}
	return resources, nil
}

// Get returns a specific job queue.
func (d *JobQueueDAO) Get(ctx context.Context, name string) (dao.Resource, error) {
	output, err := d.client.DescribeJobQueues(ctx, &batch.DescribeJobQueuesInput{
		JobQueues: []string{name},
	})
	if err != nil {
		return nil, apperrors.Wrap(err, "describe batch job queue")
	}
	if len(output.JobQueues) == 0 {
		return nil, fmt.Errorf("job queue not found: %s", name)
	}
	return NewJobQueueResource(output.JobQueues[0]), nil
}

// Delete deletes a Batch job queue.
func (d *JobQueueDAO) Delete(ctx context.Context, name string) error {
	// First disable the queue
	_, err := d.client.UpdateJobQueue(ctx, &batch.UpdateJobQueueInput{
		JobQueue: &name,
		State:    types.JQStateDisabled,
	})
	if err != nil {
		return apperrors.Wrap(err, "disable batch job queue")
	}

	// Then delete it
	_, err = d.client.DeleteJobQueue(ctx, &batch.DeleteJobQueueInput{
		JobQueue: &name,
	})
	if err != nil {
		return apperrors.Wrap(err, "delete batch job queue")
	}
	return nil
}

// JobQueueResource wraps a Batch job queue.
type JobQueueResource struct {
	dao.BaseResource
	Queue *types.JobQueueDetail
}

// NewJobQueueResource creates a new JobQueueResource.
func NewJobQueueResource(queue types.JobQueueDetail) *JobQueueResource {
	return &JobQueueResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(queue.JobQueueName),
			ARN: appaws.Str(queue.JobQueueArn),
		},
		Queue: &queue,
	}
}

// State returns the queue state.
func (r *JobQueueResource) State() string {
	if r.Queue != nil {
		return string(r.Queue.State)
	}
	return ""
}

// Status returns the queue status.
func (r *JobQueueResource) Status() string {
	if r.Queue != nil {
		return string(r.Queue.Status)
	}
	return ""
}

// Priority returns the queue priority.
func (r *JobQueueResource) Priority() int32 {
	if r.Queue != nil && r.Queue.Priority != nil {
		return *r.Queue.Priority
	}
	return 0
}

// SchedulingPolicy returns the scheduling policy ARN.
func (r *JobQueueResource) SchedulingPolicy() string {
	if r.Queue != nil && r.Queue.SchedulingPolicyArn != nil {
		return *r.Queue.SchedulingPolicyArn
	}
	return ""
}
