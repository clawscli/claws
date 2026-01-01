package statemachines

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// StateMachineDAO provides data access for Step Functions state machines
type StateMachineDAO struct {
	dao.BaseDAO
	client *sfn.Client
}

// NewStateMachineDAO creates a new StateMachineDAO
func NewStateMachineDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new stepfunctions/state-machines dao")
	}
	return &StateMachineDAO{
		BaseDAO: dao.NewBaseDAO("stepfunctions", "state-machines"),
		client:  sfn.NewFromConfig(cfg),
	}, nil
}

func (d *StateMachineDAO) List(ctx context.Context) ([]dao.Resource, error) {
	input := &sfn.ListStateMachinesInput{}

	var resources []dao.Resource
	paginator := sfn.NewListStateMachinesPaginator(d.client, input)

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, apperrors.Wrap(err, "list state machines")
		}

		for _, sm := range output.StateMachines {
			resources = append(resources, NewStateMachineResource(sm, nil))
		}
	}

	return resources, nil
}

func (d *StateMachineDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	input := &sfn.DescribeStateMachineInput{
		StateMachineArn: &id,
	}

	output, err := d.client.DescribeStateMachine(ctx, input)
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe state machine %s", id)
	}

	// Convert to list item format
	listItem := types.StateMachineListItem{
		StateMachineArn: output.StateMachineArn,
		Name:            output.Name,
		Type:            output.Type,
		CreationDate:    output.CreationDate,
	}

	return NewStateMachineResource(listItem, output), nil
}

func (d *StateMachineDAO) Delete(ctx context.Context, id string) error {
	input := &sfn.DeleteStateMachineInput{
		StateMachineArn: &id,
	}

	_, err := d.client.DeleteStateMachine(ctx, input)
	if err != nil {
		return apperrors.Wrapf(err, "delete state machine %s", id)
	}

	return nil
}

// StateMachineResource wraps a Step Functions state machine
type StateMachineResource struct {
	dao.BaseResource
	Item   types.StateMachineListItem
	Detail *sfn.DescribeStateMachineOutput
}

// NewStateMachineResource creates a new StateMachineResource
func NewStateMachineResource(sm types.StateMachineListItem, detail *sfn.DescribeStateMachineOutput) *StateMachineResource {
	arn := appaws.Str(sm.StateMachineArn)
	name := appaws.Str(sm.Name)

	return &StateMachineResource{
		BaseResource: dao.BaseResource{
			ID:   arn,
			Name: name,
			ARN:  arn,
			Tags: nil,
			Data: sm,
		},
		Item:   sm,
		Detail: detail,
	}
}

// ARN returns the state machine ARN
func (r *StateMachineResource) ARN() string {
	if r.Item.StateMachineArn != nil {
		return *r.Item.StateMachineArn
	}
	return ""
}

// Type returns the state machine type (STANDARD or EXPRESS)
func (r *StateMachineResource) Type() string {
	return string(r.Item.Type)
}

// Status returns the state machine status
func (r *StateMachineResource) Status() string {
	if r.Detail != nil {
		return string(r.Detail.Status)
	}
	return "ACTIVE"
}

// RoleARN returns the IAM role ARN
func (r *StateMachineResource) RoleARN() string {
	if r.Detail != nil && r.Detail.RoleArn != nil {
		return *r.Detail.RoleArn
	}
	return ""
}

// RoleName extracts the role name from ARN
func (r *StateMachineResource) RoleName() string {
	arn := r.RoleARN()
	if arn == "" {
		return ""
	}
	parts := strings.Split(arn, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// Definition returns the state machine definition
func (r *StateMachineResource) Definition() string {
	if r.Detail != nil && r.Detail.Definition != nil {
		return *r.Detail.Definition
	}
	return ""
}
