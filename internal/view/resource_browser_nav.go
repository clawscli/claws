package view

import (
	"fmt"
	"slices"

	tea "charm.land/bubbletea/v2"

	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/dao"
)

// handleNavigation processes navigation key shortcuts
func (r *ResourceBrowser) handleNavigation(key string) (tea.Model, tea.Cmd) {
	if r.renderer == nil || len(r.filtered) == 0 {
		return nil, nil
	}

	ctx, resource := r.contextForResource(r.filtered[r.table.Cursor()])

	helper := &NavigationHelper{
		Ctx:      ctx,
		Registry: r.registry,
		Renderer: r.renderer,
	}

	if cmd := helper.HandleKey(key, resource); cmd != nil {
		return r, cmd
	}

	return nil, nil
}

// cycleResourceType switches to the next/previous resource type
func (r *ResourceBrowser) cycleResourceType(delta int) {
	if len(r.resourceTypes) <= 1 {
		return
	}

	currentIdx := slices.Index(r.resourceTypes, r.resourceType)
	if currentIdx < 0 {
		currentIdx = 0
	}

	newIdx := (currentIdx + delta + len(r.resourceTypes)) % len(r.resourceTypes)
	r.resourceType = r.resourceTypes[newIdx]
	r.loading = true
	r.filterText = ""
	r.filterInput.SetValue("")
	r.markedResource = nil
	r.metricsEnabled = false
	r.metricsData = nil
}

// StatusLine implements View interface
func (r *ResourceBrowser) StatusLine() string {
	total := len(r.resources)
	shown := len(r.filtered)
	hasActions := len(action.Global.Get(r.service, r.resourceType)) > 0

	// Build auto-reload info
	autoReloadInfo := ""
	if r.autoReload {
		autoReloadInfo = fmt.Sprintf(" (auto-refresh: %s)", r.autoReloadInterval)
	}

	// Build filter info
	filterInfo := ""
	if r.fieldFilter != "" && r.fieldFilterValue != "" {
		filterInfo = fmt.Sprintf(" [%s=%s]", r.fieldFilter, r.fieldFilterValue)
	}

	// Build sort info
	sortInfo := r.getSortInfo()

	markInfo := ""
	markInFiltered := false
	if r.markedResource != nil {
		markInfo = fmt.Sprintf(" [◆ %s]", r.markedResource.GetName())
		for _, res := range r.filtered {
			if res.GetID() == r.markedResource.GetID() {
				markInFiltered = true
				break
			}
		}
	}

	navInfo := r.getNavigationShortcuts()

	dHint := "d:describe"
	if r.markedResource != nil && markInFiltered {
		dHint = "d:diff"
	}

	metricsHint := ""
	if r.getMetricSpec() != nil {
		if r.metricsLoading {
			metricsHint = " M:metrics(loading)"
		} else if r.metricsEnabled {
			metricsHint = " M:metrics(on)"
		} else {
			metricsHint = " M:metrics"
		}
	}

	partialWarn := ""
	if len(r.partialErrors) > 0 {
		partialWarn = fmt.Sprintf(" ⚠%d region(s) failed", len(r.partialErrors))
	}

	if r.filterText != "" || filterInfo != "" {
		base := fmt.Sprintf("%s/%s%s%s%s%s%s • %d/%d items • c:clear", r.service, r.resourceType, filterInfo, sortInfo, markInfo, autoReloadInfo, partialWarn, shown, total)
		if hasActions {
			base += " a:actions"
		}
		base += " m:mark y:copy" + metricsHint
		if navInfo != "" {
			base += " " + navInfo
		}
		return base
	}

	base := fmt.Sprintf("%s/%s%s%s%s%s • %d items • /:filter %s", r.service, r.resourceType, sortInfo, markInfo, autoReloadInfo, partialWarn, total, dHint)
	if hasActions {
		base += " a:actions"
	}
	base += " m:mark y:copy" + metricsHint
	if navInfo != "" {
		base += " " + navInfo
	}
	return base
}

// getSortInfo returns a string describing the current sort state
func (r *ResourceBrowser) getSortInfo() string {
	if r.sortColumn < 0 || r.renderer == nil {
		return ""
	}

	cols := r.renderer.Columns()
	if r.sortColumn >= len(cols) {
		return ""
	}

	colName := cols[r.sortColumn].Name
	direction := "↑"
	if !r.sortAscending {
		direction = "↓"
	}

	return fmt.Sprintf(" [sort: %s%s]", colName, direction)
}

// CanRefresh implements Refreshable interface
func (r *ResourceBrowser) CanRefresh() bool {
	return true
}

// Service returns the service name for this browser
func (r *ResourceBrowser) Service() string {
	return r.service
}

// getNavigationShortcuts returns a string of navigation shortcuts for the current resource
func (r *ResourceBrowser) getNavigationShortcuts() string {
	if r.renderer == nil || len(r.filtered) == 0 {
		return ""
	}

	helper := &NavigationHelper{Renderer: r.renderer}
	resource := dao.UnwrapResource(r.filtered[r.table.Cursor()])
	return helper.FormatShortcuts(resource)
}
