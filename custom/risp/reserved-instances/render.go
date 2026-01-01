package reservedinstances

import (
	"fmt"
	"time"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// ReservedInstanceRenderer renders EC2 Reserved Instances
type ReservedInstanceRenderer struct {
	render.BaseRenderer
}

// NewReservedInstanceRenderer creates a new ReservedInstanceRenderer
func NewReservedInstanceRenderer() render.Renderer {
	return &ReservedInstanceRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "risp",
			Resource: "reserved-instances",
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
						if ri, ok := r.(*ReservedInstanceResource); ok {
							return ri.State()
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "TYPE",
					Width: 13,
					Getter: func(r dao.Resource) string {
						if ri, ok := r.(*ReservedInstanceResource); ok {
							return ri.InstanceType()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "COUNT",
					Width: 6,
					Getter: func(r dao.Resource) string {
						if ri, ok := r.(*ReservedInstanceResource); ok {
							return fmt.Sprintf("%d", ri.InstanceCount())
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "SCOPE",
					Width: 16,
					Getter: func(r dao.Resource) string {
						if ri, ok := r.(*ReservedInstanceResource); ok {
							if ri.Scope() == "Availability Zone" {
								return ri.AvailabilityZone()
							}
							return ri.Scope()
						}
						return ""
					},
					Priority: 4,
				},
				{
					Name:  "OFFERING",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if ri, ok := r.(*ReservedInstanceResource); ok {
							return ri.OfferingClass()
						}
						return ""
					},
					Priority: 5,
				},
				{
					Name:  "TERM",
					Width: 5,
					Getter: func(r dao.Resource) string {
						if ri, ok := r.(*ReservedInstanceResource); ok {
							return ri.Duration()
						}
						return ""
					},
					Priority: 6,
				},
				{
					Name:  "EXPIRES",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if ri, ok := r.(*ReservedInstanceResource); ok {
							if end := ri.EndTime(); end != nil {
								return end.Format("2006-01-02")
							}
						}
						return ""
					},
					Priority: 7,
				},
			},
		},
	}
}

// RenderDetail renders detailed Reserved Instance information
func (r *ReservedInstanceRenderer) RenderDetail(resource dao.Resource) string {
	ri, ok := resource.(*ReservedInstanceResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Reserved Instance", ri.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Reserved Instance ID", ri.GetID())
	d.FieldStyled("State", ri.State(), render.StateColorer()(ri.State()))
	d.Field("Instance Type", ri.InstanceType())
	d.Field("Instance Count", fmt.Sprintf("%d", ri.InstanceCount()))
	d.Field("Product Description", ri.ProductDescription())

	// Scope
	d.Section("Scope")
	d.Field("Scope", ri.Scope())
	if ri.Scope() == "Availability Zone" {
		d.Field("Availability Zone", ri.AvailabilityZone())
	}
	if tenancy := ri.Tenancy(); tenancy != "" && tenancy != "default" {
		d.Field("Tenancy", tenancy)
	}

	// Offering
	d.Section("Offering Details")
	d.Field("Offering Class", ri.OfferingClass())
	d.Field("Offering Type", ri.OfferingType())
	d.Field("Term", ri.Duration())

	// Pricing
	d.Section("Pricing")
	d.Field("Currency", ri.CurrencyCode())
	d.Field("Upfront Cost", fmt.Sprintf("%.2f", ri.FixedPrice()))
	d.Field("Hourly Price", fmt.Sprintf("%.4f", ri.UsagePrice()))

	// Recurring Charges
	if len(ri.Item.RecurringCharges) > 0 {
		d.Section("Recurring Charges")
		for _, charge := range ri.Item.RecurringCharges {
			freq := string(charge.Frequency)
			amount := float64(0)
			if charge.Amount != nil {
				amount = *charge.Amount
			}
			d.Field(freq, fmt.Sprintf("%.4f", amount))
		}
	}

	// Time
	d.Section("Duration")
	if start := ri.StartTime(); start != nil {
		d.Field("Start Date", start.Format(time.RFC3339))
	}
	if end := ri.EndTime(); end != nil {
		d.Field("End Date", end.Format(time.RFC3339))
		remaining := time.Until(*end)
		if remaining > 0 {
			d.Field("Remaining", formatDuration(remaining))
		} else {
			d.Field("Status", "Expired")
		}
	}

	// Tags
	d.Tags(ri.GetTags())

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *ReservedInstanceRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	ri, ok := resource.(*ReservedInstanceResource)
	if !ok {
		return nil
	}

	stateStyle := render.StateColorer()(ri.State())

	fields := []render.SummaryField{
		{Label: "ID", Value: ri.GetID()},
		{Label: "State", Value: ri.State(), Style: stateStyle},
		{Label: "Type", Value: ri.InstanceType()},
		{Label: "Count", Value: fmt.Sprintf("%d", ri.InstanceCount())},
		{Label: "Scope", Value: ri.Scope()},
		{Label: "Offering Class", Value: ri.OfferingClass()},
		{Label: "Payment Option", Value: ri.OfferingType()},
		{Label: "Term", Value: ri.Duration()},
	}

	if end := ri.EndTime(); end != nil {
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
