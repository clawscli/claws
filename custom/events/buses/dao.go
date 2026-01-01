package buses

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// BusDAO provides data access for EventBridge event buses
type BusDAO struct {
	dao.BaseDAO
	client *eventbridge.Client
}

// NewBusDAO creates a new BusDAO
func NewBusDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new events/buses dao")
	}
	return &BusDAO{
		BaseDAO: dao.NewBaseDAO("events", "buses"),
		client:  eventbridge.NewFromConfig(cfg),
	}, nil
}

func (d *BusDAO) List(ctx context.Context) ([]dao.Resource, error) {
	input := &eventbridge.ListEventBusesInput{}

	var resources []dao.Resource
	// EventBridge doesn't have a paginator for ListEventBuses
	output, err := d.client.ListEventBuses(ctx, input)
	if err != nil {
		return nil, apperrors.Wrap(err, "list event buses")
	}

	for _, bus := range output.EventBuses {
		resources = append(resources, NewBusResource(bus))
	}

	return resources, nil
}

func (d *BusDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	input := &eventbridge.DescribeEventBusInput{
		Name: &id,
	}

	output, err := d.client.DescribeEventBus(ctx, input)
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe event bus %s", id)
	}

	// Convert DescribeEventBusOutput to types.EventBus
	bus := types.EventBus{
		Name: output.Name,
		Arn:  output.Arn,
	}

	return NewBusResource(bus), nil
}

func (d *BusDAO) Delete(ctx context.Context, id string) error {
	input := &eventbridge.DeleteEventBusInput{
		Name: &id,
	}

	_, err := d.client.DeleteEventBus(ctx, input)
	if err != nil {
		return apperrors.Wrapf(err, "delete event bus %s", id)
	}

	return nil
}

// BusResource wraps an EventBridge event bus
type BusResource struct {
	dao.BaseResource
	Item types.EventBus
}

// NewBusResource creates a new BusResource
func NewBusResource(bus types.EventBus) *BusResource {
	name := appaws.Str(bus.Name)

	return &BusResource{
		BaseResource: dao.BaseResource{
			ID:   name,
			Name: name,
			Tags: nil,
			Data: bus,
		},
		Item: bus,
	}
}

// ARN returns the event bus ARN
func (r *BusResource) ARN() string {
	if r.Item.Arn != nil {
		return *r.Item.Arn
	}
	return ""
}

// IsDefault returns whether this is the default event bus
func (r *BusResource) IsDefault() bool {
	return r.GetName() == "default"
}
