package view

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	tagtypes "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"

	"github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/log"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/ui"
)

const tagSearchLimit = 100

type taggedARN struct {
	ARN    *aws.ARN
	Region string
	Tags   map[string]string
	RawARN string
}

type tagSearchViewStyles struct {
	header       lipgloss.Style
	status       lipgloss.Style
	filterWrap   lipgloss.Style
	filterActive lipgloss.Style
}

func newTagSearchViewStyles() tagSearchViewStyles {
	return tagSearchViewStyles{
		header:       ui.TableHeaderStyle().Padding(0, 1),
		status:       ui.DimStyle().Padding(0, 1),
		filterWrap:   ui.NoStyle().Padding(0, 1),
		filterActive: ui.AccentStyle().Italic(true),
	}
}

type TagSearchView struct {
	ctx       context.Context
	registry  *registry.Registry
	tagFilter string
	styles    tagSearchViewStyles

	tc           TableCursor
	tableContent string

	resources []taggedARN
	filtered  []taggedARN
	loading   bool
	err       error
	width     int
	height    int
	spinner   spinner.Model

	filterActive bool
	filterText   string
	filterInput  textinput.Model

	hasMorePages  bool
	isLoadingMore bool
	pageTokens    map[string]string
	partialErrors []string
}

func NewTagSearchView(ctx context.Context, reg *registry.Registry, tagFilter string) *TagSearchView {
	ti := textinput.New()
	ti.Placeholder = FilterPlaceholder
	ti.Prompt = "/"
	ti.CharLimit = 100

	return &TagSearchView{
		ctx:         ctx,
		registry:    reg,
		tagFilter:   tagFilter,
		styles:      newTagSearchViewStyles(),
		loading:     true,
		filterInput: ti,
		spinner:     ui.NewSpinner(),
		pageTokens:  make(map[string]string),
	}
}

func (v *TagSearchView) Init() tea.Cmd {
	return tea.Batch(v.loadResources, v.spinner.Tick)
}

type tagSearchLoadedMsg struct {
	resources     []taggedARN
	pageTokens    map[string]string
	hasMore       bool
	partialErrors []string
}

type tagSearchErrorMsg struct {
	err error
}

type tagSearchNextPageMsg struct {
	resources  []taggedARN
	pageTokens map[string]string
	hasMore    bool
}

func (v *TagSearchView) loadResources() tea.Msg {
	regions := config.Global().Regions()
	if len(regions) == 0 {
		regions = []string{config.Global().Region()}
	}

	result := v.fetchTaggedResources(regions, nil)
	if len(result.resources) == 0 && len(result.errors) > 0 {
		return tagSearchErrorMsg{err: fmt.Errorf("all regions failed: %s", strings.Join(result.errors, "; "))}
	}

	return tagSearchLoadedMsg{
		resources:     result.resources,
		pageTokens:    result.pageTokens,
		hasMore:       len(result.pageTokens) > 0,
		partialErrors: result.errors,
	}
}

type fetchResult struct {
	resources  []taggedARN
	pageTokens map[string]string
	errors     []string
}

func (v *TagSearchView) fetchTaggedResources(regions []string, existingTokens map[string]string) fetchResult {
	type regionResult struct {
		region    string
		resources []taggedARN
		nextToken string
		err       error
	}

	ctx, cancel := context.WithTimeout(v.ctx, config.File().TagSearchTimeout())
	defer cancel()

	results := make(chan regionResult, len(regions))
	var wg sync.WaitGroup

	tagFilters := v.parseTagFilters()

	for _, region := range regions {
		wg.Add(1)
		go func(region string) {
			defer wg.Done()

			regionCtx := aws.WithRegionOverride(ctx, region)
			cfg, err := aws.NewConfig(regionCtx)
			if err != nil {
				results <- regionResult{region: region, err: err}
				return
			}

			client := resourcegroupstaggingapi.NewFromConfig(cfg)
			input := &resourcegroupstaggingapi.GetResourcesInput{
				TagFilters:       tagFilters,
				ResourcesPerPage: aws.Int32Ptr(int32(tagSearchLimit)),
			}

			if existingTokens != nil {
				if token, ok := existingTokens[region]; ok {
					input.PaginationToken = aws.StringPtr(token)
				}
			}

			output, err := client.GetResources(regionCtx, input)
			if err != nil {
				results <- regionResult{region: region, err: err}
				return
			}

			resources := make([]taggedARN, 0, len(output.ResourceTagMappingList))
			for _, mapping := range output.ResourceTagMappingList {
				rawARN := aws.Str(mapping.ResourceARN)
				parsed := aws.ParseARN(rawARN)

				tags := make(map[string]string)
				for _, tag := range mapping.Tags {
					tags[aws.Str(tag.Key)] = aws.Str(tag.Value)
				}

				resources = append(resources, taggedARN{
					ARN:    parsed,
					Region: region,
					Tags:   tags,
					RawARN: rawARN,
				})
			}

			nextToken := ""
			if output.PaginationToken != nil {
				nextToken = *output.PaginationToken
			}

			results <- regionResult{region: region, resources: resources, nextToken: nextToken}
		}(region)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	resultsByRegion := make(map[string]regionResult)
	for result := range results {
		resultsByRegion[result.region] = result
	}

	var allResources []taggedARN
	var errors []string
	pageTokens := make(map[string]string)

	for _, region := range regions {
		result, ok := resultsByRegion[region]
		if !ok {
			continue
		}
		if result.err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", result.region, result.err))
			log.Warn("failed to fetch tags from region", "region", result.region, "error", result.err)
		} else {
			allResources = append(allResources, result.resources...)
			if result.nextToken != "" {
				pageTokens[result.region] = result.nextToken
			}
		}
	}

	return fetchResult{resources: allResources, pageTokens: pageTokens, errors: errors}
}

func (v *TagSearchView) parseTagFilters() []tagtypes.TagFilter {
	if v.tagFilter == "" {
		return nil
	}

	parts := strings.SplitN(v.tagFilter, "=", 2)
	key := parts[0]
	filter := tagtypes.TagFilter{Key: aws.StringPtr(key)}

	if len(parts) == 2 {
		filter.Values = []string{parts[1]}
	}

	return []tagtypes.TagFilter{filter}
}

func (v *TagSearchView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tagSearchLoadedMsg:
		v.loading = false
		v.resources = msg.resources
		v.pageTokens = msg.pageTokens
		v.hasMorePages = msg.hasMore
		v.partialErrors = msg.partialErrors
		v.applyFilter()
		v.buildTable()
		return v, nil

	case tagSearchNextPageMsg:
		v.isLoadingMore = false
		v.resources = append(v.resources, msg.resources...)
		v.pageTokens = msg.pageTokens
		v.hasMorePages = msg.hasMore
		v.applyFilter()
		v.buildTable()
		return v, nil

	case tagSearchErrorMsg:
		v.loading = false
		v.err = msg.err
		return v, nil

	case spinner.TickMsg:
		if v.loading {
			var cmd tea.Cmd
			v.spinner, cmd = v.spinner.Update(msg)
			return v, cmd
		}
		return v, nil
	case ThemeChangedMsg:
		v.styles = newTagSearchViewStyles()
		v.buildTable()
		return v, nil

	case tea.MouseWheelMsg:
		delta := 0
		switch msg.Button {
		case tea.MouseWheelUp:
			delta = -3
		case tea.MouseWheelDown:
			delta = 3
		}
		v.tc.AdjustScrollOffset(delta, len(v.filtered))
		v.buildTable()
		return v, nil

	case tea.MouseMotionMsg:
		if idx := v.getRowAtPosition(msg.Y); idx >= 0 && idx != v.tc.Cursor() {
			v.tc.SetCursor(idx, len(v.filtered))
			v.buildTable()
		}
		return v, nil

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft && len(v.filtered) > 0 {
			if idx := v.getRowAtPosition(msg.Y); idx >= 0 {
				v.tc.SetCursor(idx, len(v.filtered))
				v.buildTable()
				return v.navigateToResource()
			}
		}
		return v, nil

	case tea.KeyPressMsg:
		if v.filterActive {
			switch msg.String() {
			case "esc":
				v.filterActive = false
				v.filterInput.Blur()
				return v, nil
			case "enter":
				v.filterActive = false
				v.filterInput.Blur()
				v.filterText = v.filterInput.Value()
				v.applyFilter()
				v.buildTable()
				return v, nil
			default:
				var cmd tea.Cmd
				v.filterInput, cmd = v.filterInput.Update(msg)
				v.filterText = v.filterInput.Value()
				v.applyFilter()
				v.buildTable()
				return v, cmd
			}
		}

		switch msg.String() {
		case "/":
			v.filterActive = true
			v.filterInput.Focus()
			return v, textinput.Blink

		case "c":
			v.filterText = ""
			v.filterInput.SetValue("")
			v.applyFilter()
			v.buildTable()
			return v, nil

		case "ctrl+r":
			v.loading = true
			v.err = nil
			v.resources = nil
			v.pageTokens = make(map[string]string)
			return v, tea.Batch(v.loadResources, v.spinner.Tick)

		case "N":
			if v.hasMorePages && !v.isLoadingMore && len(v.pageTokens) > 0 {
				v.isLoadingMore = true
				return v, v.loadNextPage
			}

		case "enter", "d":
			if len(v.filtered) > 0 && v.tc.Cursor() < len(v.filtered) {
				return v.navigateToResource()
			}

		case "j", "down":
			v.tc.SetCursor(v.tc.Cursor()+1, len(v.filtered))
			v.tc.UpdateScrollOffset(len(v.filtered))
			v.buildTable()
			return v, nil

		case "k", "up":
			v.tc.SetCursor(v.tc.Cursor()-1, len(v.filtered))
			v.tc.UpdateScrollOffset(len(v.filtered))
			v.buildTable()
			return v, nil

		case "ctrl+d", "pgdown":
			v.tc.SetCursor(v.tc.Cursor()+v.tc.TableHeight()/2, len(v.filtered))
			v.tc.UpdateScrollOffset(len(v.filtered))
			v.buildTable()
			return v, nil

		case "ctrl+u", "pgup":
			v.tc.SetCursor(v.tc.Cursor()-v.tc.TableHeight()/2, len(v.filtered))
			v.tc.UpdateScrollOffset(len(v.filtered))
			v.buildTable()
			return v, nil

		case "g", "home":
			v.tc.SetCursor(0, len(v.filtered))
			v.tc.UpdateScrollOffset(len(v.filtered))
			v.buildTable()
			return v, nil

		case "G", "end":
			v.tc.SetCursor(len(v.filtered)-1, len(v.filtered))
			v.tc.UpdateScrollOffset(len(v.filtered))
			v.buildTable()
			return v, nil
		}
	}

	return v, nil
}

func (v *TagSearchView) loadNextPage() tea.Msg {
	regions := make([]string, 0, len(v.pageTokens))
	for region := range v.pageTokens {
		regions = append(regions, region)
	}

	result := v.fetchTaggedResources(regions, v.pageTokens)
	return tagSearchNextPageMsg{
		resources:  result.resources,
		pageTokens: result.pageTokens,
		hasMore:    len(result.pageTokens) > 0,
	}
}

func (v *TagSearchView) navigateToResource() (tea.Model, tea.Cmd) {
	cursor := v.tc.Cursor()
	if len(v.filtered) == 0 || cursor >= len(v.filtered) {
		return v, nil
	}

	res := v.filtered[cursor]
	if res.ARN == nil || !res.ARN.CanNavigate() {
		return v, nil
	}

	service, resourceType := res.ARN.ServiceResourceType()
	if service == "" || resourceType == "" {
		return v, nil
	}

	if _, ok := v.registry.Get(service, resourceType); !ok {
		return v, nil
	}

	ctx := v.ctx
	if res.Region != "" {
		ctx = aws.WithRegionOverride(ctx, res.Region)
	}

	if filterKey, filterValue := res.ARN.ExtractParentFilter(); filterKey != "" {
		ctx = dao.WithFilter(ctx, filterKey, filterValue)
	}

	renderer, err := v.registry.GetRenderer(service, resourceType)
	if err != nil {
		return v, nil
	}
	daoInst, err := v.registry.GetDAO(ctx, service, resourceType)
	if err != nil {
		daoInst = nil
	}

	resourceID := v.getResourceIDForGet(res.ARN)
	minimalResource := &dao.BaseResource{
		ID:   resourceID,
		Name: res.ARN.ShortID(),
		ARN:  res.RawARN,
		Tags: res.Tags,
	}

	detailView := NewDetailView(ctx, minimalResource, renderer, service, resourceType, v.registry, daoInst)
	return v, func() tea.Msg {
		return NavigateMsg{View: detailView}
	}
}

func (v *TagSearchView) getResourceIDForGet(arn *aws.ARN) string {
	switch arn.Service {
	case "states":
		return arn.Raw
	case "bedrock-agentcore":
		if idx := strings.Index(arn.ResourceID, "/"); idx > 0 {
			return arn.ResourceID[:idx]
		}
		return arn.ResourceID
	default:
		if arn.ResourceID != "" {
			return arn.ResourceID
		}
		return arn.Raw
	}
}

func (v *TagSearchView) applyFilter() {
	if v.filterText == "" {
		v.filtered = v.resources
		return
	}

	filter := strings.ToLower(v.filterText)
	v.filtered = nil

	for _, res := range v.resources {
		if fuzzyMatch(res.RawARN, filter) ||
			fuzzyMatch(res.Region, filter) {
			v.filtered = append(v.filtered, res)
			continue
		}

		if res.ARN != nil {
			if fuzzyMatch(res.ARN.Service, filter) ||
				fuzzyMatch(res.ARN.ResourceType, filter) ||
				fuzzyMatch(res.ARN.ResourceID, filter) {
				v.filtered = append(v.filtered, res)
				continue
			}
		}

		for k, val := range res.Tags {
			if fuzzyMatch(k, filter) || fuzzyMatch(val, filter) {
				v.filtered = append(v.filtered, res)
				break
			}
		}
	}
}

func (v *TagSearchView) Cursor() int {
	return v.tc.Cursor()
}

func (v *TagSearchView) SetCursor(n int) {
	v.tc.SetCursor(n, len(v.filtered))
}

func (v *TagSearchView) buildTable() {
	v.tc.SetCursor(v.tc.Cursor(), len(v.filtered))

	isMultiRegion := config.Global().IsMultiRegion()

	headers := []string{"Service", "Type", "ID"}
	if isMultiRegion {
		headers = append(headers, "Region")
	}
	headers = append(headers, "Tags")

	tableHeight := v.height - 1
	if tableHeight < 1 {
		tableHeight = 1
	}
	v.tc.SetTableHeight(tableHeight)

	tableWidth := v.width
	if tableWidth < 80 {
		tableWidth = 120
	}

	cursor := v.tc.Cursor()

	numCols := len(headers)
	widths := make([]int, numCols)
	baseWidth := tableWidth / numCols
	remainder := tableWidth % numCols
	for i := range widths {
		widths[i] = baseWidth
		if i < remainder {
			widths[i]++
		}
	}

	t := table.New().
		Headers(headers...).
		Width(tableWidth).
		Height(tableHeight).
		Wrap(false).
		BorderTop(false).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderColumn(false).
		BorderHeader(true).
		BorderStyle(TableBorderStyle()).
		StyleFunc(NewTableStyleFunc(widths, cursor))

	for _, res := range v.filtered {
		service := ""
		resType := ""
		resID := ""

		if res.ARN != nil {
			service = res.ARN.Service
			resType = res.ARN.ResourceType
			resID = res.ARN.ShortID()
		} else {
			resID = res.RawARN
		}

		tagStr := formatTags(res.Tags, 50)

		row := []string{service, resType, resID}
		if isMultiRegion {
			row = append(row, res.Region)
		}
		row = append(row, tagStr)
		t = t.Row(row...)
	}

	if v.tc.ScrollOffset() > 0 {
		t = t.YOffset(v.tc.ScrollOffset())
	}

	v.tableContent = t.String()
}

func (v *TagSearchView) ViewString() string {
	s := v.styles

	title := "Tag Search"
	if v.tagFilter != "" {
		title = fmt.Sprintf("Tag Search: %s", v.tagFilter)
	}
	header := s.header.Width(v.width).Render(title)

	if v.loading {
		return header + "\n" + v.spinner.View() + " Searching..."
	}

	if v.err != nil {
		return header + "\n" + ui.DangerStyle().Render(fmt.Sprintf("Error: %v", v.err))
	}

	statusLine := ""
	if v.filterText != "" {
		statusLine = fmt.Sprintf("Found %d/%d resources", len(v.filtered), len(v.resources))
	} else {
		statusLine = fmt.Sprintf("Found %d resources", len(v.resources))
	}
	if v.isLoadingMore {
		statusLine += " (loading more...)"
	} else if v.hasMorePages {
		statusLine += " (N for more)"
	}
	if len(v.partialErrors) > 0 {
		statusLine += fmt.Sprintf(" [%d region errors]", len(v.partialErrors))
	}

	status := s.status.Render(statusLine)

	filterView := ""
	if v.filterActive {
		filterView = s.filterWrap.Render(v.filterInput.View()) + "\n"
	} else if v.filterText != "" {
		filterView = s.filterActive.Render(fmt.Sprintf("filter: %s", v.filterText)) + "\n"
	}

	if len(v.filtered) == 0 && len(v.resources) > 0 {
		return header + "\n" + status + "\n" + filterView +
			ui.DimStyle().Render("No matching resources (press 'c' to clear filter)")
	}

	if len(v.resources) == 0 {
		msg := "No tagged resources found"
		if v.tagFilter != "" {
			msg = fmt.Sprintf("No resources with tag '%s' found", v.tagFilter)
		}
		return header + "\n" + status + "\n" + ui.DimStyle().Render(msg)
	}

	return header + "\n" + status + "\n" + filterView + v.tableContent
}

func (v *TagSearchView) View() tea.View {
	return tea.NewView(v.ViewString())
}

func (v *TagSearchView) SetSize(width, height int) tea.Cmd {
	v.width = width
	v.height = height
	v.filterInput.SetWidth(width - 4)
	if len(v.resources) > 0 {
		v.buildTable()
	}
	return nil
}

func (v *TagSearchView) StatusLine() string {
	if v.filterActive {
		return fmt.Sprintf("/%s • %d/%d items • Esc:done Enter:apply", v.filterInput.Value(), len(v.filtered), len(v.resources))
	}

	count := len(v.filtered)
	regions := config.Global().Regions()
	regionInfo := ""
	if len(regions) > 1 {
		regionInfo = fmt.Sprintf(" (%d regions)", len(regions))
	}

	if v.tagFilter != "" {
		if v.filterText != "" {
			return fmt.Sprintf("Tag Search: %s • %d/%d%s (/%s)", v.tagFilter, count, len(v.resources), regionInfo, v.filterText)
		}
		return fmt.Sprintf("Tag Search: %s • %d resources%s", v.tagFilter, count, regionInfo)
	}
	if v.filterText != "" {
		return fmt.Sprintf("Tag Search • %d/%d%s (/%s)", count, len(v.resources), regionInfo, v.filterText)
	}
	return fmt.Sprintf("Tag Search • %d resources%s", count, regionInfo)
}

func (v *TagSearchView) HasActiveInput() bool {
	return v.filterActive
}

func (v *TagSearchView) getRowAtPosition(y int) int {
	headerHeight := 4
	if v.filterActive || v.filterText != "" {
		headerHeight++
	}

	visualRow := y - headerHeight
	dataIdx := visualRow + v.tc.ScrollOffset()
	if visualRow >= 0 && dataIdx >= 0 && dataIdx < len(v.filtered) {
		return dataIdx
	}
	return -1
}

func (v *TagSearchView) GetTagKeys() []string {
	keySet := make(map[string]struct{})
	for _, res := range v.resources {
		for key := range res.Tags {
			keySet[key] = struct{}{}
		}
	}

	keys := slices.Collect(maps.Keys(keySet))
	slices.Sort(keys)
	return keys
}

func (v *TagSearchView) GetTagValues(key string) []string {
	valueSet := make(map[string]struct{})
	keyLower := strings.ToLower(key)

	for _, res := range v.resources {
		for k, val := range res.Tags {
			if strings.ToLower(k) == keyLower {
				valueSet[val] = struct{}{}
			}
		}
	}

	values := slices.Collect(maps.Keys(valueSet))
	slices.Sort(values)
	return values
}

func formatTags(tags map[string]string, maxLen int) string {
	if tags == nil {
		return ""
	}

	keys := slices.Collect(maps.Keys(tags))
	slices.Sort(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", sanitizeTagValue(k), sanitizeTagValue(tags[k])))
	}

	result := strings.Join(parts, ", ")
	if len([]rune(result)) > maxLen {
		runes := []rune(result)
		result = string(runes[:maxLen-1]) + "…"
	}
	return result
}

func sanitizeTagValue(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= 32 && r != 127 {
			b.WriteRune(r)
		}
	}
	return b.String()
}
