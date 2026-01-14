package view

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/metrics"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
	"github.com/clawscli/claws/internal/ui"
)

// ResourceBrowser displays resources of a specific type

const logTokenMaxLen = 20

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
	return resourceBrowserStyles{
		count:        ui.DimStyle(),
		filterBg:     ui.InputFieldStyle(),
		filterActive: ui.AccentStyle().Italic(true),
		tabSingle:    ui.PrimaryStyle(),
		tabActive:    ui.SelectedStyle().Padding(0, 1),
		tabInactive:  ui.DimStyle().Padding(0, 1),
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

	tc           TableCursor
	tableContent string

	dao       dao.DAO
	renderer  render.Renderer
	resources []dao.Resource
	filtered  []dao.Resource
	loading   bool
	err       error
	width     int
	height    int

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
	nextPageToken       string
	nextPageTokens      map[string]string
	nextMultiPageTokens map[profileRegionKey]string
	hasMorePages        bool
	isLoadingMore       bool
	pageSize            int

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

	// List-level toggles (e.g., show resolved findings)
	toggleStates map[string]bool
}

// NewResourceBrowser creates a new ResourceBrowser
func NewResourceBrowser(ctx context.Context, reg *registry.Registry, service string) *ResourceBrowser {
	resourceType := reg.DefaultResource(service)
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
	ti.Placeholder = FilterPlaceholder
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
		sortColumn:    -1,
		sortAscending: true,
		toggleStates:  make(map[string]bool),
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

func (r *ResourceBrowser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case resourcesLoadedMsg:
		return r.handleResourcesLoaded(msg)
	case nextPageLoadedMsg:
		return r.handleNextPageLoaded(msg)
	case resourcesErrorMsg:
		return r.handleResourcesError(msg)
	case metricsLoadedMsg:
		return r.handleMetricsLoaded(msg)
	case autoReloadTickMsg:
		return r.handleAutoReloadTick()
	case RefreshMsg:
		return r.handleRefreshMsg()
	case ThemeChangedMsg:
		r.styles = newResourceBrowserStyles()
		r.headerPanel.ReloadStyles()
		r.buildTable()
		return r, nil
	case CompactHeaderChangedMsg:
		r.buildTable()
		return r, nil
	case SortMsg:
		return r.handleSortMsg(msg)
	case TagFilterMsg:
		return r.handleTagFilterMsg(msg)
	case DiffMsg:
		return r.handleDiffMsg(msg)
	case tea.KeyPressMsg:
		if model, cmd := r.handleKeyPress(msg); model != nil || cmd != nil {
			if model == nil {
				model = r
			}
			return model, cmd
		}

	case spinner.TickMsg:
		if r.loading {
			var cmd tea.Cmd
			r.spinner, cmd = r.spinner.Update(msg)
			return r, cmd
		}
		return r, nil

	case tea.MouseWheelMsg:
		return r.handleMouseWheel(msg)

	case tea.MouseMotionMsg:
		return r.handleMouseMotion(msg)

	case tea.MouseClickMsg:
		if model, cmd := r.handleMouseClickMsg(msg); cmd != nil {
			return model, cmd
		}
	}

	// Check if we should load more pages (infinite scroll)
	if r.shouldLoadNextPage() {
		r.isLoadingMore = true
		return r, r.loadNextPage
	}

	return r, nil
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

	var summaryFields []render.SummaryField
	if len(r.filtered) > 0 && r.tc.Cursor() < len(r.filtered) && r.renderer != nil {
		selectedResource := dao.UnwrapResource(r.filtered[r.tc.Cursor()])
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

	return headerPanel + "\n" + tabsView + "\n" + filterView + r.tableContent
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

func (r *ResourceBrowser) contextForResource(res dao.Resource) (context.Context, dao.Resource) {
	ctx := r.ctx
	if profile := dao.GetResourceProfile(res); profile != "" {
		sel := config.ProfileSelectionFromID(profile)
		ctx = aws.WithSelectionOverride(ctx, sel)
	}
	if region := dao.GetResourceRegion(res); region != "" {
		ctx = aws.WithRegionOverride(ctx, region)
	}
	return ctx, res
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

	keys := slices.Collect(maps.Keys(keySet))
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

	values := slices.Collect(maps.Keys(valueSet))
	slices.Sort(values)
	return values
}

// GetResourceIDs implements DiffCompletionProvider
func (r *ResourceBrowser) GetResourceIDs() []string {
	ids := make([]string, 0, len(r.filtered))
	for _, res := range r.filtered {
		ids = append(ids, res.GetID())
	}
	return ids
}

// GetMarkedResourceID implements DiffCompletionProvider
func (r *ResourceBrowser) GetMarkedResourceID() string {
	if r.markedResource == nil {
		return ""
	}
	return r.markedResource.GetID()
}
