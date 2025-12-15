package view

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/ui"
)

// ProfileSelector allows switching AWS profiles
type ProfileSelector struct {
	list     list.Model
	profiles []string
	width    int
	height   int
}

type profileItem string

func (p profileItem) Title() string       { return string(p) }
func (p profileItem) Description() string { return "" }
func (p profileItem) FilterValue() string { return string(p) }

// ProfileChangedMsg is sent when profile is changed
type ProfileChangedMsg struct {
	Profile string
}

// NewProfileSelector creates a new profile selector
func NewProfileSelector() *ProfileSelector {
	t := ui.Current()

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(t.Primary).
		BorderLeftForeground(t.Primary)

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Select Profile"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().
		Background(t.TableHeader).
		Foreground(t.TableHeaderText).
		Padding(0, 1)

	return &ProfileSelector{
		list: l,
	}
}

// Init implements tea.Model
func (p *ProfileSelector) Init() tea.Cmd {
	return p.loadProfiles
}

func (p *ProfileSelector) loadProfiles() tea.Msg {
	profiles := config.FetchAvailableProfiles()
	return profilesLoadedMsg{profiles: profiles}
}

type profilesLoadedMsg struct {
	profiles []string
}

// Update implements tea.Model
func (p *ProfileSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case profilesLoadedMsg:
		p.profiles = msg.profiles
		items := make([]list.Item, len(p.profiles))
		currentProfile := config.Global().Profile()
		if currentProfile == "" {
			currentProfile = "default"
		}
		selectedIdx := 0
		for i, profile := range p.profiles {
			items[i] = profileItem(profile)
			if profile == currentProfile {
				selectedIdx = i
			}
		}
		p.list.SetItems(items)
		p.list.Select(selectedIdx)
		return p, nil

	case tea.KeyMsg:
		if !p.list.SettingFilter() {
			switch msg.String() {
			case "enter", "l":
				if item, ok := p.list.SelectedItem().(profileItem); ok {
					profile := string(item)
					config.Global().SetProfile(profile)
					return p, func() tea.Msg {
						return ProfileChangedMsg{Profile: profile}
					}
				}
			}
		}
	}

	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)
	return p, cmd
}

// View implements tea.Model
func (p *ProfileSelector) View() string {
	current := config.Global().Profile()
	if current == "" {
		current = "default"
	}
	header := ui.DimStyle().Render("Current: " + current)
	return header + "\n\n" + p.list.View()
}

// SetSize implements View
func (p *ProfileSelector) SetSize(width, height int) tea.Cmd {
	p.width = width
	p.height = height
	p.list.SetSize(width, height-3)
	return nil
}

// StatusLine implements View
func (p *ProfileSelector) StatusLine() string {
	return "Select profile • / to filter • Enter to select • Esc to cancel"
}

// HasActiveInput implements InputCapture
func (p *ProfileSelector) HasActiveInput() bool {
	return p.list.SettingFilter()
}
