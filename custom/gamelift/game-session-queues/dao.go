package gamesessionqueues

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/gamelift"
	"github.com/aws/aws-sdk-go-v2/service/gamelift/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// QueueDAO provides data access for GameLift game session queues.
type QueueDAO struct {
	dao.BaseDAO
	client *gamelift.Client
}

// NewQueueDAO creates a new QueueDAO.
func NewQueueDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &QueueDAO{
		BaseDAO: dao.NewBaseDAO("gamelift", "game-session-queues"),
		client:  gamelift.NewFromConfig(cfg),
	}, nil
}

// List returns all GameLift game session queues.
func (d *QueueDAO) List(ctx context.Context) ([]dao.Resource, error) {
	queues, err := appaws.Paginate(ctx, func(token *string) ([]types.GameSessionQueue, *string, error) {
		output, err := d.client.DescribeGameSessionQueues(ctx, &gamelift.DescribeGameSessionQueuesInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "describe gamelift game session queues")
		}
		return output.GameSessionQueues, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(queues))
	for i, queue := range queues {
		resources[i] = NewQueueResource(queue)
	}
	return resources, nil
}

// Get returns a specific GameLift game session queue by name.
func (d *QueueDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeGameSessionQueues(ctx, &gamelift.DescribeGameSessionQueuesInput{
		Names: []string{id},
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe gamelift game session queue %s", id)
	}
	if len(output.GameSessionQueues) == 0 {
		return nil, fmt.Errorf("gamelift game session queue %s not found", id)
	}
	return NewQueueResource(output.GameSessionQueues[0]), nil
}

// Delete deletes a GameLift game session queue by name.
func (d *QueueDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteGameSessionQueue(ctx, &gamelift.DeleteGameSessionQueueInput{
		Name: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete gamelift game session queue %s", id)
	}
	return nil
}

// QueueResource wraps a GameLift game session queue.
type QueueResource struct {
	dao.BaseResource
	Queue types.GameSessionQueue
}

// NewQueueResource creates a new QueueResource.
func NewQueueResource(queue types.GameSessionQueue) *QueueResource {
	return &QueueResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(queue.Name),
			Name: appaws.Str(queue.Name),
			ARN:  appaws.Str(queue.GameSessionQueueArn),
			Data: queue,
		},
		Queue: queue,
	}
}

// TimeoutInSeconds returns the queue timeout.
func (r *QueueResource) TimeoutInSeconds() int32 {
	return appaws.Int32(r.Queue.TimeoutInSeconds)
}

// Destinations returns the queue destinations.
func (r *QueueResource) Destinations() []types.GameSessionQueueDestination {
	return r.Queue.Destinations
}

// DestinationCount returns the number of destinations.
func (r *QueueResource) DestinationCount() int {
	return len(r.Queue.Destinations)
}

// NotificationTarget returns the SNS notification target.
func (r *QueueResource) NotificationTarget() string {
	return appaws.Str(r.Queue.NotificationTarget)
}

// CustomEventData returns the custom event data.
func (r *QueueResource) CustomEventData() string {
	return appaws.Str(r.Queue.CustomEventData)
}

// PlayerLatencyPolicies returns the player latency policies.
func (r *QueueResource) PlayerLatencyPolicies() []types.PlayerLatencyPolicy {
	return r.Queue.PlayerLatencyPolicies
}

// FilterConfiguration returns the filter configuration.
func (r *QueueResource) FilterConfiguration() *types.FilterConfiguration {
	return r.Queue.FilterConfiguration
}

// PriorityConfiguration returns the priority configuration.
func (r *QueueResource) PriorityConfiguration() *types.PriorityConfiguration {
	return r.Queue.PriorityConfiguration
}
