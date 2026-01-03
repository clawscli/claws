package topics

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
	"github.com/clawscli/claws/internal/log"
)

// TopicDAO provides data access for SNS topics
type TopicDAO struct {
	dao.BaseDAO
	client *sns.Client
}

// NewTopicDAO creates a new TopicDAO
func NewTopicDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &TopicDAO{
		BaseDAO: dao.NewBaseDAO("sns", "topics"),
		client:  sns.NewFromConfig(cfg),
	}, nil
}

func (d *TopicDAO) List(ctx context.Context) ([]dao.Resource, error) {
	input := &sns.ListTopicsInput{}

	var resources []dao.Resource
	paginator := sns.NewListTopicsPaginator(d.client, input)

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, apperrors.Wrap(err, "list topics")
		}

		for _, topic := range output.Topics {
			if topic.TopicArn == nil {
				continue
			}
			// Get topic attributes for more details
			attrs, err := d.client.GetTopicAttributes(ctx, &sns.GetTopicAttributesInput{
				TopicArn: topic.TopicArn,
			})
			if err != nil {
				log.Warn("failed to get topic attributes", "arn", appaws.Str(topic.TopicArn), "error", err)
				continue
			}
			resources = append(resources, NewTopicResource(topic, attrs.Attributes))
		}
	}

	return resources, nil
}

func (d *TopicDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	// id is the topic ARN
	attrs, err := d.client.GetTopicAttributes(ctx, &sns.GetTopicAttributesInput{
		TopicArn: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get topic %s", id)
	}

	topic := types.Topic{TopicArn: &id}
	return NewTopicResource(topic, attrs.Attributes), nil
}

func (d *TopicDAO) Delete(ctx context.Context, id string) error {
	input := &sns.DeleteTopicInput{
		TopicArn: &id,
	}

	_, err := d.client.DeleteTopic(ctx, input)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil // Already deleted
		}
		if apperrors.IsResourceInUse(err) {
			return apperrors.Wrapf(err, "topic %s is in use", id)
		}
		return apperrors.Wrapf(err, "delete topic %s", id)
	}

	return nil
}

// TopicResource wraps an SNS topic
type TopicResource struct {
	dao.BaseResource
	Item  types.Topic
	Attrs map[string]string
}

// NewTopicResource creates a new TopicResource
func NewTopicResource(topic types.Topic, attrs map[string]string) *TopicResource {
	arn := appaws.Str(topic.TopicArn)
	name := ""
	if arn != "" {
		// Extract name from ARN (last part after :)
		parts := strings.Split(arn, ":")
		if len(parts) > 0 {
			name = parts[len(parts)-1]
		}
	}

	return &TopicResource{
		BaseResource: dao.BaseResource{
			ID:   arn,
			Name: name,
			ARN:  arn,
			Tags: nil,
			Data: topic,
		},
		Item:  topic,
		Attrs: attrs,
	}
}

// ARN returns the topic ARN
func (r *TopicResource) ARN() string {
	if r.Item.TopicArn != nil {
		return *r.Item.TopicArn
	}
	return ""
}

// DisplayName returns the display name
func (r *TopicResource) DisplayName() string {
	return r.Attrs["DisplayName"]
}

// SubscriptionCount returns the number of subscriptions
func (r *TopicResource) SubscriptionCount() string {
	if count, ok := r.Attrs["SubscriptionsConfirmed"]; ok {
		return count
	}
	return "0"
}

// PendingSubscriptions returns the number of pending subscriptions
func (r *TopicResource) PendingSubscriptions() string {
	if count, ok := r.Attrs["SubscriptionsPending"]; ok {
		return count
	}
	return "0"
}

// IsFIFO returns whether the topic is a FIFO topic
func (r *TopicResource) IsFIFO() bool {
	return r.Attrs["FifoTopic"] == "true"
}

// Owner returns the topic owner
func (r *TopicResource) Owner() string {
	return r.Attrs["Owner"]
}
