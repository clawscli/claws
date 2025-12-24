package recommendations

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/computeoptimizer"
	"github.com/aws/aws-sdk-go-v2/service/computeoptimizer/types"
	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
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
		return nil, fmt.Errorf("new computeoptimizer/recommendations dao: %w", err)
	}
	return &RecommendationDAO{
		BaseDAO: dao.NewBaseDAO("computeoptimizer", "recommendations"),
		client:  computeoptimizer.NewFromConfig(cfg),
	}, nil
}

// numRecommendationAPIs is the number of recommendation APIs we call.
const numRecommendationAPIs = 5

// List returns all recommendations from multiple resource types.
// Partial failures are logged but don't prevent returning results from successful APIs.
func (d *RecommendationDAO) List(ctx context.Context) ([]dao.Resource, error) {
	var resources []dao.Resource
	var errs []error

	// Fetch EC2 recommendations
	ec2Recs, err := d.listEC2Recommendations(ctx)
	if err != nil {
		log.Warn("failed to list EC2 recommendations", "error", err)
		errs = append(errs, fmt.Errorf("ec2: %w", err))
	} else {
		resources = append(resources, ec2Recs...)
	}

	// Fetch ASG recommendations
	asgRecs, err := d.listASGRecommendations(ctx)
	if err != nil {
		log.Warn("failed to list ASG recommendations", "error", err)
		errs = append(errs, fmt.Errorf("asg: %w", err))
	} else {
		resources = append(resources, asgRecs...)
	}

	// Fetch EBS recommendations
	ebsRecs, err := d.listEBSRecommendations(ctx)
	if err != nil {
		log.Warn("failed to list EBS recommendations", "error", err)
		errs = append(errs, fmt.Errorf("ebs: %w", err))
	} else {
		resources = append(resources, ebsRecs...)
	}

	// Fetch Lambda recommendations
	lambdaRecs, err := d.listLambdaRecommendations(ctx)
	if err != nil {
		log.Warn("failed to list Lambda recommendations", "error", err)
		errs = append(errs, fmt.Errorf("lambda: %w", err))
	} else {
		resources = append(resources, lambdaRecs...)
	}

	// Fetch ECS recommendations
	ecsRecs, err := d.listECSRecommendations(ctx)
	if err != nil {
		log.Warn("failed to list ECS recommendations", "error", err)
		errs = append(errs, fmt.Errorf("ecs: %w", err))
	} else {
		resources = append(resources, ecsRecs...)
	}

	// If all APIs failed, return combined error
	if len(errs) == numRecommendationAPIs {
		return nil, errors.Join(errs...)
	}

	return resources, nil
}

func (d *RecommendationDAO) listEC2Recommendations(ctx context.Context) ([]dao.Resource, error) {
	recs, err := appaws.Paginate(ctx, func(token *string) ([]types.InstanceRecommendation, *string, error) {
		output, err := d.client.GetEC2InstanceRecommendations(ctx, &computeoptimizer.GetEC2InstanceRecommendationsInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("list ec2 recommendations: %w", err)
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
			return nil, nil, fmt.Errorf("list asg recommendations: %w", err)
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
			return nil, nil, fmt.Errorf("list ebs recommendations: %w", err)
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
			return nil, nil, fmt.Errorf("list lambda recommendations: %w", err)
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
			return nil, nil, fmt.Errorf("list ecs recommendations: %w", err)
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

// SavingsCurrency returns the currency for savings.
func (r *RecommendationResource) SavingsCurrency() string {
	if r.savingsCurrency == "" {
		return "USD"
	}
	return r.savingsCurrency
}

// PerformanceRisk returns the current performance risk level.
func (r *RecommendationResource) PerformanceRisk() string {
	return r.performanceRisk
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
			Data: rec,
		},
		resourceType:    "Lambda",
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
