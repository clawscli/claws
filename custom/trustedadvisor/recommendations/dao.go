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

// Get returns a specific recommendation by ID with full details.
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

	return NewRecommendationResourceFull(*output.Recommendation), nil
}

// Delete is not supported for recommendations.
func (d *RecommendationDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported for trusted advisor recommendations")
}

// Supports returns true for List and Get operations only.
func (d *RecommendationDAO) Supports(op dao.Operation) bool {
	return op == dao.OpList || op == dao.OpGet
}

// RecommendationResource wraps a Trusted Advisor Recommendation.
type RecommendationResource struct {
	dao.BaseResource
}

// NewRecommendationResource creates a new RecommendationResource from summary.
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

// NewRecommendationResourceFull creates a new RecommendationResource from full recommendation.
func NewRecommendationResourceFull(rec types.Recommendation) *RecommendationResource {
	return &RecommendationResource{
		BaseResource: dao.BaseResource{
			ID:   appaws.Str(rec.Id),
			Name: appaws.Str(rec.Name),
			ARN:  appaws.Str(rec.Arn),
			Data: rec,
		},
	}
}

// itemSummary returns the underlying SDK type as RecommendationSummary if available.
func (r *RecommendationResource) itemSummary() (types.RecommendationSummary, bool) {
	s, ok := r.Data.(types.RecommendationSummary)
	return s, ok
}

// itemFull returns the underlying SDK type as full Recommendation if available.
func (r *RecommendationResource) itemFull() (types.Recommendation, bool) {
	f, ok := r.Data.(types.Recommendation)
	return f, ok
}

// Name returns the recommendation name.
func (r *RecommendationResource) Name() string {
	if f, ok := r.itemFull(); ok {
		return appaws.Str(f.Name)
	}
	if s, ok := r.itemSummary(); ok {
		return appaws.Str(s.Name)
	}
	return ""
}

// Status returns the recommendation status.
func (r *RecommendationResource) Status() string {
	if f, ok := r.itemFull(); ok {
		return string(f.Status)
	}
	if s, ok := r.itemSummary(); ok {
		return string(s.Status)
	}
	return ""
}

// Pillars returns the pillars as a comma-separated string.
func (r *RecommendationResource) Pillars() string {
	list := r.PillarList()
	pillars := make([]string, len(list))
	for i, p := range list {
		pillars[i] = string(p)
	}
	return strings.Join(pillars, ", ")
}

// PillarList returns the pillars as a slice.
func (r *RecommendationResource) PillarList() []types.RecommendationPillar {
	if f, ok := r.itemFull(); ok {
		return f.Pillars
	}
	if s, ok := r.itemSummary(); ok {
		return s.Pillars
	}
	return nil
}

// Source returns the recommendation source.
func (r *RecommendationResource) Source() string {
	if f, ok := r.itemFull(); ok {
		return string(f.Source)
	}
	if s, ok := r.itemSummary(); ok {
		return string(s.Source)
	}
	return ""
}

// Type returns the recommendation type.
func (r *RecommendationResource) Type() string {
	if f, ok := r.itemFull(); ok {
		return string(f.Type)
	}
	if s, ok := r.itemSummary(); ok {
		return string(s.Type)
	}
	return ""
}

// resourcesAggregates returns the resources aggregates.
func (r *RecommendationResource) resourcesAggregates() *types.RecommendationResourcesAggregates {
	if f, ok := r.itemFull(); ok {
		return f.ResourcesAggregates
	}
	if s, ok := r.itemSummary(); ok {
		return s.ResourcesAggregates
	}
	return nil
}

// ErrorCount returns the number of resources with errors.
func (r *RecommendationResource) ErrorCount() int64 {
	if agg := r.resourcesAggregates(); agg != nil {
		return appaws.Int64(agg.ErrorCount)
	}
	return 0
}

// WarningCount returns the number of resources with warnings.
func (r *RecommendationResource) WarningCount() int64 {
	if agg := r.resourcesAggregates(); agg != nil {
		return appaws.Int64(agg.WarningCount)
	}
	return 0
}

// OkCount returns the number of resources that are OK.
func (r *RecommendationResource) OkCount() int64 {
	if agg := r.resourcesAggregates(); agg != nil {
		return appaws.Int64(agg.OkCount)
	}
	return 0
}

// AwsServices returns the AWS services this recommendation applies to.
func (r *RecommendationResource) AwsServices() []string {
	if f, ok := r.itemFull(); ok {
		return f.AwsServices
	}
	if s, ok := r.itemSummary(); ok {
		return s.AwsServices
	}
	return nil
}

// CreatedAt returns the creation time as a formatted string.
func (r *RecommendationResource) CreatedAt() string {
	if f, ok := r.itemFull(); ok && f.CreatedAt != nil {
		return f.CreatedAt.Format("2006-01-02 15:04:05")
	}
	if s, ok := r.itemSummary(); ok && s.CreatedAt != nil {
		return s.CreatedAt.Format("2006-01-02 15:04:05")
	}
	return ""
}

// LastUpdatedAt returns the last update time as a formatted string.
func (r *RecommendationResource) LastUpdatedAt() string {
	if f, ok := r.itemFull(); ok && f.LastUpdatedAt != nil {
		return f.LastUpdatedAt.Format("2006-01-02 15:04:05")
	}
	if s, ok := r.itemSummary(); ok && s.LastUpdatedAt != nil {
		return s.LastUpdatedAt.Format("2006-01-02 15:04:05")
	}
	return ""
}

// LifecycleStage returns the lifecycle stage.
func (r *RecommendationResource) LifecycleStage() string {
	if f, ok := r.itemFull(); ok {
		return string(f.LifecycleStage)
	}
	if s, ok := r.itemSummary(); ok {
		return string(s.LifecycleStage)
	}
	return ""
}

// --- Full Recommendation only fields ---

// Description returns the recommendation description (full only).
func (r *RecommendationResource) Description() string {
	if f, ok := r.itemFull(); ok {
		return appaws.Str(f.Description)
	}
	return ""
}

// CreatedBy returns the creator (full only).
func (r *RecommendationResource) CreatedBy() string {
	if f, ok := r.itemFull(); ok {
		return appaws.Str(f.CreatedBy)
	}
	return ""
}

// ResolvedAt returns when the recommendation was resolved (full only).
func (r *RecommendationResource) ResolvedAt() string {
	if f, ok := r.itemFull(); ok && f.ResolvedAt != nil {
		return f.ResolvedAt.Format("2006-01-02 15:04:05")
	}
	return ""
}

// UpdateReason returns the reason for lifecycle stage change (full only).
func (r *RecommendationResource) UpdateReason() string {
	if f, ok := r.itemFull(); ok {
		return appaws.Str(f.UpdateReason)
	}
	return ""
}

// UpdateReasonCode returns the reason code for lifecycle state change (full only).
func (r *RecommendationResource) UpdateReasonCode() string {
	if f, ok := r.itemFull(); ok {
		return string(f.UpdateReasonCode)
	}
	return ""
}

// pillarSpecificAggregates returns the pillar specific aggregates.
func (r *RecommendationResource) pillarSpecificAggregates() *types.RecommendationPillarSpecificAggregates {
	if f, ok := r.itemFull(); ok {
		return f.PillarSpecificAggregates
	}
	if s, ok := r.itemSummary(); ok {
		return s.PillarSpecificAggregates
	}
	return nil
}

// EstimatedMonthlySavings returns the estimated monthly savings.
func (r *RecommendationResource) EstimatedMonthlySavings() float64 {
	if agg := r.pillarSpecificAggregates(); agg != nil && agg.CostOptimizing != nil {
		return appaws.Float64(agg.CostOptimizing.EstimatedMonthlySavings)
	}
	return 0
}

// EstimatedPercentMonthlySavings returns the estimated percent monthly savings.
func (r *RecommendationResource) EstimatedPercentMonthlySavings() float64 {
	if agg := r.pillarSpecificAggregates(); agg != nil && agg.CostOptimizing != nil {
		return appaws.Float64(agg.CostOptimizing.EstimatedPercentMonthlySavings)
	}
	return 0
}
