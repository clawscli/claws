package view

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/filter"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/ui"
)

// TagBrowser displays resources filtered by tags across all services
type TagBrowser struct {
	ctx       context.Context
	registry  *registry.Registry
	tagFilter string // e.g., "Environment=production" or "Environment"
	table     table.Model
	resources []taggedResource
	filtered  []taggedResource
	loading   bool
	err       error
	width     int
	height    int

	// Filter input
	filterActive bool
	filterText   string
	filterInput  textinput.Model
}

// taggedResource holds a resource with its service/type context
type taggedResource struct {
	Service      string
	ResourceType string
	Resource     dao.Resource
}

// NewTagBrowser creates a new TagBrowser
func NewTagBrowser(ctx context.Context, reg *registry.Registry, tagFilter string) *TagBrowser {
	ti := textinput.New()
	ti.Placeholder = "filter..."
	ti.Prompt = "/"
	ti.CharLimit = 100

	return &TagBrowser{
		ctx:         ctx,
		registry:    reg,
		tagFilter:   tagFilter,
		loading:     true,
		filterInput: ti,
	}
}

// Init implements tea.Model
func (t *TagBrowser) Init() tea.Cmd {
	return t.loadResources
}

type tagResourcesLoadedMsg struct {
	resources []taggedResource
}

type tagResourcesErrorMsg struct {
	err error
}

func (t *TagBrowser) loadResources() tea.Msg {
	var results []taggedResource

	// Iterate through all services and resources
	for _, service := range t.registry.ListServices() {
		for _, resourceType := range t.registry.ListResources(service) {
			d, err := t.registry.GetDAO(t.ctx, service, resourceType)
			if err != nil {
				continue
			}

			resources, err := d.List(t.ctx)
			if err != nil {
				continue
			}

			for _, res := range resources {
				if t.matchesTagFilter(res) {
					results = append(results, taggedResource{
						Service:      service,
						ResourceType: resourceType,
						Resource:     res,
					})
				}
			}
		}
	}

	return tagResourcesLoadedMsg{resources: results}
}

func (t *TagBrowser) matchesTagFilter(res dao.Resource) bool {
	return filter.MatchesTagFilter(res.GetTags(), t.tagFilter)
}

// Update implements tea.Model
func (t *TagBrowser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tagResourcesLoadedMsg:
		t.loading = false
		t.resources = msg.resources
		t.applyFilter()
		t.buildTable()
		return t, nil

	case tagResourcesErrorMsg:
		t.loading = false
		t.err = msg.err
		return t, nil

	case tea.KeyMsg:
		// Handle filter input mode
		if t.filterActive {
			switch msg.String() {
			case "esc":
				t.filterActive = false
				t.filterInput.Blur()
				return t, nil
			case "enter":
				t.filterActive = false
				t.filterInput.Blur()
				t.filterText = t.filterInput.Value()
				t.applyFilter()
				t.buildTable()
				return t, nil
			default:
				var cmd tea.Cmd
				t.filterInput, cmd = t.filterInput.Update(msg)
				// Live filter as user types
				t.filterText = t.filterInput.Value()
				t.applyFilter()
				t.buildTable()
				return t, cmd
			}
		}

		switch msg.String() {
		case "/":
			t.filterActive = true
			t.filterInput.Focus()
			return t, textinput.Blink

		case "c":
			// Clear filter
			t.filterText = ""
			t.filterInput.SetValue("")
			t.applyFilter()
			t.buildTable()
			return t, nil

		case "enter", "d":
			if len(t.filtered) > 0 && t.table.Cursor() < len(t.filtered) {
				selected := t.filtered[t.table.Cursor()]
				// Navigate to the resource's service/type view
				return t, func() tea.Msg {
					return NavigateMsg{
						View: NewResourceBrowserWithType(t.ctx, t.registry, selected.Service, selected.ResourceType),
					}
				}
			}

		case "j", "down":
			t.table.MoveDown(1)
			return t, nil

		case "k", "up":
			t.table.MoveUp(1)
			return t, nil
		}
	}

	var cmd tea.Cmd
	t.table, cmd = t.table.Update(msg)
	return t, cmd
}

// applyFilter filters resources based on filterText using fuzzy matching
func (t *TagBrowser) applyFilter() {
	if t.filterText == "" {
		t.filtered = t.resources
		return
	}

	filter := strings.ToLower(t.filterText)
	t.filtered = nil

	for _, tr := range t.resources {
		// Fuzzy search in service, type, ID, name
		if fuzzyMatch(tr.Service, filter) ||
			fuzzyMatch(tr.ResourceType, filter) ||
			fuzzyMatch(tr.Resource.GetID(), filter) ||
			fuzzyMatch(tr.Resource.GetName(), filter) {
			t.filtered = append(t.filtered, tr)
			continue
		}

		// Fuzzy search in tags
		tags := tr.Resource.GetTags()
		for k, v := range tags {
			if fuzzyMatch(k, filter) || fuzzyMatch(v, filter) {
				t.filtered = append(t.filtered, tr)
				break
			}
		}
	}
}

// fuzzyMatch checks if pattern characters appear in order in str (case insensitive)
func fuzzyMatch(str, pattern string) bool {
	str = strings.ToLower(str)
	pi := 0
	for i := 0; i < len(str) && pi < len(pattern); i++ {
		if str[i] == pattern[pi] {
			pi++
		}
	}
	return pi == len(pattern)
}

func (t *TagBrowser) buildTable() {
	columns := []table.Column{
		{Title: "Service", Width: 15},
		{Title: "Type", Width: 15},
		{Title: "ID", Width: 25},
		{Title: "Name", Width: 25},
		{Title: "Tags", Width: 40},
	}

	rows := make([]table.Row, len(t.filtered))
	for i, tr := range t.filtered {
		tags := tr.Resource.GetTags()
		tagStr := formatTags(tags, 40)
		rows[i] = table.Row{
			tr.Service,
			tr.ResourceType,
			tr.Resource.GetID(),
			tr.Resource.GetName(),
			tagStr,
		}
	}

	tbl := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(t.height-6),
	)

	s := table.DefaultStyles()
	theme := ui.Current()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(theme.TableBorder).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(theme.SelectionText).
		Background(theme.Selection).
		Bold(false)

	tbl.SetStyles(s)
	t.table = tbl
}

func formatTags(tags map[string]string, maxLen int) string {
	if tags == nil {
		return ""
	}

	var parts []string
	for k, v := range tags {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}

	result := strings.Join(parts, ", ")
	if len(result) > maxLen {
		result = result[:maxLen-3] + "..."
	}
	return result
}

// View implements tea.Model
func (t *TagBrowser) View() string {
	theme := ui.Current()

	// Header
	title := "Tags"
	if t.tagFilter != "" {
		title = fmt.Sprintf("Tags: %s", t.tagFilter)
	}
	header := lipgloss.NewStyle().
		Foreground(theme.TableHeaderText).
		Background(theme.TableHeader).
		Padding(0, 1).
		Width(t.width).
		Render(title)

	// Status line
	statusLine := ""
	if t.loading {
		statusLine = "Loading..."
	} else if t.err != nil {
		statusLine = fmt.Sprintf("Error: %v", t.err)
	} else if t.filterText != "" {
		statusLine = fmt.Sprintf("Found %d/%d resources (filter: %s)", len(t.filtered), len(t.resources), t.filterText)
	} else {
		statusLine = fmt.Sprintf("Found %d resources", len(t.resources))
	}
	status := lipgloss.NewStyle().
		Foreground(theme.TextDim).
		Padding(0, 1).
		Render(statusLine)

	// Filter input (if active)
	filterView := ""
	if t.filterActive {
		filterView = lipgloss.NewStyle().
			Padding(0, 1).
			Render(t.filterInput.View()) + "\n"
	}

	// Help
	helpText := "↑/k up • ↓/j down • / filter • c clear • enter select • esc back"
	if t.filterActive {
		helpText = "Type to filter • enter confirm • esc cancel"
	}
	help := lipgloss.NewStyle().
		Foreground(theme.TextDim).
		Padding(0, 1).
		Render(helpText)

	return header + "\n" + status + "\n" + filterView + t.table.View() + "\n" + help
}

// SetSize sets the view size
func (t *TagBrowser) SetSize(width, height int) tea.Cmd {
	t.width = width
	t.height = height
	if t.table.Columns() != nil {
		t.table.SetHeight(height - 6)
	}
	return nil
}

// StatusLine returns the status line for this view
func (t *TagBrowser) StatusLine() string {
	count := len(t.filtered)
	if t.tagFilter != "" {
		if t.filterText != "" {
			return fmt.Sprintf("Tags: %s • %d/%d (/%s)", t.tagFilter, count, len(t.resources), t.filterText)
		}
		return fmt.Sprintf("Tags: %s • %d resources", t.tagFilter, count)
	}
	if t.filterText != "" {
		return fmt.Sprintf("Tags • %d/%d (/%s)", count, len(t.resources), t.filterText)
	}
	return fmt.Sprintf("Tags • %d resources", count)
}

// HasActiveInput returns true when filter input is active
func (t *TagBrowser) HasActiveInput() bool {
	return t.filterActive
}
