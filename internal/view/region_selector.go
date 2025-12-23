package view

import (
	"context"
	"sort"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/config"
	navmsg "github.com/clawscli/claws/internal/msg"
	"github.com/clawscli/claws/internal/ui"
)

// regionOrder defines geographic ordering for region prefixes
var regionOrder = map[string]int{
	"us":      0, // US
	"ca":      1, // Canada
	"sa":      2, // South America
	"eu":      3, // Europe
	"me":      4, // Middle East
	"af":      5, // Africa
	"ap":      6, // Asia Pacific
	"il":      7, // Israel
	"cn":      8, // China
	"default": 9,
}

// regionSelectorStyles holds cached styles
type regionSelectorStyles struct {
	title        lipgloss.Style
	item         lipgloss.Style
	itemSelected lipgloss.Style
	itemCurrent  lipgloss.Style
	filter       lipgloss.Style
}

func newRegionSelectorStyles() regionSelectorStyles {
	t := ui.Current()
	return regionSelectorStyles{
		title:        lipgloss.NewStyle().Background(t.TableHeader).Foreground(t.TableHeaderText).Padding(0, 1),
		item:         lipgloss.NewStyle().PaddingLeft(2),
		itemSelected: lipgloss.NewStyle().PaddingLeft(2).Background(t.Selection).Foreground(t.SelectionText),
		itemCurrent:  lipgloss.NewStyle().PaddingLeft(2).Foreground(t.Success),
		filter:       lipgloss.NewStyle().Foreground(t.Accent),
	}
}

// RegionSelector allows switching AWS regions
type RegionSelector struct {
	ctx     context.Context
	regions []string
	cursor  int
	width   int
	height  int

	// Current region (for highlighting)
	currentRegion string

	// Viewport for scrolling
	viewport viewport.Model
	ready    bool

	// Filter
	filterInput  textinput.Model
	filterActive bool
	filterText   string
	filtered     []string

	styles regionSelectorStyles
}

// NewRegionSelector creates a new region selector
func NewRegionSelector(ctx context.Context) *RegionSelector {
	ti := textinput.New()
	ti.Placeholder = "filter..."
	ti.Prompt = "/"
	ti.CharLimit = 50

	return &RegionSelector{
		ctx:           ctx,
		currentRegion: config.Global().Region(),
		filterInput:   ti,
		styles:        newRegionSelectorStyles(),
	}
}

// Init implements tea.Model
func (r *RegionSelector) Init() tea.Cmd {
	return r.loadRegions
}

func (r *RegionSelector) loadRegions() tea.Msg {
	regions, _ := aws.FetchAvailableRegions(r.ctx)
	return regionsLoadedMsg{regions: regions}
}

type regionsLoadedMsg struct {
	regions []string
}

// sortRegions sorts regions by geographic area then alphabetically
func sortRegions(regions []string) {
	sort.Slice(regions, func(i, j int) bool {
		// Get prefix (e.g., "us" from "us-east-1")
		pi := strings.Split(regions[i], "-")[0]
		pj := strings.Split(regions[j], "-")[0]

		oi, ok := regionOrder[pi]
		if !ok {
			oi = regionOrder["default"]
		}
		oj, ok := regionOrder[pj]
		if !ok {
			oj = regionOrder["default"]
		}

		if oi != oj {
			return oi < oj
		}
		return regions[i] < regions[j]
	})
}

// Update implements tea.Model
func (r *RegionSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case regionsLoadedMsg:
		r.regions = msg.regions
		sortRegions(r.regions)
		r.applyFilter()
		// Set cursor to current region
		for i, region := range r.filtered {
			if region == r.currentRegion {
				r.cursor = i
				break
			}
		}
		r.updateViewport()
		return r, nil

	case tea.MouseWheelMsg:
		var cmd tea.Cmd
		r.viewport, cmd = r.viewport.Update(msg)
		return r, cmd

	case tea.MouseMotionMsg:
		// Hover to select
		if idx := r.getItemAtPosition(msg.Y); idx >= 0 && idx != r.cursor {
			r.cursor = idx
			r.updateViewport()
		}
		return r, nil

	case tea.MouseClickMsg:
		// Click to select and apply
		if msg.Button == tea.MouseLeft {
			if idx := r.getItemAtPosition(msg.Y); idx >= 0 {
				r.cursor = idx
				return r.selectRegion()
			}
		}
		return r, nil

	case tea.KeyPressMsg:
		// Handle filter input mode
		if r.filterActive {
			switch msg.String() {
			case "esc":
				r.filterActive = false
				r.filterInput.Blur()
				return r, nil
			case "enter":
				r.filterActive = false
				r.filterInput.Blur()
				r.filterText = r.filterInput.Value()
				r.applyFilter()
				r.cursor = 0
				r.updateViewport()
				return r, nil
			default:
				var cmd tea.Cmd
				r.filterInput, cmd = r.filterInput.Update(msg)
				r.filterText = r.filterInput.Value()
				r.applyFilter()
				r.cursor = 0
				r.updateViewport()
				return r, cmd
			}
		}

		switch msg.String() {
		case "/":
			r.filterActive = true
			r.filterInput.Focus()
			return r, textinput.Blink
		case "c":
			r.filterText = ""
			r.filterInput.SetValue("")
			r.applyFilter()
			r.cursor = 0
			r.updateViewport()
			return r, nil
		case "up", "k":
			if r.cursor > 0 {
				r.cursor--
				r.updateViewport()
			}
			return r, nil
		case "down", "j":
			if r.cursor < len(r.filtered)-1 {
				r.cursor++
				r.updateViewport()
			}
			return r, nil
		case "enter", "l":
			return r.selectRegion()
		}
	}

	var cmd tea.Cmd
	r.viewport, cmd = r.viewport.Update(msg)
	return r, cmd
}

func (r *RegionSelector) selectRegion() (tea.Model, tea.Cmd) {
	if r.cursor >= 0 && r.cursor < len(r.filtered) {
		region := r.filtered[r.cursor]
		config.Global().SetRegion(region)
		return r, func() tea.Msg {
			return navmsg.RegionChangedMsg{Region: region}
		}
	}
	return r, nil
}

func (r *RegionSelector) applyFilter() {
	if r.filterText == "" {
		r.filtered = r.regions
		return
	}

	filter := strings.ToLower(r.filterText)
	r.filtered = nil
	for _, region := range r.regions {
		if strings.Contains(strings.ToLower(region), filter) {
			r.filtered = append(r.filtered, region)
		}
	}
}

func (r *RegionSelector) updateViewport() {
	if !r.ready {
		return
	}
	r.viewport.SetContent(r.renderContent())

	// Scroll to keep cursor visible
	if r.cursor >= 0 {
		viewportHeight := r.viewport.Height()
		if viewportHeight > 0 {
			if r.cursor < r.viewport.YOffset() {
				r.viewport.SetYOffset(r.cursor)
			} else if r.cursor >= r.viewport.YOffset()+viewportHeight {
				r.viewport.SetYOffset(r.cursor - viewportHeight + 1)
			}
		}
	}
}

func (r *RegionSelector) renderContent() string {
	var b strings.Builder

	for i, region := range r.filtered {
		style := r.styles.item
		if i == r.cursor {
			style = r.styles.itemSelected
		} else if region == r.currentRegion {
			style = r.styles.itemCurrent
		}

		prefix := "  "
		if region == r.currentRegion {
			prefix = "• "
		}

		b.WriteString(style.Render(prefix + region))
		b.WriteString("\n")
	}

	return b.String()
}

func (r *RegionSelector) getItemAtPosition(y int) int {
	// Layout: title (1) + filter? (1) + viewport content
	headerHeight := 1
	if r.filterActive || r.filterText != "" {
		headerHeight++
	}

	contentY := y - headerHeight + r.viewport.YOffset()
	if contentY >= 0 && contentY < len(r.filtered) {
		return contentY
	}
	return -1
}

// ViewString returns the view content as a string
func (r *RegionSelector) ViewString() string {
	s := r.styles

	// Title
	title := s.title.Render("Select Region")

	// Filter
	var filterView string
	if r.filterActive {
		filterView = r.styles.filter.Render(r.filterInput.View()) + "\n"
	} else if r.filterText != "" {
		filterView = r.styles.filter.Render("filter: "+r.filterText) + "\n"
	}

	if !r.ready {
		return title + "\n" + filterView + "Loading..."
	}

	return title + "\n" + filterView + r.viewport.View()
}

// View implements tea.Model
func (r *RegionSelector) View() tea.View {
	return tea.NewView(r.ViewString())
}

// SetSize implements View
func (r *RegionSelector) SetSize(width, height int) tea.Cmd {
	r.width = width
	r.height = height

	viewportHeight := height - 2 // title + some padding
	if r.filterActive || r.filterText != "" {
		viewportHeight--
	}

	if !r.ready {
		r.viewport = viewport.New(viewport.WithWidth(width), viewport.WithHeight(viewportHeight))
		r.ready = true
	} else {
		r.viewport.SetWidth(width)
		r.viewport.SetHeight(viewportHeight)
	}
	r.updateViewport()
	return nil
}

// StatusLine implements View
func (r *RegionSelector) StatusLine() string {
	if r.filterActive {
		return "Type to filter • Enter confirm • Esc cancel"
	}
	return "Select region • / filter • Enter select • Esc cancel"
}

// HasActiveInput implements InputCapture
func (r *RegionSelector) HasActiveInput() bool {
	return r.filterActive
}
