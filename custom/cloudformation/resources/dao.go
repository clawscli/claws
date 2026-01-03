package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// ResourceDAO provides data access for CloudFormation stack resources
type ResourceDAO struct {
	dao.BaseDAO
	client *cloudformation.Client
}

// NewResourceDAO creates a new ResourceDAO
func NewResourceDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &ResourceDAO{
		BaseDAO: dao.NewBaseDAO("cloudformation", "resources"),
		client:  cloudformation.NewFromConfig(cfg),
	}, nil
}

func (d *ResourceDAO) List(ctx context.Context) ([]dao.Resource, error) {
	// Get stack name from filter context
	stackName := dao.GetFilterFromContext(ctx, "StackName")
	if stackName == "" {
		return nil, fmt.Errorf("stack name filter required")
	}

	input := &cloudformation.DescribeStackResourcesInput{
		StackName: &stackName,
	}

	output, err := d.client.DescribeStackResources(ctx, input)
	if err != nil {
		return nil, apperrors.Wrap(err, "describe stack resources")
	}

	resources := make([]dao.Resource, 0, len(output.StackResources))
	for _, res := range output.StackResources {
		resources = append(resources, NewStackResourceResource(res))
	}

	return resources, nil
}

func (d *ResourceDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	return nil, fmt.Errorf("get by ID not supported for stack resources")
}

func (d *ResourceDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported for stack resources")
}

func (d *ResourceDAO) Supports(op dao.Operation) bool {
	switch op {
	case dao.OpList:
		return true
	default:
		return false
	}
}

// StackResourceResource wraps a CloudFormation stack resource
type StackResourceResource struct {
	dao.BaseResource
	Item types.StackResource
}

// NewStackResourceResource creates a new StackResourceResource
func NewStackResourceResource(res types.StackResource) *StackResourceResource {
	return &StackResourceResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(res.PhysicalResourceId),
			Name: appaws.Str(res.LogicalResourceId),
			Data: res,
		},
		Item: res,
	}
}

// ResourceStatus returns the resource status
func (r *StackResourceResource) ResourceStatus() string {
	return string(r.Item.ResourceStatus)
}

// ResourceType returns the resource type
func (r *StackResourceResource) ResourceType() string {
	return appaws.Str(r.Item.ResourceType)
}

// StatusReason returns the status reason
func (r *StackResourceResource) StatusReason() string {
	return appaws.Str(r.Item.ResourceStatusReason)
}

// DriftStatus returns the drift status
func (r *StackResourceResource) DriftStatus() string {
	if r.Item.DriftInformation != nil {
		return string(r.Item.DriftInformation.StackResourceDriftStatus)
	}
	return ""
}
