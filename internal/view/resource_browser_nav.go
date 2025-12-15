package view

import (
	"fmt"
	"slices"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/clawscli/claws/internal/action"
)

// handleNavigation processes navigation key shortcuts
func (r *ResourceBrowser) handleNavigation(key string) (tea.Model, tea.Cmd) {
	if r.renderer == nil || len(r.filtered) == 0 {
		return nil, nil
	}

	helper := &NavigationHelper{
		Ctx:      r.ctx,
		Registry: r.registry,
		Renderer: r.renderer,
	}

	resource := r.filtered[r.table.Cursor()]
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

	// Build navigation shortcuts string
	navInfo := r.getNavigationShortcuts()

	if r.filterText != "" || filterInfo != "" {
		base := fmt.Sprintf("%s/%s%s%s%s • %d/%d items • c:clear", r.service, r.resourceType, filterInfo, sortInfo, autoReloadInfo, shown, total)
		if hasActions {
			base += " a:actions"
		}
		if navInfo != "" {
			base += " " + navInfo
		}
		return base
	}

	base := fmt.Sprintf("%s/%s%s%s • %d items • /:filter d:describe", r.service, r.resourceType, sortInfo, autoReloadInfo, total)
	if hasActions {
		base += " a:actions"
	}
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

// getNavigationShortcuts returns a string of navigation shortcuts for the current resource
func (r *ResourceBrowser) getNavigationShortcuts() string {
	if r.renderer == nil || len(r.filtered) == 0 {
		return ""
	}

	helper := &NavigationHelper{Renderer: r.renderer}
	resource := r.filtered[r.table.Cursor()]
	return helper.FormatShortcuts(resource)
}
