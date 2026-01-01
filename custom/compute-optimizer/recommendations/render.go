package recommendations

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/computeoptimizer/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// RecommendationRenderer renders Compute Optimizer Recommendations data.
type RecommendationRenderer struct {
	render.BaseRenderer
}

// NewRecommendationRenderer creates a new RecommendationRenderer.
func NewRecommendationRenderer() render.Renderer {
	return &RecommendationRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "compute-optimizer",
			Resource: "recommendations",
			Cols: []render.Column{
				{Name: "TYPE", Width: 8, Getter: getType},
				{Name: "NAME", Width: 30, Getter: getName},
				{Name: "FINDING", Width: 16, Getter: getFinding},
				{Name: "CURRENT", Width: 16, Getter: getCurrent},
				{Name: "SAVINGS %", Width: 10, Getter: getSavingsPct},
				{Name: "EST. SAVINGS", Width: 12, Getter: getEstSavings},
			},
		},
	}
}

func getType(r dao.Resource) string {
	rec, ok := r.(*RecommendationResource)
	if !ok {
		return ""
	}
	return rec.ResourceType()
}

func getName(r dao.Resource) string {
	rec, ok := r.(*RecommendationResource)
	if !ok {
		return ""
	}
	return rec.GetName()
}

func getFinding(r dao.Resource) string {
	rec, ok := r.(*RecommendationResource)
	if !ok {
		return ""
	}
	return rec.Finding()
}

func getCurrent(r dao.Resource) string {
	rec, ok := r.(*RecommendationResource)
	if !ok {
		return ""
	}
	return rec.CurrentConfig()
}

func getSavingsPct(r dao.Resource) string {
	rec, ok := r.(*RecommendationResource)
	if !ok {
		return ""
	}
	pct := rec.SavingsPercent()
	if pct > 0 {
		return fmt.Sprintf("%.1f%%", pct)
	}
	return "-"
}

func getEstSavings(r dao.Resource) string {
	rec, ok := r.(*RecommendationResource)
	if !ok {
		return ""
	}
	savings := rec.SavingsValue()
	if savings > 0 {
		return appaws.FormatMoney(savings, rec.SavingsCurrency())
	}
	return "-"
}

// RenderDetail renders the detail view for a recommendation.
func (r *RecommendationRenderer) RenderDetail(resource dao.Resource) string {
	rec, ok := resource.(*RecommendationResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Compute Optimizer Recommendation", rec.GetName())

	// Basic Info
	d.Section("Resource Information")
	d.Field("Resource Type", rec.ResourceType())
	d.Field("Resource ARN", rec.GetID())
	d.Field("Name", rec.GetName())

	// Finding
	d.Section("Finding")
	d.Field("Classification", rec.Finding())
	d.Field("Performance Risk", rec.PerformanceRisk())

	// Current Configuration
	d.Section("Current Configuration")
	d.Field("Configuration", rec.CurrentConfig())

	// Savings Opportunity
	if rec.SavingsPercent() > 0 || rec.SavingsValue() > 0 {
		d.Section("Savings Opportunity")
		d.Field("Savings Percentage", fmt.Sprintf("%.2f%%", rec.SavingsPercent()))
		d.Field("Estimated Monthly Savings", appaws.FormatMoney(rec.SavingsValue(), rec.SavingsCurrency()))
	}

	// Type-specific details from original SDK data
	switch data := rec.Data.(type) {
	case types.InstanceRecommendation:
		renderEC2Detail(d, data)
	case types.AutoScalingGroupRecommendation:
		renderASGDetail(d, data)
	case types.VolumeRecommendation:
		renderEBSDetail(d, data)
	case types.LambdaFunctionRecommendation:
		renderLambdaDetail(d, data)
	case types.ECSServiceRecommendation:
		renderECSDetail(d, data)
	}

	return d.String()
}

// renderEC2Detail adds EC2-specific recommendation details to the detail view.
func renderEC2Detail(d *render.DetailBuilder, rec types.InstanceRecommendation) {
	// Finding Reason Codes
	if len(rec.FindingReasonCodes) > 0 {
		d.Section("Finding Reasons")
		for _, code := range rec.FindingReasonCodes {
			d.Field("", string(code))
		}
	}

	// Utilization Metrics
	if len(rec.UtilizationMetrics) > 0 {
		d.Section("Utilization Metrics")
		for _, m := range rec.UtilizationMetrics {
			d.Field(string(m.Name), fmt.Sprintf("%.2f (statistic: %s)", m.Value, m.Statistic))
		}
	}

	// Recommendation Options
	if len(rec.RecommendationOptions) > 0 {
		d.Section("Recommendation Options")
		for i, opt := range rec.RecommendationOptions {
			prefix := fmt.Sprintf("Option %d", i+1)
			d.Field(prefix+" Instance Type", appaws.Str(opt.InstanceType))
			if opt.SavingsOpportunity != nil {
				d.Field(prefix+" Savings %", fmt.Sprintf("%.2f%%", opt.SavingsOpportunity.SavingsOpportunityPercentage))
				if opt.SavingsOpportunity.EstimatedMonthlySavings != nil {
					d.Field(prefix+" Est. Savings", appaws.FormatMoney(opt.SavingsOpportunity.EstimatedMonthlySavings.Value, string(opt.SavingsOpportunity.EstimatedMonthlySavings.Currency)))
				}
			}
			d.Field(prefix+" Performance Risk", fmt.Sprintf("%.0f", opt.PerformanceRisk))
			d.Field(prefix+" Migration Effort", string(opt.MigrationEffort))
		}
	}

	// Timestamps
	d.Section("Metadata")
	d.Field("Look Back Period (Days)", fmt.Sprintf("%.0f", rec.LookBackPeriodInDays))
	if rec.LastRefreshTimestamp != nil {
		d.Field("Last Refresh", rec.LastRefreshTimestamp.Format("2006-01-02 15:04:05"))
	}

	// Tags
	d.Tags(appaws.TagsToMap(rec.Tags))
}

// renderASGDetail adds Auto Scaling Group-specific recommendation details.
func renderASGDetail(d *render.DetailBuilder, rec types.AutoScalingGroupRecommendation) {
	// Current Configuration
	if rec.CurrentConfiguration != nil {
		d.Section("Current ASG Configuration")
		d.Field("Instance Type", appaws.Str(rec.CurrentConfiguration.InstanceType))
		d.Field("Desired Capacity", fmt.Sprintf("%d", rec.CurrentConfiguration.DesiredCapacity))
		d.Field("Min Size", fmt.Sprintf("%d", rec.CurrentConfiguration.MinSize))
		d.Field("Max Size", fmt.Sprintf("%d", rec.CurrentConfiguration.MaxSize))
	}

	// Recommendation Options
	if len(rec.RecommendationOptions) > 0 {
		d.Section("Recommendation Options")
		for i, opt := range rec.RecommendationOptions {
			prefix := fmt.Sprintf("Option %d", i+1)
			if opt.Configuration != nil {
				d.Field(prefix+" Instance Type", appaws.Str(opt.Configuration.InstanceType))
			}
			if opt.SavingsOpportunity != nil {
				d.Field(prefix+" Savings %", fmt.Sprintf("%.2f%%", opt.SavingsOpportunity.SavingsOpportunityPercentage))
				if opt.SavingsOpportunity.EstimatedMonthlySavings != nil {
					d.Field(prefix+" Est. Savings", appaws.FormatMoney(opt.SavingsOpportunity.EstimatedMonthlySavings.Value, string(opt.SavingsOpportunity.EstimatedMonthlySavings.Currency)))
				}
			}
			d.Field(prefix+" Performance Risk", fmt.Sprintf("%.0f", opt.PerformanceRisk))
			d.Field(prefix+" Migration Effort", string(opt.MigrationEffort))
		}
	}

	// Utilization Metrics
	if len(rec.UtilizationMetrics) > 0 {
		d.Section("Utilization Metrics")
		for _, m := range rec.UtilizationMetrics {
			d.Field(string(m.Name), fmt.Sprintf("%.2f (statistic: %s)", m.Value, m.Statistic))
		}
	}

	// Inferred Workload Types
	if len(rec.InferredWorkloadTypes) > 0 {
		workloads := make([]string, len(rec.InferredWorkloadTypes))
		for i, w := range rec.InferredWorkloadTypes {
			workloads[i] = string(w)
		}
		d.Section("Inferred Workloads")
		d.Field("Types", strings.Join(workloads, ", "))
	}

	// Timestamps
	d.Section("Metadata")
	d.Field("Look Back Period (Days)", fmt.Sprintf("%.0f", rec.LookBackPeriodInDays))
	if rec.LastRefreshTimestamp != nil {
		d.Field("Last Refresh", rec.LastRefreshTimestamp.Format("2006-01-02 15:04:05"))
	}
}

// renderEBSDetail adds EBS volume-specific recommendation details.
func renderEBSDetail(d *render.DetailBuilder, rec types.VolumeRecommendation) {
	// Current Configuration
	if rec.CurrentConfiguration != nil {
		d.Section("Current Volume Configuration")
		d.Field("Volume Type", appaws.Str(rec.CurrentConfiguration.VolumeType))
		d.Field("Volume Size", fmt.Sprintf("%d GB", rec.CurrentConfiguration.VolumeSize))
		d.Field("Baseline IOPS", fmt.Sprintf("%d", rec.CurrentConfiguration.VolumeBaselineIOPS))
		d.Field("Baseline Throughput", fmt.Sprintf("%d MB/s", rec.CurrentConfiguration.VolumeBaselineThroughput))
	}

	// Recommendation Options
	if len(rec.VolumeRecommendationOptions) > 0 {
		d.Section("Recommendation Options")
		for i, opt := range rec.VolumeRecommendationOptions {
			prefix := fmt.Sprintf("Option %d", i+1)
			if opt.Configuration != nil {
				d.Field(prefix+" Volume Type", appaws.Str(opt.Configuration.VolumeType))
				d.Field(prefix+" Volume Size", fmt.Sprintf("%d GB", opt.Configuration.VolumeSize))
				d.Field(prefix+" IOPS", fmt.Sprintf("%d", opt.Configuration.VolumeBaselineIOPS))
			}
			if opt.SavingsOpportunity != nil {
				d.Field(prefix+" Savings %", fmt.Sprintf("%.2f%%", opt.SavingsOpportunity.SavingsOpportunityPercentage))
				if opt.SavingsOpportunity.EstimatedMonthlySavings != nil {
					d.Field(prefix+" Est. Savings", appaws.FormatMoney(opt.SavingsOpportunity.EstimatedMonthlySavings.Value, string(opt.SavingsOpportunity.EstimatedMonthlySavings.Currency)))
				}
			}
			d.Field(prefix+" Performance Risk", fmt.Sprintf("%.0f", opt.PerformanceRisk))
		}
	}

	// Utilization Metrics
	if len(rec.UtilizationMetrics) > 0 {
		d.Section("Utilization Metrics")
		for _, m := range rec.UtilizationMetrics {
			d.Field(string(m.Name), fmt.Sprintf("%.2f (statistic: %s)", m.Value, m.Statistic))
		}
	}

	// Timestamps
	d.Section("Metadata")
	d.Field("Look Back Period (Days)", fmt.Sprintf("%.0f", rec.LookBackPeriodInDays))
	if rec.LastRefreshTimestamp != nil {
		d.Field("Last Refresh", rec.LastRefreshTimestamp.Format("2006-01-02 15:04:05"))
	}
}

// renderLambdaDetail adds Lambda function-specific recommendation details.
func renderLambdaDetail(d *render.DetailBuilder, rec types.LambdaFunctionRecommendation) {
	// Current Configuration
	d.Section("Current Lambda Configuration")
	d.Field("Memory Size", fmt.Sprintf("%d MB", rec.CurrentMemorySize))
	d.Field("Number of Invocations", fmt.Sprintf("%d", rec.NumberOfInvocations))

	// Finding Reason Codes
	if len(rec.FindingReasonCodes) > 0 {
		d.Section("Finding Reasons")
		for _, code := range rec.FindingReasonCodes {
			d.Field("", string(code))
		}
	}

	// Utilization Metrics
	if len(rec.UtilizationMetrics) > 0 {
		d.Section("Utilization Metrics")
		for _, m := range rec.UtilizationMetrics {
			d.Field(string(m.Name), fmt.Sprintf("%.2f (statistic: %s)", m.Value, m.Statistic))
		}
	}

	// Recommendation Options
	if len(rec.MemorySizeRecommendationOptions) > 0 {
		d.Section("Memory Size Options")
		for i, opt := range rec.MemorySizeRecommendationOptions {
			prefix := fmt.Sprintf("Option %d", i+1)
			d.Field(prefix+" Memory Size", fmt.Sprintf("%d MB", opt.MemorySize))
			if opt.SavingsOpportunity != nil {
				d.Field(prefix+" Savings %", fmt.Sprintf("%.2f%%", opt.SavingsOpportunity.SavingsOpportunityPercentage))
				if opt.SavingsOpportunity.EstimatedMonthlySavings != nil {
					d.Field(prefix+" Est. Savings", appaws.FormatMoney(opt.SavingsOpportunity.EstimatedMonthlySavings.Value, string(opt.SavingsOpportunity.EstimatedMonthlySavings.Currency)))
				}
			}
		}
	}

	// Timestamps
	d.Section("Metadata")
	d.Field("Look Back Period (Days)", fmt.Sprintf("%.0f", rec.LookbackPeriodInDays))
	if rec.LastRefreshTimestamp != nil {
		d.Field("Last Refresh", rec.LastRefreshTimestamp.Format("2006-01-02 15:04:05"))
	}
}

// renderECSDetail adds ECS service-specific recommendation details.
func renderECSDetail(d *render.DetailBuilder, rec types.ECSServiceRecommendation) {
	// Current Configuration
	if rec.CurrentServiceConfiguration != nil {
		d.Section("Current ECS Configuration")
		d.Field("CPU", fmt.Sprintf("%d", rec.CurrentServiceConfiguration.Cpu))
		d.Field("Memory", fmt.Sprintf("%d", rec.CurrentServiceConfiguration.Memory))
		d.Field("Task Definition ARN", appaws.Str(rec.CurrentServiceConfiguration.TaskDefinitionArn))
		d.Field("Container Configs", fmt.Sprintf("%d", len(rec.CurrentServiceConfiguration.ContainerConfigurations)))
	}

	// Finding Reason Codes
	if len(rec.FindingReasonCodes) > 0 {
		d.Section("Finding Reasons")
		for _, code := range rec.FindingReasonCodes {
			d.Field("", string(code))
		}
	}

	// Utilization Metrics
	if len(rec.UtilizationMetrics) > 0 {
		d.Section("Utilization Metrics")
		for _, m := range rec.UtilizationMetrics {
			d.Field(string(m.Name), fmt.Sprintf("%.2f (statistic: %s)", m.Value, m.Statistic))
		}
	}

	// Recommendation Options
	if len(rec.ServiceRecommendationOptions) > 0 {
		d.Section("Service Options")
		for i, opt := range rec.ServiceRecommendationOptions {
			prefix := fmt.Sprintf("Option %d", i+1)
			d.Field(prefix+" CPU", fmt.Sprintf("%d", opt.Cpu))
			d.Field(prefix+" Memory", fmt.Sprintf("%d", opt.Memory))
			if opt.SavingsOpportunity != nil {
				d.Field(prefix+" Savings %", fmt.Sprintf("%.2f%%", opt.SavingsOpportunity.SavingsOpportunityPercentage))
				if opt.SavingsOpportunity.EstimatedMonthlySavings != nil {
					d.Field(prefix+" Est. Savings", appaws.FormatMoney(opt.SavingsOpportunity.EstimatedMonthlySavings.Value, string(opt.SavingsOpportunity.EstimatedMonthlySavings.Currency)))
				}
			}
		}
	}

	// Timestamps
	d.Section("Metadata")
	d.Field("Look Back Period (Days)", fmt.Sprintf("%.0f", rec.LookbackPeriodInDays))
	if rec.LastRefreshTimestamp != nil {
		d.Field("Last Refresh", rec.LastRefreshTimestamp.Format("2006-01-02 15:04:05"))
	}

	// Tags
	d.Tags(appaws.TagsToMap(rec.Tags))
}

// RenderSummary renders summary fields.
func (r *RecommendationRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	rec, ok := resource.(*RecommendationResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Type", Value: rec.ResourceType()},
		{Label: "Finding", Value: rec.Finding()},
		{Label: "Savings", Value: fmt.Sprintf("%s (%.1f%%)", appaws.FormatMoney(rec.SavingsValue(), rec.SavingsCurrency()), rec.SavingsPercent())},
	}
}
