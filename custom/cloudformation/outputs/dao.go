package outputs

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// OutputDAO provides data access for CloudFormation stack outputs
type OutputDAO struct {
	dao.BaseDAO
	client *cloudformation.Client
}

// NewOutputDAO creates a new OutputDAO
func NewOutputDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &OutputDAO{
		BaseDAO: dao.NewBaseDAO("cloudformation", "outputs"),
		client:  cloudformation.NewFromConfig(cfg),
	}, nil
}

func (d *OutputDAO) List(ctx context.Context) ([]dao.Resource, error) {
	// Get stack name from filter context
	stackName := dao.GetFilterFromContext(ctx, "StackName")
	if stackName == "" {
		return nil, fmt.Errorf("stack name filter required")
	}

	input := &cloudformation.DescribeStacksInput{
		StackName: &stackName,
	}

	output, err := d.client.DescribeStacks(ctx, input)
	if err != nil {
		return nil, apperrors.Wrap(err, "describe stacks")
	}

	if len(output.Stacks) == 0 {
		return nil, fmt.Errorf("stack not found: %s", stackName)
	}

	stack := output.Stacks[0]
	resources := make([]dao.Resource, 0, len(stack.Outputs))
	for _, out := range stack.Outputs {
		resources = append(resources, NewOutputResource(out, stackName))
	}

	return resources, nil
}

func (d *OutputDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	return nil, fmt.Errorf("get by ID not supported for stack outputs")
}

func (d *OutputDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported for stack outputs")
}

func (d *OutputDAO) Supports(op dao.Operation) bool {
	switch op {
	case dao.OpList:
		return true
	default:
		return false
	}
}

// OutputResource wraps a CloudFormation stack output
type OutputResource struct {
	dao.BaseResource
	Item      types.Output
	StackName string
}

// NewOutputResource creates a new OutputResource
func NewOutputResource(out types.Output, stackName string) *OutputResource {
	key := appaws.Str(out.OutputKey)
	return &OutputResource{
		BaseResource: dao.BaseResource{
			ID:   key,
			Name: key,
			Data: outputWithStackName{Output: out, StackName: stackName},
		},
		Item:      out,
		StackName: stackName,
	}
}

// outputWithStackName wraps Output with StackName for field filtering
type outputWithStackName struct {
	types.Output
	StackName string
}

// OutputKey returns the output key
func (r *OutputResource) OutputKey() string {
	return appaws.Str(r.Item.OutputKey)
}

// OutputValue returns the output value
func (r *OutputResource) OutputValue() string {
	return appaws.Str(r.Item.OutputValue)
}

// Description returns the output description
func (r *OutputResource) Description() string {
	return appaws.Str(r.Item.Description)
}

// ExportName returns the export name if exported
func (r *OutputResource) ExportName() string {
	return appaws.Str(r.Item.ExportName)
}
