package summary

import (
	"fmt"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// SummaryRenderer renders Compute Optimizer Summary data.
type SummaryRenderer struct {
	render.BaseRenderer
}

// NewSummaryRenderer creates a new SummaryRenderer.
func NewSummaryRenderer() render.Renderer {
	return &SummaryRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "compute-optimizer",
			Resource: "summary",
			Cols: []render.Column{
				{Name: "RESOURCE TYPE", Width: 20, Getter: getResourceType},
				{Name: "TOTAL", Width: 8, Getter: getTotal},
				{Name: "OPTIMIZED", Width: 10, Getter: getOptimized},
				{Name: "NOT OPT", Width: 10, Getter: getNotOptimized},
				{Name: "SAVINGS %", Width: 10, Getter: getSavingsPct},
				{Name: "EST. SAVINGS", Width: 14, Getter: getEstSavings},
			},
		},
	}
}

func getResourceType(r dao.Resource) string {
	s, ok := r.(*SummaryResource)
	if !ok {
		return ""
	}
	return s.ResourceType()
}

func getTotal(r dao.Resource) string {
	s, ok := r.(*SummaryResource)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%.0f", s.TotalResources())
}

func getOptimized(r dao.Resource) string {
	s, ok := r.(*SummaryResource)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%.0f", s.OptimizedCount())
}

func getNotOptimized(r dao.Resource) string {
	s, ok := r.(*SummaryResource)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%.0f", s.NotOptimizedCount())
}

func getSavingsPct(r dao.Resource) string {
	s, ok := r.(*SummaryResource)
	if !ok {
		return ""
	}
	pct := s.SavingsOpportunityPercentage()
	if pct > 0 {
		return fmt.Sprintf("%.1f%%", pct)
	}
	return "-"
}

func getEstSavings(r dao.Resource) string {
	s, ok := r.(*SummaryResource)
	if !ok {
		return ""
	}
	savings := s.EstimatedMonthlySavings()
	if savings > 0 {
		return appaws.FormatMoney(savings, s.SavingsCurrency())
	}
	return "-"
}

// RenderDetail renders the detail view for a summary.
func (r *SummaryRenderer) RenderDetail(resource dao.Resource) string {
	s, ok := resource.(*SummaryResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Compute Optimizer Summary", s.ResourceType())

	// Basic Info
	d.Section("Resource Information")
	d.Field("Resource Type", s.ResourceType())
	d.Field("Account ID", s.AccountId())

	// Summary Counts
	d.Section("Resource Counts")
	d.Field("Total Resources", fmt.Sprintf("%.0f", s.TotalResources()))
	for _, summary := range s.Summaries() {
		if summary.Value > 0 {
			d.Field(string(summary.Name), fmt.Sprintf("%.0f", summary.Value))
		}
	}

	// Performance Risk Ratings
	if risk := s.PerformanceRiskRatings(); risk != nil {
		d.Section("Performance Risk Distribution")
		d.Field("Very Low", fmt.Sprintf("%d", risk.VeryLow))
		d.Field("Low", fmt.Sprintf("%d", risk.Low))
		d.Field("Medium", fmt.Sprintf("%d", risk.Medium))
		d.Field("High", fmt.Sprintf("%d", risk.High))
	}

	// Savings Opportunity
	d.Section("Savings Opportunity")
	d.Field("Savings Percentage", fmt.Sprintf("%.2f%%", s.SavingsOpportunityPercentage()))
	d.Field("Estimated Monthly Savings", appaws.FormatMoney(s.EstimatedMonthlySavings(), s.SavingsCurrency()))

	// Idle Savings
	if idle := s.IdleSavingsOpportunity(); idle != nil && idle.EstimatedMonthlySavings != nil {
		d.Section("Idle Resource Savings")
		d.Field("Savings Percentage", fmt.Sprintf("%.2f%%", idle.SavingsOpportunityPercentage))
		d.Field("Estimated Monthly Savings", appaws.FormatMoney(idle.EstimatedMonthlySavings.Value, string(idle.EstimatedMonthlySavings.Currency)))
	}

	// Idle Summaries
	if idleSummaries := s.IdleSummaries(); len(idleSummaries) > 0 {
		d.Section("Idle Resources")
		for _, is := range idleSummaries {
			if is.Value > 0 {
				d.Field(string(is.Name), fmt.Sprintf("%.0f", is.Value))
			}
		}
	}

	// Inferred Workload Savings
	if workloads := s.InferredWorkloadSavings(); len(workloads) > 0 {
		d.Section("Inferred Workload Savings")
		for _, w := range workloads {
			if w.EstimatedMonthlySavings != nil && w.EstimatedMonthlySavings.Value > 0 {
				for _, wt := range w.InferredWorkloadTypes {
					d.Field(string(wt), appaws.FormatMoney(w.EstimatedMonthlySavings.Value, string(w.EstimatedMonthlySavings.Currency)))
				}
			}
		}
	}

	return d.String()
}

// RenderSummary renders summary fields.
func (r *SummaryRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	s, ok := resource.(*SummaryResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Resource Type", Value: s.ResourceType()},
		{Label: "Total", Value: fmt.Sprintf("%.0f", s.TotalResources())},
		{Label: "Savings", Value: fmt.Sprintf("%s (%.1f%%)", appaws.FormatMoney(s.EstimatedMonthlySavings(), s.SavingsCurrency()), s.SavingsOpportunityPercentage())},
	}
}
