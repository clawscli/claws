package view

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/aws"
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
	nextPageToken  string
	nextPageTokens map[string]string
	hasMorePages   bool
	isLoadingMore  bool
	pageSize       int

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

	// Partial region errors (for multi-region queries)
	partialErrors []string
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

// Update implements tea.Model
func (r *ResourceBrowser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case resourcesLoadedMsg:
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

	case nextPageLoadedMsg:
		r.isLoadingMore = false
		r.resources = append(r.resources, msg.resources...)
		r.nextPageToken = msg.nextToken
		r.nextPageTokens = msg.nextPageTokens
		r.hasMorePages = msg.hasMorePages
		r.applyFilter()
		r.buildTable()
		return r, nil

	case resourcesErrorMsg:
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

		diffView := NewDiffView(r.ctx, dao.UnwrapResource(leftRes), dao.UnwrapResource(rightRes), r.renderer, r.service, r.resourceType)
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
				ctx, resource := r.contextForResource(r.filtered[r.table.Cursor()])
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
		case "a":
			if len(r.filtered) > 0 && r.table.Cursor() < len(r.filtered) {
				if actions := action.Global.Get(r.service, r.resourceType); len(actions) > 0 {
					ctx, resource := r.contextForResource(r.filtered[r.table.Cursor()])
					actionMenu := NewActionMenu(ctx, resource, r.service, r.resourceType)
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
			if r.hasMorePages && !r.isLoadingMore && (r.nextPageToken != "" || len(r.nextPageTokens) > 0) {
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
	ctx, resource := r.contextForResource(r.filtered[cursor])
	detailView := NewDetailView(ctx, resource, r.renderer, r.service, r.resourceType, r.registry, r.dao)
	return r, func() tea.Msg {
		return NavigateMsg{View: detailView}
	}
}

func (r *ResourceBrowser) contextForResource(res dao.Resource) (context.Context, dao.Resource) {
	if region := dao.GetResourceRegion(res); region != "" {
		return aws.WithRegionOverride(r.ctx, region), dao.UnwrapResource(res)
	}
	return r.ctx, dao.UnwrapResource(res)
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
