package view

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/log"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
	"github.com/clawscli/claws/internal/ui"
)

// ResourceBrowser displays resources of a specific type
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

type ResourceBrowser struct {
	ctx           context.Context
	registry      *registry.Registry
	service       string
	resourceType  string
	resourceTypes []string // All resource types for this service
	table         table.Model
	dao           dao.DAO
	renderer      render.Renderer
	resources     []dao.Resource
	filtered      []dao.Resource
	loading       bool
	err           error
	width         int
	height        int

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
		pageSize:      100, // default page size for paginated resources
		sortColumn:    -1,  // no sorting by default
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

// listResources executes the resource listing logic (shared by loadResources and reloadResources)
func (r *ResourceBrowser) listResources(d dao.DAO) listResourcesResult {
	// Use context with filter if field filter is set
	listCtx := r.ctx
	if r.fieldFilter != "" && r.fieldFilterValue != "" {
		listCtx = dao.WithFilter(r.ctx, r.fieldFilter, r.fieldFilterValue)
	}

	// Use paginated listing if supported
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

func (r *ResourceBrowser) loadResources() tea.Msg {
	start := time.Now()
	log.Debug("loading resources", "service", r.service, "resourceType", r.resourceType, "filter", r.fieldFilter, "filterValue", r.fieldFilterValue)

	d, err := r.registry.GetDAO(r.ctx, r.service, r.resourceType)
	if err != nil {
		log.Error("failed to get DAO", "service", r.service, "resourceType", r.resourceType, "error", err)
		return resourcesErrorMsg{err: err}
	}

	renderer, err := r.registry.GetRenderer(r.service, r.resourceType)
	if err != nil {
		log.Error("failed to get renderer", "service", r.service, "resourceType", r.resourceType, "error", err)
		return resourcesErrorMsg{err: err}
	}

	result := r.listResources(d)
	if result.err != nil {
		log.Error("failed to list resources", "service", r.service, "resourceType", r.resourceType, "error", result.err, "duration", time.Since(start))
		return resourcesErrorMsg{err: result.err}
	}
	log.Debug("resources loaded", "service", r.service, "resourceType", r.resourceType, "count", len(result.resources), "hasMore", result.nextToken != "", "duration", time.Since(start))

	return resourcesLoadedMsg{
		dao:          d,
		renderer:     renderer,
		resources:    result.resources,
		nextToken:    result.nextToken,
		hasMorePages: result.nextToken != "",
	}
}

// reloadResources reloads resources without changing loading state (for auto-reload)
func (r *ResourceBrowser) reloadResources() tea.Msg {
	// Use existing DAO if available
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
		// Schedule next tick if auto-reload is enabled
		if r.autoReload {
			return r, r.tickCmd()
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

	case autoReloadTickMsg:
		// Silent reload (don't show loading state to avoid flicker)
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

	case tea.KeyMsg:
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
			return r, tea.Batch(r.loadResources, r.spinner.Tick)
		case "c":
			// Clear all filters (text filter and field filter)
			r.filterText = ""
			r.filterInput.SetValue("")
			r.fieldFilter = ""
			r.fieldFilterValue = ""
			r.applyFilter()
			r.buildTable()
			return r, nil
		case "d", "enter":
			if len(r.filtered) > 0 && r.table.Cursor() < len(r.filtered) {
				resource := r.filtered[r.table.Cursor()]
				// Pass DAO for async refresh in DetailView
				detailView := NewDetailView(r.ctx, resource, r.renderer, r.service, r.resourceType, r.registry, r.dao)
				return r, func() tea.Msg {
					return NavigateMsg{View: detailView}
				}
			}
		case "a":
			// Show actions menu (only if actions exist)
			if len(r.filtered) > 0 && r.table.Cursor() < len(r.filtered) {
				if actions := action.Global.Get(r.service, r.resourceType); len(actions) > 0 {
					resource := r.filtered[r.table.Cursor()]
					actionMenu := NewActionMenu(r.ctx, resource, r.service, r.resourceType)
					return r, func() tea.Msg {
						return NavigateMsg{View: actionMenu}
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
			// Switch to resource type by number
			idx := int(msg.String()[0] - '1')
			if idx < len(r.resourceTypes) {
				r.resourceType = r.resourceTypes[idx]
				r.loading = true
				r.filterText = ""
				r.filterInput.SetValue("")
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
	log.Debug("loading next page", "service", r.service, "resourceType", r.resourceType, "token", r.nextPageToken[:min(20, len(r.nextPageToken))])

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

// buildTable rebuilds the table with current filtered resources
func (r *ResourceBrowser) buildTable() {
	if r.renderer == nil {
		return
	}

	// Save current cursor position before rebuilding
	currentCursor := r.table.Cursor()

	cols := r.renderer.Columns()
	tableCols := make([]table.Column, len(cols))
	for i, col := range cols {
		title := col.Name + r.getSortIndicator(i)
		tableCols[i] = table.Column{
			Title: title,
			Width: col.Width,
		}
	}

	rows := make([]table.Row, len(r.filtered))
	for i, res := range r.filtered {
		rows[i] = r.renderer.RenderRow(res, cols)
	}

	// Calculate header height dynamically
	var summaryFields []render.SummaryField
	if len(r.filtered) > 0 && currentCursor >= 0 && currentCursor < len(r.filtered) {
		summaryFields = r.renderer.RenderSummary(r.filtered[currentCursor])
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
	s.Cell = s.Cell.
		Foreground(th.Text)
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

// View implements tea.Model
func (r *ResourceBrowser) View() string {
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
		selectedResource := r.filtered[r.table.Cursor()]
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

// SetSize implements View
func (r *ResourceBrowser) SetSize(width, height int) tea.Cmd {
	r.width = width
	r.height = height
	r.filterInput.Width = width - 4
	r.headerPanel.SetWidth(width)
	if r.renderer != nil {
		r.buildTable()
	}
	return nil
}

func (r *ResourceBrowser) HasActiveInput() bool {
	return r.filterActive
}

func (r *ResourceBrowser) renderTabs() string {
	if len(r.resourceTypes) <= 1 {
		return r.styles.tabSingle.Render(r.resourceType)
	}

	var tabs string
	for i, rt := range r.resourceTypes {
		prefix := fmt.Sprintf("%d:", i+1)
		if rt == r.resourceType {
			tabs += r.styles.tabActive.Render(prefix + rt)
		} else {
			tabs += r.styles.tabInactive.Render(prefix + rt)
		}
		if i < len(r.resourceTypes)-1 {
			tabs += " "
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
