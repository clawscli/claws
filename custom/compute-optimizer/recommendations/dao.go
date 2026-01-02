package recommendations

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/computeoptimizer"
	"github.com/aws/aws-sdk-go-v2/service/computeoptimizer/types"
	"golang.org/x/sync/errgroup"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
	"github.com/clawscli/claws/internal/log"
)

// RecommendationDAO provides data access for Compute Optimizer Recommendations.
type RecommendationDAO struct {
	dao.BaseDAO
	client *computeoptimizer.Client
}

// NewRecommendationDAO creates a new RecommendationDAO.
func NewRecommendationDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new compute-optimizer/recommendations dao")
	}
	return &RecommendationDAO{
		BaseDAO: dao.NewBaseDAO("compute-optimizer", "recommendations"),
		client:  computeoptimizer.NewFromConfig(cfg),
	}, nil
}

// recommendationFetcher represents a function that fetches recommendations for a specific resource type.
type recommendationFetcher struct {
	name  string
	fetch func(context.Context) ([]dao.Resource, error)
}

// List returns all recommendations from multiple resource types.
// Fetches are executed in parallel for better performance.
// Partial failures are logged but don't prevent returning results from successful APIs.
func (d *RecommendationDAO) List(ctx context.Context) ([]dao.Resource, error) {
	fetchers := []recommendationFetcher{
		{"EC2", d.listEC2Recommendations},
		{"ASG", d.listASGRecommendations},
		{"EBS", d.listEBSRecommendations},
		{"Lambda", d.listLambdaRecommendations},
		{"ECS", d.listECSRecommendations},
	}

	var (
		mu        sync.Mutex
		resources []dao.Resource
		errs      []error
	)

	// Use errgroup for parallel execution. We tolerate partial failures,
	// so goroutines always return nil to avoid early cancellation.
	g, ctx := errgroup.WithContext(ctx)
	for _, f := range fetchers {
		f := f // capture for goroutine
		g.Go(func() error {
			recs, err := f.fetch(ctx)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				log.Warn("failed to list recommendations", "type", f.name, "error", err)
				errs = append(errs, apperrors.Wrapf(err, "%s", f.name))
			} else {
				resources = append(resources, recs...)
			}
			return nil // always return nil to continue fetching other types
		})
	}

	_ = g.Wait() // errors are collected in errs, not returned by goroutines

	// If all APIs failed, return combined error
	if len(errs) == len(fetchers) {
		return nil, errors.Join(errs...)
	}

	// Sort for stable ordering: by type, then by savings (descending)
	sort.Slice(resources, func(i, j int) bool {
		ri := resources[i].(*RecommendationResource)
		rj := resources[j].(*RecommendationResource)
		if ri.resourceType != rj.resourceType {
			return ri.resourceType < rj.resourceType
		}
		return ri.savingsValue > rj.savingsValue
	})

	return resources, nil
}

func (d *RecommendationDAO) listEC2Recommendations(ctx context.Context) ([]dao.Resource, error) {
	recs, err := appaws.Paginate(ctx, func(token *string) ([]types.InstanceRecommendation, *string, error) {
		output, err := d.client.GetEC2InstanceRecommendations(ctx, &computeoptimizer.GetEC2InstanceRecommendationsInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list ec2 recommendations")
		}
		return output.InstanceRecommendations, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(recs))
	for i, rec := range recs {
		resources[i] = NewEC2RecommendationResource(rec)
	}
	return resources, nil
}

func (d *RecommendationDAO) listASGRecommendations(ctx context.Context) ([]dao.Resource, error) {
	recs, err := appaws.Paginate(ctx, func(token *string) ([]types.AutoScalingGroupRecommendation, *string, error) {
		output, err := d.client.GetAutoScalingGroupRecommendations(ctx, &computeoptimizer.GetAutoScalingGroupRecommendationsInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list asg recommendations")
		}
		return output.AutoScalingGroupRecommendations, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(recs))
	for i, rec := range recs {
		resources[i] = NewASGRecommendationResource(rec)
	}
	return resources, nil
}

func (d *RecommendationDAO) listEBSRecommendations(ctx context.Context) ([]dao.Resource, error) {
	recs, err := appaws.Paginate(ctx, func(token *string) ([]types.VolumeRecommendation, *string, error) {
		output, err := d.client.GetEBSVolumeRecommendations(ctx, &computeoptimizer.GetEBSVolumeRecommendationsInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list ebs recommendations")
		}
		return output.VolumeRecommendations, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(recs))
	for i, rec := range recs {
		resources[i] = NewEBSRecommendationResource(rec)
	}
	return resources, nil
}

func (d *RecommendationDAO) listLambdaRecommendations(ctx context.Context) ([]dao.Resource, error) {
	recs, err := appaws.Paginate(ctx, func(token *string) ([]types.LambdaFunctionRecommendation, *string, error) {
		output, err := d.client.GetLambdaFunctionRecommendations(ctx, &computeoptimizer.GetLambdaFunctionRecommendationsInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list lambda recommendations")
		}
		return output.LambdaFunctionRecommendations, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(recs))
	for i, rec := range recs {
		resources[i] = NewLambdaRecommendationResource(rec)
	}
	return resources, nil
}

func (d *RecommendationDAO) listECSRecommendations(ctx context.Context) ([]dao.Resource, error) {
	recs, err := appaws.Paginate(ctx, func(token *string) ([]types.ECSServiceRecommendation, *string, error) {
		output, err := d.client.GetECSServiceRecommendations(ctx, &computeoptimizer.GetECSServiceRecommendationsInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list ecs recommendations")
		}
		return output.EcsServiceRecommendations, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(recs))
	for i, rec := range recs {
		resources[i] = NewECSRecommendationResource(rec)
	}
	return resources, nil
}

// Get returns a specific recommendation by ID.
func (d *RecommendationDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	resources, err := d.List(ctx)
	if err != nil {
		return nil, err
	}

	for _, r := range resources {
		if r.GetID() == id {
			return r, nil
		}
	}
	return nil, fmt.Errorf("recommendation not found: %s", id)
}

// Delete is not supported.
func (d *RecommendationDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported for compute optimizer recommendations")
}

// Supports returns true only for List operation.
// Get() is implemented via List() scan, so we disable auto-refresh in DetailView.
func (d *RecommendationDAO) Supports(op dao.Operation) bool {
	return op == dao.OpList
}

// RecommendationResource is a unified wrapper for all recommendation types.
type RecommendationResource struct {
	dao.BaseResource
	resourceType    string
	finding         string
	currentConfig   string
	savingsPercent  float64
	savingsValue    float64
	savingsCurrency string
	performanceRisk string
}

// extractSavings extracts savings info from SavingsOpportunity.
func extractSavings(opportunity *types.SavingsOpportunity) (pct, val float64, currency string) {
	if opportunity == nil {
		return 0, 0, ""
	}
	pct = opportunity.SavingsOpportunityPercentage
	if opportunity.EstimatedMonthlySavings != nil {
		val = opportunity.EstimatedMonthlySavings.Value
		currency = string(opportunity.EstimatedMonthlySavings.Currency)
	}
	return
}

// ResourceType returns the resource type (EC2, ASG, EBS, Lambda, ECS).
func (r *RecommendationResource) ResourceType() string {
	return r.resourceType
}

// Finding returns the finding classification.
func (r *RecommendationResource) Finding() string {
	return r.finding
}

// CurrentConfig returns a summary of current configuration.
func (r *RecommendationResource) CurrentConfig() string {
	return r.currentConfig
}

// SavingsPercent returns the savings opportunity percentage.
func (r *RecommendationResource) SavingsPercent() float64 {
	return r.savingsPercent
}

// SavingsValue returns the estimated monthly savings.
func (r *RecommendationResource) SavingsValue() float64 {
	return r.savingsValue
}

// PerformanceRisk returns the current performance risk level.
func (r *RecommendationResource) PerformanceRisk() string {
	return r.performanceRisk
}

// SavingsCurrency returns the currency for savings values.
func (r *RecommendationResource) SavingsCurrency() string {
	return r.savingsCurrency
}

// NewEC2RecommendationResource creates a resource from EC2 recommendation.
func NewEC2RecommendationResource(rec types.InstanceRecommendation) *RecommendationResource {
	arn := appaws.Str(rec.InstanceArn)
	instanceType := appaws.Str(rec.CurrentInstanceType)

	var savingsPercent, savingsValue float64
	var savingsCurrency string
	if len(rec.RecommendationOptions) > 0 {
		savingsPercent, savingsValue, savingsCurrency = extractSavings(rec.RecommendationOptions[0].SavingsOpportunity)
	}

	return &RecommendationResource{
		BaseResource: dao.BaseResource{
			ID:   arn,
			Name: appaws.ExtractResourceName(arn),
			ARN:  arn,
			Tags: appaws.TagsToMap(rec.Tags),
			Data: rec,
		},
		resourceType:    "EC2",
		finding:         string(rec.Finding),
		currentConfig:   instanceType,
		savingsPercent:  savingsPercent,
		savingsValue:    savingsValue,
		savingsCurrency: savingsCurrency,
		performanceRisk: string(rec.CurrentPerformanceRisk),
	}
}

// NewASGRecommendationResource creates a resource from ASG recommendation.
func NewASGRecommendationResource(rec types.AutoScalingGroupRecommendation) *RecommendationResource {
	arn := appaws.Str(rec.AutoScalingGroupArn)
	name := appaws.Str(rec.AutoScalingGroupName)

	var currentConfig string
	if rec.CurrentConfiguration != nil {
		currentConfig = appaws.Str(rec.CurrentConfiguration.InstanceType)
	}

	var savingsPercent, savingsValue float64
	var savingsCurrency string
	if len(rec.RecommendationOptions) > 0 {
		savingsPercent, savingsValue, savingsCurrency = extractSavings(rec.RecommendationOptions[0].SavingsOpportunity)
	}

	return &RecommendationResource{
		BaseResource: dao.BaseResource{
			ID:   arn,
			Name: name,
			ARN:  arn,
			Data: rec,
		},
		resourceType:    "ASG",
		finding:         string(rec.Finding),
		currentConfig:   currentConfig,
		savingsPercent:  savingsPercent,
		savingsValue:    savingsValue,
		savingsCurrency: savingsCurrency,
		performanceRisk: string(rec.CurrentPerformanceRisk),
	}
}

// NewEBSRecommendationResource creates a resource from EBS recommendation.
func NewEBSRecommendationResource(rec types.VolumeRecommendation) *RecommendationResource {
	arn := appaws.Str(rec.VolumeArn)

	var currentConfig string
	if rec.CurrentConfiguration != nil {
		currentConfig = fmt.Sprintf("%s/%dGB", appaws.Str(rec.CurrentConfiguration.VolumeType), rec.CurrentConfiguration.VolumeSize)
	}

	var savingsPercent, savingsValue float64
	var savingsCurrency string
	if len(rec.VolumeRecommendationOptions) > 0 {
		savingsPercent, savingsValue, savingsCurrency = extractSavings(rec.VolumeRecommendationOptions[0].SavingsOpportunity)
	}

	return &RecommendationResource{
		BaseResource: dao.BaseResource{
			ID:   arn,
			Name: appaws.ExtractResourceName(arn),
			ARN:  arn,
			Data: rec,
		},
		resourceType:    "EBS",
		finding:         string(rec.Finding),
		currentConfig:   currentConfig,
		savingsPercent:  savingsPercent,
		savingsValue:    savingsValue,
		savingsCurrency: savingsCurrency,
		performanceRisk: string(rec.CurrentPerformanceRisk),
	}
}

// NewLambdaRecommendationResource creates a resource from Lambda recommendation.
func NewLambdaRecommendationResource(rec types.LambdaFunctionRecommendation) *RecommendationResource {
	arn := appaws.Str(rec.FunctionArn)

	currentConfig := fmt.Sprintf("%dMB", rec.CurrentMemorySize)

	var savingsPercent, savingsValue float64
	var savingsCurrency string
	if len(rec.MemorySizeRecommendationOptions) > 0 {
		savingsPercent, savingsValue, savingsCurrency = extractSavings(rec.MemorySizeRecommendationOptions[0].SavingsOpportunity)
	}

	return &RecommendationResource{
		BaseResource: dao.BaseResource{
			ID:   arn,
			Name: appaws.ExtractResourceName(arn),
			ARN:  arn,
			Data: rec,
		},
		resourceType:    "LAMBDA",
		finding:         string(rec.Finding),
		currentConfig:   currentConfig,
		savingsPercent:  savingsPercent,
		savingsValue:    savingsValue,
		savingsCurrency: savingsCurrency,
		performanceRisk: string(rec.CurrentPerformanceRisk),
	}
}

// NewECSRecommendationResource creates a resource from ECS recommendation.
func NewECSRecommendationResource(rec types.ECSServiceRecommendation) *RecommendationResource {
	arn := appaws.Str(rec.ServiceArn)

	var currentConfig string
	if rec.CurrentServiceConfiguration != nil {
		cpu := rec.CurrentServiceConfiguration.Cpu
		mem := rec.CurrentServiceConfiguration.Memory
		currentConfig = fmt.Sprintf("CPU:%d/Mem:%d", cpu, mem)
	}

	var savingsPercent, savingsValue float64
	var savingsCurrency string
	if len(rec.ServiceRecommendationOptions) > 0 {
		savingsPercent, savingsValue, savingsCurrency = extractSavings(rec.ServiceRecommendationOptions[0].SavingsOpportunity)
	}

	return &RecommendationResource{
		BaseResource: dao.BaseResource{
			ID:   arn,
			Name: appaws.ExtractResourceName(arn),
			ARN:  arn,
			Tags: appaws.TagsToMap(rec.Tags),
			Data: rec,
		},
		resourceType:    "ECS",
		finding:         string(rec.Finding),
		currentConfig:   currentConfig,
		savingsPercent:  savingsPercent,
		savingsValue:    savingsValue,
		savingsCurrency: savingsCurrency,
		performanceRisk: string(rec.CurrentPerformanceRisk),
	}
}
