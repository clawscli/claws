package anomalies

import (
	"fmt"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// AnomalyRenderer renders Cost Anomaly Detection data.
type AnomalyRenderer struct {
	render.BaseRenderer
}

// NewAnomalyRenderer creates a new AnomalyRenderer.
func NewAnomalyRenderer() render.Renderer {
	return &AnomalyRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "ce",
			Resource: "anomalies",
			Cols: []render.Column{
				{Name: "SERVICE", Width: 30, Getter: getDimensionValue},
				{Name: "IMPACT", Width: 12, Getter: getImpact},
				{Name: "IMPACT%", Width: 10, Getter: getImpactPct},
				{Name: "START", Width: 12, Getter: getStartDate},
				{Name: "END", Width: 12, Getter: getEndDate},
				{Name: "SCORE", Width: 8, Getter: getMaxScore},
				{Name: "FEEDBACK", Width: 12, Getter: getFeedback},
			},
		},
	}
}

func getDimensionValue(r dao.Resource) string {
	a, ok := r.(*AnomalyResource)
	if !ok {
		return ""
	}
	return a.DimensionValue()
}

func getImpact(r dao.Resource) string {
	a, ok := r.(*AnomalyResource)
	if !ok {
		return ""
	}
	return appaws.FormatMoney(a.TotalImpact(), "")
}

func getImpactPct(r dao.Resource) string {
	a, ok := r.(*AnomalyResource)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%.1f%%", a.TotalImpactPercentage())
}

func getStartDate(r dao.Resource) string {
	a, ok := r.(*AnomalyResource)
	if !ok {
		return ""
	}
	return a.StartDate()
}

func getEndDate(r dao.Resource) string {
	a, ok := r.(*AnomalyResource)
	if !ok {
		return ""
	}
	return a.EndDate()
}

func getMaxScore(r dao.Resource) string {
	a, ok := r.(*AnomalyResource)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%.1f", a.MaxScore())
}

func getFeedback(r dao.Resource) string {
	a, ok := r.(*AnomalyResource)
	if !ok {
		return ""
	}
	return a.Feedback()
}

// RenderDetail renders the detail view for an anomaly.
func (r *AnomalyRenderer) RenderDetail(resource dao.Resource) string {
	a, ok := resource.(*AnomalyResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Cost Anomaly", a.DimensionValue())

	// Basic Info
	d.Section("Anomaly Information")
	d.Field("Anomaly ID", a.GetID())
	d.Field("Service/Dimension", a.DimensionValue())
	d.Field("Monitor ARN", a.MonitorArn())

	// Time Period
	d.Section("Time Period")
	d.Field("Start Date", a.StartDate())
	d.Field("End Date", a.EndDate())

	// Impact
	d.Section("Cost Impact")
	d.Field("Total Impact", appaws.FormatMoney(a.TotalImpact(), ""))
	d.Field("Impact Percentage", fmt.Sprintf("%.2f%%", a.TotalImpactPercentage()))
	d.Field("Actual Spend", appaws.FormatMoney(a.TotalActualSpend(), ""))
	d.Field("Expected Spend", appaws.FormatMoney(a.TotalExpectedSpend(), ""))

	// Score
	d.Section("Anomaly Score")
	d.Field("Max Score", fmt.Sprintf("%.2f", a.MaxScore()))
	d.Field("Current Score", fmt.Sprintf("%.2f", a.CurrentScore()))

	// Feedback
	d.Section("Status")
	d.Field("Feedback", a.Feedback())

	// Root Causes
	if len(a.RootCauses()) > 0 {
		d.Section("Root Causes")
		for i, cause := range a.RootCauses() {
			prefix := fmt.Sprintf("Cause %d", i+1)
			if cause.Service != nil {
				d.Field(prefix+" Service", *cause.Service)
			}
			if cause.Region != nil {
				d.Field(prefix+" Region", *cause.Region)
			}
			if cause.UsageType != nil {
				d.Field(prefix+" Usage Type", *cause.UsageType)
			}
			if cause.LinkedAccount != nil {
				acct := *cause.LinkedAccount
				if cause.LinkedAccountName != nil {
					acct = fmt.Sprintf("%s (%s)", *cause.LinkedAccountName, acct)
				}
				d.Field(prefix+" Account", acct)
			}
			if cause.Impact != nil {
				d.Field(prefix+" Contribution", appaws.FormatMoney(cause.Impact.Contribution, ""))
			}
		}
	}

	return d.String()
}

// RenderSummary renders summary fields for an anomaly.
func (r *AnomalyRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	a, ok := resource.(*AnomalyResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Service", Value: a.DimensionValue()},
		{Label: "Period", Value: fmt.Sprintf("%s to %s", a.StartDate(), a.EndDate())},
		{Label: "Impact", Value: fmt.Sprintf("%s (%.1f%%)", appaws.FormatMoney(a.TotalImpact(), ""), a.TotalImpactPercentage())},
		{Label: "Score", Value: fmt.Sprintf("%.1f", a.MaxScore())},
	}
}
