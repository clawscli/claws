package events

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/health"
	"github.com/aws/aws-sdk-go-v2/service/health/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// EventDAO provides data access for AWS Health events.
type EventDAO struct {
	dao.BaseDAO
	client *health.Client
}

// NewEventDAO creates a new EventDAO.
func NewEventDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	// Health API requires us-east-1 region
	return &EventDAO{
		BaseDAO: dao.NewBaseDAO("health", "events"),
		client:  health.NewFromConfig(cfg, func(o *health.Options) { o.Region = "us-east-1" }),
	}, nil
}

// List returns Health events (first page only for backwards compatibility).
// For paginated access, use ListPage instead.
func (d *EventDAO) List(ctx context.Context) ([]dao.Resource, error) {
	resources, _, err := d.ListPage(ctx, 100, "")
	return resources, err
}

// ListPage returns a page of AWS Health events.
// Implements dao.PaginatedDAO interface.
func (d *EventDAO) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
	maxResults := int32(pageSize)
	if maxResults > 100 {
		maxResults = 100 // AWS API max
	}

	input := &health.DescribeEventsInput{
		MaxResults: &maxResults,
	}
	if pageToken != "" {
		input.NextToken = &pageToken
	}

	output, err := d.client.DescribeEvents(ctx, input)
	if err != nil {
		return nil, "", apperrors.Wrap(err, "describe health events")
	}

	// Sort by StartTime descending (newest first)
	events := output.Events
	slices.SortFunc(events, func(a, b types.Event) int {
		aTime := a.StartTime
		bTime := b.StartTime
		if aTime == nil && bTime == nil {
			return 0
		}
		if aTime == nil {
			return 1 // nil times go to end
		}
		if bTime == nil {
			return -1
		}
		return bTime.Compare(*aTime) // descending order
	})

	resources := make([]dao.Resource, len(events))
	for i, event := range events {
		resources[i] = NewEventResource(event)
	}

	nextToken := ""
	if output.NextToken != nil {
		nextToken = *output.NextToken
	}

	return resources, nextToken, nil
}

// Get returns a specific Health event by ARN.
func (d *EventDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeEvents(ctx, &health.DescribeEventsInput{
		Filter: &types.EventFilter{
			EventArns: []string{id},
		},
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe health event %s", id)
	}
	if len(output.Events) == 0 {
		return nil, fmt.Errorf("health event not found: %s", id)
	}
	return NewEventResource(output.Events[0]), nil
}

// Delete is not supported for Health events.
func (d *EventDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported for health events")
}

// EventResource wraps an AWS Health event.
type EventResource struct {
	dao.BaseResource
	Item types.Event
}

// NewEventResource creates a new EventResource.
func NewEventResource(event types.Event) *EventResource {
	return &EventResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(event.Arn),
			ARN: appaws.Str(event.Arn),
		},
		Item: event,
	}
}

// Service returns the affected service.
func (r *EventResource) Service() string {
	return appaws.Str(r.Item.Service)
}

// EventTypeCode returns the event type code.
func (r *EventResource) EventTypeCode() string {
	return appaws.Str(r.Item.EventTypeCode)
}

// EventTypeCategory returns the event type category.
func (r *EventResource) EventTypeCategory() string {
	return string(r.Item.EventTypeCategory)
}

// Region returns the affected region.
func (r *EventResource) Region() string {
	return appaws.Str(r.Item.Region)
}

// AvailabilityZone returns the affected AZ.
func (r *EventResource) AvailabilityZone() string {
	return appaws.Str(r.Item.AvailabilityZone)
}

// StartTime returns when the event started.
func (r *EventResource) StartTime() *time.Time {
	return r.Item.StartTime
}

// EndTime returns when the event ended.
func (r *EventResource) EndTime() *time.Time {
	return r.Item.EndTime
}

// LastUpdatedTime returns when the event was last updated.
func (r *EventResource) LastUpdatedTime() *time.Time {
	return r.Item.LastUpdatedTime
}

// StatusCode returns the event status.
func (r *EventResource) StatusCode() string {
	return string(r.Item.StatusCode)
}

// EventScopeCode returns the event scope.
func (r *EventResource) EventScopeCode() string {
	return string(r.Item.EventScopeCode)
}
