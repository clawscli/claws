package recommendations

import (
	"fmt"
	"strings"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// RecommendationRenderer renders Trusted Advisor Recommendation data.
type RecommendationRenderer struct {
	render.BaseRenderer
}

// NewRecommendationRenderer creates a new RecommendationRenderer.
func NewRecommendationRenderer() render.Renderer {
	return &RecommendationRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "trustedadvisor",
			Resource: "recommendations",
			Cols: []render.Column{
				{Name: "NAME", Width: 45, Getter: getName},
				{Name: "STATUS", Width: 10, Getter: getStatus},
				{Name: "PILLAR", Width: 18, Getter: getPillar},
				{Name: "ERR", Width: 5, Getter: getErrCount},
				{Name: "WARN", Width: 5, Getter: getWarnCount},
				{Name: "OK", Width: 5, Getter: getOkCount},
			},
		},
	}
}

func getName(r dao.Resource) string {
	rec, ok := r.(*RecommendationResource)
	if !ok {
		return ""
	}
	return rec.Name()
}

func getStatus(r dao.Resource) string {
	rec, ok := r.(*RecommendationResource)
	if !ok {
		return ""
	}
	return rec.Status()
}

func getPillar(r dao.Resource) string {
	rec, ok := r.(*RecommendationResource)
	if !ok {
		return ""
	}
	return rec.Pillars()
}

func getErrCount(r dao.Resource) string {
	rec, ok := r.(*RecommendationResource)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%d", rec.ErrorCount())
}

func getWarnCount(r dao.Resource) string {
	rec, ok := r.(*RecommendationResource)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%d", rec.WarningCount())
}

func getOkCount(r dao.Resource) string {
	rec, ok := r.(*RecommendationResource)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%d", rec.OkCount())
}

// RenderDetail renders the detail view for a recommendation.
func (r *RecommendationRenderer) RenderDetail(resource dao.Resource) string {
	rec, ok := resource.(*RecommendationResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Trusted Advisor Recommendation", rec.Name())

	// Basic Info
	d.Section("Recommendation Information")
	d.Field("Name", rec.Name())
	d.Field("ID", rec.GetID())
	d.Field("ARN", rec.GetARN())
	d.Field("Status", rec.Status())
	d.Field("Type", rec.Type())
	d.Field("Source", rec.Source())

	// Description (full recommendation only)
	if desc := rec.Description(); desc != "" {
		d.Section("Description")
		d.Field("", desc)
	}

	// Pillars
	d.Section("Pillars")
	for _, pillar := range rec.PillarList() {
		d.Field("", string(pillar))
	}

	// Resource Counts
	d.Section("Resource Summary")
	d.Field("Errors", fmt.Sprintf("%d", rec.ErrorCount()))
	d.Field("Warnings", fmt.Sprintf("%d", rec.WarningCount()))
	d.Field("OK", fmt.Sprintf("%d", rec.OkCount()))

	// Cost Savings (if available)
	if savings := rec.EstimatedMonthlySavings(); savings > 0 {
		d.Section("Cost Optimization")
		d.Field("Estimated Monthly Savings", appaws.FormatMoney(savings, ""))
		d.Field("Estimated Savings %", fmt.Sprintf("%.1f%%", rec.EstimatedPercentMonthlySavings()))
	}

	// AWS Services
	if len(rec.AwsServices()) > 0 {
		d.Section("AWS Services")
		d.Field("Services", strings.Join(rec.AwsServices(), ", "))
	}

	// Dates
	d.Section("Dates")
	if rec.CreatedAt() != "" {
		d.Field("Created", rec.CreatedAt())
	}
	if rec.LastUpdatedAt() != "" {
		d.Field("Last Updated", rec.LastUpdatedAt())
	}
	if resolved := rec.ResolvedAt(); resolved != "" {
		d.Field("Resolved", resolved)
	}

	// Creator (full recommendation only)
	if createdBy := rec.CreatedBy(); createdBy != "" {
		d.Field("Created By", createdBy)
	}

	// Lifecycle
	if rec.LifecycleStage() != "" {
		d.Section("Lifecycle")
		d.Field("Stage", rec.LifecycleStage())
		if reason := rec.UpdateReason(); reason != "" {
			d.Field("Update Reason", reason)
		}
		if code := rec.UpdateReasonCode(); code != "" {
			d.Field("Reason Code", code)
		}
	}

	return d.String()
}

// RenderSummary renders summary fields for a recommendation.
func (r *RecommendationRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	rec, ok := resource.(*RecommendationResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Name", Value: rec.Name()},
		{Label: "Status", Value: rec.Status()},
		{Label: "Pillars", Value: rec.Pillars()},
		{Label: "Resources", Value: fmt.Sprintf("Err:%d Warn:%d OK:%d", rec.ErrorCount(), rec.WarningCount(), rec.OkCount())},
	}
}
