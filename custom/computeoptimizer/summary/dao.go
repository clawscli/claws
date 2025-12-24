package summary

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/computeoptimizer"
	"github.com/aws/aws-sdk-go-v2/service/computeoptimizer/types"
	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
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
		return nil, fmt.Errorf("new computeoptimizer/summary dao: %w", err)
	}
	return &SummaryDAO{
		BaseDAO: dao.NewBaseDAO("compute-optimizer", "summary"),
		client:  computeoptimizer.NewFromConfig(cfg),
	}, nil
}

// List returns recommendation summaries for all resource types.
func (d *SummaryDAO) List(ctx context.Context) ([]dao.Resource, error) {
	var resources []dao.Resource
	var nextToken *string

	for {
		input := &computeoptimizer.GetRecommendationSummariesInput{
			NextToken: nextToken,
		}

		output, err := d.client.GetRecommendationSummaries(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("get recommendation summaries: %w", err)
		}

		for _, summary := range output.RecommendationSummaries {
			resources = append(resources, NewSummaryResource(summary))
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
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

// SummaryResource wraps a Compute Optimizer Recommendation Summary.
type SummaryResource struct {
	dao.BaseResource
	Item types.RecommendationSummary
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
		Item: summary,
	}
}

// ResourceType returns the resource type.
func (r *SummaryResource) ResourceType() string {
	return string(r.Item.RecommendationResourceType)
}

// AccountId returns the AWS account ID.
func (r *SummaryResource) AccountId() string {
	return appaws.Str(r.Item.AccountId)
}

// Summaries returns the summary findings.
func (r *SummaryResource) Summaries() []types.Summary {
	return r.Item.Summaries
}

// SummaryString returns a formatted summary of findings.
func (r *SummaryResource) SummaryString() string {
	result := ""
	for _, s := range r.Item.Summaries {
		if s.Value > 0 {
			if result != "" {
				result += ", "
			}
			result += fmt.Sprintf("%s:%.0f", string(s.Name), s.Value)
		}
	}
	return result
}

// SavingsOpportunityPercentage returns the savings opportunity percentage.
func (r *SummaryResource) SavingsOpportunityPercentage() float64 {
	if r.Item.SavingsOpportunity != nil {
		return r.Item.SavingsOpportunity.SavingsOpportunityPercentage
	}
	return 0
}

// EstimatedMonthlySavings returns the estimated monthly savings value.
func (r *SummaryResource) EstimatedMonthlySavings() float64 {
	if r.Item.SavingsOpportunity != nil && r.Item.SavingsOpportunity.EstimatedMonthlySavings != nil {
		return r.Item.SavingsOpportunity.EstimatedMonthlySavings.Value
	}
	return 0
}

// EstimatedMonthlySavingsCurrency returns the currency for savings.
func (r *SummaryResource) EstimatedMonthlySavingsCurrency() string {
	if r.Item.SavingsOpportunity != nil && r.Item.SavingsOpportunity.EstimatedMonthlySavings != nil {
		return string(r.Item.SavingsOpportunity.EstimatedMonthlySavings.Currency)
	}
	return "USD"
}

// TotalResources returns the total count of resources.
func (r *SummaryResource) TotalResources() float64 {
	var total float64
	for _, s := range r.Item.Summaries {
		total += s.Value
	}
	return total
}

// OptimizedCount returns count of optimized resources.
func (r *SummaryResource) OptimizedCount() float64 {
	for _, s := range r.Item.Summaries {
		if s.Name == types.FindingOptimized {
			return s.Value
		}
	}
	return 0
}

// NotOptimizedCount returns count of not optimized resources.
func (r *SummaryResource) NotOptimizedCount() float64 {
	var count float64
	for _, s := range r.Item.Summaries {
		if s.Name == types.FindingUnderProvisioned ||
			s.Name == types.FindingOverProvisioned ||
			s.Name == types.FindingNotOptimized {
			count += s.Value
		}
	}
	return count
}
