package versions

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

// VersionDAO provides data access for Bedrock AgentCore Runtime Versions
type VersionDAO struct {
	dao.BaseDAO
	client *bedrockagentcorecontrol.Client
}

// NewVersionDAO creates a new VersionDAO
func NewVersionDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new bedrock-agentcore/versions dao")
	}
	return &VersionDAO{
		BaseDAO: dao.NewBaseDAO("bedrock-agentcore", "versions"),
		client:  bedrockagentcorecontrol.NewFromConfig(cfg),
	}, nil
}

// List returns runtime versions (first page only for backwards compatibility).
// For paginated access, use ListPage instead.
func (d *VersionDAO) List(ctx context.Context) ([]dao.Resource, error) {
	resources, _, err := d.ListPage(ctx, 50, "")
	return resources, err
}

// ListPage returns a page of Bedrock AgentCore runtime versions.
// Implements dao.PaginatedDAO interface.
func (d *VersionDAO) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
	runtimeID := dao.GetFilterFromContext(ctx, "AgentRuntimeId")
	if runtimeID == "" {
		return nil, "", fmt.Errorf("AgentRuntimeId filter required")
	}

	maxResults := int32(pageSize)
	if maxResults > 50 {
		maxResults = 50 // AWS API max
	}

	input := &bedrockagentcorecontrol.ListAgentRuntimeVersionsInput{
		AgentRuntimeId: &runtimeID,
		MaxResults:     &maxResults,
	}
	if pageToken != "" {
		input.NextToken = &pageToken
	}

	output, err := d.client.ListAgentRuntimeVersions(ctx, input)
	if err != nil {
		return nil, "", apperrors.Wrap(err, "list agent runtime versions")
	}

	resources := make([]dao.Resource, len(output.AgentRuntimes))
	for i, version := range output.AgentRuntimes {
		resources[i] = NewVersionResource(version)
	}

	nextToken := ""
	if output.NextToken != nil {
		nextToken = *output.NextToken
	}

	return resources, nextToken, nil
}

func (d *VersionDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	// For versions, we use GetAgentRuntime with the version
	output, err := d.client.GetAgentRuntime(ctx, &bedrockagentcorecontrol.GetAgentRuntimeInput{
		AgentRuntimeId: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get agent runtime version %s", id)
	}

	return NewVersionResourceFromDetail(output), nil
}

func (d *VersionDAO) Delete(ctx context.Context, id string) error {
	// Versions cannot be directly deleted
	return fmt.Errorf("runtime versions cannot be deleted directly")
}

// VersionResource wraps a Bedrock AgentCore Runtime Version
type VersionResource struct {
	dao.BaseResource
	Item       types.AgentRuntime
	DetailItem *bedrockagentcorecontrol.GetAgentRuntimeOutput
}

// NewVersionResource creates a new VersionResource from list output
func NewVersionResource(runtime types.AgentRuntime) *VersionResource {
	id := appaws.Str(runtime.AgentRuntimeId)
	version := appaws.Str(runtime.AgentRuntimeVersion)
	name := fmt.Sprintf("%s (v%s)", appaws.Str(runtime.AgentRuntimeName), version)
	arn := appaws.Str(runtime.AgentRuntimeArn)

	return &VersionResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: name,
			ARN:  arn,
			Data: runtime,
		},
		Item: runtime,
	}
}

// NewVersionResourceFromDetail creates a VersionResource from detail output
func NewVersionResourceFromDetail(output *bedrockagentcorecontrol.GetAgentRuntimeOutput) *VersionResource {
	id := appaws.Str(output.AgentRuntimeId)
	version := appaws.Str(output.AgentRuntimeVersion)
	name := fmt.Sprintf("%s (v%s)", appaws.Str(output.AgentRuntimeName), version)
	arn := appaws.Str(output.AgentRuntimeArn)

	return &VersionResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: name,
			ARN:  arn,
			Data: output,
		},
		DetailItem: output,
	}
}

// Version returns the runtime version
func (r *VersionResource) Version() string {
	if r.DetailItem != nil {
		return appaws.Str(r.DetailItem.AgentRuntimeVersion)
	}
	return appaws.Str(r.Item.AgentRuntimeVersion)
}

// Status returns the runtime status
func (r *VersionResource) Status() string {
	if r.DetailItem != nil {
		return string(r.DetailItem.Status)
	}
	return string(r.Item.Status)
}

// Description returns the runtime description
func (r *VersionResource) Description() string {
	if r.DetailItem != nil {
		return appaws.Str(r.DetailItem.Description)
	}
	return appaws.Str(r.Item.Description)
}

// RuntimeName returns the runtime name
func (r *VersionResource) RuntimeName() string {
	if r.DetailItem != nil {
		return appaws.Str(r.DetailItem.AgentRuntimeName)
	}
	return appaws.Str(r.Item.AgentRuntimeName)
}

// LastUpdatedAt returns the last update time
func (r *VersionResource) LastUpdatedAt() *time.Time {
	if r.DetailItem != nil {
		return r.DetailItem.LastUpdatedAt
	}
	return r.Item.LastUpdatedAt
}

// CreatedAt returns the creation time
func (r *VersionResource) CreatedAt() *time.Time {
	if r.DetailItem != nil {
		return r.DetailItem.CreatedAt
	}
	return nil
}
