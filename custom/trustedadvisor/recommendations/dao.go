package recommendations

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/trustedadvisor"
	"github.com/aws/aws-sdk-go-v2/service/trustedadvisor/types"
	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// RecommendationDAO provides data access for Trusted Advisor Recommendations.
type RecommendationDAO struct {
	dao.BaseDAO
	client *trustedadvisor.Client
}

// NewRecommendationDAO creates a new RecommendationDAO.
func NewRecommendationDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new trustedadvisor/recommendations dao: %w", err)
	}
	return &RecommendationDAO{
		BaseDAO: dao.NewBaseDAO("trustedadvisor", "recommendations"),
		client:  trustedadvisor.NewFromConfig(cfg),
	}, nil
}

// List returns all Trusted Advisor recommendations.
func (d *RecommendationDAO) List(ctx context.Context) ([]dao.Resource, error) {
	recs, err := appaws.Paginate(ctx, func(token *string) ([]types.RecommendationSummary, *string, error) {
		output, err := d.client.ListRecommendations(ctx, &trustedadvisor.ListRecommendationsInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("list recommendations: %w", err)
		}
		return output.RecommendationSummaries, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(recs))
	for i, rec := range recs {
		resources[i] = NewRecommendationResource(rec)
	}
	return resources, nil
}

// Get returns a specific recommendation by ID.
func (d *RecommendationDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.GetRecommendation(ctx, &trustedadvisor.GetRecommendationInput{
		RecommendationIdentifier: &id,
	})
	if err != nil {
		return nil, fmt.Errorf("get recommendation %s: %w", id, err)
	}

	if output.Recommendation == nil {
		return nil, fmt.Errorf("recommendation not found: %s", id)
	}

	// Convert Recommendation to RecommendationSummary for consistent resource type
	summary := types.RecommendationSummary{
		Arn:                 output.Recommendation.Arn,
		Id:                  output.Recommendation.Id,
		Name:                output.Recommendation.Name,
		Pillars:             output.Recommendation.Pillars,
		ResourcesAggregates: output.Recommendation.ResourcesAggregates,
		Source:              output.Recommendation.Source,
		Status:              output.Recommendation.Status,
		Type:                output.Recommendation.Type,
		AwsServices:         output.Recommendation.AwsServices,
		CheckArn:            output.Recommendation.CheckArn,
		CreatedAt:           output.Recommendation.CreatedAt,
		LastUpdatedAt:       output.Recommendation.LastUpdatedAt,
		LifecycleStage:      output.Recommendation.LifecycleStage,
	}

	return NewRecommendationResource(summary), nil
}

// Delete is not supported for recommendations.
func (d *RecommendationDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported for trusted advisor recommendations")
}

// RecommendationResource wraps a Trusted Advisor Recommendation.
type RecommendationResource struct {
	dao.BaseResource
}

// NewRecommendationResource creates a new RecommendationResource.
func NewRecommendationResource(rec types.RecommendationSummary) *RecommendationResource {
	return &RecommendationResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(rec.Id),
			Name: appaws.Str(rec.Name),
			ARN:  appaws.Str(rec.Arn),
			Data: rec,
		},
	}
}

// item returns the underlying SDK type.
func (r *RecommendationResource) item() types.RecommendationSummary {
	return r.Data.(types.RecommendationSummary)
}

// Name returns the recommendation name.
func (r *RecommendationResource) Name() string {
	return appaws.Str(r.item().Name)
}

// Status returns the recommendation status.
func (r *RecommendationResource) Status() string {
	return string(r.item().Status)
}

// Pillars returns the pillars as a comma-separated string.
func (r *RecommendationResource) Pillars() string {
	pillars := make([]string, len(r.item().Pillars))
	for i, p := range r.item().Pillars {
		pillars[i] = string(p)
	}
	return strings.Join(pillars, ", ")
}

// PillarList returns the pillars as a slice.
func (r *RecommendationResource) PillarList() []types.RecommendationPillar {
	return r.item().Pillars
}

// Source returns the recommendation source.
func (r *RecommendationResource) Source() string {
	return string(r.item().Source)
}

// Type returns the recommendation type.
func (r *RecommendationResource) Type() string {
	return string(r.item().Type)
}

// ErrorCount returns the number of resources with errors.
func (r *RecommendationResource) ErrorCount() int64 {
	if r.item().ResourcesAggregates != nil {
		return appaws.Int64(r.item().ResourcesAggregates.ErrorCount)
	}
	return 0
}

// WarningCount returns the number of resources with warnings.
func (r *RecommendationResource) WarningCount() int64 {
	if r.item().ResourcesAggregates != nil {
		return appaws.Int64(r.item().ResourcesAggregates.WarningCount)
	}
	return 0
}

// OkCount returns the number of resources that are OK.
func (r *RecommendationResource) OkCount() int64 {
	if r.item().ResourcesAggregates != nil {
		return appaws.Int64(r.item().ResourcesAggregates.OkCount)
	}
	return 0
}

// AwsServices returns the AWS services this recommendation applies to.
func (r *RecommendationResource) AwsServices() []string {
	return r.item().AwsServices
}

// CreatedAt returns the creation time as a formatted string.
func (r *RecommendationResource) CreatedAt() string {
	if r.item().CreatedAt != nil {
		return r.item().CreatedAt.Format("2006-01-02 15:04:05")
	}
	return ""
}

// LastUpdatedAt returns the last update time as a formatted string.
func (r *RecommendationResource) LastUpdatedAt() string {
	if r.item().LastUpdatedAt != nil {
		return r.item().LastUpdatedAt.Format("2006-01-02 15:04:05")
	}
	return ""
}

// LifecycleStage returns the lifecycle stage.
func (r *RecommendationResource) LifecycleStage() string {
	return string(r.item().LifecycleStage)
}
