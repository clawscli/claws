package view

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

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
// itemPosition stores the rendered position of an item for mouse hit testing
type itemPosition struct {
	startLine, endLine int
	startCol, endCol   int
	itemIdx            int
}

type ServiceBrowser struct {
	ctx      context.Context
	registry *registry.Registry

	// Category-based data
	categories []categoryGroup
	flatItems  []flatItem // Flattened list for navigation

	cursor int // Current selection index in flatItems
	cols   int // Number of columns in grid

	// Mouse hit testing - populated during render
	itemPositions []itemPosition

	// Header panel
	headerPanel *HeaderPanel

	// Viewport for scrolling
	vp ViewportState

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
	return serviceBrowserStyles{
		category: ui.DimStyle().
			Bold(true).
			MarginTop(1).
			MarginBottom(0),
		cell: lipgloss.NewStyle().
			Width(cellWidth).
			Height(cellHeight).
			Padding(0, 1),
		cellSelected: ui.SelectedStyle().
			Width(cellWidth).
			Height(cellHeight).
			Padding(0, 1),
		serviceName:   ui.TextStyle().Bold(true),
		serviceNameSe: ui.TitleStyle(),
		aliases:       ui.DimStyle(),
		aliasesSel:    ui.DimStyle(),
		filterPrompt:  ui.PrimaryStyle(),
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
	ti.Placeholder = FilterPlaceholder
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

	case tea.KeyPressMsg:
		if s.filterActive {
			return s.handleFilterInput(msg)
		}
		return s.handleNavigation(msg)

	case tea.MouseWheelMsg:
		var cmd tea.Cmd
		s.vp.Model, cmd = s.vp.Model.Update(msg)
		return s, cmd

	case tea.MouseMotionMsg:
		// Hover: update cursor to item under mouse
		if idx := s.getItemAtPosition(msg.X, msg.Y); idx >= 0 && idx != s.cursor {
			s.cursor = idx
			s.updateViewport()
		}
		return s, nil

	case tea.MouseClickMsg:
		// Click: select item at position and navigate
		if msg.Button == tea.MouseLeft {
			if idx := s.getItemAtPosition(msg.X, msg.Y); idx >= 0 {
				s.cursor = idx
				return s.selectCurrentService()
			}
		}
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

func (s *ServiceBrowser) handleFilterInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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

func (s *ServiceBrowser) handleNavigation(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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

	case "~":
		dashboard := NewDashboardView(s.ctx, s.registry)
		return s, func() tea.Msg {
			return NavigateMsg{View: dashboard, ClearStack: true}
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

func (s *ServiceBrowser) updateViewport() {
	if !s.vp.Ready {
		return
	}
	content := s.renderContent()
	s.vp.Model.SetContent(content)

	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	if len(s.flatItems) == 0 {
		return
	}

	cursorRatio := float64(s.cursor) / float64(len(s.flatItems))
	targetLine := int(cursorRatio * float64(totalLines))

	vpHeight := s.vp.Model.Height()
	currentTop := s.vp.Model.YOffset()

	if targetLine < currentTop {
		s.vp.Model.SetYOffset(max(0, targetLine-2))
	} else if targetLine > currentTop+vpHeight-cellHeight {
		newOffset := targetLine - vpHeight + cellHeight + 2
		newOffset = max(0, min(newOffset, totalLines-vpHeight))
		s.vp.Model.SetYOffset(newOffset)
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

func (s *ServiceBrowser) getItemAtPosition(x, y int) int {
	if !s.vp.Ready || len(s.itemPositions) == 0 {
		return -1
	}

	headerStr := s.headerPanel.RenderHome()
	headerHeight := s.headerPanel.Height(headerStr)

	contentY := y - headerHeight + s.vp.Model.YOffset()
	if contentY < 0 {
		return -1
	}

	// Search through recorded positions
	for _, pos := range s.itemPositions {
		if contentY >= pos.startLine && contentY < pos.endLine &&
			x >= pos.startCol && x < pos.endCol {
			// Safety check: ensure itemIdx is within bounds
			if pos.itemIdx >= 0 && pos.itemIdx < len(s.flatItems) {
				return pos.itemIdx
			}
			return -1
		}
	}

	return -1
}

func (s *ServiceBrowser) ViewString() string {
	header := s.headerPanel.RenderHome()

	if !s.vp.Ready {
		return header + "\n" + LoadingMessage
	}

	var footer string
	if s.filterActive {
		footer = "\n" + s.styles.filterPrompt.Render(s.filterInput.View())
	}

	return header + "\n" + s.vp.Model.View() + footer
}

// View implements tea.Model
func (s *ServiceBrowser) View() tea.View {
	return tea.NewView(s.ViewString())
}

// renderContent renders the service grid content for the viewport
func (s *ServiceBrowser) renderContent() string {
	var b strings.Builder

	// Reset item positions for mouse hit testing
	s.itemPositions = s.itemPositions[:0]

	if len(s.flatItems) == 0 {
		b.WriteString(s.styles.aliases.Render("\n  No services found"))
		return b.String()
	}

	// Track current line for position recording
	currentLine := 0

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
		catHeader := s.styles.category.Render("── " + cat.name + " ")
		catHeaderHeight := strings.Count(catHeader, "\n") + 1 // +1 for the \n we add
		b.WriteString(catHeader)
		b.WriteString("\n")
		currentLine += catHeaderHeight

		// Render services in grid
		rows := (len(catItems) + s.cols - 1) / s.cols
		for row := range rows {
			var cells []string
			for col := range s.cols {
				idx := row*s.cols + col
				if idx < len(catItems) {
					selected := globalIdx+idx == s.cursor
					cells = append(cells, s.renderCell(catItems[idx].service, selected))
				}
			}
			rowContent := lipgloss.JoinHorizontal(lipgloss.Top, cells...)
			rowHeight := strings.Count(rowContent, "\n") + 1 // +1 for the line after

			// Record positions for items in this row
			for col := range s.cols {
				idx := row*s.cols + col
				if idx < len(catItems) {
					s.itemPositions = append(s.itemPositions, itemPosition{
						startLine: currentLine,
						endLine:   currentLine + rowHeight,
						startCol:  col * cellWidth,
						endCol:    (col + 1) * cellWidth,
						itemIdx:   globalIdx + idx,
					})
				}
			}

			b.WriteString(rowContent)
			b.WriteString("\n")
			currentLine += rowHeight
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
	// Set header panel width
	s.headerPanel.SetWidth(width)

	// Calculate columns based on width
	s.cols = max(minColumns, min((width-cellPaddingX)/cellWidth, maxColumns))

	// Calculate header height dynamically
	headerStr := s.headerPanel.RenderHome()
	headerHeight := s.headerPanel.Height(headerStr)

	vpHeight := max(height-headerHeight+1, 5)

	s.vp.SetSize(width, vpHeight)
	s.vp.Model.SetContent(s.renderContent())

	return nil
}

// StatusLine implements View
func (s *ServiceBrowser) StatusLine() string {
	if s.filterActive {
		return "type to filter • enter:confirm • esc:cancel"
	}
	if s.filterText != "" {
		return "~:home • c:clear • enter:select • ?:help"
	}
	return "~:home • /:filter • enter:select • ?:help"
}

// HasActiveInput implements InputCapture
func (s *ServiceBrowser) HasActiveInput() bool {
	return s.filterActive
}

// CanRefresh implements Refreshable interface
func (s *ServiceBrowser) CanRefresh() bool {
	return true
}
