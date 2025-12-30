package foundationmodels

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	"github.com/aws/aws-sdk-go-v2/service/bedrock/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// FoundationModelDAO provides data access for Bedrock Foundation Models
type FoundationModelDAO struct {
	dao.BaseDAO
	client *bedrock.Client
}

// NewFoundationModelDAO creates a new FoundationModelDAO
func NewFoundationModelDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new bedrock/foundationmodels dao")
	}
	return &FoundationModelDAO{
		BaseDAO: dao.NewBaseDAO("bedrock", "foundation-models"),
		client:  bedrock.NewFromConfig(cfg),
	}, nil
}

// Client returns the bedrock client for shared use
func (d *FoundationModelDAO) Client() *bedrock.Client {
	return d.client
}

func (d *FoundationModelDAO) List(ctx context.Context) ([]dao.Resource, error) {
	// ListFoundationModels does not have pagination
	output, err := d.client.ListFoundationModels(ctx, &bedrock.ListFoundationModelsInput{})
	if err != nil {
		return nil, apperrors.Wrap(err, "list foundation models")
	}

	resources := make([]dao.Resource, len(output.ModelSummaries))
	for i, model := range output.ModelSummaries {
		resources[i] = NewFoundationModelResource(model)
	}

	return resources, nil
}

func (d *FoundationModelDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.GetFoundationModel(ctx, &bedrock.GetFoundationModelInput{
		ModelIdentifier: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get foundation model %s", id)
	}

	return NewFoundationModelResourceFromDetail(output.ModelDetails), nil
}

func (d *FoundationModelDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("foundation models cannot be deleted")
}

// FoundationModelResource wraps a Bedrock Foundation Model
type FoundationModelResource struct {
	dao.BaseResource
	Item       types.FoundationModelSummary
	DetailItem *types.FoundationModelDetails
	IsFromList bool
}

// NewFoundationModelResource creates a new FoundationModelResource from list output
func NewFoundationModelResource(model types.FoundationModelSummary) *FoundationModelResource {
	id := appaws.Str(model.ModelId)
	name := appaws.Str(model.ModelName)
	arn := appaws.Str(model.ModelArn)

	return &FoundationModelResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: name,
			ARN:  arn,
			Data: model,
		},
		Item:       model,
		IsFromList: true,
	}
}

// NewFoundationModelResourceFromDetail creates a FoundationModelResource from detail output
func NewFoundationModelResourceFromDetail(model *types.FoundationModelDetails) *FoundationModelResource {
	id := appaws.Str(model.ModelId)
	name := appaws.Str(model.ModelName)
	arn := appaws.Str(model.ModelArn)

	return &FoundationModelResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: name,
			ARN:  arn,
			Data: model,
		},
		DetailItem: model,
		IsFromList: false,
	}
}

// Provider returns the model provider name
func (r *FoundationModelResource) Provider() string {
	if r.IsFromList {
		return appaws.Str(r.Item.ProviderName)
	}
	if r.DetailItem != nil {
		return appaws.Str(r.DetailItem.ProviderName)
	}
	return ""
}

// InputModalities returns the input modalities
func (r *FoundationModelResource) InputModalities() string {
	var modalities []types.ModelModality
	if r.IsFromList {
		modalities = r.Item.InputModalities
	} else if r.DetailItem != nil {
		modalities = r.DetailItem.InputModalities
	}
	strs := make([]string, len(modalities))
	for i, m := range modalities {
		strs[i] = string(m)
	}
	return strings.Join(strs, ", ")
}

// OutputModalities returns the output modalities
func (r *FoundationModelResource) OutputModalities() string {
	var modalities []types.ModelModality
	if r.IsFromList {
		modalities = r.Item.OutputModalities
	} else if r.DetailItem != nil {
		modalities = r.DetailItem.OutputModalities
	}
	strs := make([]string, len(modalities))
	for i, m := range modalities {
		strs[i] = string(m)
	}
	return strings.Join(strs, ", ")
}

// InferenceTypes returns the supported inference types
func (r *FoundationModelResource) InferenceTypes() string {
	var inferenceTypes []types.InferenceType
	if r.IsFromList {
		inferenceTypes = r.Item.InferenceTypesSupported
	} else if r.DetailItem != nil {
		inferenceTypes = r.DetailItem.InferenceTypesSupported
	}
	strs := make([]string, len(inferenceTypes))
	for i, t := range inferenceTypes {
		strs[i] = string(t)
	}
	return strings.Join(strs, ", ")
}

// StreamingSupported returns whether streaming is supported
func (r *FoundationModelResource) StreamingSupported() bool {
	if r.IsFromList {
		if r.Item.ResponseStreamingSupported != nil {
			return *r.Item.ResponseStreamingSupported
		}
	} else if r.DetailItem != nil {
		if r.DetailItem.ResponseStreamingSupported != nil {
			return *r.DetailItem.ResponseStreamingSupported
		}
	}
	return false
}

// CustomizationsSupported returns the supported customization types
func (r *FoundationModelResource) CustomizationsSupported() string {
	var customizations []types.ModelCustomization
	if r.IsFromList {
		customizations = r.Item.CustomizationsSupported
	} else if r.DetailItem != nil {
		customizations = r.DetailItem.CustomizationsSupported
	}
	strs := make([]string, len(customizations))
	for i, c := range customizations {
		strs[i] = string(c)
	}
	return strings.Join(strs, ", ")
}

// LifecycleStatus returns the model lifecycle status
func (r *FoundationModelResource) LifecycleStatus() string {
	if r.IsFromList && r.Item.ModelLifecycle != nil {
		return string(r.Item.ModelLifecycle.Status)
	}
	if r.DetailItem != nil && r.DetailItem.ModelLifecycle != nil {
		return string(r.DetailItem.ModelLifecycle.Status)
	}
	return ""
}
