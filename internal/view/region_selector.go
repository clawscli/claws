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
	"github.com/clawscli/claws/internal/log"
	navmsg "github.com/clawscli/claws/internal/msg"
	"github.com/clawscli/claws/internal/ui"
)

var regionOrder = map[string]int{
	"us":      0,
	"ca":      1,
	"sa":      2,
	"eu":      3,
	"me":      4,
	"af":      5,
	"ap":      6,
	"il":      7,
	"cn":      8,
	"default": 9,
}

type regionSelectorStyles struct {
	title        lipgloss.Style
	item         lipgloss.Style
	itemSelected lipgloss.Style
	itemChecked  lipgloss.Style
	filter       lipgloss.Style
}

func newRegionSelectorStyles() regionSelectorStyles {
	t := ui.Current()
	return regionSelectorStyles{
		title:        lipgloss.NewStyle().Background(t.TableHeader).Foreground(t.TableHeaderText).Padding(0, 1),
		item:         lipgloss.NewStyle().PaddingLeft(2),
		itemSelected: lipgloss.NewStyle().PaddingLeft(2).Background(t.Selection).Foreground(t.SelectionText),
		itemChecked:  lipgloss.NewStyle().PaddingLeft(2).Foreground(t.Success),
		filter:       lipgloss.NewStyle().Foreground(t.Accent),
	}
}

type RegionSelector struct {
	ctx     context.Context
	regions []string
	cursor  int
	width   int
	height  int

	selected map[string]bool

	viewport viewport.Model
	ready    bool

	filterInput  textinput.Model
	filterActive bool
	filterText   string
	filtered     []string

	styles regionSelectorStyles
}

func NewRegionSelector(ctx context.Context) *RegionSelector {
	ti := textinput.New()
	ti.Placeholder = "filter..."
	ti.Prompt = "/"
	ti.CharLimit = 50

	selected := make(map[string]bool)
	for _, r := range config.Global().Regions() {
		selected[r] = true
	}

	return &RegionSelector{
		ctx:         ctx,
		selected:    selected,
		filterInput: ti,
		styles:      newRegionSelectorStyles(),
	}
}

func (r *RegionSelector) Init() tea.Cmd {
	return r.loadRegions
}

func (r *RegionSelector) loadRegions() tea.Msg {
	regions, err := aws.FetchAvailableRegions(r.ctx)
	if err != nil {
		log.Error("failed to fetch regions", "error", err)
	}
	return regionsLoadedMsg{regions: regions}
}

type regionsLoadedMsg struct {
	regions []string
}

func sortRegions(regions []string) {
	sort.Slice(regions, func(i, j int) bool {
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

func (r *RegionSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case regionsLoadedMsg:
		r.regions = msg.regions
		sortRegions(r.regions)
		r.applyFilter()
		r.clampCursor()
		for i, region := range r.filtered {
			if r.selected[region] {
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
		if idx := r.getItemAtPosition(msg.Y); idx >= 0 && idx != r.cursor {
			r.cursor = idx
			r.updateViewport()
		}
		return r, nil

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			if idx := r.getItemAtPosition(msg.Y); idx >= 0 {
				r.cursor = idx
				r.toggleCurrent()
				r.updateViewport()
			}
		}
		return r, nil

	case tea.KeyPressMsg:
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
				r.clampCursor()
				r.updateViewport()
				return r, nil
			default:
				var cmd tea.Cmd
				r.filterInput, cmd = r.filterInput.Update(msg)
				r.filterText = r.filterInput.Value()
				r.applyFilter()
				r.clampCursor()
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
			r.clampCursor()
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
		case "space":
			r.toggleCurrent()
			r.updateViewport()
			return r, nil
		case "a":
			for _, region := range r.filtered {
				r.selected[region] = true
			}
			r.updateViewport()
			return r, nil
		case "n":
			for _, region := range r.filtered {
				delete(r.selected, region)
			}
			r.updateViewport()
			return r, nil
		case "enter", "l":
			return r.applySelection()
		}
	}

	var cmd tea.Cmd
	r.viewport, cmd = r.viewport.Update(msg)
	return r, cmd
}

func (r *RegionSelector) toggleCurrent() {
	if r.cursor >= 0 && r.cursor < len(r.filtered) {
		region := r.filtered[r.cursor]
		if r.selected[region] {
			delete(r.selected, region)
		} else {
			r.selected[region] = true
		}
	}
}

func (r *RegionSelector) applySelection() (tea.Model, tea.Cmd) {
	var regions []string
	for _, region := range r.regions {
		if r.selected[region] {
			regions = append(regions, region)
		}
	}
	if len(regions) == 0 {
		return r, nil
	}
	config.Global().SetRegions(regions)
	return r, func() tea.Msg {
		return navmsg.RegionChangedMsg{Regions: regions}
	}
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

func (r *RegionSelector) clampCursor() {
	if len(r.filtered) == 0 {
		r.cursor = -1
	} else if r.cursor >= len(r.filtered) {
		r.cursor = len(r.filtered) - 1
	} else if r.cursor < 0 {
		r.cursor = 0
	}
}

func (r *RegionSelector) updateViewport() {
	if !r.ready {
		return
	}
	r.viewport.SetContent(r.renderContent())

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
		isChecked := r.selected[region]

		if i == r.cursor {
			style = r.styles.itemSelected
		} else if isChecked {
			style = r.styles.itemChecked
		}

		checkbox := "☐ "
		if isChecked {
			checkbox = "☑ "
		}

		b.WriteString(style.Render(checkbox + region))
		b.WriteString("\n")
	}

	return b.String()
}

func (r *RegionSelector) getItemAtPosition(y int) int {
	if !r.ready {
		return -1
	}
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

func (r *RegionSelector) ViewString() string {
	s := r.styles

	title := s.title.Render("Select Regions")

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

func (r *RegionSelector) View() tea.View {
	return tea.NewView(r.ViewString())
}

func (r *RegionSelector) SetSize(width, height int) tea.Cmd {
	r.width = width
	r.height = height

	viewportHeight := height - 2
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

func (r *RegionSelector) StatusLine() string {
	count := len(r.selected)
	if r.filterActive {
		return "Type to filter • Enter confirm • Esc cancel"
	}
	return "Space:toggle • a:all • n:none • Enter:apply • " + strings.Repeat("●", count) + " selected"
}

func (r *RegionSelector) HasActiveInput() bool {
	return r.filterActive
}
