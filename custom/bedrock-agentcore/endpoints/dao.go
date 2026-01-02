package endpoints

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/bedrockagentcorecontrol"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentcorecontrol/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// EndpointDAO provides data access for Bedrock AgentCore Runtime Endpoints
type EndpointDAO struct {
	dao.BaseDAO
	client *bedrockagentcorecontrol.Client
}

// NewEndpointDAO creates a new EndpointDAO
func NewEndpointDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new bedrock-agentcore/endpoints dao")
	}
	return &EndpointDAO{
		BaseDAO: dao.NewBaseDAO("bedrock-agentcore", "endpoints"),
		client:  bedrockagentcorecontrol.NewFromConfig(cfg),
	}, nil
}

func (d *EndpointDAO) List(ctx context.Context) ([]dao.Resource, error) {
	// Get runtime ID from filter
	runtimeID := dao.GetFilterFromContext(ctx, "AgentRuntimeId")
	if runtimeID == "" {
		return nil, fmt.Errorf("AgentRuntimeId filter required")
	}

	endpoints, err := appaws.Paginate(ctx, func(token *string) ([]types.AgentRuntimeEndpoint, *string, error) {
		output, err := d.client.ListAgentRuntimeEndpoints(ctx, &bedrockagentcorecontrol.ListAgentRuntimeEndpointsInput{
			AgentRuntimeId: &runtimeID,
			NextToken:      token,
			MaxResults:     appaws.Int32Ptr(50),
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list agent runtime endpoints")
		}
		return output.RuntimeEndpoints, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(endpoints))
	for i, endpoint := range endpoints {
		resources[i] = NewEndpointResource(endpoint, runtimeID)
	}

	return resources, nil
}

func (d *EndpointDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	// Get runtime ID from filter
	runtimeID := dao.GetFilterFromContext(ctx, "AgentRuntimeId")
	if runtimeID == "" {
		return nil, fmt.Errorf("AgentRuntimeId filter required")
	}

	output, err := d.client.GetAgentRuntimeEndpoint(ctx, &bedrockagentcorecontrol.GetAgentRuntimeEndpointInput{
		AgentRuntimeId: &runtimeID,
		EndpointName:   &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get agent runtime endpoint %s", id)
	}

	return NewEndpointResourceFromDetail(output, runtimeID), nil
}

func (d *EndpointDAO) Delete(ctx context.Context, id string) error {
	runtimeID := dao.GetFilterFromContext(ctx, "AgentRuntimeId")
	if runtimeID == "" {
		return fmt.Errorf("AgentRuntimeId filter required")
	}

	_, err := d.client.DeleteAgentRuntimeEndpoint(ctx, &bedrockagentcorecontrol.DeleteAgentRuntimeEndpointInput{
		AgentRuntimeId: &runtimeID,
		EndpointName:   &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete agent runtime endpoint %s", id)
	}

	return nil
}

// EndpointResource wraps a Bedrock AgentCore Runtime Endpoint
type EndpointResource struct {
	dao.BaseResource
	Item       types.AgentRuntimeEndpoint
	DetailItem *bedrockagentcorecontrol.GetAgentRuntimeEndpointOutput
	RuntimeID  string
}

// NewEndpointResource creates a new EndpointResource from list output
// Note: Uses Name as ID since Get/Delete APIs require EndpointName
func NewEndpointResource(endpoint types.AgentRuntimeEndpoint, runtimeID string) *EndpointResource {
	name := appaws.Str(endpoint.Name)
	arn := appaws.Str(endpoint.AgentRuntimeEndpointArn)

	return &EndpointResource{
		BaseResource: dao.BaseResource{
			ID:   name, // Use name as ID since Get/Delete APIs require EndpointName
			Name: name,
			ARN:  arn,
			Data: endpoint,
		},
		Item:      endpoint,
		RuntimeID: runtimeID,
	}
}

// NewEndpointResourceFromDetail creates an EndpointResource from detail output
func NewEndpointResourceFromDetail(output *bedrockagentcorecontrol.GetAgentRuntimeEndpointOutput, runtimeID string) *EndpointResource {
	name := appaws.Str(output.Name)
	arn := appaws.Str(output.AgentRuntimeEndpointArn)

	return &EndpointResource{
		BaseResource: dao.BaseResource{
			ID:   name, // Use name as ID since Get/Delete APIs require EndpointName
			Name: name,
			ARN:  arn,
			Data: output,
		},
		DetailItem: output,
		RuntimeID:  runtimeID,
	}
}

// Status returns the endpoint status
func (r *EndpointResource) Status() string {
	if r.DetailItem != nil {
		return string(r.DetailItem.Status)
	}
	return string(r.Item.Status)
}

// Description returns the endpoint description
func (r *EndpointResource) Description() string {
	if r.DetailItem != nil {
		return appaws.Str(r.DetailItem.Description)
	}
	return appaws.Str(r.Item.Description)
}

// LiveVersion returns the live version
func (r *EndpointResource) LiveVersion() string {
	if r.DetailItem != nil {
		return appaws.Str(r.DetailItem.LiveVersion)
	}
	return appaws.Str(r.Item.LiveVersion)
}

// TargetVersion returns the target version
func (r *EndpointResource) TargetVersion() string {
	if r.DetailItem != nil {
		return appaws.Str(r.DetailItem.TargetVersion)
	}
	return appaws.Str(r.Item.TargetVersion)
}

// CreatedAt returns the creation time
func (r *EndpointResource) CreatedAt() *time.Time {
	if r.DetailItem != nil {
		return r.DetailItem.CreatedAt
	}
	return r.Item.CreatedAt
}

// LastUpdatedAt returns the last update time
func (r *EndpointResource) LastUpdatedAt() *time.Time {
	if r.DetailItem != nil {
		return r.DetailItem.LastUpdatedAt
	}
	return r.Item.LastUpdatedAt
}
