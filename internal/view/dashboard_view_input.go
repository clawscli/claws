package view

import (
	tea "charm.land/bubbletea/v2"
)

func (d *DashboardView) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "~":
		// Toggle to ServiceBrowser
		browser := NewServiceBrowser(d.ctx, d.registry)
		return d, func() tea.Msg {
			return NavigateMsg{View: browser, ClearStack: false}
		}
	case "s":
		browser := NewServiceBrowser(d.ctx, d.registry)
		return d, func() tea.Msg {
			return NavigateMsg{View: browser}
		}
	case "ctrl+r":
		return d.Update(RefreshMsg{})
	case "h", "left":
		d.cyclePanelFocus(-1)
	case "l", "right":
		d.cyclePanelFocus(1)
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
	return d, nil
}

func (d *DashboardView) handleMouseClick(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if msg.Button == tea.MouseLeft {
		panelIdx, rowIdx := d.hitTestRow(msg.X, msg.Y)
		if panelIdx >= 0 {
			d.focusedPanel = panelIdx
			d.focusedRow = rowIdx
			return d.activateCurrentRow()
		}
	}
	return d, nil
}

func (d *DashboardView) handleMouseMotion(msg tea.MouseMotionMsg) {
	panelIdx, rowIdx := d.hitTestRow(msg.X, msg.Y)
	d.hoverIdx = panelIdx
	if panelIdx >= 0 {
		d.focusedPanel = panelIdx
		d.focusedRow = rowIdx
	}
}

func (d *DashboardView) handleRefresh() (tea.Model, tea.Cmd) {
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
