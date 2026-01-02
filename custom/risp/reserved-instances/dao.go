package reservedinstances

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// ReservedInstanceDAO provides data access for EC2 Reserved Instances
type ReservedInstanceDAO struct {
	dao.BaseDAO
	client *ec2.Client
}

// NewReservedInstanceDAO creates a new ReservedInstanceDAO
func NewReservedInstanceDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new risp/reserved-instances dao")
	}
	return &ReservedInstanceDAO{
		BaseDAO: dao.NewBaseDAO("risp", "reserved-instances"),
		client:  ec2.NewFromConfig(cfg),
	}, nil
}

func (d *ReservedInstanceDAO) List(ctx context.Context) ([]dao.Resource, error) {
	input := &ec2.DescribeReservedInstancesInput{}

	output, err := d.client.DescribeReservedInstances(ctx, input)
	if err != nil {
		return nil, apperrors.Wrap(err, "describe reserved instances")
	}

	var resources []dao.Resource
	for _, ri := range output.ReservedInstances {
		resources = append(resources, NewReservedInstanceResource(ri))
	}

	return resources, nil
}

func (d *ReservedInstanceDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	input := &ec2.DescribeReservedInstancesInput{
		ReservedInstancesIds: []string{id},
	}

	output, err := d.client.DescribeReservedInstances(ctx, input)
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe reserved instance %s", id)
	}

	if len(output.ReservedInstances) == 0 {
		return nil, fmt.Errorf("reserved instance not found: %s", id)
	}

	return NewReservedInstanceResource(output.ReservedInstances[0]), nil
}

func (d *ReservedInstanceDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported for reserved instances")
}

// ReservedInstanceResource wraps an EC2 Reserved Instance
type ReservedInstanceResource struct {
	dao.BaseResource
	Item types.ReservedInstances
}

// NewReservedInstanceResource creates a new ReservedInstanceResource
func NewReservedInstanceResource(ri types.ReservedInstances) *ReservedInstanceResource {
	return &ReservedInstanceResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(ri.ReservedInstancesId),
			Name: appaws.Str(ri.ReservedInstancesId),
			Tags: appaws.TagsToMap(ri.Tags),
			Data: ri,
		},
		Item: ri,
	}
}

// State returns the reservation state
func (r *ReservedInstanceResource) State() string {
	return string(r.Item.State)
}

// InstanceType returns the instance type
func (r *ReservedInstanceResource) InstanceType() string {
	return string(r.Item.InstanceType)
}

// InstanceCount returns the number of instances
func (r *ReservedInstanceResource) InstanceCount() int32 {
	if r.Item.InstanceCount != nil {
		return *r.Item.InstanceCount
	}
	return 0
}

// Scope returns the scope (Region or Availability Zone)
func (r *ReservedInstanceResource) Scope() string {
	return string(r.Item.Scope)
}

// AvailabilityZone returns the AZ (if scope is AZ)
func (r *ReservedInstanceResource) AvailabilityZone() string {
	return appaws.Str(r.Item.AvailabilityZone)
}

// OfferingClass returns the offering class (standard or convertible)
func (r *ReservedInstanceResource) OfferingClass() string {
	return string(r.Item.OfferingClass)
}

// OfferingType returns the payment option
func (r *ReservedInstanceResource) OfferingType() string {
	return string(r.Item.OfferingType)
}

// ProductDescription returns the platform
func (r *ReservedInstanceResource) ProductDescription() string {
	return string(r.Item.ProductDescription)
}

// Duration returns the term duration as a formatted string
func (r *ReservedInstanceResource) Duration() string {
	if r.Item.Duration == nil {
		return ""
	}
	seconds := *r.Item.Duration
	years := seconds / (365 * 24 * 60 * 60)
	if years >= 1 {
		return fmt.Sprintf("%dy", years)
	}
	return fmt.Sprintf("%ds", seconds)
}

// StartTime returns the start time
func (r *ReservedInstanceResource) StartTime() *time.Time {
	return r.Item.Start
}

// EndTime returns the end time
func (r *ReservedInstanceResource) EndTime() *time.Time {
	return r.Item.End
}

// FixedPrice returns the upfront cost
func (r *ReservedInstanceResource) FixedPrice() float32 {
	if r.Item.FixedPrice != nil {
		return *r.Item.FixedPrice
	}
	return 0
}

// UsagePrice returns the hourly price
func (r *ReservedInstanceResource) UsagePrice() float32 {
	if r.Item.UsagePrice != nil {
		return *r.Item.UsagePrice
	}
	return 0
}

// CurrencyCode returns the currency
func (r *ReservedInstanceResource) CurrencyCode() string {
	return string(r.Item.CurrencyCode)
}

// Tenancy returns the instance tenancy
func (r *ReservedInstanceResource) Tenancy() string {
	return string(r.Item.InstanceTenancy)
}
