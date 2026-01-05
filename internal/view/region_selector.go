package view

import (
	"context"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/log"
	navmsg "github.com/clawscli/claws/internal/msg"
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

type regionItem string

func (r regionItem) GetID() string    { return string(r) }
func (r regionItem) GetLabel() string { return string(r) }

type RegionSelector struct {
	ctx      context.Context
	selector *MultiSelector[regionItem]
	regions  []regionItem
}

func NewRegionSelector(ctx context.Context) *RegionSelector {
	return &RegionSelector{
		ctx:      ctx,
		selector: NewMultiSelector[regionItem]("Select Regions", config.Global().Regions()),
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
		sortRegions(msg.regions)
		r.regions = make([]regionItem, len(msg.regions))
		for i, region := range msg.regions {
			r.regions[i] = regionItem(region)
		}
		r.selector.SetItems(r.regions)
		return r, nil
	case ThemeChangedMsg:
		r.selector.ReloadStyles()
		return r, nil
	}

	cmd, result := r.selector.HandleUpdate(msg)
	if result == KeyApply {
		return r.applySelection()
	}
	return r, cmd
}

func (r *RegionSelector) applySelection() (tea.Model, tea.Cmd) {
	selected := r.selector.SelectedItems()
	if len(selected) == 0 {
		return r, nil
	}

	regions := make([]string, len(selected))
	for i, item := range selected {
		regions[i] = string(item)
	}

	config.Global().SetRegions(regions)
	return r, func() tea.Msg {
		return navmsg.RegionChangedMsg{Regions: regions}
	}
}

func (r *RegionSelector) ViewString() string {
	return r.selector.ViewString()
}

func (r *RegionSelector) View() tea.View {
	return tea.NewView(r.ViewString())
}

func (r *RegionSelector) SetSize(width, height int) tea.Cmd {
	r.selector.SetSize(width, height)
	return nil
}

func (r *RegionSelector) StatusLine() string {
	count := r.selector.SelectedCount()
	if r.selector.FilterActive() {
		return "Type to filter • Enter confirm • Esc cancel"
	}
	return "Space:toggle • a:all • n:none • Enter:apply • " + strings.Repeat("●", count) + " selected"
}

func (r *RegionSelector) HasActiveInput() bool {
	return r.selector.FilterActive()
}
