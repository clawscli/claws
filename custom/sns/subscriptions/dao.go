package subscriptions

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// SubscriptionDAO provides data access for SNS subscriptions
type SubscriptionDAO struct {
	dao.BaseDAO
	client *sns.Client
}

// NewSubscriptionDAO creates a new SubscriptionDAO
func NewSubscriptionDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &SubscriptionDAO{
		BaseDAO: dao.NewBaseDAO("sns", "subscriptions"),
		client:  sns.NewFromConfig(cfg),
	}, nil
}

func (d *SubscriptionDAO) List(ctx context.Context) ([]dao.Resource, error) {
	input := &sns.ListSubscriptionsInput{}

	var resources []dao.Resource
	paginator := sns.NewListSubscriptionsPaginator(d.client, input)

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, apperrors.Wrap(err, "list subscriptions")
		}

		for _, sub := range output.Subscriptions {
			resources = append(resources, NewSubscriptionResource(sub))
		}
	}

	return resources, nil
}

func (d *SubscriptionDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	// id is the subscription ARN
	attrs, err := d.client.GetSubscriptionAttributes(ctx, &sns.GetSubscriptionAttributesInput{
		SubscriptionArn: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get subscription %s", id)
	}

	// Reconstruct subscription from attributes
	sub := types.Subscription{
		SubscriptionArn: &id,
	}
	if topicArn, ok := attrs.Attributes["TopicArn"]; ok {
		sub.TopicArn = &topicArn
	}
	if endpoint, ok := attrs.Attributes["Endpoint"]; ok {
		sub.Endpoint = &endpoint
	}
	if protocol, ok := attrs.Attributes["Protocol"]; ok {
		sub.Protocol = &protocol
	}
	if owner, ok := attrs.Attributes["Owner"]; ok {
		sub.Owner = &owner
	}

	return NewSubscriptionResourceWithAttrs(sub, attrs.Attributes), nil
}

func (d *SubscriptionDAO) Delete(ctx context.Context, id string) error {
	input := &sns.UnsubscribeInput{
		SubscriptionArn: &id,
	}

	_, err := d.client.Unsubscribe(ctx, input)
	if err != nil {
		return apperrors.Wrapf(err, "unsubscribe %s", id)
	}

	return nil
}

// SubscriptionResource wraps an SNS subscription
type SubscriptionResource struct {
	dao.BaseResource
	Item  types.Subscription
	Attrs map[string]string
}

// NewSubscriptionResource creates a new SubscriptionResource
func NewSubscriptionResource(sub types.Subscription) *SubscriptionResource {
	return NewSubscriptionResourceWithAttrs(sub, nil)
}

// NewSubscriptionResourceWithAttrs creates a new SubscriptionResource with attributes
func NewSubscriptionResourceWithAttrs(sub types.Subscription, attrs map[string]string) *SubscriptionResource {
	arn := appaws.Str(sub.SubscriptionArn)
	name := appaws.Str(sub.Endpoint) // Use endpoint as display name

	return &SubscriptionResource{
		BaseResource: dao.BaseResource{
			ID:   arn,
			Name: name,
			Tags: nil,
			Data: sub,
		},
		Item:  sub,
		Attrs: attrs,
	}
}

// ARN returns the subscription ARN
func (r *SubscriptionResource) ARN() string {
	if r.Item.SubscriptionArn != nil {
		return *r.Item.SubscriptionArn
	}
	return ""
}

// TopicARN returns the topic ARN
func (r *SubscriptionResource) TopicARN() string {
	if r.Item.TopicArn != nil {
		return *r.Item.TopicArn
	}
	return ""
}

// TopicName returns the topic name extracted from ARN
func (r *SubscriptionResource) TopicName() string {
	if r.Item.TopicArn != nil {
		parts := strings.Split(*r.Item.TopicArn, ":")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}
	return ""
}

// Protocol returns the subscription protocol
func (r *SubscriptionResource) Protocol() string {
	if r.Item.Protocol != nil {
		return *r.Item.Protocol
	}
	return ""
}

// Endpoint returns the subscription endpoint
func (r *SubscriptionResource) Endpoint() string {
	if r.Item.Endpoint != nil {
		return *r.Item.Endpoint
	}
	return ""
}

// Owner returns the subscription owner
func (r *SubscriptionResource) Owner() string {
	if r.Item.Owner != nil {
		return *r.Item.Owner
	}
	return ""
}

// IsPending returns whether the subscription is pending confirmation
func (r *SubscriptionResource) IsPending() bool {
	return r.ARN() == "PendingConfirmation"
}
