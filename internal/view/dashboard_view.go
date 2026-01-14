package view

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/clawscli/claws/internal/dao"
	navmsg "github.com/clawscli/claws/internal/msg"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/ui"
)

type hitArea struct {
	y1, y2 int
	x1, x2 int
	target string
}

type dashboardStyles struct {
	text      lipgloss.Style
	warning   lipgloss.Style
	danger    lipgloss.Style
	success   lipgloss.Style
	dim       lipgloss.Style
	highlight lipgloss.Style
}

func newDashboardStyles() dashboardStyles {
	return dashboardStyles{
		text:      ui.TextStyle(),
		warning:   ui.WarningStyle(),
		danger:    ui.DangerStyle(),
		success:   ui.SuccessStyle(),
		dim:       ui.MutedStyle(),
		highlight: ui.SelectedStyle(),
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
		return d.handleKeyPress(msg)

	case RefreshMsg:
		return d.handleRefresh()
	case ThemeChangedMsg:
		d.styles = newDashboardStyles()
		d.headerPanel.ReloadStyles()
		return d, nil
	case CompactHeaderChangedMsg:
		d.lastHeaderHeight = 0 // force hitAreas rebuild
		return d, nil

	case tea.MouseClickMsg:
		return d.handleMouseClick(msg)

	case tea.MouseMotionMsg:
		d.handleMouseMotion(msg)

	case navmsg.ProfilesChangedMsg:
		return d.handleRefresh()

	case navmsg.RegionChangedMsg:
		return d.handleRefresh()
	}

	return d, nil
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
	contentStartY := h.y1 + 1

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
			for i := range d.alarms {
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
			for i := range d.healthItems {
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
	d.hoverIdx = d.focusedPanel
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
			return d.navigateToFiltered("ce", "costs", "ServiceName", item.service)
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
	switch d.focusedPanel {
	case panelCost:
		costFocusRow = d.focusedRow
	case panelOperations:
		opsFocusRow = d.focusedRow
	case panelSecurity:
		secFocusRow = d.focusedRow
	case panelOptimization:
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

	if panelWidth != d.lastPanelWidth || panelHeight != d.lastPanelHeight || headerHeight != d.lastHeaderHeight {
		d.buildHitAreas(panelWidth, panelHeight, headerHeight)
		d.lastPanelWidth = panelWidth
		d.lastPanelHeight = panelHeight
		d.lastHeaderHeight = headerHeight
	}

	return header + "\n" + grid
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
	available := d.height - headerHeight + 1
	return max(available/2, minPanelHeight)
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
