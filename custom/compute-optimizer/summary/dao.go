package summary

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/computeoptimizer"
	"github.com/aws/aws-sdk-go-v2/service/computeoptimizer/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// SummaryDAO provides data access for Compute Optimizer Recommendation Summaries.
type SummaryDAO struct {
	dao.BaseDAO
	client *computeoptimizer.Client
}

// NewSummaryDAO creates a new SummaryDAO.
func NewSummaryDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new compute-optimizer/summary dao")
	}
	return &SummaryDAO{
		BaseDAO: dao.NewBaseDAO("compute-optimizer", "summary"),
		client:  computeoptimizer.NewFromConfig(cfg),
	}, nil
}

// List returns recommendation summaries for all resource types.
func (d *SummaryDAO) List(ctx context.Context) ([]dao.Resource, error) {
	summaries, err := appaws.Paginate(ctx, func(token *string) ([]types.RecommendationSummary, *string, error) {
		output, err := d.client.GetRecommendationSummaries(ctx, &computeoptimizer.GetRecommendationSummariesInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list recommendation summaries")
		}
		return output.RecommendationSummaries, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(summaries))
	for i, summary := range summaries {
		resources[i] = NewSummaryResource(summary)
	}
	return resources, nil
}

// Get returns a specific summary by resource type.
func (d *SummaryDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	resources, err := d.List(ctx)
	if err != nil {
		return nil, err
	}

	for _, r := range resources {
		if r.GetID() == id {
			return r, nil
		}
	}
	return nil, fmt.Errorf("summary not found: %s", id)
}

// Delete is not supported.
func (d *SummaryDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported for compute optimizer summaries")
}

// Supports returns true only for List operation.
// Get() is implemented via List() scan, so we disable auto-refresh in DetailView.
func (d *SummaryDAO) Supports(op dao.Operation) bool {
	return op == dao.OpList
}

// SummaryResource wraps a Compute Optimizer Recommendation Summary.
type SummaryResource struct {
	dao.BaseResource
}

// NewSummaryResource creates a new SummaryResource.
func NewSummaryResource(summary types.RecommendationSummary) *SummaryResource {
	resourceType := string(summary.RecommendationResourceType)

	return &SummaryResource{
		BaseResource: dao.BaseResource{
			ID:   resourceType,
			Name: resourceType,
			Data: summary,
		},
	}
}

// item returns the underlying SDK type.
func (r *SummaryResource) item() types.RecommendationSummary {
	return r.Data.(types.RecommendationSummary)
}

// ResourceType returns the resource type.
func (r *SummaryResource) ResourceType() string {
	return string(r.item().RecommendationResourceType)
}

// AccountId returns the AWS account ID.
func (r *SummaryResource) AccountId() string {
	return appaws.Str(r.item().AccountId)
}

// Summaries returns the summary findings.
func (r *SummaryResource) Summaries() []types.Summary {
	return r.item().Summaries
}

// SavingsOpportunityPercentage returns the savings opportunity percentage.
func (r *SummaryResource) SavingsOpportunityPercentage() float64 {
	if r.item().SavingsOpportunity != nil {
		return r.item().SavingsOpportunity.SavingsOpportunityPercentage
	}
	return 0
}

// EstimatedMonthlySavings returns the estimated monthly savings value.
func (r *SummaryResource) EstimatedMonthlySavings() float64 {
	if r.item().SavingsOpportunity != nil && r.item().SavingsOpportunity.EstimatedMonthlySavings != nil {
		return r.item().SavingsOpportunity.EstimatedMonthlySavings.Value
	}
	return 0
}

// SavingsCurrency returns the currency for savings values.
func (r *SummaryResource) SavingsCurrency() string {
	if r.item().SavingsOpportunity != nil && r.item().SavingsOpportunity.EstimatedMonthlySavings != nil {
		return string(r.item().SavingsOpportunity.EstimatedMonthlySavings.Currency)
	}
	return ""
}

// TotalResources returns the total count of resources.
func (r *SummaryResource) TotalResources() float64 {
	var total float64
	for _, s := range r.item().Summaries {
		total += s.Value
	}
	return total
}

// OptimizedCount returns count of optimized resources.
func (r *SummaryResource) OptimizedCount() float64 {
	for _, s := range r.item().Summaries {
		if s.Name == types.FindingOptimized {
			return s.Value
		}
	}
	return 0
}

// NotOptimizedCount returns count of not optimized resources.
func (r *SummaryResource) NotOptimizedCount() float64 {
	var count float64
	for _, s := range r.item().Summaries {
		if s.Name == types.FindingUnderProvisioned ||
			s.Name == types.FindingOverProvisioned ||
			s.Name == types.FindingNotOptimized {
			count += s.Value
		}
	}
	return count
}

// PerformanceRiskRatings returns the performance risk ratings.
func (r *SummaryResource) PerformanceRiskRatings() *types.CurrentPerformanceRiskRatings {
	return r.item().CurrentPerformanceRiskRatings
}

// IdleSummaries returns the idle resource summaries.
func (r *SummaryResource) IdleSummaries() []types.IdleSummary {
	return r.item().IdleSummaries
}

// IdleSavingsOpportunity returns the idle savings opportunity.
func (r *SummaryResource) IdleSavingsOpportunity() *types.SavingsOpportunity {
	return r.item().IdleSavingsOpportunity
}

// InferredWorkloadSavings returns the inferred workload savings.
func (r *SummaryResource) InferredWorkloadSavings() []types.InferredWorkloadSaving {
	return r.item().InferredWorkloadSavings
}
