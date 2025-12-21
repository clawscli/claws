package view

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/ui"
)

// Grid layout constants
const (
	cellWidth    = 24 // Width of each grid cell
	cellHeight   = 2  // Height of each cell (service name + aliases)
	minColumns   = 2
	maxColumns   = 6
	cellPaddingX = 2
)

// ServiceBrowser displays available AWS services in a grid layout grouped by category
type ServiceBrowser struct {
	ctx      context.Context
	registry *registry.Registry

	// Category-based data
	categories []categoryGroup
	flatItems  []flatItem // Flattened list for navigation

	cursor int // Current selection index in flatItems
	cols   int // Number of columns in grid
	width  int
	height int

	// Header panel
	headerPanel *HeaderPanel

	// Viewport for scrolling
	viewport viewport.Model
	ready    bool

	// Filter
	filterInput  textinput.Model
	filterActive bool
	filterText   string

	// Cached styles
	styles serviceBrowserStyles
}

// categoryGroup holds services for a category
type categoryGroup struct {
	name     string
	services []serviceItem
}

// flatItem represents a service with its global index for navigation
type flatItem struct {
	service      serviceItem
	categoryIdx  int
	indexInGroup int
}

type serviceBrowserStyles struct {
	title         lipgloss.Style
	category      lipgloss.Style
	cell          lipgloss.Style
	cellSelected  lipgloss.Style
	serviceName   lipgloss.Style
	serviceNameSe lipgloss.Style // Selected service name
	aliases       lipgloss.Style
	aliasesSel    lipgloss.Style // Selected aliases
	filterPrompt  lipgloss.Style
}

func newServiceBrowserStyles() serviceBrowserStyles {
	t := ui.Current()
	return serviceBrowserStyles{
		title: lipgloss.NewStyle().
			Background(t.TableHeader).
			Foreground(t.TableHeaderText).
			Padding(0, 1).
			MarginBottom(1),
		category: lipgloss.NewStyle().
			Foreground(t.TextDim).
			Bold(true).
			MarginTop(1).
			MarginBottom(0),
		cell: lipgloss.NewStyle().
			Width(cellWidth).
			Height(cellHeight).
			Padding(0, 1),
		cellSelected: lipgloss.NewStyle().
			Width(cellWidth).
			Height(cellHeight).
			Padding(0, 1).
			Background(t.Selection),
		serviceName: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Text),
		serviceNameSe: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Primary),
		aliases: lipgloss.NewStyle().
			Foreground(t.TextDim),
		aliasesSel: lipgloss.NewStyle().
			Foreground(t.TextDim),
		filterPrompt: lipgloss.NewStyle().
			Foreground(t.Primary),
	}
}

type serviceItem struct {
	name        string   // internal service name (e.g., "ssm")
	displayName string   // display name (e.g., "Systems Manager")
	aliases     []string // command aliases
}

// filterValue returns searchable text for filtering
func (i serviceItem) filterValue() string {
	return strings.ToLower(i.name + " " + i.displayName + " " + strings.Join(i.aliases, " "))
}

// NewServiceBrowser creates a new ServiceBrowser
func NewServiceBrowser(ctx context.Context, reg *registry.Registry) *ServiceBrowser {
	ti := textinput.New()
	ti.Placeholder = "filter..."
	ti.Prompt = "/"
	ti.CharLimit = 30

	hp := NewHeaderPanel()
	hp.SetWidth(120)

	return &ServiceBrowser{
		ctx:         ctx,
		registry:    reg,
		cols:        4, // Default columns
		headerPanel: hp,
		styles:      newServiceBrowserStyles(),
		filterInput: ti,
	}
}

// Init implements tea.Model
func (s *ServiceBrowser) Init() tea.Cmd {
	return s.loadServices
}

func (s *ServiceBrowser) loadServices() tea.Msg {
	cats := s.registry.ListServicesByCategory()
	groups := make([]categoryGroup, 0, len(cats))

	for _, cat := range cats {
		items := make([]serviceItem, 0, len(cat.Services))
		for _, svc := range cat.Services {
			aliases := s.registry.GetAliasesForService(svc)
			items = append(items, serviceItem{
				name:        svc,
				displayName: s.registry.GetDisplayName(svc),
				aliases:     aliases,
			})
		}
		groups = append(groups, categoryGroup{
			name:     cat.Name,
			services: items,
		})
	}

	return servicesLoadedMsg{categories: groups}
}

type servicesLoadedMsg struct {
	categories []categoryGroup
}

// Update implements tea.Model
func (s *ServiceBrowser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case servicesLoadedMsg:
		s.categories = msg.categories
		s.rebuildFlatItems()
		return s, nil

	case RefreshMsg:
		return s, s.loadServices

	case tea.KeyMsg:
		if s.filterActive {
			return s.handleFilterInput(msg)
		}
		return s.handleNavigation(msg)
	}

	return s, nil
}

func (s *ServiceBrowser) rebuildFlatItems() {
	s.flatItems = nil
	filter := strings.ToLower(s.filterText)

	for catIdx, cat := range s.categories {
		idxInGroup := 0
		for _, svc := range cat.services {
			if filter == "" || strings.Contains(svc.filterValue(), filter) {
				s.flatItems = append(s.flatItems, flatItem{
					service:      svc,
					categoryIdx:  catIdx,
					indexInGroup: idxInGroup,
				})
				idxInGroup++
			}
		}
	}

	// Reset cursor if out of bounds
	if s.cursor >= len(s.flatItems) {
		s.cursor = len(s.flatItems) - 1
	}
	if s.cursor < 0 {
		s.cursor = 0
	}

	// Update viewport content after rebuilding items
	s.updateViewport()
}

func (s *ServiceBrowser) handleFilterInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if IsEscKey(msg) {
		s.filterActive = false
		s.filterInput.Blur()
		return s, nil
	}

	switch msg.String() {

	case "enter":
		s.filterActive = false
		s.filterInput.Blur()
		if len(s.flatItems) == 1 {
			return s.selectCurrentService()
		}
		return s, nil
	}

	var cmd tea.Cmd
	s.filterInput, cmd = s.filterInput.Update(msg)
	s.filterText = s.filterInput.Value()
	s.rebuildFlatItems()
	s.cursor = 0
	s.updateViewport()
	return s, cmd
}

func (s *ServiceBrowser) handleNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(s.flatItems) == 0 {
		if msg.String() == "/" {
			s.filterActive = true
			s.filterInput.Focus()
			return s, textinput.Blink
		}
		return s, nil
	}

	switch msg.String() {
	case "j", "down":
		// Move to next category
		s.moveToNextCategory()

	case "k", "up":
		// Move to previous category
		s.moveToPrevCategory()

	case "h", "left":
		// Move within category (previous item)
		if s.cursor > 0 {
			currentCat := s.flatItems[s.cursor].categoryIdx
			if s.flatItems[s.cursor-1].categoryIdx == currentCat {
				s.cursor--
			}
		}

	case "l", "right":
		// Move within category (next item)
		if s.cursor < len(s.flatItems)-1 {
			currentCat := s.flatItems[s.cursor].categoryIdx
			if s.flatItems[s.cursor+1].categoryIdx == currentCat {
				s.cursor++
			}
		}

	case "enter":
		return s.selectCurrentService()

	case "/":
		s.filterActive = true
		s.filterInput.Focus()
		return s, textinput.Blink

	case "c":
		if s.filterText != "" {
			s.filterText = ""
			s.filterInput.SetValue("")
			s.rebuildFlatItems()
			s.cursor = 0
		}
	}

	// Also allow esc to clear filter (handles various escape sequences)
	if IsEscKey(msg) && s.filterText != "" {
		s.filterText = ""
		s.filterInput.SetValue("")
		s.rebuildFlatItems()
		s.cursor = 0
	}

	// Update viewport content and scroll to cursor
	s.updateViewport()

	return s, nil
}

// updateViewport updates the viewport content and scrolls to make cursor visible
func (s *ServiceBrowser) updateViewport() {
	if !s.ready {
		return
	}
	content := s.renderContent()
	s.viewport.SetContent(content)

	// Count total lines and find cursor line by scanning rendered content
	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	// Estimate cursor position based on proportion through flatItems
	if len(s.flatItems) == 0 {
		return
	}

	// Calculate approximate line position
	cursorRatio := float64(s.cursor) / float64(len(s.flatItems))
	targetLine := int(cursorRatio * float64(totalLines))

	vpHeight := s.viewport.Height
	currentTop := s.viewport.YOffset

	// Scroll if cursor is outside visible area
	if targetLine < currentTop {
		s.viewport.SetYOffset(max(0, targetLine-2))
	} else if targetLine > currentTop+vpHeight-cellHeight {
		newOffset := targetLine - vpHeight + cellHeight + 2
		if newOffset > totalLines-vpHeight {
			newOffset = totalLines - vpHeight
		}
		if newOffset < 0 {
			newOffset = 0
		}
		s.viewport.SetYOffset(newOffset)
	}
}

func (s *ServiceBrowser) moveToNextCategory() {
	if s.cursor >= len(s.flatItems)-1 {
		return
	}

	currentCat := s.flatItems[s.cursor].categoryIdx

	// Find first item of next category
	for i := s.cursor + 1; i < len(s.flatItems); i++ {
		if s.flatItems[i].categoryIdx != currentCat {
			s.cursor = i
			return
		}
	}
}

func (s *ServiceBrowser) moveToPrevCategory() {
	if s.cursor <= 0 {
		return
	}

	currentCat := s.flatItems[s.cursor].categoryIdx

	// If we're not at the first item of current category, go to first item
	for i := s.cursor - 1; i >= 0; i-- {
		if s.flatItems[i].categoryIdx != currentCat {
			// Found previous category, now find its first item
			prevCat := s.flatItems[i].categoryIdx
			for j := i; j >= 0; j-- {
				if s.flatItems[j].categoryIdx != prevCat {
					s.cursor = j + 1
					return
				}
				if j == 0 {
					s.cursor = 0
					return
				}
			}
			return
		}
	}

	// We're at first category, go to first item
	s.cursor = 0
}

func (s *ServiceBrowser) selectCurrentService() (tea.Model, tea.Cmd) {
	if s.cursor >= 0 && s.cursor < len(s.flatItems) {
		item := s.flatItems[s.cursor]
		resourceBrowser := NewResourceBrowser(s.ctx, s.registry, item.service.name)
		return s, func() tea.Msg {
			return NavigateMsg{View: resourceBrowser}
		}
	}
	return s, nil
}

// View implements tea.Model
func (s *ServiceBrowser) View() string {
	// Header panel (always visible at top)
	header := s.headerPanel.RenderHome()

	if !s.ready {
		return header + "\n" + "Loading..."
	}

	// Filter input
	var footer string
	if s.filterActive {
		footer = "\n" + s.styles.filterPrompt.Render(s.filterInput.View())
	}

	return header + "\n" + s.viewport.View() + footer
}

// renderContent renders the service grid content for the viewport
func (s *ServiceBrowser) renderContent() string {
	var b strings.Builder

	if len(s.flatItems) == 0 {
		b.WriteString(s.styles.aliases.Render("\n  No services found"))
		return b.String()
	}

	// Render by category
	globalIdx := 0
	for catIdx, cat := range s.categories {
		// Collect items for this category
		var catItems []flatItem
		for _, fi := range s.flatItems {
			if fi.categoryIdx == catIdx {
				catItems = append(catItems, fi)
			}
		}

		if len(catItems) == 0 {
			continue
		}

		// Category header
		b.WriteString(s.styles.category.Render("── " + cat.name + " "))
		b.WriteString("\n")

		// Render services in grid
		rows := (len(catItems) + s.cols - 1) / s.cols
		for row := 0; row < rows; row++ {
			var cells []string
			for col := 0; col < s.cols; col++ {
				idx := row*s.cols + col
				if idx < len(catItems) {
					selected := globalIdx+idx == s.cursor
					cells = append(cells, s.renderCell(catItems[idx].service, selected))
				}
			}
			b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, cells...))
			b.WriteString("\n")
		}

		globalIdx += len(catItems)
	}

	return b.String()
}

func (s *ServiceBrowser) renderCell(item serviceItem, selected bool) string {
	var nameStyle, aliasStyle, cellStyle lipgloss.Style
	if selected {
		nameStyle = s.styles.serviceNameSe
		aliasStyle = s.styles.aliasesSel
		cellStyle = s.styles.cellSelected
	} else {
		nameStyle = s.styles.serviceName
		aliasStyle = s.styles.aliases
		cellStyle = s.styles.cell
	}

	// Service name (truncate if too long)
	name := item.displayName
	maxNameLen := cellWidth - 2
	if len(name) > maxNameLen {
		name = name[:maxNameLen-1] + "…"
	}

	// Aliases line
	var aliasLine string
	if len(item.aliases) > 0 {
		aliasLine = strings.Join(item.aliases, ", ")
		if len(aliasLine) > maxNameLen {
			aliasLine = aliasLine[:maxNameLen-1] + "…"
		}
	}

	content := nameStyle.Render(name) + "\n" + aliasStyle.Render(aliasLine)
	return cellStyle.Render(content)
}

// SetSize implements View
func (s *ServiceBrowser) SetSize(width, height int) tea.Cmd {
	s.width = width
	s.height = height

	// Set header panel width
	s.headerPanel.SetWidth(width)

	// Calculate columns based on width
	s.cols = (width - cellPaddingX) / cellWidth
	if s.cols < minColumns {
		s.cols = minColumns
	}
	if s.cols > maxColumns {
		s.cols = maxColumns
	}

	// Calculate header height dynamically
	headerStr := s.headerPanel.RenderHome()
	headerHeight := s.headerPanel.Height(headerStr)

	// height - header + extra space
	vpHeight := height - headerHeight + 1
	if vpHeight < 5 {
		vpHeight = 5
	}

	if !s.ready {
		s.viewport = viewport.New(width, vpHeight)
		s.ready = true
	} else {
		s.viewport.Width = width
		s.viewport.Height = vpHeight
	}

	// Update viewport content
	s.viewport.SetContent(s.renderContent())

	return nil
}

// StatusLine implements View
func (s *ServiceBrowser) StatusLine() string {
	if s.filterActive {
		return "Type to filter • Enter to confirm • Esc to cancel"
	}
	if s.filterText != "" {
		return "/ to filter • c to clear • Enter to select • ? for help"
	}
	return "/ to filter • Enter to select • ? for help"
}

// HasActiveInput implements InputCapture
func (s *ServiceBrowser) HasActiveInput() bool {
	return s.filterActive
}

// CanRefresh implements Refreshable interface
func (s *ServiceBrowser) CanRefresh() bool {
	return true
}
