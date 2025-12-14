package view

import (
	"context"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/ui"
)

// RegionSelector allows switching AWS regions
type RegionSelector struct {
	ctx     context.Context
	list    list.Model
	regions []string
	width   int
	height  int
}

type regionItem string

func (r regionItem) Title() string       { return string(r) }
func (r regionItem) Description() string { return "" }
func (r regionItem) FilterValue() string { return string(r) }

// RegionChangedMsg is sent when region is changed
type RegionChangedMsg struct {
	Region string
}

// NewRegionSelector creates a new region selector
func NewRegionSelector(ctx context.Context) *RegionSelector {
	t := ui.Current()

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(t.Primary).
		BorderLeftForeground(t.Primary)

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Select Region"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().
		Background(t.TableHeader).
		Foreground(t.TableHeaderText).
		Padding(0, 1)

	return &RegionSelector{
		ctx:  ctx,
		list: l,
	}
}

// Init implements tea.Model
func (r *RegionSelector) Init() tea.Cmd {
	return r.loadRegions
}

func (r *RegionSelector) loadRegions() tea.Msg {
	regions, _ := config.FetchAvailableRegions(r.ctx)
	return regionsLoadedMsg{regions: regions}
}

type regionsLoadedMsg struct {
	regions []string
}

// Update implements tea.Model
func (r *RegionSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case regionsLoadedMsg:
		r.regions = msg.regions
		items := make([]list.Item, len(r.regions))
		currentRegion := config.Global().Region()
		selectedIdx := 0
		for i, region := range r.regions {
			items[i] = regionItem(region)
			if region == currentRegion {
				selectedIdx = i
			}
		}
		r.list.SetItems(items)
		r.list.Select(selectedIdx)
		return r, nil

	case tea.KeyMsg:
		if !r.list.SettingFilter() {
			switch msg.String() {
			case "enter", "l":
				if item, ok := r.list.SelectedItem().(regionItem); ok {
					region := string(item)
					config.Global().SetRegion(region)
					return r, func() tea.Msg {
						return RegionChangedMsg{Region: region}
					}
				}
			}
		}
	}

	var cmd tea.Cmd
	r.list, cmd = r.list.Update(msg)
	return r, cmd
}

// View implements tea.Model
func (r *RegionSelector) View() string {
	current := config.Global().Region()
	header := ui.DimStyle().Render("Current: " + current)
	return header + "\n\n" + r.list.View()
}

// SetSize implements View
func (r *RegionSelector) SetSize(width, height int) tea.Cmd {
	r.width = width
	r.height = height
	r.list.SetSize(width, height-3)
	return nil
}

// StatusLine implements View
func (r *RegionSelector) StatusLine() string {
	return "Select region • / to filter • Enter to select • Esc to cancel"
}

// HasActiveInput implements InputCapture
func (r *RegionSelector) HasActiveInput() bool {
	return r.list.SettingFilter()
}
