package prompts

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/bedrockagent"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagent/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// PromptDAO provides data access for Bedrock Prompts
type PromptDAO struct {
	dao.BaseDAO
	client *bedrockagent.Client
}

// NewPromptDAO creates a new PromptDAO
func NewPromptDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new bedrock-agent/prompts dao")
	}
	return &PromptDAO{
		BaseDAO: dao.NewBaseDAO("bedrock-agent", "prompts"),
		client:  bedrockagent.NewFromConfig(cfg),
	}, nil
}

func (d *PromptDAO) List(ctx context.Context) ([]dao.Resource, error) {
	prompts, err := appaws.Paginate(ctx, func(token *string) ([]types.PromptSummary, *string, error) {
		output, err := d.client.ListPrompts(ctx, &bedrockagent.ListPromptsInput{
			NextToken:  token,
			MaxResults: appaws.Int32Ptr(100),
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list prompts")
		}
		return output.PromptSummaries, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(prompts))
	for i, prompt := range prompts {
		resources[i] = NewPromptResource(prompt)
	}

	return resources, nil
}

func (d *PromptDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.GetPrompt(ctx, &bedrockagent.GetPromptInput{
		PromptIdentifier: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get prompt %s", id)
	}

	return NewPromptResourceFromDetail(output), nil
}

func (d *PromptDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeletePrompt(ctx, &bedrockagent.DeletePromptInput{
		PromptIdentifier: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete prompt %s", id)
	}
	return nil
}

// PromptResource wraps a Bedrock Prompt
type PromptResource struct {
	dao.BaseResource
	Item       types.PromptSummary
	DetailItem *bedrockagent.GetPromptOutput
	IsFromList bool
}

// NewPromptResource creates a new PromptResource from list output
func NewPromptResource(prompt types.PromptSummary) *PromptResource {
	id := appaws.Str(prompt.Id)
	name := appaws.Str(prompt.Name)
	arn := appaws.Str(prompt.Arn)

	return &PromptResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: name,
			ARN:  arn,
			Data: prompt,
		},
		Item:       prompt,
		IsFromList: true,
	}
}

// NewPromptResourceFromDetail creates a PromptResource from detail output
func NewPromptResourceFromDetail(output *bedrockagent.GetPromptOutput) *PromptResource {
	id := appaws.Str(output.Id)
	name := appaws.Str(output.Name)
	arn := appaws.Str(output.Arn)

	return &PromptResource{
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

// Description returns the prompt description
func (r *PromptResource) Description() string {
	if r.IsFromList {
		return appaws.Str(r.Item.Description)
	}
	if r.DetailItem != nil {
		return appaws.Str(r.DetailItem.Description)
	}
	return ""
}

// Version returns the prompt version
func (r *PromptResource) Version() string {
	if r.IsFromList {
		return appaws.Str(r.Item.Version)
	}
	if r.DetailItem != nil {
		return appaws.Str(r.DetailItem.Version)
	}
	return ""
}

// UpdatedAt returns the last update time
func (r *PromptResource) UpdatedAt() *time.Time {
	if r.IsFromList {
		return r.Item.UpdatedAt
	}
	if r.DetailItem != nil {
		return r.DetailItem.UpdatedAt
	}
	return nil
}

// CreatedAt returns the creation time
func (r *PromptResource) CreatedAt() *time.Time {
	if r.IsFromList {
		return r.Item.CreatedAt
	}
	if r.DetailItem != nil {
		return r.DetailItem.CreatedAt
	}
	return nil
}

// DefaultVariant returns the default variant name
func (r *PromptResource) DefaultVariant() string {
	if r.DetailItem != nil {
		return appaws.Str(r.DetailItem.DefaultVariant)
	}
	return ""
}

// VariantCount returns the number of variants
func (r *PromptResource) VariantCount() int {
	if r.DetailItem != nil {
		return len(r.DetailItem.Variants)
	}
	return 0
}

// Variants returns the prompt variants
func (r *PromptResource) Variants() []types.PromptVariant {
	if r.DetailItem != nil {
		return r.DetailItem.Variants
	}
	return nil
}
