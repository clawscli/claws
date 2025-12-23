package view

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/log"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
	"github.com/clawscli/claws/internal/ui"
)

// DetailView displays detailed information about a single resource
// detailViewStyles holds cached lipgloss styles for performance
type detailViewStyles struct {
	title lipgloss.Style
	label lipgloss.Style
	value lipgloss.Style
}

func newDetailViewStyles() detailViewStyles {
	t := ui.Current()
	return detailViewStyles{
		title: lipgloss.NewStyle().Bold(true).Foreground(t.Primary),
		label: lipgloss.NewStyle().Foreground(t.TextDim).Width(15),
		value: lipgloss.NewStyle().Foreground(t.Text),
	}
}

type DetailView struct {
	ctx         context.Context
	resource    dao.Resource
	renderer    render.Renderer
	service     string
	resType     string
	viewport    viewport.Model
	headerPanel *HeaderPanel
	ready       bool
	width       int
	height      int
	registry    *registry.Registry
	dao         dao.DAO // for async refresh
	refreshing  bool    // true while fetching extended details
	refreshErr  error   // error from last refresh attempt
	spinner     spinner.Model
	styles      detailViewStyles
}

// NewDetailView creates a new DetailView
func NewDetailView(ctx context.Context, resource dao.Resource, renderer render.Renderer, service, resType string, reg *registry.Registry, d dao.DAO) *DetailView {
	hp := NewHeaderPanel()
	hp.SetWidth(120) // Default width until SetSize is called

	return &DetailView{
		ctx:         ctx,
		resource:    resource,
		renderer:    renderer,
		service:     service,
		resType:     resType,
		registry:    reg,
		dao:         d,
		headerPanel: hp,
		spinner:     ui.NewSpinner(),
		styles:      newDetailViewStyles(),
	}
}

// detailRefreshMsg is sent when async resource refresh completes
type detailRefreshMsg struct {
	resource dao.Resource
	err      error
}

// Init implements tea.Model
func (d *DetailView) Init() tea.Cmd {
	// Start async refresh for extended details if DAO supports Get operation
	if d.dao != nil && d.dao.Supports(dao.OpGet) {
		d.refreshing = true
		return tea.Batch(d.spinner.Tick, d.refreshResource)
	}
	return nil
}

// refreshResource fetches extended resource details in background
func (d *DetailView) refreshResource() tea.Msg {
	if d.dao == nil || d.resource == nil {
		return detailRefreshMsg{resource: d.resource}
	}
	refreshed, err := d.dao.Get(d.ctx, d.resource.GetID())
	if err != nil {
		return detailRefreshMsg{resource: d.resource, err: err}
	}
	return detailRefreshMsg{resource: refreshed}
}

// Update implements tea.Model
func (d *DetailView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case detailRefreshMsg:
		d.refreshing = false
		if msg.err != nil {
			log.Warn("failed to refresh resource details", "error", msg.err)
			d.refreshErr = msg.err
		} else {
			d.refreshErr = nil
			// Merge refreshed resource with original to preserve List-only fields
			d.resource = mergeResources(d.resource, msg.resource)
			// Re-render content with refreshed data
			if d.ready {
				content := d.renderContent()
				d.viewport.SetContent(content)
			}
		}
		return d, nil

	case spinner.TickMsg:
		if d.refreshing {
			var cmd tea.Cmd
			d.spinner, cmd = d.spinner.Update(msg)
			return d, cmd
		}
		return d, nil

	case tea.KeyMsg:
		// Let app handle back navigation
		if IsEscKey(msg) {
			return d, nil
		}

		// Check navigation shortcuts
		if model, cmd := d.handleNavigation(msg.String()); model != nil {
			return model, cmd
		}

		// Open action menu (only if actions exist)
		if msg.String() == "a" {
			if actions := action.Global.Get(d.service, d.resType); len(actions) > 0 {
				actionMenu := NewActionMenu(d.ctx, d.resource, d.service, d.resType)
				return d, func() tea.Msg {
					return NavigateMsg{View: actionMenu}
				}
			}
		}
	}

	// Pass other messages to viewport for scrolling
	var cmd tea.Cmd
	d.viewport, cmd = d.viewport.Update(msg)
	return d, cmd
}

// handleNavigation checks if a key matches a navigation shortcut
func (d *DetailView) handleNavigation(key string) (tea.Model, tea.Cmd) {
	if d.renderer == nil || d.registry == nil {
		return nil, nil
	}

	helper := &NavigationHelper{
		Ctx:      d.ctx,
		Registry: d.registry,
		Renderer: d.renderer,
	}

	if cmd := helper.HandleKey(key, d.resource); cmd != nil {
		return d, cmd
	}

	return nil, nil
}

// View implements tea.Model
func (d *DetailView) View() string {
	if !d.ready {
		return "Loading..."
	}

	// Get summary fields for header
	var summaryFields []render.SummaryField
	if d.renderer != nil {
		summaryFields = d.renderer.RenderSummary(d.resource)
	}

	// Render header panel
	header := d.headerPanel.Render(d.service, d.resType, summaryFields)

	return header + "\n" + d.viewport.View()
}

// SetSize implements View
func (d *DetailView) SetSize(width, height int) tea.Cmd {
	d.width = width
	d.height = height

	// Set header panel width
	d.headerPanel.SetWidth(width)

	// Calculate header height dynamically
	var summaryFields []render.SummaryField
	if d.renderer != nil {
		summaryFields = d.renderer.RenderSummary(d.resource)
	}
	headerStr := d.headerPanel.Render(d.service, d.resType, summaryFields)
	headerHeight := d.headerPanel.Height(headerStr)

	// height - header + extra space
	viewportHeight := height - headerHeight + 1
	if viewportHeight < 5 {
		viewportHeight = 5
	}

	if !d.ready {
		d.viewport = viewport.New(width, viewportHeight)
		d.ready = true
	} else {
		d.viewport.Width = width
		d.viewport.Height = viewportHeight
	}

	// Render content
	content := d.renderContent()
	d.viewport.SetContent(content)

	return nil
}

// StatusLine implements View
func (d *DetailView) StatusLine() string {
	parts := []string{d.resource.GetID()}

	if d.refreshing {
		parts = append(parts, d.spinner.View()+" refreshing...")
	} else if d.refreshErr != nil {
		parts = append(parts, "⚠ refresh failed")
	}

	parts = append(parts, "↑/↓:scroll")

	if actions := action.Global.Get(d.service, d.resType); len(actions) > 0 {
		parts = append(parts, "a:actions")
	}

	// Add navigation shortcuts
	if navInfo := d.getNavigationShortcuts(); navInfo != "" {
		parts = append(parts, navInfo)
	}

	parts = append(parts, "esc:back")
	return strings.Join(parts, " • ")
}

// getNavigationShortcuts returns a string of navigation shortcuts for the current resource
func (d *DetailView) getNavigationShortcuts() string {
	if d.renderer == nil {
		return ""
	}

	helper := &NavigationHelper{Renderer: d.renderer}
	return helper.FormatShortcuts(d.resource)
}

func (d *DetailView) renderContent() string {
	var detail string

	// Try to use renderer's RenderDetail if available
	if d.renderer != nil {
		detail = d.renderer.RenderDetail(d.resource)
	}

	// Fallback to generic detail view
	if detail == "" {
		detail = d.renderGenericDetail()
	}

	// Replace placeholder values with "Loading..." during async refresh.
	// Match placeholders only at line endings to avoid replacing substrings
	// (e.g., "Not configured server" should not be replaced).
	if d.refreshing && detail != "" {
		loading := ui.DimStyle().Render("Loading...")

		// Replace placeholders at end of line or end of content
		for _, placeholder := range []string{render.NotConfigured, render.Empty, render.NoValue} {
			detail = strings.ReplaceAll(detail, placeholder+"\n", loading+"\n")
			if strings.HasSuffix(detail, placeholder) {
				detail = detail[:len(detail)-len(placeholder)] + loading
				break // Only one placeholder can be at EOF
			}
		}
	}

	return detail
}

func (d *DetailView) renderGenericDetail() string {
	s := d.styles

	var out string
	out += s.title.Render("Resource Details") + "\n\n"
	out += s.label.Render("ID:") + s.value.Render(d.resource.GetID()) + "\n"
	out += s.label.Render("Name:") + s.value.Render(d.resource.GetName()) + "\n"

	if arn := d.resource.GetARN(); arn != "" {
		out += s.label.Render("ARN:") + s.value.Render(arn) + "\n"
	}

	out += "\n" + ui.DimStyle().Render("(Raw data view not implemented)")

	return out
}

// mergeResources merges the refreshed resource with the original to preserve
// fields that are only available from List() but not from Get().
func mergeResources(original, refreshed dao.Resource) dao.Resource {
	if original == nil {
		return refreshed
	}
	if refreshed == nil {
		return original
	}
	// If refreshed resource implements Mergeable, let it copy fields from original
	if m, ok := refreshed.(dao.Mergeable); ok {
		m.MergeFrom(original)
	}
	return refreshed
}
