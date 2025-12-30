package view

import (
	tea "charm.land/bubbletea/v2"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/log"
)

func (r *ResourceBrowser) handleResourcesLoaded(msg resourcesLoadedMsg) (tea.Model, tea.Cmd) {
	r.loading = false
	r.dao = msg.dao
	r.renderer = msg.renderer
	r.resources = msg.resources
	r.nextPageToken = msg.nextToken
	r.nextPageTokens = msg.nextPageTokens
	r.hasMorePages = msg.hasMorePages
	r.partialErrors = msg.partialErrors
	r.applyFilter()
	r.buildTable()

	var cmds []tea.Cmd
	if r.autoReload {
		cmds = append(cmds, r.tickCmd())
	}
	if r.metricsEnabled && r.metricsLoading {
		cmds = append(cmds, r.loadMetricsCmd())
	}
	if len(cmds) > 0 {
		return r, tea.Batch(cmds...)
	}
	return r, nil
}

func (r *ResourceBrowser) handleNextPageLoaded(msg nextPageLoadedMsg) (tea.Model, tea.Cmd) {
	r.isLoadingMore = false
	r.resources = append(r.resources, msg.resources...)
	r.nextPageToken = msg.nextToken
	r.nextPageTokens = msg.nextPageTokens
	r.hasMorePages = msg.hasMorePages
	r.applyFilter()
	r.buildTable()
	return r, nil
}

func (r *ResourceBrowser) handleResourcesError(msg resourcesErrorMsg) (tea.Model, tea.Cmd) {
	r.loading = false
	r.isLoadingMore = false
	if r.hasMorePages && len(r.resources) > 0 {
		r.hasMorePages = false
		r.nextPageToken = ""
		r.nextPageTokens = nil
		log.Warn("pagination stopped due to error", "error", msg.err)
		return r, nil
	}
	r.err = msg.err
	if r.autoReload {
		return r, r.tickCmd()
	}
	return r, nil
}

func (r *ResourceBrowser) handleMetricsLoaded(msg metricsLoadedMsg) (tea.Model, tea.Cmd) {
	r.metricsLoading = false
	if msg.resourceType != r.resourceType {
		return r, nil
	}
	if msg.err != nil {
		log.Warn("failed to load metrics", "error", msg.err, "service", r.service, "resource", r.resourceType)
	} else {
		r.metricsData = msg.data
	}
	r.buildTable()
	return r, nil
}

func (r *ResourceBrowser) handleAutoReloadTick() (tea.Model, tea.Cmd) {
	if r.metricsEnabled && r.getMetricSpec() != nil {
		return r, tea.Batch(r.reloadResources, r.loadMetricsCmd())
	}
	return r, r.reloadResources
}

func (r *ResourceBrowser) handleRefreshMsg() (tea.Model, tea.Cmd) {
	r.loading = true
	r.err = nil
	return r, tea.Batch(r.loadResources, r.spinner.Tick)
}

func (r *ResourceBrowser) handleSortMsg(msg SortMsg) (tea.Model, tea.Cmd) {
	if msg.Column == "" {
		r.ClearSort()
	} else {
		colIdx := r.FindColumnByName(msg.Column)
		if colIdx >= 0 {
			r.SetSort(colIdx, msg.Ascending)
		}
	}
	r.applyFilter()
	r.buildTable()
	return r, nil
}

func (r *ResourceBrowser) handleTagFilterMsg(msg TagFilterMsg) (tea.Model, tea.Cmd) {
	if msg.Filter == "" {
		r.tagFilterText = ""
	} else {
		r.tagFilterText = msg.Filter
	}
	r.applyFilter()
	r.buildTable()
	return r, nil
}

func (r *ResourceBrowser) handleDiffMsg(msg DiffMsg) (tea.Model, tea.Cmd) {
	var leftRes, rightRes dao.Resource

	for _, res := range r.filtered {
		if res.GetName() == msg.RightName {
			rightRes = res
			break
		}
	}
	if rightRes == nil {
		return r, nil
	}

	if msg.LeftName == "" {
		if len(r.filtered) > 0 && r.table.Cursor() < len(r.filtered) {
			leftRes = r.filtered[r.table.Cursor()]
		}
	} else {
		for _, res := range r.filtered {
			if res.GetName() == msg.LeftName {
				leftRes = res
				break
			}
		}
	}

	if leftRes == nil || leftRes.GetID() == rightRes.GetID() {
		return r, nil
	}

	diffView := NewDiffView(r.ctx, dao.UnwrapResource(leftRes), dao.UnwrapResource(rightRes), r.renderer, r.service, r.resourceType)
	return r, func() tea.Msg {
		return NavigateMsg{View: diffView}
	}
}
