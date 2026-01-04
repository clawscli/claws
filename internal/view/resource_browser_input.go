package view

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/clipboard"
	"github.com/clawscli/claws/internal/dao"
)

func (r *ResourceBrowser) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if r.filterActive {
		return r.handleFilterInput(msg)
	}

	if len(r.filtered) > 0 && r.table.Cursor() < len(r.filtered) {
		if nav, cmd := r.handleNavigation(msg.String()); cmd != nil {
			return nav, cmd
		}
	}

	switch msg.String() {
	case "/":
		r.filterActive = true
		r.filterInput.Focus()
		return r, textinput.Blink
	case "ctrl+r":
		return r.handleRefresh()
	case "c":
		return r.handleClearFilter()
	case "esc":
		return r.handleEsc()
	case "m":
		return r.handleMark()
	case "M":
		return r.handleMetricsToggle()
	case "d", "enter":
		return r.handleEnter()
	case "a":
		return r.handleAction()
	case "tab":
		r.cycleResourceType(1)
		return r, tea.Batch(r.loadResources, r.spinner.Tick)
	case "shift+tab":
		r.cycleResourceType(-1)
		return r, tea.Batch(r.loadResources, r.spinner.Tick)
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		return r.handleNumberKey(msg.String())
	case "N":
		return r.handleLoadNextPage()
	case "y":
		return r.handleCopyID()
	case "Y":
		return r.handleCopyARN()
	}

	return nil, nil
}

func (r *ResourceBrowser) handleFilterInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if IsEscKey(msg) {
		r.filterActive = false
		r.filterInput.Blur()
		return r, nil
	}
	switch msg.String() {
	case "enter":
		r.filterActive = false
		r.filterInput.Blur()
		r.filterText = r.filterInput.Value()
		r.applyFilter()
		r.buildTable()
		return r, nil
	default:
		var cmd tea.Cmd
		r.filterInput, cmd = r.filterInput.Update(msg)
		r.filterText = r.filterInput.Value()
		r.applyFilter()
		r.buildTable()
		return r, cmd
	}
}

func (r *ResourceBrowser) handleRefresh() (tea.Model, tea.Cmd) {
	r.loading = true
	r.err = nil
	if r.metricsEnabled {
		r.metricsLoading = true
		r.metricsData = nil
	}
	return r, tea.Batch(r.loadResources, r.spinner.Tick)
}

func (r *ResourceBrowser) handleClearFilter() (tea.Model, tea.Cmd) {
	r.filterText = ""
	r.filterInput.SetValue("")
	r.fieldFilter = ""
	r.fieldFilterValue = ""
	r.markedResource = nil
	r.applyFilter()
	r.buildTable()
	return r, nil
}

func (r *ResourceBrowser) handleEsc() (tea.Model, tea.Cmd) {
	if r.markedResource != nil {
		r.markedResource = nil
		r.buildTable()
		return r, nil
	}
	return nil, nil
}

func (r *ResourceBrowser) handleMark() (tea.Model, tea.Cmd) {
	cursor := r.table.Cursor()
	if len(r.filtered) > 0 && cursor >= 0 && cursor < len(r.filtered) {
		resource := r.filtered[cursor]
		if r.markedResource != nil && r.markedResource.GetID() == resource.GetID() {
			r.markedResource = nil
		} else {
			r.markedResource = resource
		}
		r.buildTable()
	}
	return r, nil
}

func (r *ResourceBrowser) handleMetricsToggle() (tea.Model, tea.Cmd) {
	if r.getMetricSpec() != nil {
		r.metricsEnabled = !r.metricsEnabled
		if r.metricsEnabled && r.metricsData == nil {
			r.metricsLoading = true
			return r, r.loadMetricsCmd()
		}
		r.buildTable()
	}
	return r, nil
}

func (r *ResourceBrowser) handleEnter() (tea.Model, tea.Cmd) {
	cursor := r.table.Cursor()
	if len(r.filtered) > 0 && cursor >= 0 && cursor < len(r.filtered) {
		ctx, resource := r.contextForResource(r.filtered[cursor])
		if r.markedResource != nil && r.markedResource.GetID() != resource.GetID() {
			diffView := NewDiffView(ctx, dao.UnwrapResource(r.markedResource), resource, r.renderer, r.service, r.resourceType)
			return r, func() tea.Msg {
				return NavigateMsg{View: diffView}
			}
		}
		detailView := NewDetailView(ctx, resource, r.renderer, r.service, r.resourceType, r.registry, r.dao)
		return r, func() tea.Msg {
			return NavigateMsg{View: detailView}
		}
	}
	return r, nil
}

func (r *ResourceBrowser) handleAction() (tea.Model, tea.Cmd) {
	cursor := r.table.Cursor()
	if len(r.filtered) > 0 && cursor >= 0 && cursor < len(r.filtered) {
		if actions := action.Global.Get(r.service, r.resourceType); len(actions) > 0 {
			ctx, resource := r.contextForResource(r.filtered[cursor])
			actionMenu := NewActionMenu(ctx, resource, r.service, r.resourceType)
			return r, func() tea.Msg {
				return ShowModalMsg{Modal: &Modal{Content: actionMenu, Width: ModalWidthActionMenu}}
			}
		}
	}
	return r, nil
}

func (r *ResourceBrowser) handleNumberKey(key string) (tea.Model, tea.Cmd) {
	idx := int(key[0] - '1')
	if idx < len(r.resourceTypes) {
		r.resourceType = r.resourceTypes[idx]
		r.loading = true
		r.filterText = ""
		r.filterInput.SetValue("")
		r.markedResource = nil
		r.metricsEnabled = false
		r.metricsData = nil
		return r, tea.Batch(r.loadResources, r.spinner.Tick)
	}
	return r, nil
}

func (r *ResourceBrowser) handleLoadNextPage() (tea.Model, tea.Cmd) {
	if r.hasMorePages && !r.isLoadingMore && (r.nextPageToken != "" || len(r.nextPageTokens) > 0) {
		r.isLoadingMore = true
		return r, r.loadNextPage
	}
	return r, nil
}

func (r *ResourceBrowser) handleMouseWheel(msg tea.MouseWheelMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	r.table, cmd = r.table.Update(msg)
	return r, cmd
}

func (r *ResourceBrowser) handleMouseMotion(msg tea.MouseMotionMsg) (tea.Model, tea.Cmd) {
	if idx := r.getRowAtPosition(msg.Y); idx >= 0 && idx != r.table.Cursor() {
		r.table.SetCursor(idx)
	}
	return r, nil
}

func (r *ResourceBrowser) handleMouseClickMsg(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if msg.Button == tea.MouseLeft {
		if idx := r.getTabAtPosition(msg.X, msg.Y); idx >= 0 {
			return r.switchToTab(idx)
		}
		if len(r.filtered) > 0 {
			return r.handleMouseClick(msg.X, msg.Y)
		}
	}
	return r, nil
}

func (r *ResourceBrowser) getHeaderPanelHeight() int {
	headerStr := r.headerPanel.Render(r.service, r.resourceType, nil)
	return r.headerPanel.Height(headerStr)
}

func (r *ResourceBrowser) getRowAtPosition(y int) int {
	headerHeight := r.getHeaderPanelHeight() + 1 + 1
	if r.filterActive || r.filterText != "" {
		headerHeight++
	}
	tableHeaderRows := 1
	row := y - headerHeight - tableHeaderRows
	if row >= 0 && row < len(r.filtered) {
		return row
	}
	return -1
}

func (r *ResourceBrowser) handleMouseClick(x, y int) (tea.Model, tea.Cmd) {
	if row := r.getRowAtPosition(y); row >= 0 {
		r.table.SetCursor(row)
		return r.openDetailView()
	}
	return r, nil
}

func (r *ResourceBrowser) getTabAtPosition(x, y int) int {
	if len(r.tabPositions) == 0 {
		return -1
	}
	tabsY := r.getHeaderPanelHeight()
	if y != tabsY {
		return -1
	}
	for _, tp := range r.tabPositions {
		if x >= tp.startX && x < tp.endX {
			return tp.tabIdx
		}
	}
	return -1
}

func (r *ResourceBrowser) switchToTab(idx int) (tea.Model, tea.Cmd) {
	if idx < 0 || idx >= len(r.resourceTypes) {
		return r, nil
	}
	r.resourceType = r.resourceTypes[idx]
	r.markedResource = nil
	r.metricsEnabled = false
	r.metricsData = nil
	return r, r.loadResources
}

func (r *ResourceBrowser) openDetailView() (tea.Model, tea.Cmd) {
	cursor := r.table.Cursor()
	if len(r.filtered) == 0 || cursor < 0 || cursor >= len(r.filtered) {
		return r, nil
	}
	ctx, resource := r.contextForResource(r.filtered[cursor])
	detailView := NewDetailView(ctx, resource, r.renderer, r.service, r.resourceType, r.registry, r.dao)
	return r, func() tea.Msg {
		return NavigateMsg{View: detailView}
	}
}

func (r *ResourceBrowser) handleCopyID() (tea.Model, tea.Cmd) {
	cursor := r.table.Cursor()
	if len(r.filtered) > 0 && cursor >= 0 && cursor < len(r.filtered) {
		resource := dao.UnwrapResource(r.filtered[cursor])
		return r, clipboard.CopyID(resource.GetID())
	}
	return r, nil
}

func (r *ResourceBrowser) handleCopyARN() (tea.Model, tea.Cmd) {
	cursor := r.table.Cursor()
	if len(r.filtered) > 0 && cursor >= 0 && cursor < len(r.filtered) {
		resource := dao.UnwrapResource(r.filtered[cursor])
		if arn := resource.GetARN(); arn != "" {
			return r, clipboard.CopyARN(arn)
		}
		return r, clipboard.NoARN()
	}
	return r, nil
}
