package runtimes

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/bedrockagentcorecontrol"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentcorecontrol/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// RuntimeDAO provides data access for Bedrock AgentCore Runtimes
type RuntimeDAO struct {
	dao.BaseDAO
	client *bedrockagentcorecontrol.Client
}

// NewRuntimeDAO creates a new RuntimeDAO
func NewRuntimeDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new bedrock-agentcore/runtimes dao")
	}
	return &RuntimeDAO{
		BaseDAO: dao.NewBaseDAO("bedrock-agentcore", "runtimes"),
		client:  bedrockagentcorecontrol.NewFromConfig(cfg),
	}, nil
}

func (d *RuntimeDAO) List(ctx context.Context) ([]dao.Resource, error) {
	runtimes, err := appaws.Paginate(ctx, func(token *string) ([]types.AgentRuntime, *string, error) {
		output, err := d.client.ListAgentRuntimes(ctx, &bedrockagentcorecontrol.ListAgentRuntimesInput{
			NextToken:  token,
			MaxResults: appaws.Int32Ptr(50),
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list agent runtimes")
		}
		return output.AgentRuntimes, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(runtimes))
	for i, runtime := range runtimes {
		resources[i] = NewRuntimeResource(runtime)
	}

	return resources, nil
}

func (d *RuntimeDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	input := &bedrockagentcorecontrol.GetAgentRuntimeInput{
		AgentRuntimeId: &id,
	}

	output, err := d.client.GetAgentRuntime(ctx, input)
	if err != nil {
		return nil, apperrors.Wrapf(err, "get agent runtime %s", id)
	}

	return NewRuntimeResourceFromDetail(output), nil
}

func (d *RuntimeDAO) Delete(ctx context.Context, id string) error {
	input := &bedrockagentcorecontrol.DeleteAgentRuntimeInput{
		AgentRuntimeId: &id,
	}

	_, err := d.client.DeleteAgentRuntime(ctx, input)
	if err != nil {
		return apperrors.Wrapf(err, "delete agent runtime %s", id)
	}

	return nil
}

// RuntimeResource wraps a Bedrock AgentCore Runtime
type RuntimeResource struct {
	dao.BaseResource
	Item       types.AgentRuntime
	DetailItem *bedrockagentcorecontrol.GetAgentRuntimeOutput
	IsFromList bool
}

// NewRuntimeResource creates a new RuntimeResource from list output
func NewRuntimeResource(runtime types.AgentRuntime) *RuntimeResource {
	id := appaws.Str(runtime.AgentRuntimeId)
	name := appaws.Str(runtime.AgentRuntimeName)
	arn := appaws.Str(runtime.AgentRuntimeArn)

	return &RuntimeResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: name,
			ARN:  arn,
			Tags: nil,
			Data: runtime,
		},
		Item:       runtime,
		IsFromList: true,
	}
}

// NewRuntimeResourceFromDetail creates a RuntimeResource from detail output
func NewRuntimeResourceFromDetail(output *bedrockagentcorecontrol.GetAgentRuntimeOutput) *RuntimeResource {
	id := appaws.Str(output.AgentRuntimeId)
	name := appaws.Str(output.AgentRuntimeName)
	arn := appaws.Str(output.AgentRuntimeArn)

	return &RuntimeResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: name,
			ARN:  arn,
			Tags: nil,
			Data: output,
		},
		DetailItem: output,
		IsFromList: false,
	}
}

// Status returns the runtime status
func (r *RuntimeResource) Status() string {
	if r.IsFromList {
		return string(r.Item.Status)
	}
	if r.DetailItem != nil {
		return string(r.DetailItem.Status)
	}
	return ""
}

// Description returns the runtime description
func (r *RuntimeResource) Description() string {
	if r.IsFromList {
		return appaws.Str(r.Item.Description)
	}
	if r.DetailItem != nil {
		return appaws.Str(r.DetailItem.Description)
	}
	return ""
}

// CreatedAt returns the creation time
func (r *RuntimeResource) CreatedAt() *time.Time {
	// List response doesn't include CreatedAt, only detail does
	if r.DetailItem != nil {
		return r.DetailItem.CreatedAt
	}
	return nil
}

// LastUpdatedAt returns the last update time
func (r *RuntimeResource) LastUpdatedAt() *time.Time {
	if r.IsFromList {
		return r.Item.LastUpdatedAt
	}
	if r.DetailItem != nil {
		return r.DetailItem.LastUpdatedAt
	}
	return nil
}

// Version returns the runtime version
func (r *RuntimeResource) Version() string {
	if r.IsFromList {
		return appaws.Str(r.Item.AgentRuntimeVersion)
	}
	if r.DetailItem != nil {
		return appaws.Str(r.DetailItem.AgentRuntimeVersion)
	}
	return ""
}

// RoleArn returns the IAM role ARN
func (r *RuntimeResource) RoleArn() string {
	if r.DetailItem != nil {
		return appaws.Str(r.DetailItem.RoleArn)
	}
	return ""
}
