package inferenceprofiles

import (
	"context"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	"github.com/aws/aws-sdk-go-v2/service/bedrock/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// InferenceProfileDAO provides data access for Bedrock Inference Profiles
type InferenceProfileDAO struct {
	dao.BaseDAO
	client *bedrock.Client
}

// NewInferenceProfileDAO creates a new InferenceProfileDAO
func NewInferenceProfileDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new bedrock/inferenceprofiles dao")
	}
	return &InferenceProfileDAO{
		BaseDAO: dao.NewBaseDAO("bedrock", "inference-profiles"),
		client:  bedrock.NewFromConfig(cfg),
	}, nil
}

func (d *InferenceProfileDAO) List(ctx context.Context) ([]dao.Resource, error) {
	profiles, err := appaws.Paginate(ctx, func(token *string) ([]types.InferenceProfileSummary, *string, error) {
		output, err := d.client.ListInferenceProfiles(ctx, &bedrock.ListInferenceProfilesInput{
			NextToken:  token,
			MaxResults: appaws.Int32Ptr(100),
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list inference profiles")
		}
		return output.InferenceProfileSummaries, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(profiles))
	for i, profile := range profiles {
		resources[i] = NewInferenceProfileResource(profile)
	}

	return resources, nil
}

func (d *InferenceProfileDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.GetInferenceProfile(ctx, &bedrock.GetInferenceProfileInput{
		InferenceProfileIdentifier: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get inference profile %s", id)
	}

	return NewInferenceProfileResourceFromDetail(output), nil
}

func (d *InferenceProfileDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteInferenceProfile(ctx, &bedrock.DeleteInferenceProfileInput{
		InferenceProfileIdentifier: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete inference profile %s", id)
	}
	return nil
}

// InferenceProfileResource wraps a Bedrock Inference Profile
type InferenceProfileResource struct {
	dao.BaseResource
	Item       types.InferenceProfileSummary
	DetailItem *bedrock.GetInferenceProfileOutput
	IsFromList bool
}

// NewInferenceProfileResource creates a new InferenceProfileResource from list output
func NewInferenceProfileResource(profile types.InferenceProfileSummary) *InferenceProfileResource {
	id := appaws.Str(profile.InferenceProfileId)
	name := appaws.Str(profile.InferenceProfileName)
	arn := appaws.Str(profile.InferenceProfileArn)

	return &InferenceProfileResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: name,
			ARN:  arn,
			Data: profile,
		},
		Item:       profile,
		IsFromList: true,
	}
}

// NewInferenceProfileResourceFromDetail creates an InferenceProfileResource from detail output
func NewInferenceProfileResourceFromDetail(output *bedrock.GetInferenceProfileOutput) *InferenceProfileResource {
	id := appaws.Str(output.InferenceProfileId)
	name := appaws.Str(output.InferenceProfileName)
	arn := appaws.Str(output.InferenceProfileArn)

	return &InferenceProfileResource{
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

// Status returns the inference profile status
func (r *InferenceProfileResource) Status() string {
	if r.IsFromList {
		return string(r.Item.Status)
	}
	if r.DetailItem != nil {
		return string(r.DetailItem.Status)
	}
	return ""
}

// ProfileType returns the inference profile type
func (r *InferenceProfileResource) ProfileType() string {
	if r.IsFromList {
		return string(r.Item.Type)
	}
	if r.DetailItem != nil {
		return string(r.DetailItem.Type)
	}
	return ""
}

// Description returns the inference profile description
func (r *InferenceProfileResource) Description() string {
	if r.IsFromList {
		return appaws.Str(r.Item.Description)
	}
	if r.DetailItem != nil {
		return appaws.Str(r.DetailItem.Description)
	}
	return ""
}

// UpdatedAt returns the last update time
func (r *InferenceProfileResource) UpdatedAt() *time.Time {
	if r.IsFromList {
		return r.Item.UpdatedAt
	}
	if r.DetailItem != nil {
		return r.DetailItem.UpdatedAt
	}
	return nil
}

// CreatedAt returns the creation time
func (r *InferenceProfileResource) CreatedAt() *time.Time {
	if r.IsFromList {
		return r.Item.CreatedAt
	}
	if r.DetailItem != nil {
		return r.DetailItem.CreatedAt
	}
	return nil
}

// Models returns a comma-separated list of model ARNs
func (r *InferenceProfileResource) Models() string {
	var models []types.InferenceProfileModel
	if r.IsFromList {
		models = r.Item.Models
	} else if r.DetailItem != nil {
		models = r.DetailItem.Models
	}

	arns := make([]string, len(models))
	for i, m := range models {
		arns[i] = appaws.Str(m.ModelArn)
	}
	return strings.Join(arns, ", ")
}

// ModelCount returns the number of models in the profile
func (r *InferenceProfileResource) ModelCount() int {
	if r.IsFromList {
		return len(r.Item.Models)
	}
	if r.DetailItem != nil {
		return len(r.DetailItem.Models)
	}
	return 0
}
