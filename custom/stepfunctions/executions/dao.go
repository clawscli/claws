package executions

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// ExecutionDAO provides data access for Step Functions executions
type ExecutionDAO struct {
	dao.BaseDAO
	client *sfn.Client
}

// NewExecutionDAO creates a new ExecutionDAO
func NewExecutionDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new stepfunctions/executions dao")
	}
	return &ExecutionDAO{
		BaseDAO: dao.NewBaseDAO("stepfunctions", "executions"),
		client:  sfn.NewFromConfig(cfg),
	}, nil
}

// List returns executions (first page only for backwards compatibility).
// For paginated access, use ListPage instead.
func (d *ExecutionDAO) List(ctx context.Context) ([]dao.Resource, error) {
	resources, _, err := d.ListPage(ctx, 100, "")
	return resources, err
}

// ListPage returns a page of Step Functions executions.
// Implements dao.PaginatedDAO interface.
func (d *ExecutionDAO) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
	smName := dao.GetFilterFromContext(ctx, "StateMachineName")
	if smName == "" {
		return nil, "", fmt.Errorf("StateMachineName filter required - navigate from a state machine")
	}

	// First find the state machine ARN by name
	smInput := &sfn.ListStateMachinesInput{}
	smPaginator := sfn.NewListStateMachinesPaginator(d.client, smInput)

	var smArn string
	for smPaginator.HasMorePages() {
		smOutput, err := smPaginator.NextPage(ctx)
		if err != nil {
			return nil, "", apperrors.Wrap(err, "list state machines")
		}
		for _, sm := range smOutput.StateMachines {
			if appaws.Str(sm.Name) == smName {
				smArn = appaws.Str(sm.StateMachineArn)
				break
			}
		}
		if smArn != "" {
			break
		}
	}

	if smArn == "" {
		return []dao.Resource{}, "", nil // State machine not found
	}

	maxResults := int32(pageSize)
	if maxResults > 100 {
		maxResults = 100 // AWS API max
	}

	input := &sfn.ListExecutionsInput{
		StateMachineArn: &smArn,
		MaxResults:      maxResults,
	}
	if pageToken != "" {
		input.NextToken = &pageToken
	}

	output, err := d.client.ListExecutions(ctx, input)
	if err != nil {
		return nil, "", apperrors.Wrapf(err, "list executions for %s", smName)
	}

	resources := make([]dao.Resource, 0, len(output.Executions))
	for _, exec := range output.Executions {
		resources = append(resources, NewExecutionResource(exec, nil))
	}

	nextToken := ""
	if output.NextToken != nil {
		nextToken = *output.NextToken
	}

	return resources, nextToken, nil
}

func (d *ExecutionDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	input := &sfn.DescribeExecutionInput{
		ExecutionArn: &id,
	}

	output, err := d.client.DescribeExecution(ctx, input)
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe execution %s", id)
	}

	// Convert to list item format
	listItem := types.ExecutionListItem{
		ExecutionArn:    output.ExecutionArn,
		StateMachineArn: output.StateMachineArn,
		Name:            output.Name,
		Status:          output.Status,
		StartDate:       output.StartDate,
		StopDate:        output.StopDate,
	}

	return NewExecutionResource(listItem, output), nil
}

func (d *ExecutionDAO) Delete(ctx context.Context, id string) error {
	// Stop the execution (can't actually delete, but can stop running ones)
	input := &sfn.StopExecutionInput{
		ExecutionArn: &id,
	}

	_, err := d.client.StopExecution(ctx, input)
	if err != nil {
		return apperrors.Wrapf(err, "stop execution %s", id)
	}

	return nil
}

// ExecutionResource wraps a Step Functions execution
type ExecutionResource struct {
	dao.BaseResource
	Item   types.ExecutionListItem
	Detail *sfn.DescribeExecutionOutput
}

// NewExecutionResource creates a new ExecutionResource
func NewExecutionResource(exec types.ExecutionListItem, detail *sfn.DescribeExecutionOutput) *ExecutionResource {
	arn := appaws.Str(exec.ExecutionArn)
	name := appaws.Str(exec.Name)

	return &ExecutionResource{
		BaseResource: dao.BaseResource{
			ID:   arn,
			Name: name,
			Tags: nil,
			Data: exec,
		},
		Item:   exec,
		Detail: detail,
	}
}

// ARN returns the execution ARN
func (r *ExecutionResource) ARN() string {
	if r.Item.ExecutionArn != nil {
		return *r.Item.ExecutionArn
	}
	return ""
}

// StateMachineARN returns the state machine ARN
func (r *ExecutionResource) StateMachineARN() string {
	if r.Item.StateMachineArn != nil {
		return *r.Item.StateMachineArn
	}
	return ""
}

// StateMachineName extracts the state machine name from ARN
func (r *ExecutionResource) StateMachineName() string {
	arn := r.StateMachineARN()
	if arn == "" {
		return ""
	}
	parts := strings.Split(arn, ":")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// Status returns the execution status
func (r *ExecutionResource) Status() string {
	return string(r.Item.Status)
}

// Input returns the execution input
func (r *ExecutionResource) Input() string {
	if r.Detail != nil && r.Detail.Input != nil {
		return *r.Detail.Input
	}
	return ""
}

// Output returns the execution output
func (r *ExecutionResource) Output() string {
	if r.Detail != nil && r.Detail.Output != nil {
		return *r.Detail.Output
	}
	return ""
}

// Error returns the execution error
func (r *ExecutionResource) Error() string {
	if r.Detail != nil && r.Detail.Error != nil {
		return *r.Detail.Error
	}
	return ""
}

// Cause returns the execution error cause
func (r *ExecutionResource) Cause() string {
	if r.Detail != nil && r.Detail.Cause != nil {
		return *r.Detail.Cause
	}
	return ""
}
