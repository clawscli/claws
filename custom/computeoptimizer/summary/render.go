package summary

import (
	"fmt"

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
		return fmt.Sprintf("$%.2f", savings)
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

	// Savings Opportunity
	d.Section("Savings Opportunity")
	d.Field("Savings Percentage", fmt.Sprintf("%.2f%%", s.SavingsOpportunityPercentage()))
	d.Field("Estimated Monthly Savings", fmt.Sprintf("$%.2f %s", s.EstimatedMonthlySavings(), s.EstimatedMonthlySavingsCurrency()))

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
		{Label: "Savings", Value: fmt.Sprintf("$%.2f (%.1f%%)", s.EstimatedMonthlySavings(), s.SavingsOpportunityPercentage())},
	}
}
