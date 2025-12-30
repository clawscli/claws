package capacityreservations

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

// CapacityReservationDAO provides data access for EC2 Capacity Reservations
type CapacityReservationDAO struct {
	dao.BaseDAO
	client *ec2.Client
}

// NewCapacityReservationDAO creates a new CapacityReservationDAO
func NewCapacityReservationDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new ec2/capacityreservations dao")
	}
	return &CapacityReservationDAO{
		BaseDAO: dao.NewBaseDAO("ec2", "capacity-reservations"),
		client:  ec2.NewFromConfig(cfg),
	}, nil
}

func (d *CapacityReservationDAO) List(ctx context.Context) ([]dao.Resource, error) {
	input := &ec2.DescribeCapacityReservationsInput{}
	paginator := ec2.NewDescribeCapacityReservationsPaginator(d.client, input)

	var resources []dao.Resource
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, apperrors.Wrap(err, "describe capacity reservations")
		}

		for _, cr := range output.CapacityReservations {
			resources = append(resources, NewCapacityReservationResource(cr))
		}
	}

	return resources, nil
}

func (d *CapacityReservationDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	input := &ec2.DescribeCapacityReservationsInput{
		CapacityReservationIds: []string{id},
	}

	output, err := d.client.DescribeCapacityReservations(ctx, input)
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe capacity reservation %s", id)
	}

	if len(output.CapacityReservations) == 0 {
		return nil, fmt.Errorf("capacity reservation not found: %s", id)
	}

	return NewCapacityReservationResource(output.CapacityReservations[0]), nil
}

func (d *CapacityReservationDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported for capacity reservations (use cancel)")
}

// CapacityReservationResource wraps an EC2 Capacity Reservation
type CapacityReservationResource struct {
	dao.BaseResource
	Item types.CapacityReservation
}

// NewCapacityReservationResource creates a new CapacityReservationResource
func NewCapacityReservationResource(cr types.CapacityReservation) *CapacityReservationResource {
	id := appaws.Str(cr.CapacityReservationId)
	return &CapacityReservationResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: id,
			Tags: appaws.TagsToMap(cr.Tags),
			Data: cr,
		},
		Item: cr,
	}
}

// State returns the reservation state
func (r *CapacityReservationResource) State() string {
	return string(r.Item.State)
}

// InstanceType returns the instance type
func (r *CapacityReservationResource) InstanceType() string {
	return appaws.Str(r.Item.InstanceType)
}

// InstancePlatform returns the platform
func (r *CapacityReservationResource) InstancePlatform() string {
	return string(r.Item.InstancePlatform)
}

// AvailabilityZone returns the AZ
func (r *CapacityReservationResource) AvailabilityZone() string {
	return appaws.Str(r.Item.AvailabilityZone)
}

// TotalInstanceCount returns the total reserved count
func (r *CapacityReservationResource) TotalInstanceCount() int32 {
	if r.Item.TotalInstanceCount != nil {
		return *r.Item.TotalInstanceCount
	}
	return 0
}

// AvailableInstanceCount returns the available count
func (r *CapacityReservationResource) AvailableInstanceCount() int32 {
	if r.Item.AvailableInstanceCount != nil {
		return *r.Item.AvailableInstanceCount
	}
	return 0
}

// UsedInstanceCount returns the used count
func (r *CapacityReservationResource) UsedInstanceCount() int32 {
	return r.TotalInstanceCount() - r.AvailableInstanceCount()
}

// Tenancy returns the tenancy
func (r *CapacityReservationResource) Tenancy() string {
	return string(r.Item.Tenancy)
}

// EndDateType returns the end date type
func (r *CapacityReservationResource) EndDateType() string {
	return string(r.Item.EndDateType)
}

// InstanceMatchCriteria returns the match criteria
func (r *CapacityReservationResource) InstanceMatchCriteria() string {
	return string(r.Item.InstanceMatchCriteria)
}

// EbsOptimized returns whether EBS optimized
func (r *CapacityReservationResource) EbsOptimized() bool {
	if r.Item.EbsOptimized != nil {
		return *r.Item.EbsOptimized
	}
	return false
}

// EphemeralStorage returns whether ephemeral storage is enabled
func (r *CapacityReservationResource) EphemeralStorage() bool {
	if r.Item.EphemeralStorage != nil {
		return *r.Item.EphemeralStorage
	}
	return false
}

// StartDate returns the start date
func (r *CapacityReservationResource) StartDate() *time.Time {
	return r.Item.StartDate
}

// EndDate returns the end date
func (r *CapacityReservationResource) EndDate() *time.Time {
	return r.Item.EndDate
}

// CreateDate returns the creation date
func (r *CapacityReservationResource) CreateDate() *time.Time {
	return r.Item.CreateDate
}

// ARN returns the ARN
func (r *CapacityReservationResource) ARN() string {
	return appaws.Str(r.Item.CapacityReservationArn)
}

// OwnerID returns the owner ID
func (r *CapacityReservationResource) OwnerID() string {
	return appaws.Str(r.Item.OwnerId)
}
