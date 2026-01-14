package view

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/clipboard"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/log"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
	"github.com/clawscli/claws/internal/ui"
)

const minViewportHeight = 5

// DetailView displays detailed information about a single resource
// detailViewStyles holds cached lipgloss styles for performance
type detailViewStyles struct {
	title lipgloss.Style
	label lipgloss.Style
	value lipgloss.Style
}

func newDetailViewStyles() detailViewStyles {
	return detailViewStyles{
		title: ui.TitleStyle(),
		label: ui.DimStyle().Width(15),
		value: ui.TextStyle(),
	}
}

type DetailView struct {
	ctx         context.Context
	resource    dao.Resource
	renderer    render.Renderer
	service     string
	resType     string
	vp          ViewportState
	headerPanel *HeaderPanel
	registry    *registry.Registry
	dao         dao.DAO
	refreshing  bool
	refreshErr  error
	spinner     spinner.Model
	styles      detailViewStyles
	width       int
	height      int
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
			d.resource = mergeResources(d.resource, msg.resource)
			if d.vp.Ready {
				content := d.renderContent()
				d.vp.Model.SetContent(content)
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
	case ThemeChangedMsg:
		d.styles = newDetailViewStyles()
		d.headerPanel.ReloadStyles()
		if d.vp.Ready {
			d.vp.Model.SetContent(d.renderContent())
		}
		return d, nil
	case CompactHeaderChangedMsg:
		d.recalcViewport()
		return d, nil

	case tea.KeyPressMsg:
		// Let app handle back navigation (esc/backspace/q handled by app.go)
		if IsEscKey(msg) {
			return d, nil
		}

		// Check navigation shortcuts
		if model, cmd := d.handleNavigation(msg.String()); model != nil {
			return model, cmd
		}

		switch msg.String() {
		case "a":
			if actions := action.Global.Get(d.service, d.resType); len(actions) > 0 {
				actionMenu := NewActionMenu(d.ctx, dao.UnwrapResource(d.resource), d.service, d.resType)
				return d, func() tea.Msg {
					return ShowModalMsg{Modal: &Modal{Content: actionMenu, Width: ModalWidthActionMenu}}
				}
			}
		case "y":
			return d, clipboard.CopyID(dao.UnwrapResource(d.resource).GetID())
		case "Y":
			resource := dao.UnwrapResource(d.resource)
			if arn := resource.GetARN(); arn != "" {
				return d, clipboard.CopyARN(arn)
			}
			return d, clipboard.NoARN()
		}
	}

	var cmd tea.Cmd
	d.vp.Model, cmd = d.vp.Model.Update(msg)
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

	if cmd := helper.HandleKey(key, dao.UnwrapResource(d.resource)); cmd != nil {
		return d, cmd
	}

	return nil, nil
}

func (d *DetailView) ViewString() string {
	if !d.vp.Ready {
		return LoadingMessage
	}

	var summaryFields []render.SummaryField
	if d.renderer != nil {
		summaryFields = d.renderer.RenderSummary(dao.UnwrapResource(d.resource))
	}

	header := d.headerPanel.Render(d.service, d.resType, summaryFields)

	return header + "\n" + d.vp.Model.View()
}

// View implements tea.Model
func (d *DetailView) View() tea.View {
	return tea.NewView(d.ViewString())
}

// SetSize implements View
func (d *DetailView) SetSize(width, height int) tea.Cmd {
	d.width = width
	d.height = height
	d.headerPanel.SetWidth(width)

	d.recalcViewport()

	return nil
}

func (d *DetailView) recalcViewport() {
	// Calculate header height dynamically
	var summaryFields []render.SummaryField
	if d.renderer != nil {
		summaryFields = d.renderer.RenderSummary(dao.UnwrapResource(d.resource))
	}
	headerStr := d.headerPanel.Render(d.service, d.resType, summaryFields)
	headerHeight := d.headerPanel.Height(headerStr)

	// +1 compensates for border overlap
	viewportHeight := max(d.height-headerHeight+1, minViewportHeight)

	d.vp.SetSize(d.width, viewportHeight)

	if !d.vp.Ready {
		return
	}
	content := d.renderContent()
	d.vp.Model.SetContent(content)
}

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

	parts = append(parts, "y:copy")

	if navInfo := d.getNavigationShortcuts(); navInfo != "" {
		parts = append(parts, navInfo)
	}

	parts = append(parts, "q/esc:back")
	return strings.Join(parts, " • ")
}

func (d *DetailView) Resource() dao.Resource {
	return d.resource
}

func (d *DetailView) Service() string {
	return d.service
}

func (d *DetailView) ResourceType() string {
	return d.resType
}

// getNavigationShortcuts returns a string of navigation shortcuts for the current resource
func (d *DetailView) getNavigationShortcuts() string {
	if d.renderer == nil {
		return ""
	}

	helper := &NavigationHelper{Renderer: d.renderer}
	return helper.FormatShortcuts(dao.UnwrapResource(d.resource))
}

func (d *DetailView) renderContent() string {
	var detail string

	// Try to use renderer's RenderDetail if available
	if d.renderer != nil {
		detail = d.renderer.RenderDetail(dao.UnwrapResource(d.resource))
	}

	// Fallback to generic detail view
	if detail == "" {
		detail = d.renderGenericDetail()
	}

	// Replace placeholder values with "Loading..." during async refresh.
	// Match placeholders only at line endings to avoid replacing substrings
	// (e.g., "Not configured server" should not be replaced).
	if d.refreshing && detail != "" {
		loading := ui.DimStyle().Render(LoadingMessage)

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
