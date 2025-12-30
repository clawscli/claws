package instances

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// InstanceDAO provides data access for RDS instances
type InstanceDAO struct {
	dao.BaseDAO
	client *rds.Client
}

// NewInstanceDAO creates a new InstanceDAO
func NewInstanceDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new rds/instances dao: %w", err)
	}
	return &InstanceDAO{
		BaseDAO: dao.NewBaseDAO("rds", "instances"),
		client:  rds.NewFromConfig(cfg),
	}, nil
}

func (d *InstanceDAO) List(ctx context.Context) ([]dao.Resource, error) {
	input := &rds.DescribeDBInstancesInput{}
	paginator := rds.NewDescribeDBInstancesPaginator(d.client, input)

	var resources []dao.Resource
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describe db instances: %w", err)
		}

		for _, instance := range output.DBInstances {
			resources = append(resources, NewInstanceResource(instance))
		}
	}

	return resources, nil
}

func (d *InstanceDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	input := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: &id,
	}

	output, err := d.client.DescribeDBInstances(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("describe db instance %s: %w", id, err)
	}

	if len(output.DBInstances) == 0 {
		return nil, fmt.Errorf("db instance not found: %s", id)
	}

	return NewInstanceResource(output.DBInstances[0]), nil
}

func (d *InstanceDAO) Delete(ctx context.Context, id string) error {
	skipFinalSnapshot := true
	input := &rds.DeleteDBInstanceInput{
		DBInstanceIdentifier:   &id,
		SkipFinalSnapshot:      &skipFinalSnapshot,
		DeleteAutomatedBackups: appaws.BoolPtr(true),
	}

	_, err := d.client.DeleteDBInstance(ctx, input)
	if err != nil {
		if appaws.IsNotFound(err) {
			return nil // Already deleted
		}
		if appaws.IsResourceInUse(err) {
			return fmt.Errorf("db instance %s is in use", id)
		}
		return fmt.Errorf("delete db instance %s: %w", id, err)
	}

	return nil
}

// InstanceResource wraps an RDS instance
type InstanceResource struct {
	dao.BaseResource
	Item types.DBInstance
}

// NewInstanceResource creates a new InstanceResource
func NewInstanceResource(instance types.DBInstance) *InstanceResource {
	return &InstanceResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(instance.DBInstanceIdentifier),
			Name: appaws.Str(instance.DBInstanceIdentifier),
			ARN:  appaws.Str(instance.DBInstanceArn),
			Tags: appaws.TagsToMap(instance.TagList),
			Data: instance,
		},
		Item: instance,
	}
}

// State returns the instance status
func (r *InstanceResource) State() string {
	if r.Item.DBInstanceStatus != nil {
		return *r.Item.DBInstanceStatus
	}
	return "unknown"
}

// Engine returns the database engine
func (r *InstanceResource) Engine() string {
	if r.Item.Engine != nil {
		return *r.Item.Engine
	}
	return ""
}

// EngineVersion returns the engine version
func (r *InstanceResource) EngineVersion() string {
	if r.Item.EngineVersion != nil {
		return *r.Item.EngineVersion
	}
	return ""
}

// InstanceClass returns the DB instance class
func (r *InstanceResource) InstanceClass() string {
	if r.Item.DBInstanceClass != nil {
		return *r.Item.DBInstanceClass
	}
	return ""
}

// Endpoint returns the endpoint address
func (r *InstanceResource) Endpoint() string {
	if r.Item.Endpoint != nil && r.Item.Endpoint.Address != nil {
		return *r.Item.Endpoint.Address
	}
	return ""
}

// Port returns the endpoint port
func (r *InstanceResource) Port() int32 {
	if r.Item.Endpoint != nil && r.Item.Endpoint.Port != nil {
		return *r.Item.Endpoint.Port
	}
	return 0
}

// AZ returns the availability zone
func (r *InstanceResource) AZ() string {
	if r.Item.AvailabilityZone != nil {
		return *r.Item.AvailabilityZone
	}
	return ""
}

// MultiAZ returns whether multi-AZ is enabled
func (r *InstanceResource) MultiAZ() bool {
	return r.Item.MultiAZ != nil && *r.Item.MultiAZ
}

// StorageType returns the storage type
func (r *InstanceResource) StorageType() string {
	if r.Item.StorageType != nil {
		return *r.Item.StorageType
	}
	return ""
}

// AllocatedStorage returns the allocated storage in GB
func (r *InstanceResource) AllocatedStorage() int32 {
	if r.Item.AllocatedStorage != nil {
		return *r.Item.AllocatedStorage
	}
	return 0
}
