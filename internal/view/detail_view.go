package view

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/dao"
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
	styles      detailViewStyles
}

// NewDetailView creates a new DetailView
func NewDetailView(ctx context.Context, resource dao.Resource, renderer render.Renderer, service, resType string, reg *registry.Registry) *DetailView {
	hp := NewHeaderPanel()
	hp.SetWidth(120) // Default width until SetSize is called

	return &DetailView{
		ctx:         ctx,
		resource:    resource,
		renderer:    renderer,
		service:     service,
		resType:     resType,
		registry:    reg,
		headerPanel: hp,
		styles:      newDetailViewStyles(),
	}
}

// Init implements tea.Model
func (d *DetailView) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (d *DetailView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		// Check for esc (both string and raw byte) - let app handle back navigation
		isEsc := keyMsg.String() == "esc" || keyMsg.Type == tea.KeyEsc || keyMsg.Type == tea.KeyEscape ||
			(keyMsg.Type == tea.KeyRunes && len(keyMsg.Runes) == 1 && keyMsg.Runes[0] == 27)
		if isEsc {
			return d, nil
		}

		// Check navigation shortcuts
		if model, cmd := d.handleNavigation(keyMsg.String()); model != nil {
			return model, cmd
		}

		// Open action menu (only if actions exist)
		if keyMsg.String() == "a" {
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
	parts := []string{d.resource.GetID(), "↑/↓:scroll"}

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
	// Try to use renderer's RenderDetail if available
	if d.renderer != nil {
		detail := d.renderer.RenderDetail(d.resource)
		if detail != "" {
			return detail
		}
	}

	// Fallback to generic detail view
	return d.renderGenericDetail()
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
