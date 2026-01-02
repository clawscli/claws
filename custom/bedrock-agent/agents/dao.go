package agents

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/bedrockagent"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagent/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// AgentDAO provides data access for Bedrock Agents
type AgentDAO struct {
	dao.BaseDAO
	client *bedrockagent.Client
}

// NewAgentDAO creates a new AgentDAO
func NewAgentDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new bedrock-agent/agents dao")
	}
	return &AgentDAO{
		BaseDAO: dao.NewBaseDAO("bedrock-agent", "agents"),
		client:  bedrockagent.NewFromConfig(cfg),
	}, nil
}

func (d *AgentDAO) List(ctx context.Context) ([]dao.Resource, error) {
	agents, err := appaws.Paginate(ctx, func(token *string) ([]types.AgentSummary, *string, error) {
		output, err := d.client.ListAgents(ctx, &bedrockagent.ListAgentsInput{
			NextToken:  token,
			MaxResults: appaws.Int32Ptr(100),
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list agents")
		}
		return output.AgentSummaries, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(agents))
	for i, agent := range agents {
		resources[i] = NewAgentResource(agent)
	}

	return resources, nil
}

func (d *AgentDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.GetAgent(ctx, &bedrockagent.GetAgentInput{
		AgentId: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get agent %s", id)
	}

	return NewAgentResourceFromDetail(output.Agent), nil
}

func (d *AgentDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteAgent(ctx, &bedrockagent.DeleteAgentInput{
		AgentId: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete agent %s", id)
	}
	return nil
}

// AgentResource wraps a Bedrock Agent
type AgentResource struct {
	dao.BaseResource
	Item       types.AgentSummary
	DetailItem *types.Agent
	IsFromList bool
}

// NewAgentResource creates a new AgentResource from list output
func NewAgentResource(agent types.AgentSummary) *AgentResource {
	id := appaws.Str(agent.AgentId)
	name := appaws.Str(agent.AgentName)

	return &AgentResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: name,
			Data: agent,
		},
		Item:       agent,
		IsFromList: true,
	}
}

// NewAgentResourceFromDetail creates an AgentResource from detail output
func NewAgentResourceFromDetail(agent *types.Agent) *AgentResource {
	id := appaws.Str(agent.AgentId)
	name := appaws.Str(agent.AgentName)
	arn := appaws.Str(agent.AgentArn)

	return &AgentResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: name,
			ARN:  arn,
			Data: agent,
		},
		DetailItem: agent,
		IsFromList: false,
	}
}

// Status returns the agent status
func (r *AgentResource) Status() string {
	if r.IsFromList {
		return string(r.Item.AgentStatus)
	}
	if r.DetailItem != nil {
		return string(r.DetailItem.AgentStatus)
	}
	return ""
}

// Description returns the agent description
func (r *AgentResource) Description() string {
	if r.IsFromList {
		return appaws.Str(r.Item.Description)
	}
	if r.DetailItem != nil {
		return appaws.Str(r.DetailItem.Description)
	}
	return ""
}

// LatestVersion returns the latest agent version
func (r *AgentResource) LatestVersion() string {
	if r.IsFromList {
		return appaws.Str(r.Item.LatestAgentVersion)
	}
	if r.DetailItem != nil {
		return appaws.Str(r.DetailItem.AgentVersion)
	}
	return ""
}

// UpdatedAt returns the last update time
func (r *AgentResource) UpdatedAt() *time.Time {
	if r.IsFromList {
		return r.Item.UpdatedAt
	}
	if r.DetailItem != nil {
		return r.DetailItem.UpdatedAt
	}
	return nil
}

// CreatedAt returns the creation time
func (r *AgentResource) CreatedAt() *time.Time {
	if r.DetailItem != nil {
		return r.DetailItem.CreatedAt
	}
	return nil
}

// FoundationModel returns the foundation model used
func (r *AgentResource) FoundationModel() string {
	if r.DetailItem != nil {
		return appaws.Str(r.DetailItem.FoundationModel)
	}
	return ""
}

// RoleArn returns the IAM role ARN
func (r *AgentResource) RoleArn() string {
	if r.DetailItem != nil {
		return appaws.Str(r.DetailItem.AgentResourceRoleArn)
	}
	return ""
}

// Instruction returns the agent instructions
func (r *AgentResource) Instruction() string {
	if r.DetailItem != nil {
		return appaws.Str(r.DetailItem.Instruction)
	}
	return ""
}

// IdleSessionTTL returns the idle session TTL in seconds
func (r *AgentResource) IdleSessionTTL() int32 {
	if r.DetailItem != nil && r.DetailItem.IdleSessionTTLInSeconds != nil {
		return *r.DetailItem.IdleSessionTTLInSeconds
	}
	return 0
}

// GuardrailId returns the guardrail ID if configured
func (r *AgentResource) GuardrailId() string {
	if r.IsFromList && r.Item.GuardrailConfiguration != nil {
		return appaws.Str(r.Item.GuardrailConfiguration.GuardrailIdentifier)
	}
	if r.DetailItem != nil && r.DetailItem.GuardrailConfiguration != nil {
		return appaws.Str(r.DetailItem.GuardrailConfiguration.GuardrailIdentifier)
	}
	return ""
}

// PreparedAt returns when the agent was last prepared
func (r *AgentResource) PreparedAt() *time.Time {
	if r.DetailItem != nil {
		return r.DetailItem.PreparedAt
	}
	return nil
}

// FailureReasons returns any failure reasons
func (r *AgentResource) FailureReasons() []string {
	if r.DetailItem != nil {
		return r.DetailItem.FailureReasons
	}
	return nil
}

// RecommendedActions returns recommended actions
func (r *AgentResource) RecommendedActions() []string {
	if r.DetailItem != nil {
		return r.DetailItem.RecommendedActions
	}
	return nil
}
