package events

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// EventDAO provides data access for CloudTrail events.
type EventDAO struct {
	dao.BaseDAO
	client *cloudtrail.Client
	// Pagination state - CloudTrail requires same StartTime/EndTime for NextToken
	paginationStartTime *time.Time
	paginationEndTime   *time.Time
}

// NewEventDAO creates a new EventDAO.
func NewEventDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &EventDAO{
		BaseDAO: dao.NewBaseDAO("cloudtrail", "events"),
		client:  cloudtrail.NewFromConfig(cfg),
	}, nil
}

// List returns CloudTrail events for the last 24 hours (first page only for backwards compatibility).
// For paginated access, use ListPage instead.
func (d *EventDAO) List(ctx context.Context) ([]dao.Resource, error) {
	resources, _, err := d.ListPage(ctx, 100, "")
	return resources, err
}

// ListPage returns a page of CloudTrail events for the last 24 hours.
// Implements dao.PaginatedDAO interface.
func (d *EventDAO) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
	// CloudTrail requires same StartTime/EndTime for pagination
	// Reset time range on first page, reuse for subsequent pages
	if pageToken == "" {
		endTime := time.Now()
		startTime := endTime.Add(-24 * time.Hour)
		d.paginationStartTime = &startTime
		d.paginationEndTime = &endTime
	}

	// CloudTrail LookupEvents MaxResults is capped at 50
	maxResults := int32(pageSize)
	if maxResults > 50 {
		maxResults = 50
	}

	input := &cloudtrail.LookupEventsInput{
		StartTime:  d.paginationStartTime,
		EndTime:    d.paginationEndTime,
		MaxResults: &maxResults,
	}
	if pageToken != "" {
		input.NextToken = &pageToken
	}

	output, err := d.client.LookupEvents(ctx, input)
	if err != nil {
		return nil, "", apperrors.Wrap(err, "lookup cloudtrail events")
	}

	resources := make([]dao.Resource, len(output.Events))
	for i, event := range output.Events {
		resources[i] = NewEventResource(event)
	}

	nextToken := ""
	if output.NextToken != nil {
		nextToken = *output.NextToken
	}

	return resources, nextToken, nil
}

// Get returns a specific event by ID.
func (d *EventDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	// CloudTrail doesn't have a GetEvent API, so we lookup by event ID
	endTime := time.Now()
	startTime := endTime.Add(-90 * 24 * time.Hour) // Look back 90 days

	output, err := d.client.LookupEvents(ctx, &cloudtrail.LookupEventsInput{
		StartTime: &startTime,
		EndTime:   &endTime,
		LookupAttributes: []types.LookupAttribute{
			{
				AttributeKey:   types.LookupAttributeKeyEventId,
				AttributeValue: &id,
			},
		},
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "lookup cloudtrail event %s", id)
	}
	if len(output.Events) == 0 {
		return nil, fmt.Errorf("event not found: %s", id)
	}
	return NewEventResource(output.Events[0]), nil
}

// Delete is not supported for events.
func (d *EventDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported for cloudtrail events")
}

// EventResource wraps a CloudTrail event.
type EventResource struct {
	dao.BaseResource
	Item types.Event
}

// NewEventResource creates a new EventResource.
func NewEventResource(event types.Event) *EventResource {
	return &EventResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(event.EventId),
			ARN:  appaws.Str(event.EventId),
			Data: event,
		},
		Item: event,
	}
}

// EventId returns the event ID.
func (r *EventResource) EventId() string {
	return appaws.Str(r.Item.EventId)
}

// EventName returns the event name.
func (r *EventResource) EventName() string {
	return appaws.Str(r.Item.EventName)
}

// EventSource returns the event source.
func (r *EventResource) EventSource() string {
	return appaws.Str(r.Item.EventSource)
}

// EventTime returns when the event occurred.
func (r *EventResource) EventTime() *time.Time {
	return r.Item.EventTime
}

// Username returns the username who initiated the event.
func (r *EventResource) Username() string {
	return appaws.Str(r.Item.Username)
}

// AccessKeyId returns the access key ID used.
func (r *EventResource) AccessKeyId() string {
	return appaws.Str(r.Item.AccessKeyId)
}

// ReadOnly returns whether this is a read-only event.
func (r *EventResource) ReadOnly() string {
	return appaws.Str(r.Item.ReadOnly)
}

// CloudTrailEvent returns the raw CloudTrail event JSON.
func (r *EventResource) CloudTrailEvent() string {
	return appaws.Str(r.Item.CloudTrailEvent)
}

// Resources returns the resources affected by this event.
func (r *EventResource) Resources() []types.Resource {
	return r.Item.Resources
}
