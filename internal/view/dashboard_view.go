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
	"github.com/clawscli/claws/custom/cloudwatch/alarms"
	"github.com/clawscli/claws/custom/costexplorer/anomalies"
	"github.com/clawscli/claws/custom/costexplorer/costs"
	"github.com/clawscli/claws/custom/health/events"
	"github.com/clawscli/claws/custom/securityhub/findings"
	"github.com/clawscli/claws/custom/trustedadvisor/recommendations"
	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/ui"
)

type alarmItem struct {
	name     string
	state    string
	resource *alarms.AlarmResource
}

type costItem struct {
	service string
	cost    float64
	// No resource ref needed; use ServiceName for filter
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

type hitArea struct {
	y1, y2 int
	x1, x2 int
	target string
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
	warning   lipgloss.Style
	danger    lipgloss.Style
	success   lipgloss.Style
	dim       lipgloss.Style
	highlight lipgloss.Style
}

func newDashboardStyles() dashboardStyles {
	t := ui.Current()
	return dashboardStyles{
		warning:   lipgloss.NewStyle().Foreground(t.Warning),
		danger:    lipgloss.NewStyle().Foreground(t.Danger),
		success:   lipgloss.NewStyle().Foreground(t.Success),
		dim:       lipgloss.NewStyle().Foreground(t.TextMuted),
		highlight: lipgloss.NewStyle().Background(t.Selection).Foreground(t.SelectionText),
	}
}

// Panel indices for hover detection (must be 0-3 to match hitAreas order)
const (
	panelCost = iota
	panelOperations
	panelSecurity
	panelOptimization
)

const (
	minPanelWidth  = 30
	minPanelHeight = 6
	panelGap       = 1

	dashboardMaxRecords = 100

	targetCost         = "costexplorer/costs"
	targetOperations   = "health/events"
	targetSecurity     = "securityhub/findings"
	targetOptimization = "trustedadvisor/recommendations"

	// Cost panel layout constants
	costValueWidth     = 9  // Width for cost value display (e.g., "   12345")
	costPadding        = 2  // Padding between name and bar
	minCostBarWidth    = 8  // Minimum bar graph width
	minCostNameWidth   = 15 // Minimum service name width
	costNameWidthRatio = 60 // Name takes 60% of available space, bar 40%

	bulletIndentWidth = 4
)

func renderPanel(title, content string, width, height int, t *ui.Theme, hovered bool) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(t.Primary)
	boxHeight := height - 1
	if boxHeight < 3 {
		boxHeight = 3
	}

	borderColor := t.Border
	if hovered {
		borderColor = t.Primary
	}

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
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

	hitAreas         []hitArea
	hoverIdx         int
	focusedPanel     int
	focusedRow       int
	lastPanelWidth   int
	lastPanelHeight  int
	lastHeaderHeight int

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
		hoverIdx:       -1,
		focusedPanel:   panelCost,
		focusedRow:     -1,
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
	if d.ctx.Err() != nil {
		return healthErrorMsg{err: d.ctx.Err()}
	}

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
				items = append(items, taItem{name: rr.Name(), status: status, savings: rr.EstimatedMonthlySavings(), resource: rr})
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
		case "ctrl+r":
			return d.Update(RefreshMsg{})
		case "h", "left":
			d.movePanelFocus(-1, 0)
		case "l", "right":
			d.movePanelFocus(1, 0)
		case "j", "down":
			d.moveRowFocus(1)
		case "k", "up":
			d.moveRowFocus(-1)
		case "tab":
			d.cyclePanelFocus(1)
		case "shift+tab":
			d.cyclePanelFocus(-1)
		case "enter":
			return d.activateCurrentRow()
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

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			panelIdx, rowIdx := d.hitTestRow(msg.X, msg.Y)
			if panelIdx >= 0 {
				d.focusedPanel = panelIdx
				d.focusedRow = rowIdx
				return d.activateCurrentRow()
			}
		}

	case tea.MouseMotionMsg:
		panelIdx, rowIdx := d.hitTestRow(msg.X, msg.Y)
		d.hoverIdx = panelIdx
		if panelIdx >= 0 {
			d.focusedPanel = panelIdx
			d.focusedRow = rowIdx
		}
	}

	return d, nil
}

func (d *DashboardView) hitTest(x, y int) string {
	for _, h := range d.hitAreas {
		if y >= h.y1 && y <= h.y2 && x >= h.x1 && x <= h.x2 {
			return h.target
		}
	}
	return ""
}

func (d *DashboardView) hitTestIdx(x, y int) int {
	for i, h := range d.hitAreas {
		if y >= h.y1 && y <= h.y2 && x >= h.x1 && x <= h.x2 {
			return i
		}
	}
	return -1
}

func (d *DashboardView) hitTestRow(x, y int) (panelIdx, rowIdx int) {
	panelIdx = d.hitTestIdx(x, y)
	if panelIdx < 0 {
		return -1, -1
	}

	h := d.hitAreas[panelIdx]
	contentStartY := h.y1 + 2

	rowY := y - contentStartY
	if rowY < 0 {
		return panelIdx, -1
	}

	rowIdx = d.computeRowFromContentLine(panelIdx, rowY)
	return panelIdx, rowIdx
}

func (d *DashboardView) computeRowFromContentLine(panelIdx, lineY int) int {
	switch panelIdx {
	case panelCost:
		if lineY == 0 {
			return -1
		}
		rowIdx := lineY - 1
		if rowIdx >= 0 && rowIdx < len(d.costTop) {
			return rowIdx
		}

	case panelOperations:
		line := 0
		if len(d.alarms) > 0 {
			line++
			for i := 0; i < len(d.alarms); i++ {
				if lineY == line {
					return i
				}
				line++
			}
		} else {
			line++
		}
		if len(d.healthItems) > 0 {
			line++
			alarmCount := len(d.alarms)
			for i := 0; i < len(d.healthItems); i++ {
				if lineY == line {
					return alarmCount + i
				}
				line++
			}
		}

	case panelSecurity:
		headerLines := 0
		for _, item := range d.secItems {
			if item.severity == "CRITICAL" {
				headerLines = 1
				break
			}
		}
		for _, item := range d.secItems {
			if item.severity == "HIGH" {
				if headerLines == 0 {
					headerLines = 1
				} else {
					headerLines = 2
				}
				break
			}
		}
		rowIdx := lineY - headerLines
		if rowIdx >= 0 && rowIdx < len(d.secItems) {
			return rowIdx
		}

	case panelOptimization:
		headerLines := 0
		for _, item := range d.taItems {
			if item.status == "error" {
				headerLines++
				break
			}
		}
		for _, item := range d.taItems {
			if item.status == "warning" {
				headerLines++
				break
			}
		}
		if d.taSavings > 0 {
			headerLines++
		}
		rowIdx := lineY - headerLines
		if rowIdx >= 0 && rowIdx < len(d.taItems) {
			return rowIdx
		}
	}
	return -1
}

func (d *DashboardView) navigateTo(target string) (tea.Model, tea.Cmd) {
	parts := strings.SplitN(target, "/", 2)
	if len(parts) != 2 {
		return d, nil
	}

	browser := NewResourceBrowserWithType(d.ctx, d.registry, parts[0], parts[1])
	return d, func() tea.Msg {
		return NavigateMsg{View: browser}
	}
}

func (d *DashboardView) navigateToFiltered(service, resType, filterKey, filterVal string) (tea.Model, tea.Cmd) {
	browser := NewResourceBrowserWithFilter(d.ctx, d.registry, service, resType, filterKey, filterVal)
	return d, func() tea.Msg {
		return NavigateMsg{View: browser}
	}
}

func (d *DashboardView) getRowCount(panelIdx int) int {
	switch panelIdx {
	case panelCost:
		return len(d.costTop)
	case panelOperations:
		return len(d.alarms) + len(d.healthItems)
	case panelSecurity:
		return len(d.secItems)
	case panelOptimization:
		return len(d.taItems)
	}
	return 0
}

func (d *DashboardView) clampFocusedRow() {
	count := d.getRowCount(d.focusedPanel)
	if count == 0 {
		d.focusedRow = -1
	} else if d.focusedRow >= count {
		d.focusedRow = count - 1
	} else if d.focusedRow < 0 {
		d.focusedRow = 0
	}
}

func (d *DashboardView) movePanelFocus(dx, dy int) {
	col := d.focusedPanel % 2
	row := d.focusedPanel / 2

	col = (col + dx + 2) % 2
	row = (row + dy + 2) % 2

	d.focusedPanel = row*2 + col
	d.clampFocusedRow()
}

func (d *DashboardView) moveRowFocus(delta int) {
	count := d.getRowCount(d.focusedPanel)
	if count == 0 {
		return
	}
	if d.focusedRow < 0 {
		if delta > 0 {
			d.focusedRow = 0
		} else {
			d.focusedRow = count - 1
		}
		return
	}
	d.focusedRow += delta
	if d.focusedRow < 0 {
		d.focusedRow = 0
	} else if d.focusedRow >= count {
		d.focusedRow = count - 1
	}
}

func (d *DashboardView) cyclePanelFocus(delta int) {
	d.focusedPanel = (d.focusedPanel + delta + 4) % 4
	d.clampFocusedRow()
}

func (d *DashboardView) panelTarget(panelIdx int) string {
	switch panelIdx {
	case panelCost:
		return targetCost
	case panelOperations:
		return targetOperations
	case panelSecurity:
		return targetSecurity
	case panelOptimization:
		return targetOptimization
	}
	return ""
}

func (d *DashboardView) openDetailViewForResource(resource dao.Resource, service, resType string) (tea.Model, tea.Cmd) {
	renderer, err := d.registry.GetRenderer(service, resType)
	if err != nil {
		return d.navigateTo(service + "/" + resType)
	}
	daoInst, err := d.registry.GetDAO(d.ctx, service, resType)
	if err != nil {
		daoInst = nil
	}
	detailView := NewDetailView(d.ctx, resource, renderer, service, resType, d.registry, daoInst)
	return d, func() tea.Msg {
		return NavigateMsg{View: detailView}
	}
}

func (d *DashboardView) activateCurrentRow() (tea.Model, tea.Cmd) {
	if d.focusedRow < 0 {
		return d.navigateTo(d.panelTarget(d.focusedPanel))
	}

	switch d.focusedPanel {
	case panelCost:
		if d.focusedRow < len(d.costTop) {
			item := d.costTop[d.focusedRow]
			return d.navigateToFiltered("costexplorer", "costs", "ServiceName", item.service)
		}

	case panelOperations:
		alarmCount := len(d.alarms)
		if d.focusedRow < alarmCount {
			item := d.alarms[d.focusedRow]
			if item.resource != nil {
				return d.openDetailViewForResource(item.resource, "cloudwatch", "alarms")
			}
		} else {
			healthIdx := d.focusedRow - alarmCount
			if healthIdx < len(d.healthItems) {
				item := d.healthItems[healthIdx]
				if item.resource != nil {
					return d.openDetailViewForResource(item.resource, "health", "events")
				}
			}
		}

	case panelSecurity:
		if d.focusedRow < len(d.secItems) {
			item := d.secItems[d.focusedRow]
			if item.resource != nil {
				return d.openDetailViewForResource(item.resource, "securityhub", "findings")
			}
		}

	case panelOptimization:
		if d.focusedRow < len(d.taItems) {
			item := d.taItems[d.focusedRow]
			if item.resource != nil {
				return d.openDetailViewForResource(item.resource, "trustedadvisor", "recommendations")
			}
		}
	}

	return d.navigateTo(d.panelTarget(d.focusedPanel))
}

func (d *DashboardView) isLoading() bool {
	return d.alarmLoading || d.costLoading || d.anomalyLoading ||
		d.healthLoading || d.secLoading || d.taLoading
}

func (d *DashboardView) ViewString() string {
	header := d.headerPanel.RenderHome()
	headerHeight := d.headerPanel.Height(header)
	t := ui.Current()

	panelWidth := d.calcPanelWidth()
	panelHeight := d.calcPanelHeight(headerHeight)
	contentWidth := panelWidth - 4
	contentHeight := panelHeight - 3

	costFocusRow := -1
	opsFocusRow := -1
	secFocusRow := -1
	optFocusRow := -1
	if d.focusedPanel == panelCost {
		costFocusRow = d.focusedRow
	} else if d.focusedPanel == panelOperations {
		opsFocusRow = d.focusedRow
	} else if d.focusedPanel == panelSecurity {
		secFocusRow = d.focusedRow
	} else if d.focusedPanel == panelOptimization {
		optFocusRow = d.focusedRow
	}

	costContent := d.renderCostContent(contentWidth, contentHeight, t, costFocusRow)
	opsContent := d.renderOpsContent(contentWidth, contentHeight, opsFocusRow)
	secContent := d.renderSecurityContent(contentWidth, contentHeight, secFocusRow)
	optContent := d.renderOptimizationContent(contentWidth, contentHeight, optFocusRow)

	costPanel := renderPanel("Cost", costContent, panelWidth, panelHeight, t, d.hoverIdx == panelCost)
	opsPanel := renderPanel("Operations", opsContent, panelWidth, panelHeight, t, d.hoverIdx == panelOperations)
	secPanel := renderPanel("Security", secContent, panelWidth, panelHeight, t, d.hoverIdx == panelSecurity)
	optPanel := renderPanel("Optimization", optContent, panelWidth, panelHeight, t, d.hoverIdx == panelOptimization)

	gap := strings.Repeat(" ", panelGap)
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, costPanel, gap, opsPanel)
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, secPanel, gap, optPanel)
	grid := lipgloss.JoinVertical(lipgloss.Left, topRow, bottomRow)

	hint := d.styles.dim.Render("h/l:panel • j/k:row • enter:select • s:services • Ctrl+r:refresh")

	if panelWidth != d.lastPanelWidth || panelHeight != d.lastPanelHeight || headerHeight != d.lastHeaderHeight {
		d.buildHitAreas(panelWidth, panelHeight, headerHeight)
		d.lastPanelWidth = panelWidth
		d.lastPanelHeight = panelHeight
		d.lastHeaderHeight = headerHeight
	}

	return header + "\n" + grid + "\n" + hint
}

func (d *DashboardView) buildHitAreas(panelWidth, panelHeight, headerHeight int) {
	d.hitAreas = d.hitAreas[:0]

	topRowY := headerHeight + 1
	bottomRowY := topRowY + panelHeight

	leftX1, leftX2 := 0, panelWidth
	rightX1, rightX2 := panelWidth+panelGap, panelWidth+panelGap+panelWidth

	d.hitAreas = append(d.hitAreas,
		hitArea{y1: topRowY, y2: topRowY + panelHeight - 1, x1: leftX1, x2: leftX2, target: targetCost},
		hitArea{y1: topRowY, y2: topRowY + panelHeight - 1, x1: rightX1, x2: rightX2, target: targetOperations},
		hitArea{y1: bottomRowY, y2: bottomRowY + panelHeight - 1, x1: leftX1, x2: leftX2, target: targetSecurity},
		hitArea{y1: bottomRowY, y2: bottomRowY + panelHeight - 1, x1: rightX1, x2: rightX2, target: targetOptimization},
	)
}

func (d *DashboardView) calcPanelWidth() int {
	return max((d.width-panelGap)/2, minPanelWidth)
}

func (d *DashboardView) calcPanelHeight(headerHeight int) int {
	hintHeight := 2
	available := d.height - headerHeight - hintHeight
	return max(available/2, minPanelHeight)
}

func (d *DashboardView) renderCostContent(contentWidth, contentHeight int, t *ui.Theme, focusRow int) string {
	s := d.styles
	var lines []string

	if d.costLoading {
		lines = append(lines, d.spinner.View()+" loading...")
	} else if d.costErr != nil {
		lines = append(lines, s.dim.Render("Cost: N/A"))
	} else {
		lines = append(lines, "MTD: "+appaws.FormatMoney(d.costMTD, ""))

		if len(d.costTop) > 0 {
			maxCost := d.costTop[0].cost
			available := contentWidth - costValueWidth - costPadding
			nameWidth := available * costNameWidthRatio / 100
			barWidth := available - nameWidth
			if nameWidth < minCostNameWidth {
				nameWidth = minCostNameWidth
			}
			if barWidth < minCostBarWidth {
				barWidth = minCostBarWidth
			}
			maxServices := contentHeight - 2
			if maxServices < 3 {
				maxServices = 3
			}
			showCount := min(len(d.costTop), maxServices)

			for i := 0; i < showCount; i++ {
				c := d.costTop[i]
				bar := renderBar(c.cost, maxCost, barWidth, t)
				name := truncateValue(c.service, nameWidth)
				line := fmt.Sprintf("%-*s %s %8.0f", nameWidth, name, bar, c.cost)
				if i == focusRow {
					line = s.highlight.Render(line)
				}
				lines = append(lines, line)
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

func (d *DashboardView) renderOpsContent(contentWidth, contentHeight int, focusRow int) string {
	s := d.styles
	var lines []string
	alarmCount := len(d.alarms)

	if d.alarmLoading {
		lines = append(lines, "Alarms: "+d.spinner.View())
	} else if d.alarmErr != nil {
		lines = append(lines, s.dim.Render("Alarms: N/A"))
	} else if alarmCount > 0 {
		lines = append(lines, s.danger.Render(fmt.Sprintf("Alarms: %d in ALARM", alarmCount)))
		maxShow := min(alarmCount, contentHeight-3)
		for i := 0; i < maxShow; i++ {
			line := "  " + s.danger.Render("• ") + truncateValue(d.alarms[i].name, contentWidth-bulletIndentWidth)
			if i == focusRow {
				line = s.highlight.Render(line)
			}
			lines = append(lines, line)
		}
	} else {
		lines = append(lines, "Alarms: "+s.success.Render("0 ✓"))
	}

	if d.healthLoading {
		lines = append(lines, "Health: "+d.spinner.View())
	} else if d.healthErr != nil {
		lines = append(lines, s.dim.Render("Health: N/A"))
	} else if len(d.healthItems) > 0 {
		lines = append(lines, s.warning.Render(fmt.Sprintf("Health: %d open", len(d.healthItems))))
		remaining := contentHeight - len(lines) - 1
		maxShow := min(len(d.healthItems), remaining)
		for i := 0; i < maxShow; i++ {
			h := d.healthItems[i]
			line := "  " + s.warning.Render("• ") + truncateValue(h.service+": "+h.eventType, contentWidth-bulletIndentWidth)
			if alarmCount+i == focusRow {
				line = s.highlight.Render(line)
			}
			lines = append(lines, line)
		}
	} else {
		lines = append(lines, "Health: "+s.success.Render("0 open ✓"))
	}

	return strings.Join(lines, "\n")
}

func (d *DashboardView) renderSecurityContent(contentWidth, contentHeight int, focusRow int) string {
	s := d.styles
	var lines []string

	if d.secLoading {
		lines = append(lines, d.spinner.View()+" loading...")
	} else if d.secErr != nil {
		lines = append(lines, s.dim.Render("Security: N/A"))
	} else if len(d.secItems) > 0 {
		var critical, high int
		for _, item := range d.secItems {
			if item.severity == "CRITICAL" {
				critical++
			} else if item.severity == "HIGH" {
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
			line := "  " + style.Render("• ") + truncateValue(item.title, contentWidth-bulletIndentWidth)
			if i == focusRow {
				line = s.highlight.Render(line)
			}
			lines = append(lines, line)
		}
	} else {
		lines = append(lines, s.success.Render("No critical/high ✓"))
	}

	return strings.Join(lines, "\n")
}

func (d *DashboardView) renderOptimizationContent(contentWidth, contentHeight int, focusRow int) string {
	s := d.styles
	var lines []string

	if d.taLoading {
		lines = append(lines, d.spinner.View()+" loading...")
	} else if d.taErr != nil {
		lines = append(lines, s.dim.Render("Optimization: N/A"))
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
			lines = append(lines, s.success.Render("Savings: "+appaws.FormatMoney(d.taSavings, "")+"/mo 💰"))
		}
		if len(d.taItems) > 0 {
			maxShow := min(len(d.taItems), contentHeight-len(lines)-1)
			for i := 0; i < maxShow; i++ {
				item := d.taItems[i]
				style := s.warning
				if item.status == "error" {
					style = s.danger
				}
				line := "  " + style.Render("• ") + truncateValue(item.name, contentWidth-bulletIndentWidth)
				if i == focusRow {
					line = s.highlight.Render(line)
				}
				lines = append(lines, line)
			}
		}
		if len(lines) == 0 {
			lines = append(lines, s.success.Render("All good ✓"))
		}
	}

	return strings.Join(lines, "\n")
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
	return "h/l:panel • j/k:row • enter:select • s:services • R:region • P:profile • Ctrl+r:refresh • ?:help"
}

func (d *DashboardView) CanRefresh() bool {
	return true
}
