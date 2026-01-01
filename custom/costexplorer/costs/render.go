package costs

import (
	"fmt"
	"strconv"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// CostRenderer renders AWS Cost Explorer data.
type CostRenderer struct {
	render.BaseRenderer
}

// NewCostRenderer creates a new CostRenderer.
func NewCostRenderer() render.Renderer {
	return &CostRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "ce",
			Resource: "costs",
			Cols: []render.Column{
				{Name: "SERVICE", Width: 45, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "COST", Width: 15, Getter: getCost},
				{Name: "UNIT", Width: 8, Getter: getCostUnit},
				{Name: "USAGE", Width: 20, Getter: getUsage},
			},
		},
	}
}

func getCost(r dao.Resource) string {
	cost, ok := r.(*CostResource)
	if !ok {
		return ""
	}
	// Format cost to 2 decimal places
	if cost.Cost != "" {
		if f, err := strconv.ParseFloat(cost.Cost, 64); err == nil {
			return fmt.Sprintf("%.2f", f)
		}
	}
	return cost.Cost
}

func getCostUnit(r dao.Resource) string {
	cost, ok := r.(*CostResource)
	if !ok {
		return ""
	}
	return cost.CostUnit
}

func getUsage(r dao.Resource) string {
	cost, ok := r.(*CostResource)
	if !ok {
		return ""
	}
	if cost.UsageQuantity == "" {
		return ""
	}
	// Format usage quantity
	if f, err := strconv.ParseFloat(cost.UsageQuantity, 64); err == nil {
		// Don't show unit if it's N/A or empty
		if cost.UsageUnit != "" && cost.UsageUnit != "N/A" {
			return fmt.Sprintf("%.2f %s", f, cost.UsageUnit)
		}
		return fmt.Sprintf("%.2f", f)
	}
	return cost.UsageQuantity
}

// RenderDetail renders the detail view for cost data.
func (r *CostRenderer) RenderDetail(resource dao.Resource) string {
	cost, ok := resource.(*CostResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("AWS Cost", cost.ServiceName)

	// Basic Info
	d.Section("Service Information")
	d.Field("Service Name", cost.ServiceName)

	// Time Period
	d.Section("Time Period")
	d.Field("Start Date", cost.StartDate)
	d.Field("End Date", cost.EndDate)

	// Cost
	d.Section("Cost")
	if cost.Cost != "" {
		if f, err := strconv.ParseFloat(cost.Cost, 64); err == nil {
			d.Field("Unblended Cost", fmt.Sprintf("%.2f %s", f, cost.CostUnit))
		} else {
			d.Field("Unblended Cost", fmt.Sprintf("%s %s", cost.Cost, cost.CostUnit))
		}
	}

	// Usage
	if cost.UsageQuantity != "" {
		d.Section("Usage")
		if f, err := strconv.ParseFloat(cost.UsageQuantity, 64); err == nil {
			d.Field("Usage Quantity", fmt.Sprintf("%.2f %s", f, cost.UsageUnit))
		} else {
			d.Field("Usage Quantity", fmt.Sprintf("%s %s", cost.UsageQuantity, cost.UsageUnit))
		}
	}

	return d.String()
}

// RenderSummary renders summary fields for cost data.
func (r *CostRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	cost, ok := resource.(*CostResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Service", Value: cost.ServiceName},
		{Label: "Period", Value: fmt.Sprintf("%s to %s", cost.StartDate, cost.EndDate)},
	}

	if cost.Cost != "" {
		if f, err := strconv.ParseFloat(cost.Cost, 64); err == nil {
			fields = append(fields, render.SummaryField{Label: "Cost", Value: fmt.Sprintf("%.2f %s", f, cost.CostUnit)})
		}
	}

	if cost.UsageQuantity != "" {
		if f, err := strconv.ParseFloat(cost.UsageQuantity, 64); err == nil {
			fields = append(fields, render.SummaryField{Label: "Usage", Value: fmt.Sprintf("%.2f %s", f, cost.UsageUnit)})
		}
	}

	return fields
}
