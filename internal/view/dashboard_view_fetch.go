package view

import (
	"sort"
	"strconv"

	tea "charm.land/bubbletea/v2"

	"github.com/clawscli/claws/custom/ce/anomalies"
	"github.com/clawscli/claws/custom/ce/costs"
	"github.com/clawscli/claws/custom/cloudwatch/alarms"
	"github.com/clawscli/claws/custom/health/events"
	"github.com/clawscli/claws/custom/securityhub/findings"
	"github.com/clawscli/claws/custom/trustedadvisor/recommendations"
	"github.com/clawscli/claws/internal/dao"
)

type alarmItem struct {
	name     string
	state    string
	resource *alarms.AlarmResource
}

type costItem struct {
	service string
	cost    float64
}

type healthItem struct {
	service   string
	eventType string
	resource  *events.EventResource
}

type securityItem struct {
	title    string
	severity string
	resource *findings.FindingResource
}

type taItem struct {
	name     string
	status   string
	savings  float64
	resource *recommendations.RecommendationResource
}

type alarmLoadedMsg struct{ items []alarmItem }
type alarmErrorMsg struct{ err error }

type costLoadedMsg struct {
	mtd      float64
	topCosts []costItem
}
type costErrorMsg struct{ err error }

type anomalyLoadedMsg struct{ count int }
type anomalyErrorMsg struct{ err error }

type healthLoadedMsg struct{ items []healthItem }
type healthErrorMsg struct{ err error }

type securityLoadedMsg struct{ items []securityItem }
type securityErrorMsg struct{ err error }

type taLoadedMsg struct {
	items   []taItem
	savings float64
}
type taErrorMsg struct{ err error }

func (d *DashboardView) loadAlarms() tea.Msg {
	if d.ctx.Err() != nil {
		return alarmErrorMsg{err: d.ctx.Err()}
	}

	alarmDAO, err := alarms.NewAlarmDAO(d.ctx)
	if err != nil {
		return alarmErrorMsg{err: err}
	}

	ctx := dao.WithFilter(d.ctx, "StateValue", "ALARM")
	resources, err := alarmDAO.List(ctx)
	if err != nil {
		return alarmErrorMsg{err: err}
	}

	if len(resources) > dashboardMaxRecords {
		resources = resources[:dashboardMaxRecords]
	}

	items := make([]alarmItem, 0, len(resources))
	for _, r := range resources {
		if ar, ok := r.(*alarms.AlarmResource); ok {
			items = append(items, alarmItem{name: ar.GetName(), state: ar.StateValue, resource: ar})
		}
	}
	return alarmLoadedMsg{items: items}
}

func (d *DashboardView) loadCosts() tea.Msg {
	if d.ctx.Err() != nil {
		return costErrorMsg{err: d.ctx.Err()}
	}

	costDAO, err := costs.NewCostDAO(d.ctx)
	if err != nil {
		return costErrorMsg{err: err}
	}

	resources, err := costDAO.List(d.ctx)
	if err != nil {
		return costErrorMsg{err: err}
	}

	var items []costItem
	var total float64
	for _, r := range resources {
		if cr, ok := r.(*costs.CostResource); ok {
			c, err := strconv.ParseFloat(cr.Cost, 64)
			if err != nil {
				continue
			}
			if c > 0 {
				items = append(items, costItem{service: cr.ServiceName, cost: c})
				total += c
			}
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].cost > items[j].cost
	})

	return costLoadedMsg{mtd: total, topCosts: items}
}

func (d *DashboardView) loadAnomalies() tea.Msg {
	if d.ctx.Err() != nil {
		return anomalyErrorMsg{err: d.ctx.Err()}
	}

	anomalyDAO, err := anomalies.NewAnomalyDAO(d.ctx)
	if err != nil {
		return anomalyErrorMsg{err: err}
	}

	resources, err := anomalyDAO.List(d.ctx)
	if err != nil {
		return anomalyErrorMsg{err: err}
	}

	return anomalyLoadedMsg{count: len(resources)}
}

func (d *DashboardView) loadHealth() tea.Msg {
	if d.ctx.Err() != nil {
		return healthErrorMsg{err: d.ctx.Err()}
	}

	eventDAO, err := events.NewEventDAO(d.ctx)
	if err != nil {
		return healthErrorMsg{err: err}
	}

	resources, err := eventDAO.List(d.ctx)
	if err != nil {
		return healthErrorMsg{err: err}
	}

	var items []healthItem
	for _, r := range resources {
		if er, ok := r.(*events.EventResource); ok {
			if er.StatusCode() != "closed" {
				items = append(items, healthItem{service: er.Service(), eventType: er.EventTypeCode(), resource: er})
			}
		}
	}
	return healthLoadedMsg{items: items}
}

func (d *DashboardView) loadSecurity() tea.Msg {
	if d.ctx.Err() != nil {
		return securityErrorMsg{err: d.ctx.Err()}
	}

	findingDAO, err := findings.NewFindingDAO(d.ctx)
	if err != nil {
		return securityErrorMsg{err: err}
	}

	resources, err := findingDAO.List(d.ctx)
	if err != nil {
		return securityErrorMsg{err: err}
	}

	var items []securityItem
	for _, r := range resources {
		if fr, ok := r.(*findings.FindingResource); ok {
			sev := fr.Severity()
			if sev == "CRITICAL" || sev == "HIGH" {
				items = append(items, securityItem{title: fr.Title(), severity: sev, resource: fr})
			}
		}
	}
	return securityLoadedMsg{items: items}
}

func (d *DashboardView) loadTrustedAdvisor() tea.Msg {
	if d.ctx.Err() != nil {
		return taErrorMsg{err: d.ctx.Err()}
	}

	taDAO, err := recommendations.NewRecommendationDAO(d.ctx)
	if err != nil {
		return taErrorMsg{err: err}
	}

	resources, err := taDAO.List(d.ctx)
	if err != nil {
		return taErrorMsg{err: err}
	}

	var items []taItem
	var totalSavings float64
	for _, r := range resources {
		if rr, ok := r.(*recommendations.RecommendationResource); ok {
			status := rr.Status()
			if status == "error" || status == "warning" {
				items = append(items, taItem{name: rr.Name(), status: status, savings: rr.EstimatedMonthlySavings(), resource: rr})
			}
			totalSavings += rr.EstimatedMonthlySavings()
		}
	}
	return taLoadedMsg{items: items, savings: totalSavings}
}
