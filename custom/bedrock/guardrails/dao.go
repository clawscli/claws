package guardrails

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	"github.com/aws/aws-sdk-go-v2/service/bedrock/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// GuardrailDAO provides data access for Bedrock Guardrails
type GuardrailDAO struct {
	dao.BaseDAO
	client *bedrock.Client
}

// NewGuardrailDAO creates a new GuardrailDAO
func NewGuardrailDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new bedrock/guardrails dao: %w", err)
	}
	return &GuardrailDAO{
		BaseDAO: dao.NewBaseDAO("bedrock", "guardrails"),
		client:  bedrock.NewFromConfig(cfg),
	}, nil
}

func (d *GuardrailDAO) List(ctx context.Context) ([]dao.Resource, error) {
	guardrails, err := appaws.Paginate(ctx, func(token *string) ([]types.GuardrailSummary, *string, error) {
		output, err := d.client.ListGuardrails(ctx, &bedrock.ListGuardrailsInput{
			NextToken:  token,
			MaxResults: appaws.Int32Ptr(100),
		})
		if err != nil {
			return nil, nil, fmt.Errorf("list guardrails: %w", err)
		}
		return output.Guardrails, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(guardrails))
	for i, guardrail := range guardrails {
		resources[i] = NewGuardrailResource(guardrail)
	}

	return resources, nil
}

func (d *GuardrailDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.GetGuardrail(ctx, &bedrock.GetGuardrailInput{
		GuardrailIdentifier: &id,
	})
	if err != nil {
		return nil, fmt.Errorf("get guardrail %s: %w", id, err)
	}

	return NewGuardrailResourceFromDetail(output), nil
}

func (d *GuardrailDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteGuardrail(ctx, &bedrock.DeleteGuardrailInput{
		GuardrailIdentifier: &id,
	})
	if err != nil {
		return fmt.Errorf("delete guardrail %s: %w", id, err)
	}
	return nil
}

// GuardrailResource wraps a Bedrock Guardrail
type GuardrailResource struct {
	dao.BaseResource
	Item       types.GuardrailSummary
	DetailItem *bedrock.GetGuardrailOutput
	IsFromList bool
}

// NewGuardrailResource creates a new GuardrailResource from list output
func NewGuardrailResource(guardrail types.GuardrailSummary) *GuardrailResource {
	id := appaws.Str(guardrail.Id)
	name := appaws.Str(guardrail.Name)
	arn := appaws.Str(guardrail.Arn)

	return &GuardrailResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: name,
			ARN:  arn,
			Data: guardrail,
		},
		Item:       guardrail,
		IsFromList: true,
	}
}

// NewGuardrailResourceFromDetail creates a GuardrailResource from detail output
func NewGuardrailResourceFromDetail(output *bedrock.GetGuardrailOutput) *GuardrailResource {
	id := appaws.Str(output.GuardrailId)
	name := appaws.Str(output.Name)
	arn := appaws.Str(output.GuardrailArn)

	return &GuardrailResource{
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

// Status returns the guardrail status
func (r *GuardrailResource) Status() string {
	if r.IsFromList {
		return string(r.Item.Status)
	}
	if r.DetailItem != nil {
		return string(r.DetailItem.Status)
	}
	return ""
}

// Description returns the guardrail description
func (r *GuardrailResource) Description() string {
	if r.IsFromList {
		return appaws.Str(r.Item.Description)
	}
	if r.DetailItem != nil {
		return appaws.Str(r.DetailItem.Description)
	}
	return ""
}

// Version returns the guardrail version
func (r *GuardrailResource) Version() string {
	if r.IsFromList {
		return appaws.Str(r.Item.Version)
	}
	if r.DetailItem != nil {
		return appaws.Str(r.DetailItem.Version)
	}
	return ""
}

// UpdatedAt returns the last update time
func (r *GuardrailResource) UpdatedAt() *time.Time {
	if r.IsFromList {
		return r.Item.UpdatedAt
	}
	if r.DetailItem != nil {
		return r.DetailItem.UpdatedAt
	}
	return nil
}

// CreatedAt returns the creation time
func (r *GuardrailResource) CreatedAt() *time.Time {
	if r.IsFromList {
		return r.Item.CreatedAt
	}
	if r.DetailItem != nil {
		return r.DetailItem.CreatedAt
	}
	return nil
}

// BlockedInputMessaging returns the blocked input message
func (r *GuardrailResource) BlockedInputMessaging() string {
	if r.DetailItem != nil {
		return appaws.Str(r.DetailItem.BlockedInputMessaging)
	}
	return ""
}

// BlockedOutputsMessaging returns the blocked output message
func (r *GuardrailResource) BlockedOutputsMessaging() string {
	if r.DetailItem != nil {
		return appaws.Str(r.DetailItem.BlockedOutputsMessaging)
	}
	return ""
}

// FailureRecommendations returns failure recommendations
func (r *GuardrailResource) FailureRecommendations() []string {
	if r.DetailItem != nil {
		return r.DetailItem.FailureRecommendations
	}
	return nil
}
