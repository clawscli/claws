package stacks

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// StackDAO provides data access for CloudFormation stacks
type StackDAO struct {
	dao.BaseDAO
	client *cloudformation.Client
}

// NewStackDAO creates a new StackDAO
func NewStackDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new cfn/stacks dao: %w", err)
	}
	return &StackDAO{
		BaseDAO: dao.NewBaseDAO("cloudformation", "stacks"),
		client:  cloudformation.NewFromConfig(cfg),
	}, nil
}

func (d *StackDAO) List(ctx context.Context) ([]dao.Resource, error) {
	input := &cloudformation.DescribeStacksInput{}
	paginator := cloudformation.NewDescribeStacksPaginator(d.client, input)

	var resources []dao.Resource
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describe stacks: %w", err)
		}

		for _, stack := range output.Stacks {
			resources = append(resources, NewStackResource(stack))
		}
	}

	return resources, nil
}

func (d *StackDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	input := &cloudformation.DescribeStacksInput{
		StackName: &id,
	}

	output, err := d.client.DescribeStacks(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("describe stack %s: %w", id, err)
	}

	if len(output.Stacks) == 0 {
		return nil, fmt.Errorf("stack not found: %s", id)
	}

	return NewStackResource(output.Stacks[0]), nil
}

func (d *StackDAO) Delete(ctx context.Context, id string) error {
	input := &cloudformation.DeleteStackInput{
		StackName: &id,
	}

	_, err := d.client.DeleteStack(ctx, input)
	if err != nil {
		if appaws.IsNotFound(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("delete stack %s: %w", id, err)
	}

	return nil
}

// StackResource wraps a CloudFormation stack
type StackResource struct {
	dao.BaseResource
	Item types.Stack
}

// NewStackResource creates a new StackResource
func NewStackResource(stack types.Stack) *StackResource {
	return &StackResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(stack.StackId),
			Name: appaws.Str(stack.StackName),
			ARN:  appaws.Str(stack.StackId), // StackId is the ARN
			Tags: appaws.TagsToMap(stack.Tags),
			Data: stack,
		},
		Item: stack,
	}
}

// Status returns the stack status
func (r *StackResource) Status() string {
	return string(r.Item.StackStatus)
}

// DriftStatus returns the drift status
func (r *StackResource) DriftStatus() string {
	if r.Item.DriftInformation != nil {
		return string(r.Item.DriftInformation.StackDriftStatus)
	}
	return ""
}

// Description returns the stack description
func (r *StackResource) Description() string {
	return appaws.Str(r.Item.Description)
}

// TerminationProtection returns whether termination protection is enabled
func (r *StackResource) TerminationProtection() bool {
	return appaws.Bool(r.Item.EnableTerminationProtection)
}
