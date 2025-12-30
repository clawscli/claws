package events

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// EventDAO provides data access for CloudFormation stack events
type EventDAO struct {
	dao.BaseDAO
	client *cloudformation.Client
}

// NewEventDAO creates a new EventDAO
func NewEventDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new cfn/events dao: %w", err)
	}
	return &EventDAO{
		BaseDAO: dao.NewBaseDAO("cloudformation", "events"),
		client:  cloudformation.NewFromConfig(cfg),
	}, nil
}

// List returns stack events (first page only for backwards compatibility).
// For paginated access, use ListPage instead.
func (d *EventDAO) List(ctx context.Context) ([]dao.Resource, error) {
	resources, _, err := d.ListPage(ctx, 0, "")
	return resources, err
}

// ListPage returns a page of CloudFormation stack events.
// Implements dao.PaginatedDAO interface.
// Note: DescribeStackEvents API does not support MaxResults, so pageSize is ignored.
// The API controls page size internally.
func (d *EventDAO) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
	// Get stack name from filter context
	stackName := dao.GetFilterFromContext(ctx, "StackName")
	if stackName == "" {
		return nil, "", fmt.Errorf("stack name filter required")
	}

	// Events are returned in reverse chronological order by default
	// Note: This API does not support MaxResults parameter
	input := &cloudformation.DescribeStackEventsInput{
		StackName: &stackName,
	}
	if pageToken != "" {
		input.NextToken = &pageToken
	}

	output, err := d.client.DescribeStackEvents(ctx, input)
	if err != nil {
		return nil, "", fmt.Errorf("describe stack events: %w", err)
	}

	resources := make([]dao.Resource, 0, len(output.StackEvents))
	for _, event := range output.StackEvents {
		resources = append(resources, NewEventResource(event))
	}

	nextToken := ""
	if output.NextToken != nil {
		nextToken = *output.NextToken
	}

	return resources, nextToken, nil
}

func (d *EventDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	// Events don't have a direct get by ID, return not supported
	return nil, fmt.Errorf("get by ID not supported for stack events")
}

func (d *EventDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported for stack events")
}

func (d *EventDAO) Supports(op dao.Operation) bool {
	switch op {
	case dao.OpList:
		return true
	default:
		return false
	}
}

// EventResource wraps a CloudFormation stack event
type EventResource struct {
	dao.BaseResource
	Item types.StackEvent
}

// NewEventResource creates a new EventResource
func NewEventResource(event types.StackEvent) *EventResource {
	return &EventResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(event.EventId),
			Name: appaws.Str(event.LogicalResourceId),
			Data: event,
		},
		Item: event,
	}
}

// ResourceStatus returns the resource status
func (r *EventResource) ResourceStatus() string {
	return string(r.Item.ResourceStatus)
}

// ResourceType returns the resource type
func (r *EventResource) ResourceType() string {
	return appaws.Str(r.Item.ResourceType)
}

// StatusReason returns the status reason
func (r *EventResource) StatusReason() string {
	return appaws.Str(r.Item.ResourceStatusReason)
}

// PhysicalResourceId returns the physical resource ID
func (r *EventResource) PhysicalResourceId() string {
	return appaws.Str(r.Item.PhysicalResourceId)
}
