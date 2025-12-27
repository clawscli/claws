package view

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/clawscli/claws/custom/costexplorer/anomalies"
	"github.com/clawscli/claws/custom/costexplorer/costs"
	"github.com/clawscli/claws/custom/health/events"
	"github.com/clawscli/claws/custom/securityhub/findings"
	"github.com/clawscli/claws/custom/trustedadvisor/recommendations"
	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/ui"
)

// Widget messages
type alarmLoadedMsg struct{ count int }
type alarmErrorMsg struct{ err error }

type costLoadedMsg struct {
	mtd      float64
	topCosts []costItem
}
type costErrorMsg struct{ err error }

type anomalyLoadedMsg struct{ count int }
type anomalyErrorMsg struct{ err error }

type healthLoadedMsg struct {
	openCount   int
	recentEvent string
}
type healthErrorMsg struct{ err error }

type securityLoadedMsg struct {
	criticalCount int
	highCount     int
}
type securityErrorMsg struct{ err error }

type taLoadedMsg struct {
	errorCount   int
	warningCount int
	savings      float64
}
type taErrorMsg struct{ err error }

type costItem struct {
	service string
	cost    float64
}

type dashboardStyles struct {
	title   lipgloss.Style
	section lipgloss.Style
	label   lipgloss.Style
	value   lipgloss.Style
	warning lipgloss.Style
	danger  lipgloss.Style
	success lipgloss.Style
	dim     lipgloss.Style
}

func newDashboardStyles() dashboardStyles {
	t := ui.Current()
	return dashboardStyles{
		title:   lipgloss.NewStyle().Bold(true).Foreground(t.Primary),
		section: lipgloss.NewStyle().Foreground(t.TextDim).Bold(true),
		label:   lipgloss.NewStyle().Foreground(t.TextDim).Width(32),
		value:   lipgloss.NewStyle().Foreground(t.Text),
		warning: lipgloss.NewStyle().Foreground(t.Warning),
		danger:  lipgloss.NewStyle().Foreground(t.Danger),
		success: lipgloss.NewStyle().Foreground(t.Success),
		dim:     lipgloss.NewStyle().Foreground(t.TextMuted),
	}
}

type DashboardView struct {
	ctx         context.Context
	registry    *registry.Registry
	width       int
	height      int
	headerPanel *HeaderPanel
	spinner     spinner.Model
	styles      dashboardStyles

	// Alarm widget
	alarmCount   int
	alarmLoading bool
	alarmErr     error

	// Cost widget
	costMTD     float64
	costTop     []costItem
	costLoading bool
	costErr     error

	// Anomaly widget
	anomalyCount   int
	anomalyLoading bool
	anomalyErr     error

	// Health widget
	healthOpen    int
	healthRecent  string
	healthLoading bool
	healthErr     error

	// Security Hub widget
	secCritical int
	secHigh     int
	secLoading  bool
	secErr      error

	// Trusted Advisor widget
	taErrors   int
	taWarnings int
	taSavings  float64
	taLoading  bool
	taErr      error
}

func NewDashboardView(ctx context.Context, reg *registry.Registry) *DashboardView {
	hp := NewHeaderPanel()
	hp.SetWidth(120)

	return &DashboardView{
		ctx:            ctx,
		registry:       reg,
		headerPanel:    hp,
		spinner:        ui.NewSpinner(),
		styles:         newDashboardStyles(),
		alarmLoading:   true,
		costLoading:    true,
		anomalyLoading: true,
		healthLoading:  true,
		secLoading:     true,
		taLoading:      true,
	}
}

func (d *DashboardView) Init() tea.Cmd {
	return tea.Batch(
		d.spinner.Tick,
		d.loadAlarms,
		d.loadCosts,
		d.loadAnomalies,
		d.loadHealth,
		d.loadSecurity,
		d.loadTrustedAdvisor,
	)
}

func (d *DashboardView) loadAlarms() tea.Msg {
	cfg, err := appaws.NewConfig(d.ctx)
	if err != nil {
		return alarmErrorMsg{err: err}
	}

	client := cloudwatch.NewFromConfig(cfg)
	output, err := client.DescribeAlarms(d.ctx, &cloudwatch.DescribeAlarmsInput{
		StateValue: types.StateValueAlarm,
	})
	if err != nil {
		return alarmErrorMsg{err: err}
	}

	count := len(output.MetricAlarms) + len(output.CompositeAlarms)
	return alarmLoadedMsg{count: count}
}

func (d *DashboardView) loadCosts() tea.Msg {
	dao, err := costs.NewCostDAO(d.ctx)
	if err != nil {
		return costErrorMsg{err: err}
	}

	resources, err := dao.List(d.ctx)
	if err != nil {
		return costErrorMsg{err: err}
	}

	var items []costItem
	var total float64
	for _, r := range resources {
		if cr, ok := r.(*costs.CostResource); ok {
			c, _ := strconv.ParseFloat(cr.Cost, 64)
			if c > 0 {
				items = append(items, costItem{service: cr.ServiceName, cost: c})
				total += c
			}
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].cost > items[j].cost
	})

	top := items
	if len(top) > 3 {
		top = top[:3]
	}

	return costLoadedMsg{mtd: total, topCosts: top}
}

func (d *DashboardView) loadAnomalies() tea.Msg {
	dao, err := anomalies.NewAnomalyDAO(d.ctx)
	if err != nil {
		return anomalyErrorMsg{err: err}
	}

	resources, err := dao.List(d.ctx)
	if err != nil {
		return anomalyErrorMsg{err: err}
	}

	return anomalyLoadedMsg{count: len(resources)}
}

func (d *DashboardView) loadHealth() tea.Msg {
	dao, err := events.NewEventDAO(d.ctx)
	if err != nil {
		return healthErrorMsg{err: err}
	}

	resources, err := dao.List(d.ctx)
	if err != nil {
		return healthErrorMsg{err: err}
	}

	var openCount int
	var recentEvent string
	for _, r := range resources {
		if er, ok := r.(*events.EventResource); ok {
			if er.StatusCode() != "closed" {
				openCount++
				if recentEvent == "" {
					recentEvent = fmt.Sprintf("%s: %s", er.Service(), er.EventTypeCode())
				}
			}
		}
	}

	return healthLoadedMsg{openCount: openCount, recentEvent: recentEvent}
}

func (d *DashboardView) loadSecurity() tea.Msg {
	dao, err := findings.NewFindingDAO(d.ctx)
	if err != nil {
		return securityErrorMsg{err: err}
	}

	resources, err := dao.List(d.ctx)
	if err != nil {
		return securityErrorMsg{err: err}
	}

	var critical, high int
	for _, r := range resources {
		if fr, ok := r.(*findings.FindingResource); ok {
			switch fr.Severity() {
			case "CRITICAL":
				critical++
			case "HIGH":
				high++
			}
		}
	}

	return securityLoadedMsg{criticalCount: critical, highCount: high}
}

func (d *DashboardView) loadTrustedAdvisor() tea.Msg {
	dao, err := recommendations.NewRecommendationDAO(d.ctx)
	if err != nil {
		return taErrorMsg{err: err}
	}

	resources, err := dao.List(d.ctx)
	if err != nil {
		return taErrorMsg{err: err}
	}

	var errors, warnings int
	var savings float64
	for _, r := range resources {
		if rr, ok := r.(*recommendations.RecommendationResource); ok {
			switch rr.Status() {
			case "error":
				errors++
			case "warning":
				warnings++
			}
			savings += rr.EstimatedMonthlySavings()
		}
	}

	return taLoadedMsg{errorCount: errors, warningCount: warnings, savings: savings}
}

func (d *DashboardView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case alarmLoadedMsg:
		d.alarmLoading = false
		d.alarmCount = msg.count
		return d, nil
	case alarmErrorMsg:
		d.alarmLoading = false
		d.alarmErr = msg.err
		return d, nil

	case costLoadedMsg:
		d.costLoading = false
		d.costMTD = msg.mtd
		d.costTop = msg.topCosts
		return d, nil
	case costErrorMsg:
		d.costLoading = false
		d.costErr = msg.err
		return d, nil

	case anomalyLoadedMsg:
		d.anomalyLoading = false
		d.anomalyCount = msg.count
		return d, nil
	case anomalyErrorMsg:
		d.anomalyLoading = false
		d.anomalyErr = msg.err
		return d, nil

	case healthLoadedMsg:
		d.healthLoading = false
		d.healthOpen = msg.openCount
		d.healthRecent = msg.recentEvent
		return d, nil
	case healthErrorMsg:
		d.healthLoading = false
		d.healthErr = msg.err
		return d, nil

	case securityLoadedMsg:
		d.secLoading = false
		d.secCritical = msg.criticalCount
		d.secHigh = msg.highCount
		return d, nil
	case securityErrorMsg:
		d.secLoading = false
		d.secErr = msg.err
		return d, nil

	case taLoadedMsg:
		d.taLoading = false
		d.taErrors = msg.errorCount
		d.taWarnings = msg.warningCount
		d.taSavings = msg.savings
		return d, nil
	case taErrorMsg:
		d.taLoading = false
		d.taErr = msg.err
		return d, nil

	case spinner.TickMsg:
		if d.isLoading() {
			var cmd tea.Cmd
			d.spinner, cmd = d.spinner.Update(msg)
			return d, cmd
		}

	case tea.KeyPressMsg:
		switch msg.String() {
		case "s":
			browser := NewServiceBrowser(d.ctx, d.registry)
			return d, func() tea.Msg {
				return NavigateMsg{View: browser}
			}
		}

	case RefreshMsg:
		d.alarmLoading = true
		d.costLoading = true
		d.anomalyLoading = true
		d.healthLoading = true
		d.secLoading = true
		d.taLoading = true
		d.alarmErr = nil
		d.costErr = nil
		d.anomalyErr = nil
		d.healthErr = nil
		d.secErr = nil
		d.taErr = nil
		return d, d.Init()
	}

	return d, nil
}

func (d *DashboardView) isLoading() bool {
	return d.alarmLoading || d.costLoading || d.anomalyLoading ||
		d.healthLoading || d.secLoading || d.taLoading
}

func (d *DashboardView) ViewString() string {
	header := d.headerPanel.RenderHome()
	s := d.styles

	var body string

	// Cost Section
	body += "\n" + s.section.Render("── Cost ──") + "\n"
	body += d.renderWidget("  MTD Spend", d.costLoading, d.costErr, func() string {
		return fmt.Sprintf("$%.2f", d.costMTD)
	})
	body += d.renderWidget("  Top Services", d.costLoading, d.costErr, func() string {
		if len(d.costTop) == 0 {
			return s.dim.Render("none")
		}
		var out string
		for i, c := range d.costTop {
			if i > 0 {
				out += ", "
			}
			out += fmt.Sprintf("%s ($%.0f)", c.service, c.cost)
		}
		return out
	})
	body += d.renderWidget("  Anomalies (90d)", d.anomalyLoading, d.anomalyErr, func() string {
		if d.anomalyCount > 0 {
			return s.warning.Render(fmt.Sprintf("%d", d.anomalyCount))
		}
		return s.success.Render("0")
	})

	// Operations Section
	body += "\n" + s.section.Render("── Operations ──") + "\n"
	body += d.renderWidget("  CloudWatch Alarms", d.alarmLoading, d.alarmErr, func() string {
		if d.alarmCount > 0 {
			return s.danger.Render(fmt.Sprintf("%d in ALARM", d.alarmCount))
		}
		return s.success.Render("0 ✓")
	})
	body += d.renderWidget("  Health Events", d.healthLoading, d.healthErr, func() string {
		if d.healthOpen > 0 {
			out := s.warning.Render(fmt.Sprintf("%d open", d.healthOpen))
			if d.healthRecent != "" {
				out += s.dim.Render(" · " + d.healthRecent)
			}
			return out
		}
		return s.success.Render("0 open")
	})

	// Security Section
	body += "\n" + s.section.Render("── Security ──") + "\n"
	body += d.renderWidget("  Security Hub", d.secLoading, d.secErr, func() string {
		if d.secCritical > 0 || d.secHigh > 0 {
			parts := []string{}
			if d.secCritical > 0 {
				parts = append(parts, s.danger.Render(fmt.Sprintf("%d CRITICAL", d.secCritical)))
			}
			if d.secHigh > 0 {
				parts = append(parts, s.warning.Render(fmt.Sprintf("%d HIGH", d.secHigh)))
			}
			out := parts[0]
			if len(parts) > 1 {
				out += ", " + parts[1]
			}
			return out
		}
		return s.success.Render("0 critical/high")
	})

	// Optimization Section
	body += "\n" + s.section.Render("── Optimization ──") + "\n"
	body += d.renderWidget("  Trusted Advisor", d.taLoading, d.taErr, func() string {
		parts := []string{}
		if d.taErrors > 0 {
			parts = append(parts, s.danger.Render(fmt.Sprintf("%d errors", d.taErrors)))
		}
		if d.taWarnings > 0 {
			parts = append(parts, s.warning.Render(fmt.Sprintf("%d warnings", d.taWarnings)))
		}
		if d.taSavings > 0 {
			parts = append(parts, s.success.Render(fmt.Sprintf("$%.0f/mo savings", d.taSavings)))
		}
		if len(parts) == 0 {
			return s.success.Render("all good")
		}
		out := parts[0]
		for i := 1; i < len(parts); i++ {
			out += " · " + parts[i]
		}
		return out
	})

	// Navigation hint
	body += "\n" + s.dim.Render("  s:services • Ctrl+r:refresh")

	return header + body
}

func (d *DashboardView) renderWidget(label string, loading bool, err error, valueFn func() string) string {
	s := d.styles
	line := s.label.Render(label + ":")
	if loading {
		line += d.spinner.View() + " loading..."
	} else if err != nil {
		line += s.dim.Render("N/A")
	} else {
		line += valueFn()
	}
	return line + "\n"
}

func (d *DashboardView) View() tea.View {
	return tea.NewView(d.ViewString())
}

func (d *DashboardView) SetSize(width, height int) tea.Cmd {
	d.width = width
	d.height = height
	d.headerPanel.SetWidth(width)
	return nil
}

func (d *DashboardView) StatusLine() string {
	return "s:services • R:region • P:profile • Ctrl+r:refresh • ?:help"
}

func (d *DashboardView) CanRefresh() bool {
	return true
}
