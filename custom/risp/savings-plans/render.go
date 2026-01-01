package savingsplans

import (
	"fmt"
	"time"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// SavingsPlanRenderer renders Savings Plans
type SavingsPlanRenderer struct {
	render.BaseRenderer
}

// NewSavingsPlanRenderer creates a new SavingsPlanRenderer
func NewSavingsPlanRenderer() render.Renderer {
	return &SavingsPlanRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "risp",
			Resource: "savings-plans",
			Cols: []render.Column{
				{
					Name:  "ID",
					Width: 38,
					Getter: func(r dao.Resource) string {
						return r.GetID()
					},
					Priority: 0,
				},
				{
					Name:  "STATE",
					Width: 10,
					Getter: func(r dao.Resource) string {
						if sp, ok := r.(*SavingsPlanResource); ok {
							return sp.State()
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "TYPE",
					Width: 14,
					Getter: func(r dao.Resource) string {
						if sp, ok := r.(*SavingsPlanResource); ok {
							return sp.PlanType()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "COMMITMENT",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if sp, ok := r.(*SavingsPlanResource); ok {
							commitment := sp.Commitment()
							currency := sp.Currency()
							if commitment != "" {
								return fmt.Sprintf("%s %s/hr", commitment, currency)
							}
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "PAYMENT",
					Width: 16,
					Getter: func(r dao.Resource) string {
						if sp, ok := r.(*SavingsPlanResource); ok {
							return sp.PaymentOption()
						}
						return ""
					},
					Priority: 4,
				},
				{
					Name:  "TERM",
					Width: 5,
					Getter: func(r dao.Resource) string {
						if sp, ok := r.(*SavingsPlanResource); ok {
							return sp.Duration()
						}
						return ""
					},
					Priority: 5,
				},
				{
					Name:  "EXPIRES",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if sp, ok := r.(*SavingsPlanResource); ok {
							if end := sp.EndTime(); end != nil {
								return end.Format("2006-01-02")
							}
						}
						return ""
					},
					Priority: 6,
				},
				{
					Name:  "PRODUCTS",
					Width: 20,
					Getter: func(r dao.Resource) string {
						if sp, ok := r.(*SavingsPlanResource); ok {
							return sp.ProductTypes()
						}
						return ""
					},
					Priority: 7,
				},
			},
		},
	}
}

// RenderDetail renders detailed Savings Plan information
func (r *SavingsPlanRenderer) RenderDetail(resource dao.Resource) string {
	sp, ok := resource.(*SavingsPlanResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Savings Plan", sp.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Savings Plan ID", sp.GetID())
	d.Field("ARN", sp.ARN())
	d.FieldStyled("State", sp.State(), render.StateColorer()(sp.State()))
	d.Field("Type", sp.PlanType())
	if desc := sp.Description(); desc != "" {
		d.Field("Description", desc)
	}

	// Scope (for EC2 Instance Savings Plans)
	if sp.PlanType() == "EC2Instance" {
		d.Section("Scope")
		if region := sp.Region(); region != "" {
			d.Field("Region", region)
		}
		if family := sp.EC2InstanceFamily(); family != "" {
			d.Field("Instance Family", family)
		}
	}

	// Applicable Products
	if products := sp.ProductTypes(); products != "" {
		d.Section("Applicable Products")
		d.Field("Product Types", products)
	}

	// Commitment and Pricing
	d.Section("Commitment")
	d.Field("Hourly Commitment", fmt.Sprintf("%s %s/hr", sp.Commitment(), sp.Currency()))
	d.Field("Payment Option", sp.PaymentOption())
	if upfront := sp.UpfrontPayment(); upfront != "" && upfront != "0" {
		d.Field("Upfront Payment", fmt.Sprintf("%s %s", upfront, sp.Currency()))
	}
	if recurring := sp.RecurringPayment(); recurring != "" && recurring != "0" {
		d.Field("Recurring Payment", fmt.Sprintf("%s %s/month", recurring, sp.Currency()))
	}

	// Duration
	d.Section("Duration")
	d.Field("Term", sp.Duration())
	if start := sp.StartTime(); start != nil {
		d.Field("Start Date", start.Format(time.RFC3339))
	}
	if end := sp.EndTime(); end != nil {
		d.Field("End Date", end.Format(time.RFC3339))
		remaining := time.Until(*end)
		if remaining > 0 {
			d.Field("Remaining", formatDuration(remaining))
		} else {
			d.Field("Status", "Expired")
		}
	}

	// Tags
	d.Tags(sp.GetTags())

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *SavingsPlanRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	sp, ok := resource.(*SavingsPlanResource)
	if !ok {
		return nil
	}

	stateStyle := render.StateColorer()(sp.State())

	fields := []render.SummaryField{
		{Label: "ID", Value: sp.GetID()},
		{Label: "State", Value: sp.State(), Style: stateStyle},
		{Label: "Type", Value: sp.PlanType()},
		{Label: "Commitment", Value: fmt.Sprintf("%s %s/hr", sp.Commitment(), sp.Currency())},
		{Label: "Payment Option", Value: sp.PaymentOption()},
		{Label: "Term", Value: sp.Duration()},
		{Label: "Products", Value: sp.ProductTypes()},
	}

	if end := sp.EndTime(); end != nil {
		fields = append(fields, render.SummaryField{
			Label: "Expires",
			Value: end.Format("2006-01-02"),
		})
	}

	return fields
}

// formatDuration formats a duration in a human-readable format
func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	if days > 365 {
		years := days / 365
		months := (days % 365) / 30
		return fmt.Sprintf("%dy %dm", years, months)
	}
	if days > 30 {
		months := days / 30
		remainingDays := days % 30
		return fmt.Sprintf("%dm %dd", months, remainingDays)
	}
	return fmt.Sprintf("%dd", days)
}
