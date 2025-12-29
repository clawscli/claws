package view

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/log"
	"github.com/clawscli/claws/internal/metrics"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
	"github.com/clawscli/claws/internal/ui"
)

// ResourceBrowser displays resources of a specific type

const (
	logTokenMaxLen     = 20
	metricsLoadTimeout = 30 * time.Second
)

// resourceBrowserStyles holds cached lipgloss styles for performance
type resourceBrowserStyles struct {
	count        lipgloss.Style
	filterBg     lipgloss.Style
	filterActive lipgloss.Style
	tabSingle    lipgloss.Style
	tabActive    lipgloss.Style
	tabInactive  lipgloss.Style
}

func newResourceBrowserStyles() resourceBrowserStyles {
	t := ui.Current()
	return resourceBrowserStyles{
		count:        lipgloss.NewStyle().Foreground(t.TextDim),
		filterBg:     lipgloss.NewStyle().Background(t.Background).Foreground(t.Text).Padding(0, 1),
		filterActive: lipgloss.NewStyle().Foreground(t.Accent).Italic(true),
		tabSingle:    lipgloss.NewStyle().Foreground(t.Primary),
		tabActive:    lipgloss.NewStyle().Background(t.Selection).Foreground(t.SelectionText).Padding(0, 1),
		tabInactive:  lipgloss.NewStyle().Foreground(t.TextDim).Padding(0, 1),
	}
}

// tabPosition stores rendered position of a tab for mouse click detection
type tabPosition struct {
	startX, endX int
	tabIdx       int
}

type ResourceBrowser struct {
	ctx           context.Context
	registry      *registry.Registry
	service       string
	resourceType  string
	resourceTypes []string // All resource types for this service

	// Tab positions for mouse click detection
	tabPositions []tabPosition
	table        table.Model
	dao          dao.DAO
	renderer     render.Renderer
	resources    []dao.Resource
	filtered     []dao.Resource
	loading      bool
	err          error
	width        int
	height       int

	// Header panel
	headerPanel *HeaderPanel

	// Filter
	filterInput  textinput.Model
	filterActive bool
	filterText   string

	// Tag filter (from :tag command)
	tagFilterText string // tag filter (e.g., "Env=prod")

	// Field-based filter (for navigation)
	fieldFilter      string // field name to filter by (e.g., "VpcId")
	fieldFilterValue string // value to filter by

	// Auto-reload
	autoReload         bool
	autoReloadInterval time.Duration

	// Pagination (for PaginatedDAO)
	nextPageToken string // token for next page (empty = no more pages)
	hasMorePages  bool   // whether more pages are available
	isLoadingMore bool   // whether currently loading next page
	pageSize      int    // items per page (default: 100)

	// Sorting
	sortColumn    int  // column index to sort by (-1 = no sort)
	sortAscending bool // sort direction

	// Loading spinner
	spinner spinner.Model

	// Cached styles (initialized in initStyles)
	styles resourceBrowserStyles

	// Diff mark (for comparing two resources)
	markedResource dao.Resource

	// Inline metrics
	metricsEnabled bool
	metricsLoading bool
	metricsData    *metrics.MetricData
}

// NewResourceBrowser creates a new ResourceBrowser
func NewResourceBrowser(ctx context.Context, reg *registry.Registry, service string) *ResourceBrowser {
	resources := reg.ListResources(service)
	resourceType := ""
	if len(resources) > 0 {
		resourceType = resources[0]
	}

	return newResourceBrowser(ctx, reg, service, resourceType)
}

// NewResourceBrowserWithType creates a ResourceBrowser for a specific resource type
func NewResourceBrowserWithType(ctx context.Context, reg *registry.Registry, service, resourceType string) *ResourceBrowser {
	return newResourceBrowser(ctx, reg, service, resourceType)
}

// NewResourceBrowserWithFilter creates a ResourceBrowser with a field-based filter
// fieldFilter is the field name (e.g., "VpcId"), filterValue is the value to filter by
func NewResourceBrowserWithFilter(ctx context.Context, reg *registry.Registry, service, resourceType, fieldFilter, filterValue string) *ResourceBrowser {
	rb := newResourceBrowser(ctx, reg, service, resourceType)
	rb.fieldFilter = fieldFilter
	rb.fieldFilterValue = filterValue
	return rb
}

// NewResourceBrowserWithAutoReload creates a ResourceBrowser with auto-reload enabled
func NewResourceBrowserWithAutoReload(ctx context.Context, reg *registry.Registry, service, resourceType, fieldFilter, filterValue string, interval time.Duration) *ResourceBrowser {
	rb := newResourceBrowser(ctx, reg, service, resourceType)
	rb.fieldFilter = fieldFilter
	rb.fieldFilterValue = filterValue
	rb.autoReload = true
	rb.autoReloadInterval = interval
	return rb
}

func newResourceBrowser(ctx context.Context, reg *registry.Registry, service, resourceType string) *ResourceBrowser {
	ti := textinput.New()
	ti.Placeholder = "filter..."
	ti.Prompt = "/"
	ti.CharLimit = 50

	hp := NewHeaderPanel()
	hp.SetWidth(120) // Default width until SetSize is called

	return &ResourceBrowser{
		ctx:           ctx,
		registry:      reg,
		service:       service,
		resourceType:  resourceType,
		resourceTypes: reg.ListResources(service),
		loading:       true,
		filterInput:   ti,
		headerPanel:   hp,
		spinner:       ui.NewSpinner(),
		styles:        newResourceBrowserStyles(),
		pageSize:      100,
		sortColumn:    -1, // -1 = no sort
		sortAscending: true,
	}
}

// Init implements tea.Model
func (r *ResourceBrowser) Init() tea.Cmd {
	cmds := []tea.Cmd{r.loadResources, r.spinner.Tick}
	if r.autoReload {
		cmds = append(cmds, r.tickCmd())
	}
	return tea.Batch(cmds...)
}

// tickCmd returns a command that ticks after the auto-reload interval
func (r *ResourceBrowser) tickCmd() tea.Cmd {
	return tea.Tick(r.autoReloadInterval, func(t time.Time) tea.Msg {
		return autoReloadTickMsg{time: t}
	})
}

// autoReloadTickMsg is sent when auto-reload timer fires
type autoReloadTickMsg struct {
	time time.Time
}

// listResourcesResult holds the result of listing resources
type listResourcesResult struct {
	resources []dao.Resource
	nextToken string
	err       error
}

func (r *ResourceBrowser) listResourcesWithContext(ctx context.Context, d dao.DAO) listResourcesResult {
	listCtx := ctx
	if r.fieldFilter != "" && r.fieldFilterValue != "" {
		listCtx = dao.WithFilter(ctx, r.fieldFilter, r.fieldFilterValue)
	}

	var resources []dao.Resource
	var nextToken string
	var err error
	if pagDAO, ok := d.(dao.PaginatedDAO); ok {
		resources, nextToken, err = pagDAO.ListPage(listCtx, r.pageSize, "")
	} else {
		resources, err = d.List(listCtx)
	}
	return listResourcesResult{resources: resources, nextToken: nextToken, err: err}
}

func (r *ResourceBrowser) listResources(d dao.DAO) listResourcesResult {
	return r.listResourcesWithContext(r.ctx, d)
}

type multiRegionFetchResult struct {
	resources []dao.Resource
	errors    []string
}

func (r *ResourceBrowser) fetchMultiRegionResources(regions []string) multiRegionFetchResult {
	type regionResult struct {
		region    string
		resources []dao.Resource
		err       error
	}

	results := make(chan regionResult, len(regions))
	var wg sync.WaitGroup

	for _, region := range regions {
		wg.Add(1)
		go func(region string) {
			defer wg.Done()

			regionCtx := aws.WithRegionOverride(r.ctx, region)
			d, err := r.registry.GetDAO(regionCtx, r.service, r.resourceType)
			if err != nil {
				results <- regionResult{region: region, err: err}
				return
			}

			result := r.listResourcesWithContext(regionCtx, d)
			if result.err != nil {
				results <- regionResult{region: region, err: result.err}
				return
			}

			wrapped := make([]dao.Resource, len(result.resources))
			for i, res := range result.resources {
				wrapped[i] = dao.WrapWithRegion(res, region)
			}
			results <- regionResult{region: region, resources: wrapped}
		}(region)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var allResources []dao.Resource
	var errors []string
	for result := range results {
		if result.err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", result.region, result.err))
			log.Warn("failed to fetch from region", "region", result.region, "error", result.err)
		} else {
			allResources = append(allResources, result.resources...)
		}
	}

	return multiRegionFetchResult{resources: allResources, errors: errors}
}

func (r *ResourceBrowser) loadResources() tea.Msg {
	start := time.Now()
	regions := config.Global().Regions()
	isMultiRegion := len(regions) > 1

	log.Debug("loading resources", "service", r.service, "resourceType", r.resourceType,
		"regions", regions, "multiRegion", isMultiRegion)

	renderer, err := r.registry.GetRenderer(r.service, r.resourceType)
	if err != nil {
		log.Error("failed to get renderer", "service", r.service, "resourceType", r.resourceType, "error", err)
		return resourcesErrorMsg{err: err}
	}

	if !isMultiRegion {
		d, err := r.registry.GetDAO(r.ctx, r.service, r.resourceType)
		if err != nil {
			log.Error("failed to get DAO", "service", r.service, "resourceType", r.resourceType, "error", err)
			return resourcesErrorMsg{err: err}
		}

		result := r.listResources(d)
		if result.err != nil {
			log.Error("failed to list resources", "error", result.err, "duration", time.Since(start))
			return resourcesErrorMsg{err: result.err}
		}
		log.Debug("resources loaded", "count", len(result.resources), "duration", time.Since(start))

		return resourcesLoadedMsg{
			dao:          d,
			renderer:     renderer,
			resources:    result.resources,
			nextToken:    result.nextToken,
			hasMorePages: result.nextToken != "",
		}
	}

	fetchResult := r.fetchMultiRegionResources(regions)
	if len(fetchResult.resources) == 0 && len(fetchResult.errors) > 0 {
		return resourcesErrorMsg{err: fmt.Errorf("all regions failed: %s", strings.Join(fetchResult.errors, "; "))}
	}

	log.Debug("multi-region resources loaded", "count", len(fetchResult.resources),
		"regions", len(regions), "errors", len(fetchResult.errors), "duration", time.Since(start))

	return resourcesLoadedMsg{
		dao:          nil,
		renderer:     renderer,
		resources:    fetchResult.resources,
		nextToken:    "",
		hasMorePages: false,
	}
}

func (r *ResourceBrowser) reloadResources() tea.Msg {
	regions := config.Global().Regions()
	isMultiRegion := len(regions) > 1

	if !isMultiRegion {
		d := r.dao
		if d == nil {
			var err error
			d, err = r.registry.GetDAO(r.ctx, r.service, r.resourceType)
			if err != nil {
				return resourcesErrorMsg{err: err}
			}
		}

		result := r.listResources(d)
		if result.err != nil {
			return resourcesErrorMsg{err: result.err}
		}

		return resourcesLoadedMsg{
			dao:          d,
			renderer:     r.renderer,
			resources:    result.resources,
			nextToken:    result.nextToken,
			hasMorePages: result.nextToken != "",
		}
	}

	fetchResult := r.fetchMultiRegionResources(regions)
	if len(fetchResult.resources) == 0 && len(fetchResult.errors) > 0 {
		return resourcesErrorMsg{err: fmt.Errorf("all regions failed: %s", strings.Join(fetchResult.errors, "; "))}
	}

	return resourcesLoadedMsg{
		dao:          nil,
		renderer:     r.renderer,
		resources:    fetchResult.resources,
		nextToken:    "",
		hasMorePages: false,
	}
}

type resourcesLoadedMsg struct {
	dao          dao.DAO
	renderer     render.Renderer
	resources    []dao.Resource
	nextToken    string // for pagination
	hasMorePages bool   // for pagination
}

// nextPageLoadedMsg is sent when additional resources are loaded via pagination
type nextPageLoadedMsg struct {
	resources    []dao.Resource
	nextToken    string
	hasMorePages bool
}

type resourcesErrorMsg struct {
	err error
}

type metricsLoadedMsg struct {
	data         *metrics.MetricData
	err          error
	resourceType string
}

// Update implements tea.Model
func (r *ResourceBrowser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case resourcesLoadedMsg:
		r.loading = false
		r.dao = msg.dao
		r.renderer = msg.renderer
		r.resources = msg.resources
		r.nextPageToken = msg.nextToken
		r.hasMorePages = msg.hasMorePages
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

	case nextPageLoadedMsg:
		r.isLoadingMore = false
		r.resources = append(r.resources, msg.resources...)
		r.nextPageToken = msg.nextToken
		r.hasMorePages = msg.hasMorePages
		r.applyFilter()
		r.buildTable()
		return r, nil

	case resourcesErrorMsg:
		r.loading = false
		r.isLoadingMore = false
		// If we were loading more pages, just stop pagination instead of showing error
		if r.hasMorePages && len(r.resources) > 0 {
			r.hasMorePages = false
			r.nextPageToken = ""
			log.Warn("pagination stopped due to error", "error", msg.err)
			return r, nil
		}
		r.err = msg.err
		// Keep ticking even on error
		if r.autoReload {
			return r, r.tickCmd()
		}
		return r, nil

	case metricsLoadedMsg:
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

	case autoReloadTickMsg:
		if r.metricsEnabled && r.getMetricSpec() != nil {
			return r, tea.Batch(r.reloadResources, r.loadMetricsCmd())
		}
		return r, r.reloadResources

	case RefreshMsg:
		// Reload resources (e.g., after region/profile change)
		r.loading = true
		r.err = nil
		return r, tea.Batch(r.loadResources, r.spinner.Tick)

	case SortMsg:
		// Handle sort command
		if msg.Column == "" {
			// Clear sorting
			r.ClearSort()
		} else {
			// Find column by name
			colIdx := r.FindColumnByName(msg.Column)
			if colIdx >= 0 {
				r.SetSort(colIdx, msg.Ascending)
			}
		}
		r.applyFilter() // Re-apply filter to trigger sorting
		r.buildTable()
		return r, nil

	case TagFilterMsg:
		// Handle tag filter command from :tag
		if msg.Filter == "" {
			// Clear tag filter
			r.tagFilterText = ""
		} else {
			r.tagFilterText = msg.Filter
		}
		r.applyFilter()
		r.buildTable()
		return r, nil

	case DiffMsg:
		// Handle diff command: :diff <name> or :diff <name1> <name2>
		var leftRes, rightRes dao.Resource

		// Find right resource by name
		for _, res := range r.filtered {
			if res.GetName() == msg.RightName {
				rightRes = res
				break
			}
		}
		if rightRes == nil {
			return r, nil // Right resource not found
		}

		if msg.LeftName == "" {
			// :diff <name> - use current cursor row as left
			if len(r.filtered) > 0 && r.table.Cursor() < len(r.filtered) {
				leftRes = r.filtered[r.table.Cursor()]
			}
		} else {
			// :diff <name1> <name2> - find left resource by name
			for _, res := range r.filtered {
				if res.GetName() == msg.LeftName {
					leftRes = res
					break
				}
			}
		}

		if leftRes == nil || leftRes.GetID() == rightRes.GetID() {
			return r, nil // Left not found or same resource
		}

		diffView := NewDiffView(r.ctx, leftRes, rightRes, r.renderer, r.service, r.resourceType)
		return r, func() tea.Msg {
			return NavigateMsg{View: diffView}
		}

	case tea.KeyPressMsg:
		// Handle filter mode
		if r.filterActive {
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
				// Live filter as user types
				r.filterText = r.filterInput.Value()
				r.applyFilter()
				r.buildTable()
				return r, cmd
			}
		}

		// First check navigation shortcuts (they take priority)
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
			r.loading = true
			r.err = nil
			if r.metricsEnabled {
				r.metricsLoading = true
				r.metricsData = nil
			}
			return r, tea.Batch(r.loadResources, r.spinner.Tick)
		case "c":
			r.filterText = ""
			r.filterInput.SetValue("")
			r.fieldFilter = ""
			r.fieldFilterValue = ""
			r.markedResource = nil
			r.applyFilter()
			r.buildTable()
			return r, nil
		case "esc":
			// Clear mark if set, otherwise let app handle back navigation
			if r.markedResource != nil {
				r.markedResource = nil
				r.buildTable()
				return r, nil
			}
		case "m":
			if len(r.filtered) > 0 && r.table.Cursor() < len(r.filtered) {
				resource := r.filtered[r.table.Cursor()]
				if r.markedResource != nil && r.markedResource.GetID() == resource.GetID() {
					r.markedResource = nil
				} else {
					r.markedResource = resource
				}
				r.buildTable()
			}
			return r, nil
		case "M":
			if r.getMetricSpec() != nil {
				r.metricsEnabled = !r.metricsEnabled
				if r.metricsEnabled && r.metricsData == nil {
					r.metricsLoading = true
					return r, r.loadMetricsCmd()
				}
				r.buildTable()
			}
			return r, nil
		case "d", "enter":
			if len(r.filtered) > 0 && r.table.Cursor() < len(r.filtered) {
				resource := dao.UnwrapResource(r.filtered[r.table.Cursor()])
				if r.markedResource != nil && r.markedResource.GetID() != resource.GetID() {
					diffView := NewDiffView(r.ctx, dao.UnwrapResource(r.markedResource), resource, r.renderer, r.service, r.resourceType)
					return r, func() tea.Msg {
						return NavigateMsg{View: diffView}
					}
				}
				detailView := NewDetailView(r.ctx, resource, r.renderer, r.service, r.resourceType, r.registry, r.dao)
				return r, func() tea.Msg {
					return NavigateMsg{View: detailView}
				}
			}
		case "a":
			if len(r.filtered) > 0 && r.table.Cursor() < len(r.filtered) {
				if actions := action.Global.Get(r.service, r.resourceType); len(actions) > 0 {
					resource := dao.UnwrapResource(r.filtered[r.table.Cursor()])
					actionMenu := NewActionMenu(r.ctx, resource, r.service, r.resourceType)
					return r, func() tea.Msg {
						return ShowModalMsg{Modal: &Modal{Content: actionMenu}}
					}
				}
			}
		case "tab":
			// Cycle to next resource type
			r.cycleResourceType(1)
			return r, tea.Batch(r.loadResources, r.spinner.Tick)
		case "shift+tab":
			// Cycle to previous resource type
			r.cycleResourceType(-1)
			return r, tea.Batch(r.loadResources, r.spinner.Tick)
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			idx := int(msg.String()[0] - '1')
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
		case "N":
			// Manual next page load (useful when filter is active)
			if r.hasMorePages && !r.isLoadingMore && r.nextPageToken != "" {
				r.isLoadingMore = true
				return r, r.loadNextPage
			}
		}

	case spinner.TickMsg:
		// Update spinner while loading
		if r.loading {
			var cmd tea.Cmd
			r.spinner, cmd = r.spinner.Update(msg)
			return r, cmd
		}
		return r, nil

	case tea.MouseWheelMsg:
		// Pass wheel events to table for scrolling
		var cmd tea.Cmd
		r.table, cmd = r.table.Update(msg)
		return r, cmd

	case tea.MouseMotionMsg:
		// Update cursor on hover for better UX
		if idx := r.getRowAtPosition(msg.Y); idx >= 0 && idx != r.table.Cursor() {
			r.table.SetCursor(idx)
		}
		return r, nil

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			// Check if click is on tabs
			if idx := r.getTabAtPosition(msg.X, msg.Y); idx >= 0 {
				return r.switchToTab(idx)
			}
			// Handle mouse click on table row
			if len(r.filtered) > 0 {
				return r.handleMouseClick(msg.X, msg.Y)
			}
		}
	}

	var cmd tea.Cmd
	r.table, cmd = r.table.Update(msg)

	// Check if we should load more pages (infinite scroll)
	if r.shouldLoadNextPage() {
		r.isLoadingMore = true
		return r, tea.Batch(cmd, r.loadNextPage)
	}

	return r, cmd
}

// shouldLoadNextPage checks if cursor is near bottom and more pages are available
func (r *ResourceBrowser) shouldLoadNextPage() bool {
	if !r.hasMorePages || r.isLoadingMore || r.loading || r.nextPageToken == "" {
		return false
	}
	// Don't auto-load if filter is active and matches are few
	// (prevents fetching all pages when searching for non-existent items)
	if r.filterText != "" && len(r.filtered) < 10 {
		return false
	}
	if len(r.filtered) == 0 {
		return false
	}
	buffer := 10 // load more when within 10 rows of bottom
	return r.table.Cursor() >= len(r.filtered)-buffer
}

// loadNextPage loads the next page of resources for PaginatedDAO
func (r *ResourceBrowser) loadNextPage() tea.Msg {
	if r.nextPageToken == "" {
		return nil
	}

	pagDAO, ok := r.dao.(dao.PaginatedDAO)
	if !ok {
		return nil
	}

	start := time.Now()
	log.Debug("loading next page", "service", r.service, "resourceType", r.resourceType, "token", r.nextPageToken[:min(logTokenMaxLen, len(r.nextPageToken))])

	listCtx := r.ctx
	if r.fieldFilter != "" && r.fieldFilterValue != "" {
		listCtx = dao.WithFilter(r.ctx, r.fieldFilter, r.fieldFilterValue)
	}

	resources, nextToken, err := pagDAO.ListPage(listCtx, r.pageSize, r.nextPageToken)
	if err != nil {
		log.Error("failed to load next page", "error", err, "duration", time.Since(start))
		return resourcesErrorMsg{err: err}
	}

	log.Debug("next page loaded", "count", len(resources), "hasMore", nextToken != "", "duration", time.Since(start))

	return nextPageLoadedMsg{
		resources:    resources,
		nextToken:    nextToken,
		hasMorePages: nextToken != "",
	}
}

// loadMetricsCmd captures resource IDs and type synchronously before returning the tea.Cmd,
// avoiding race conditions where r.resources could be modified while the goroutine iterates.
func (r *ResourceBrowser) loadMetricsCmd() tea.Cmd {
	spec := r.getMetricSpec()
	if spec == nil {
		return nil
	}

	resourceIDs := make([]string, len(r.resources))
	for i, res := range r.resources {
		resourceIDs[i] = res.GetID()
	}
	resourceType := r.resourceType

	return func() tea.Msg {
		if r.ctx.Err() != nil {
			return nil
		}

		ctx, cancel := context.WithTimeout(r.ctx, metricsLoadTimeout)
		defer cancel()

		fetcher, err := metrics.NewFetcher(ctx)
		if err != nil {
			return metricsLoadedMsg{err: err, resourceType: resourceType}
		}

		data, err := fetcher.Fetch(ctx, resourceIDs, spec)
		return metricsLoadedMsg{data: data, err: err, resourceType: resourceType}
	}
}

func (r *ResourceBrowser) getMetricSpec() *render.MetricSpec {
	if r.renderer == nil {
		return nil
	}
	if provider, ok := r.renderer.(render.MetricSpecProvider); ok {
		return provider.MetricSpec()
	}
	return nil
}

func (r *ResourceBrowser) buildTable() {
	if r.renderer == nil {
		return
	}

	currentCursor := r.table.Cursor()
	cols := r.renderer.Columns()

	const markColWidth = 2
	const regionColWidth = 14
	metricsColWidth := metrics.ColumnWidth

	effectiveMetricsEnabled := r.metricsEnabled && r.getMetricSpec() != nil
	isMultiRegion := config.Global().IsMultiRegion()

	numCols := len(cols) + 1
	if isMultiRegion {
		numCols++
	}
	if effectiveMetricsEnabled {
		numCols++
	}
	tableCols := make([]table.Column, numCols)
	tableCols[0] = table.Column{Title: " ", Width: markColWidth}

	totalColWidth := markColWidth
	for _, col := range cols {
		totalColWidth += col.Width
	}
	if isMultiRegion {
		totalColWidth += regionColWidth
	}
	if effectiveMetricsEnabled {
		totalColWidth += metricsColWidth
	}

	extraWidth := r.width - totalColWidth
	if extraWidth < 0 {
		extraWidth = 0
	}

	colIdx := 1
	for i, col := range cols {
		title := col.Name + r.getSortIndicator(i)
		width := col.Width
		if i == len(cols)-1 && !isMultiRegion && !effectiveMetricsEnabled {
			width += extraWidth
		}
		tableCols[colIdx] = table.Column{
			Title: title,
			Width: width,
		}
		colIdx++
	}

	if isMultiRegion {
		width := regionColWidth
		if !effectiveMetricsEnabled {
			width += extraWidth
		}
		tableCols[colIdx] = table.Column{
			Title: "REGION",
			Width: width,
		}
		colIdx++
	}

	if effectiveMetricsEnabled {
		spec := r.getMetricSpec()
		header := "METRICS"
		if spec != nil {
			header = spec.ColumnHeader
		}
		tableCols[colIdx] = table.Column{
			Title: header,
			Width: metricsColWidth + extraWidth,
		}
	}

	rows := make([]table.Row, len(r.filtered))
	for i, res := range r.filtered {
		row := r.renderer.RenderRow(dao.UnwrapResource(res), cols)
		markIndicator := "  "
		if r.markedResource != nil && r.markedResource.GetID() == res.GetID() {
			markIndicator = "◆ "
		}
		fullRow := make(table.Row, numCols)
		fullRow[0] = markIndicator
		copy(fullRow[1:], row)

		rowIdx := len(cols) + 1
		if isMultiRegion {
			fullRow[rowIdx] = dao.GetResourceRegion(res)
			rowIdx++
		}
		if effectiveMetricsEnabled && r.metricsData != nil {
			unit := ""
			if r.metricsData.Spec != nil {
				unit = r.metricsData.Spec.Unit
			}
			fullRow[rowIdx] = metrics.RenderSparkline(r.metricsData.Get(res.GetID()), unit)
		} else if effectiveMetricsEnabled {
			fullRow[rowIdx] = metrics.RenderSparkline(nil, "")
		}
		rows[i] = fullRow
	}

	// Calculate header height dynamically
	var summaryFields []render.SummaryField
	if len(r.filtered) > 0 && currentCursor >= 0 && currentCursor < len(r.filtered) {
		summaryFields = r.renderer.RenderSummary(dao.UnwrapResource(r.filtered[currentCursor]))
	}
	headerStr := r.headerPanel.Render(r.service, r.resourceType, summaryFields)
	headerHeight := r.headerPanel.Height(headerStr)

	// height - header - tabs(1)
	tableHeight := r.height - headerHeight - 1
	if tableHeight < 5 {
		tableHeight = 5
	}

	t := table.New(
		table.WithColumns(tableCols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(tableHeight),
		table.WithWidth(r.width),
	)

	th := ui.Current()
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(th.TableBorder).
		BorderBottom(true).
		Bold(true).
		Foreground(th.TableHeaderText).
		Background(th.TableHeader)
	s.Selected = s.Selected.
		Foreground(th.SelectionText).
		Background(th.Selection).
		Bold(false)
	// Note: Not setting s.Cell foreground - let Selected style take precedence
	t.SetStyles(s)

	// Restore cursor position (clamped to valid range)
	if len(rows) > 0 {
		if currentCursor >= len(rows) {
			currentCursor = len(rows) - 1
		}
		if currentCursor < 0 {
			currentCursor = 0
		}
		t.SetCursor(currentCursor)
	}

	r.table = t
}

// ViewString returns the view content as a string
func (r *ResourceBrowser) ViewString() string {
	if r.loading {
		header := r.headerPanel.Render(r.service, r.resourceType, nil)
		return header + "\n" + r.spinner.View() + " Loading..."
	}

	if r.err != nil {
		header := r.headerPanel.Render(r.service, r.resourceType, nil)
		return header + "\n" + ui.DangerStyle().Render(fmt.Sprintf("Error: %v", r.err))
	}

	// Get selected resource summary fields
	var summaryFields []render.SummaryField
	if len(r.filtered) > 0 && r.table.Cursor() < len(r.filtered) && r.renderer != nil {
		selectedResource := dao.UnwrapResource(r.filtered[r.table.Cursor()])
		summaryFields = r.renderer.RenderSummary(selectedResource)
	}

	// Render header panel
	headerPanel := r.headerPanel.Render(r.service, r.resourceType, summaryFields)

	// Render tabs with count (use cached styles)
	countText := fmt.Sprintf(" [%d]", len(r.filtered))
	if r.filterText != "" && len(r.filtered) != len(r.resources) {
		countText = fmt.Sprintf(" [%d/%d]", len(r.filtered), len(r.resources))
	}
	// Show pagination status
	if r.isLoadingMore {
		countText += " (loading more...)"
	} else if r.hasMorePages {
		countText += " (more available)"
	}

	tabsView := r.renderTabs() + r.styles.count.Render(countText)

	// Filter view (use cached styles)
	var filterView string
	if r.filterActive {
		filterView = r.styles.filterBg.Render(r.filterInput.View()) + "\n"
	} else if r.filterText != "" {
		filterView = r.styles.filterActive.Render(fmt.Sprintf("filter: %s", r.filterText)) + "\n"
	}

	// Handle empty states
	if len(r.filtered) == 0 && len(r.resources) > 0 {
		return headerPanel + "\n" + tabsView + "\n" + filterView +
			ui.DimStyle().Render("No matching resources (press 'c' to clear filter)")
	}

	if len(r.resources) == 0 {
		return headerPanel + "\n" + tabsView + "\n" +
			ui.DimStyle().Render("No resources found")
	}

	return headerPanel + "\n" + tabsView + "\n" + filterView + r.table.View()
}

// View implements tea.Model
func (r *ResourceBrowser) View() tea.View {
	return tea.NewView(r.ViewString())
}

// SetSize implements View
func (r *ResourceBrowser) SetSize(width, height int) tea.Cmd {
	r.width = width
	r.height = height
	r.filterInput.SetWidth(width - 4)
	r.headerPanel.SetWidth(width)
	if r.renderer != nil {
		r.buildTable()
	}
	return nil
}

func (r *ResourceBrowser) HasActiveInput() bool {
	return r.filterActive
}

// getHeaderPanelHeight returns the height of the header panel
func (r *ResourceBrowser) getHeaderPanelHeight() int {
	headerStr := r.headerPanel.Render(r.service, r.resourceType, nil)
	return r.headerPanel.Height(headerStr)
}

// getRowAtPosition returns the row index at given Y position, or -1 if none
func (r *ResourceBrowser) getRowAtPosition(y int) int {
	// Structure: headerPanel + \n + tabsView + \n + filterView? + tableHeader
	headerHeight := r.getHeaderPanelHeight() + 1 + 1 // headerPanel + \n + tabs
	if r.filterActive || r.filterText != "" {
		headerHeight++ // filter line
	}

	// Table header row
	tableHeaderRows := 1
	row := y - headerHeight - tableHeaderRows

	if row >= 0 && row < len(r.filtered) {
		return row
	}
	return -1
}

// handleMouseClick handles mouse click on table rows
func (r *ResourceBrowser) handleMouseClick(x, y int) (tea.Model, tea.Cmd) {
	if row := r.getRowAtPosition(y); row >= 0 {
		r.table.SetCursor(row)
		return r.openDetailView()
	}
	return r, nil
}

// getTabAtPosition returns the tab index at given position, or -1 if none
func (r *ResourceBrowser) getTabAtPosition(x, y int) int {
	if len(r.tabPositions) == 0 {
		return -1
	}

	// Tabs are on the line after header panel
	tabsY := r.getHeaderPanelHeight()
	if y != tabsY {
		return -1
	}

	// Find which tab was clicked
	for _, tp := range r.tabPositions {
		if x >= tp.startX && x < tp.endX {
			return tp.tabIdx
		}
	}
	return -1
}

// switchToTab switches to the specified tab index
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
	resource := dao.UnwrapResource(r.filtered[cursor])
	detailView := NewDetailView(r.ctx, resource, r.renderer, r.service, r.resourceType, r.registry, r.dao)
	return r, func() tea.Msg {
		return NavigateMsg{View: detailView}
	}
}

func (r *ResourceBrowser) renderTabs() string {
	// Reset tab positions
	r.tabPositions = r.tabPositions[:0]

	if len(r.resourceTypes) <= 1 {
		return r.styles.tabSingle.Render(r.resourceType)
	}

	var tabs string
	currentX := 0
	for i, rt := range r.resourceTypes {
		prefix := fmt.Sprintf("%d:", i+1)
		var tabStr string
		if rt == r.resourceType {
			tabStr = r.styles.tabActive.Render(prefix + rt)
		} else {
			tabStr = r.styles.tabInactive.Render(prefix + rt)
		}

		// Record tab position (use visible width)
		tabWidth := lipgloss.Width(tabStr)
		r.tabPositions = append(r.tabPositions, tabPosition{
			startX: currentX,
			endX:   currentX + tabWidth,
			tabIdx: i,
		})
		currentX += tabWidth

		tabs += tabStr
		if i < len(r.resourceTypes)-1 {
			tabs += " "
			currentX++ // space between tabs
		}
	}

	return tabs
}

// GetTagKeys implements TagCompletionProvider
func (r *ResourceBrowser) GetTagKeys() []string {
	keySet := make(map[string]struct{})

	for _, res := range r.resources {
		tags := res.GetTags()
		if tags == nil {
			continue
		}
		for key := range tags {
			keySet[key] = struct{}{}
		}
	}

	keys := make([]string, 0, len(keySet))
	for key := range keySet {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

// GetTagValues implements TagCompletionProvider
func (r *ResourceBrowser) GetTagValues(key string) []string {
	valueSet := make(map[string]struct{})
	keyLower := strings.ToLower(key)

	for _, res := range r.resources {
		tags := res.GetTags()
		if tags == nil {
			continue
		}
		for k, v := range tags {
			if strings.ToLower(k) == keyLower {
				valueSet[v] = struct{}{}
			}
		}
	}

	values := make([]string, 0, len(valueSet))
	for value := range valueSet {
		values = append(values, value)
	}
	slices.Sort(values)
	return values
}

// GetResourceNames implements DiffCompletionProvider
func (r *ResourceBrowser) GetResourceNames() []string {
	names := make([]string, 0, len(r.filtered))
	for _, res := range r.filtered {
		names = append(names, res.GetName())
	}
	return names
}

// GetMarkedResourceName implements DiffCompletionProvider
func (r *ResourceBrowser) GetMarkedResourceName() string {
	if r.markedResource == nil {
		return ""
	}
	return r.markedResource.GetName()
}
