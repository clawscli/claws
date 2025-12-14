package budgets

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// BudgetRenderer renders AWS Budgets.
// Ensure BudgetRenderer implements render.Navigator
var _ render.Navigator = (*BudgetRenderer)(nil)

type BudgetRenderer struct {
	render.BaseRenderer
}

// NewBudgetRenderer creates a new BudgetRenderer.
func NewBudgetRenderer() render.Renderer {
	return &BudgetRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "budgets",
			Resource: "budgets",
			Cols: []render.Column{
				{Name: "BUDGET NAME", Width: 35, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "TYPE", Width: 12, Getter: getBudgetType},
				{Name: "TIME UNIT", Width: 12, Getter: getTimeUnit},
				{Name: "LIMIT", Width: 15, Getter: getLimit},
				{Name: "ACTUAL", Width: 15, Getter: getActual},
				{Name: "FORECASTED", Width: 15, Getter: getForecasted},
			},
		},
	}
}

func getBudgetType(r dao.Resource) string {
	budget, ok := r.(*BudgetResource)
	if !ok {
		return ""
	}
	return budget.BudgetType()
}

func getTimeUnit(r dao.Resource) string {
	budget, ok := r.(*BudgetResource)
	if !ok {
		return ""
	}
	return budget.TimeUnit()
}

func getLimit(r dao.Resource) string {
	budget, ok := r.(*BudgetResource)
	if !ok {
		return ""
	}
	amount, unit := budget.BudgetLimit()
	if amount == "" {
		return ""
	}
	return fmt.Sprintf("%s %s", amount, unit)
}

func getActual(r dao.Resource) string {
	budget, ok := r.(*BudgetResource)
	if !ok {
		return ""
	}
	amount, unit := budget.ActualSpend()
	if amount == "" {
		return ""
	}
	return fmt.Sprintf("%s %s", amount, unit)
}

func getForecasted(r dao.Resource) string {
	budget, ok := r.(*BudgetResource)
	if !ok {
		return ""
	}
	amount, unit := budget.ForecastedSpend()
	if amount == "" {
		return ""
	}
	return fmt.Sprintf("%s %s", amount, unit)
}

// RenderDetail renders the detail view for a budget.
func (r *BudgetRenderer) RenderDetail(resource dao.Resource) string {
	budget, ok := resource.(*BudgetResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("AWS Budget", budget.Name())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Budget Name", budget.Name())
	d.Field("ARN", budget.GetARN())
	d.Field("Account ID", budget.AccountID)
	d.Field("Budget Type", budget.BudgetType())
	d.Field("Time Unit", budget.TimeUnit())

	// Budget Limit
	d.Section("Budget Limit")
	amount, unit := budget.BudgetLimit()
	if amount != "" {
		d.Field("Limit Amount", fmt.Sprintf("%s %s", amount, unit))
	}

	// Spend
	d.Section("Current Spend")
	actualAmount, actualUnit := budget.ActualSpend()
	if actualAmount != "" {
		d.Field("Actual Spend", fmt.Sprintf("%s %s", actualAmount, actualUnit))
	}
	forecastAmount, forecastUnit := budget.ForecastedSpend()
	if forecastAmount != "" {
		d.Field("Forecasted Spend", fmt.Sprintf("%s %s", forecastAmount, forecastUnit))
	}

	return d.String()
}

// RenderSummary renders summary fields for a budget.
func (r *BudgetRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	budget, ok := resource.(*BudgetResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Budget Name", Value: budget.Name()},
		{Label: "Type", Value: budget.BudgetType()},
		{Label: "Time Unit", Value: budget.TimeUnit()},
	}

	amount, unit := budget.BudgetLimit()
	if amount != "" {
		fields = append(fields, render.SummaryField{Label: "Limit", Value: fmt.Sprintf("%s %s", amount, unit)})
	}

	actualAmount, actualUnit := budget.ActualSpend()
	if actualAmount != "" {
		fields = append(fields, render.SummaryField{Label: "Actual Spend", Value: fmt.Sprintf("%s %s", actualAmount, actualUnit)})
	}

	return fields
}

// Navigations returns available navigations from a budget.
func (r *BudgetRenderer) Navigations(resource dao.Resource) []render.Navigation {
	budget, ok := resource.(*BudgetResource)
	if !ok {
		return nil
	}
	return []render.Navigation{
		{
			Key:         "n",
			Label:       "Notifications",
			Service:     "budgets",
			Resource:    "notifications",
			FilterField: "BudgetName",
			FilterValue: budget.Name(),
		},
	}
}
