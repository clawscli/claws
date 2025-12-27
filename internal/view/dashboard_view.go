package view

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

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

type alarmItem struct {
	name  string
	state string
}

type costItem struct {
	service string
	cost    float64
}

type healthItem struct {
	service   string
	eventType string
}

type securityItem struct {
	title    string
	severity string
}

type taItem struct {
	name    string
	status  string
	savings float64
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

const (
	minPanelWidth  = 30
	minPanelHeight = 6
	panelGap       = 1
)

func renderPanel(title, content string, width, height int, t *ui.Theme) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(t.Primary)
	boxHeight := height - 1
	if boxHeight < 3 {
		boxHeight = 3
	}
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Padding(0, 1).
		Width(width).
		Height(boxHeight)

	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(title),
		borderStyle.Render(content))
}

func renderBar(value, max float64, width int, t *ui.Theme) string {
	if max <= 0 || width <= 0 {
		return ""
	}
	ratio := value / max
	if ratio > 1 {
		ratio = 1
	}
	filled := int(ratio * float64(width))
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}

	barStyle := lipgloss.NewStyle().Foreground(t.Accent)
	emptyStyle := lipgloss.NewStyle().Foreground(t.TextMuted)

	return barStyle.Render(strings.Repeat("█", filled)) +
		emptyStyle.Render(strings.Repeat("░", width-filled))
}

type DashboardView struct {
	ctx         context.Context
	registry    *registry.Registry
	width       int
	height      int
	headerPanel *HeaderPanel
	spinner     spinner.Model
	styles      dashboardStyles

	alarms       []alarmItem
	alarmLoading bool
	alarmErr     error

	costMTD     float64
	costTop     []costItem
	costLoading bool
	costErr     error

	anomalyCount   int
	anomalyLoading bool
	anomalyErr     error

	healthItems   []healthItem
	healthLoading bool
	healthErr     error

	secItems   []securityItem
	secLoading bool
	secErr     error

	taItems   []taItem
	taSavings float64
	taLoading bool
	taErr     error
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

	var items []alarmItem
	for _, a := range output.MetricAlarms {
		items = append(items, alarmItem{name: *a.AlarmName, state: string(a.StateValue)})
	}
	for _, a := range output.CompositeAlarms {
		items = append(items, alarmItem{name: *a.AlarmName, state: string(a.StateValue)})
	}
	return alarmLoadedMsg{items: items}
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

	return costLoadedMsg{mtd: total, topCosts: items}
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

	var items []healthItem
	for _, r := range resources {
		if er, ok := r.(*events.EventResource); ok {
			if er.StatusCode() != "closed" {
				items = append(items, healthItem{service: er.Service(), eventType: er.EventTypeCode()})
			}
		}
	}
	return healthLoadedMsg{items: items}
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

	var items []securityItem
	for _, r := range resources {
		if fr, ok := r.(*findings.FindingResource); ok {
			sev := fr.Severity()
			if sev == "CRITICAL" || sev == "HIGH" {
				items = append(items, securityItem{title: fr.Title(), severity: sev})
			}
		}
	}
	return securityLoadedMsg{items: items}
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

	var items []taItem
	var totalSavings float64
	for _, r := range resources {
		if rr, ok := r.(*recommendations.RecommendationResource); ok {
			status := rr.Status()
			if status == "error" || status == "warning" {
				items = append(items, taItem{name: rr.Name(), status: status, savings: rr.EstimatedMonthlySavings()})
			}
			totalSavings += rr.EstimatedMonthlySavings()
		}
	}
	return taLoadedMsg{items: items, savings: totalSavings}
}

func (d *DashboardView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case alarmLoadedMsg:
		d.alarmLoading = false
		d.alarms = msg.items
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
		d.healthItems = msg.items
		return d, nil
	case healthErrorMsg:
		d.healthLoading = false
		d.healthErr = msg.err
		return d, nil

	case securityLoadedMsg:
		d.secLoading = false
		d.secItems = msg.items
		return d, nil
	case securityErrorMsg:
		d.secLoading = false
		d.secErr = msg.err
		return d, nil

	case taLoadedMsg:
		d.taLoading = false
		d.taItems = msg.items
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
	t := ui.Current()

	panelWidth := d.calcPanelWidth()
	panelHeight := d.calcPanelHeight()
	contentWidth := panelWidth - 4
	contentHeight := panelHeight - 3

	costContent := d.renderCostContent(contentWidth, contentHeight)
	opsContent := d.renderOpsContent(contentWidth, contentHeight)
	secContent := d.renderSecurityContent(contentWidth, contentHeight)
	optContent := d.renderOptimizationContent(contentWidth, contentHeight)

	costPanel := renderPanel("Cost", costContent, panelWidth, panelHeight, t)
	opsPanel := renderPanel("Operations", opsContent, panelWidth, panelHeight, t)
	secPanel := renderPanel("Security", secContent, panelWidth, panelHeight, t)
	optPanel := renderPanel("Optimization", optContent, panelWidth, panelHeight, t)

	gap := strings.Repeat(" ", panelGap)
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, costPanel, gap, opsPanel)
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, secPanel, gap, optPanel)
	grid := lipgloss.JoinVertical(lipgloss.Left, topRow, bottomRow)

	hint := d.styles.dim.Render("s:services • Ctrl+r:refresh")

	return header + "\n" + grid + "\n" + hint
}

func (d *DashboardView) calcPanelWidth() int {
	return max((d.width-panelGap)/2, minPanelWidth)
}

func (d *DashboardView) calcPanelHeight() int {
	headerHeight := 3
	hintHeight := 2
	available := d.height - headerHeight - hintHeight
	return max(available/2, minPanelHeight)
}

func (d *DashboardView) renderCostContent(contentWidth, contentHeight int) string {
	s := d.styles
	t := ui.Current()
	var lines []string

	if d.costLoading {
		lines = append(lines, d.spinner.View()+" loading...")
	} else if d.costErr != nil {
		lines = append(lines, s.dim.Render("N/A"))
	} else {
		lines = append(lines, fmt.Sprintf("MTD: $%.2f", d.costMTD))

		if len(d.costTop) > 0 {
			maxCost := d.costTop[0].cost
			const costWidth = 9
			const minBarWidth = 8
			const minNameWidth = 15
			available := contentWidth - costWidth - 2
			nameWidth := available * 60 / 100
			barWidth := available - nameWidth
			if nameWidth < minNameWidth {
				nameWidth = minNameWidth
			}
			if barWidth < minBarWidth {
				barWidth = minBarWidth
			}
			maxServices := contentHeight - 2
			if maxServices < 3 {
				maxServices = 3
			}
			showCount := min(len(d.costTop), maxServices)

			for i := 0; i < showCount; i++ {
				c := d.costTop[i]
				bar := renderBar(c.cost, maxCost, barWidth, t)
				name := truncate(c.service, nameWidth)
				lines = append(lines, fmt.Sprintf("%-*s %s %8.0f", nameWidth, name, bar, c.cost))
			}
		}

		if d.anomalyLoading {
			lines = append(lines, "Anomalies: "+d.spinner.View())
		} else if d.anomalyErr != nil {
			lines = append(lines, "Anomalies: "+s.dim.Render("N/A"))
		} else if d.anomalyCount > 0 {
			lines = append(lines, "Anomalies: "+s.warning.Render(fmt.Sprintf("%d", d.anomalyCount)))
		} else {
			lines = append(lines, "Anomalies: "+s.success.Render("0"))
		}
	}

	return strings.Join(lines, "\n")
}

func (d *DashboardView) renderOpsContent(contentWidth, contentHeight int) string {
	s := d.styles
	var lines []string

	if d.alarmLoading {
		lines = append(lines, "Alarms: "+d.spinner.View())
	} else if d.alarmErr != nil {
		lines = append(lines, "Alarms: "+s.dim.Render("N/A"))
	} else if len(d.alarms) > 0 {
		lines = append(lines, s.danger.Render(fmt.Sprintf("Alarms: %d in ALARM", len(d.alarms))))
		maxShow := min(len(d.alarms), contentHeight-3)
		for i := 0; i < maxShow; i++ {
			lines = append(lines, "  "+s.danger.Render("• ")+truncate(d.alarms[i].name, contentWidth-4))
		}
	} else {
		lines = append(lines, "Alarms: "+s.success.Render("0 ✓"))
	}

	if d.healthLoading {
		lines = append(lines, "Health: "+d.spinner.View())
	} else if d.healthErr != nil {
		lines = append(lines, "Health: "+s.dim.Render("N/A"))
	} else if len(d.healthItems) > 0 {
		lines = append(lines, s.warning.Render(fmt.Sprintf("Health: %d open", len(d.healthItems))))
		remaining := contentHeight - len(lines) - 1
		maxShow := min(len(d.healthItems), remaining)
		for i := 0; i < maxShow; i++ {
			h := d.healthItems[i]
			lines = append(lines, "  "+s.warning.Render("• ")+truncate(h.service+": "+h.eventType, contentWidth-4))
		}
	} else {
		lines = append(lines, "Health: "+s.success.Render("0 open ✓"))
	}

	return strings.Join(lines, "\n")
}

func (d *DashboardView) renderSecurityContent(contentWidth, contentHeight int) string {
	s := d.styles
	var lines []string

	if d.secLoading {
		lines = append(lines, d.spinner.View()+" loading...")
	} else if d.secErr != nil {
		lines = append(lines, s.dim.Render("N/A"))
	} else if len(d.secItems) > 0 {
		var critical, high int
		for _, item := range d.secItems {
			if item.severity == "CRITICAL" {
				critical++
			} else {
				high++
			}
		}
		if critical > 0 {
			lines = append(lines, s.danger.Render(fmt.Sprintf("Critical: %d 🔴", critical)))
		}
		if high > 0 {
			lines = append(lines, s.warning.Render(fmt.Sprintf("High: %d 🟠", high)))
		}
		maxShow := min(len(d.secItems), contentHeight-len(lines)-1)
		for i := 0; i < maxShow; i++ {
			item := d.secItems[i]
			style := s.warning
			if item.severity == "CRITICAL" {
				style = s.danger
			}
			lines = append(lines, "  "+style.Render("• ")+truncate(item.title, contentWidth-4))
		}
	} else {
		lines = append(lines, s.success.Render("No critical/high ✓"))
	}

	return strings.Join(lines, "\n")
}

func (d *DashboardView) renderOptimizationContent(contentWidth, contentHeight int) string {
	s := d.styles
	var lines []string

	if d.taLoading {
		lines = append(lines, d.spinner.View()+" loading...")
	} else if d.taErr != nil {
		lines = append(lines, s.dim.Render("N/A"))
	} else {
		var errors, warnings int
		for _, item := range d.taItems {
			if item.status == "error" {
				errors++
			} else {
				warnings++
			}
		}
		if errors > 0 {
			lines = append(lines, s.danger.Render(fmt.Sprintf("Errors: %d", errors)))
		}
		if warnings > 0 {
			lines = append(lines, s.warning.Render(fmt.Sprintf("Warnings: %d", warnings)))
		}
		if d.taSavings > 0 {
			lines = append(lines, s.success.Render(fmt.Sprintf("Savings: $%.0f/mo 💰", d.taSavings)))
		}
		if len(d.taItems) > 0 {
			maxShow := min(len(d.taItems), contentHeight-len(lines)-1)
			for i := 0; i < maxShow; i++ {
				item := d.taItems[i]
				style := s.warning
				if item.status == "error" {
					style = s.danger
				}
				lines = append(lines, "  "+style.Render("• ")+truncate(item.name, contentWidth-4))
			}
		}
		if len(lines) == 0 {
			lines = append(lines, s.success.Render("All good ✓"))
		}
	}

	return strings.Join(lines, "\n")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
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
