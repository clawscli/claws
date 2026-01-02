package flows

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/bedrockagent"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagent/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// FlowDAO provides data access for Bedrock Flows
type FlowDAO struct {
	dao.BaseDAO
	client *bedrockagent.Client
}

// NewFlowDAO creates a new FlowDAO
func NewFlowDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new bedrock-agent/flows dao")
	}
	return &FlowDAO{
		BaseDAO: dao.NewBaseDAO("bedrock-agent", "flows"),
		client:  bedrockagent.NewFromConfig(cfg),
	}, nil
}

func (d *FlowDAO) List(ctx context.Context) ([]dao.Resource, error) {
	flows, err := appaws.Paginate(ctx, func(token *string) ([]types.FlowSummary, *string, error) {
		output, err := d.client.ListFlows(ctx, &bedrockagent.ListFlowsInput{
			NextToken:  token,
			MaxResults: appaws.Int32Ptr(100),
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list flows")
		}
		return output.FlowSummaries, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(flows))
	for i, flow := range flows {
		resources[i] = NewFlowResource(flow)
	}

	return resources, nil
}

func (d *FlowDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.GetFlow(ctx, &bedrockagent.GetFlowInput{
		FlowIdentifier: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get flow %s", id)
	}

	return NewFlowResourceFromDetail(output), nil
}

func (d *FlowDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteFlow(ctx, &bedrockagent.DeleteFlowInput{
		FlowIdentifier: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete flow %s", id)
	}
	return nil
}

// FlowResource wraps a Bedrock Flow
type FlowResource struct {
	dao.BaseResource
	Item       types.FlowSummary
	DetailItem *bedrockagent.GetFlowOutput
	IsFromList bool
}

// NewFlowResource creates a new FlowResource from list output
func NewFlowResource(flow types.FlowSummary) *FlowResource {
	id := appaws.Str(flow.Id)
	name := appaws.Str(flow.Name)
	arn := appaws.Str(flow.Arn)

	return &FlowResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: name,
			ARN:  arn,
			Data: flow,
		},
		Item:       flow,
		IsFromList: true,
	}
}

// NewFlowResourceFromDetail creates a FlowResource from detail output
func NewFlowResourceFromDetail(output *bedrockagent.GetFlowOutput) *FlowResource {
	id := appaws.Str(output.Id)
	name := appaws.Str(output.Name)
	arn := appaws.Str(output.Arn)

	return &FlowResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: name,
			ARN:  arn,
			Data: output,
		},
		DetailItem: output,
		IsFromList: false,
	}
}

// Status returns the flow status
func (r *FlowResource) Status() string {
	if r.IsFromList {
		return string(r.Item.Status)
	}
	if r.DetailItem != nil {
		return string(r.DetailItem.Status)
	}
	return ""
}

// Description returns the flow description
func (r *FlowResource) Description() string {
	if r.IsFromList {
		return appaws.Str(r.Item.Description)
	}
	if r.DetailItem != nil {
		return appaws.Str(r.DetailItem.Description)
	}
	return ""
}

// Version returns the flow version
func (r *FlowResource) Version() string {
	if r.IsFromList {
		return appaws.Str(r.Item.Version)
	}
	if r.DetailItem != nil {
		return appaws.Str(r.DetailItem.Version)
	}
	return ""
}

// UpdatedAt returns the last update time
func (r *FlowResource) UpdatedAt() *time.Time {
	if r.IsFromList {
		return r.Item.UpdatedAt
	}
	if r.DetailItem != nil {
		return r.DetailItem.UpdatedAt
	}
	return nil
}

// CreatedAt returns the creation time
func (r *FlowResource) CreatedAt() *time.Time {
	if r.IsFromList {
		return r.Item.CreatedAt
	}
	if r.DetailItem != nil {
		return r.DetailItem.CreatedAt
	}
	return nil
}

// ExecutionRoleArn returns the execution role ARN
func (r *FlowResource) ExecutionRoleArn() string {
	if r.DetailItem != nil {
		return appaws.Str(r.DetailItem.ExecutionRoleArn)
	}
	return ""
}

// NodeCount returns the number of nodes in the flow
func (r *FlowResource) NodeCount() int {
	if r.DetailItem != nil && r.DetailItem.Definition != nil {
		return len(r.DetailItem.Definition.Nodes)
	}
	return 0
}

// ConnectionCount returns the number of connections in the flow
func (r *FlowResource) ConnectionCount() int {
	if r.DetailItem != nil && r.DetailItem.Definition != nil {
		return len(r.DetailItem.Definition.Connections)
	}
	return 0
}
