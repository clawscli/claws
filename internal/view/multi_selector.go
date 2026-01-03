package view

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/clawscli/claws/internal/ui"
)

type SelectorItem interface {
	GetID() string
	GetLabel() string
}

type selectorStyles struct {
	title        lipgloss.Style
	item         lipgloss.Style
	itemSelected lipgloss.Style
	itemChecked  lipgloss.Style
	filter       lipgloss.Style
}

func newSelectorStyles() selectorStyles {
	return selectorStyles{
		title:        ui.TableHeaderStyle().Padding(0, 1),
		item:         lipgloss.NewStyle().PaddingLeft(2),
		itemSelected: ui.SelectedStyle().PaddingLeft(2),
		itemChecked:  ui.SuccessStyle().PaddingLeft(2),
		filter:       ui.AccentStyle(),
	}
}

type MultiSelector[T SelectorItem] struct {
	title    string
	items    []T
	cursor   int
	selected map[string]bool

	vp ViewportState

	filterInput  textinput.Model
	filterActive bool
	filterText   string
	filtered     []T

	styles      selectorStyles
	renderExtra func(item T) string
	extraHeight int
}

func NewMultiSelector[T SelectorItem](title string, initialSelected []string) *MultiSelector[T] {
	ti := textinput.New()
	ti.Placeholder = FilterPlaceholder
	ti.Prompt = "/"
	ti.CharLimit = 50

	selected := make(map[string]bool)
	for _, id := range initialSelected {
		selected[id] = true
	}

	return &MultiSelector[T]{
		title:       title,
		selected:    selected,
		filterInput: ti,
		styles:      newSelectorStyles(),
	}
}

func (m *MultiSelector[T]) SetItems(items []T) {
	m.items = items
	m.applyFilter()
	m.clampCursor()
	for i, item := range m.filtered {
		if m.selected[item.GetID()] {
			m.cursor = i
			break
		}
	}
	m.updateViewport()
}

func (m *MultiSelector[T]) SetRenderExtra(fn func(T) string) {
	m.renderExtra = fn
}

func (m *MultiSelector[T]) SetExtraHeight(h int) {
	m.extraHeight = h
}

func (m *MultiSelector[T]) Selected() map[string]bool {
	return m.selected
}

func (m *MultiSelector[T]) SelectedItems() []T {
	var result []T
	for _, item := range m.items {
		if m.selected[item.GetID()] {
			result = append(result, item)
		}
	}
	return result
}

func (m *MultiSelector[T]) CurrentItem() (T, bool) {
	var zero T
	if m.cursor < 0 || m.cursor >= len(m.filtered) {
		return zero, false
	}
	return m.filtered[m.cursor], true
}

func (m *MultiSelector[T]) Cursor() int {
	return m.cursor
}

func (m *MultiSelector[T]) FilteredLen() int {
	return len(m.filtered)
}

type SelectorKeyResult int

const (
	KeyNotHandled SelectorKeyResult = iota
	KeyHandled
	KeyApply
)

func (m *MultiSelector[T]) HandleUpdate(msg tea.Msg) (tea.Cmd, SelectorKeyResult) {
	switch msg := msg.(type) {
	case tea.MouseWheelMsg:
		var cmd tea.Cmd
		m.vp.Model, cmd = m.vp.Model.Update(msg)
		return cmd, KeyHandled

	case tea.MouseMotionMsg:
		if idx := m.getItemAtPosition(msg.Y); idx >= 0 && idx != m.cursor {
			m.cursor = idx
			m.updateViewport()
		}
		return nil, KeyHandled

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			if idx := m.getItemAtPosition(msg.Y); idx >= 0 {
				m.cursor = idx
				m.toggleCurrent()
				m.updateViewport()
			}
		}
		return nil, KeyHandled

	case tea.KeyPressMsg:
		if m.filterActive {
			switch msg.String() {
			case "esc":
				m.filterActive = false
				m.filterInput.Blur()
				return nil, KeyHandled
			case "enter":
				m.filterActive = false
				m.filterInput.Blur()
				m.filterText = m.filterInput.Value()
				m.applyFilter()
				m.clampCursor()
				m.updateViewport()
				return nil, KeyHandled
			default:
				var cmd tea.Cmd
				m.filterInput, cmd = m.filterInput.Update(msg)
				m.filterText = m.filterInput.Value()
				m.applyFilter()
				m.clampCursor()
				m.updateViewport()
				return cmd, KeyHandled
			}
		}

		switch msg.String() {
		case "/":
			m.filterActive = true
			m.filterInput.Focus()
			return textinput.Blink, KeyHandled
		case "c":
			m.filterText = ""
			m.filterInput.SetValue("")
			m.applyFilter()
			m.clampCursor()
			m.updateViewport()
			return nil, KeyHandled
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.updateViewport()
			}
			return nil, KeyHandled
		case "down", "j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
				m.updateViewport()
			}
			return nil, KeyHandled
		case "space":
			m.toggleCurrent()
			m.updateViewport()
			return nil, KeyHandled
		case "a":
			for _, item := range m.filtered {
				m.selected[item.GetID()] = true
			}
			m.updateViewport()
			return nil, KeyHandled
		case "n":
			for _, item := range m.filtered {
				delete(m.selected, item.GetID())
			}
			m.updateViewport()
			return nil, KeyHandled
		case "enter":
			return nil, KeyApply
		}
	}

	var cmd tea.Cmd
	m.vp.Model, cmd = m.vp.Model.Update(msg)
	if cmd != nil {
		return cmd, KeyHandled
	}
	return nil, KeyNotHandled
}

func (m *MultiSelector[T]) ClearResult() {
	m.updateViewport()
}

func (m *MultiSelector[T]) toggleCurrent() {
	if m.cursor >= 0 && m.cursor < len(m.filtered) {
		item := m.filtered[m.cursor]
		id := item.GetID()
		if m.selected[id] {
			delete(m.selected, id)
		} else {
			m.selected[id] = true
		}
	}
}

func (m *MultiSelector[T]) applyFilter() {
	if m.filterText == "" {
		m.filtered = m.items
		return
	}

	filter := strings.ToLower(m.filterText)
	m.filtered = nil
	for _, item := range m.items {
		if strings.Contains(strings.ToLower(item.GetLabel()), filter) {
			m.filtered = append(m.filtered, item)
		}
	}
}

func (m *MultiSelector[T]) clampCursor() {
	if len(m.filtered) == 0 {
		m.cursor = -1
	} else if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	} else if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *MultiSelector[T]) updateViewport() {
	if !m.vp.Ready {
		return
	}
	m.vp.Model.SetContent(m.renderContent())

	if m.cursor >= 0 {
		viewportHeight := m.vp.Model.Height()
		if viewportHeight > 0 {
			if m.cursor < m.vp.Model.YOffset() {
				m.vp.Model.SetYOffset(m.cursor)
			} else if m.cursor >= m.vp.Model.YOffset()+viewportHeight {
				m.vp.Model.SetYOffset(m.cursor - viewportHeight + 1)
			}
		}
	}
}

func (m *MultiSelector[T]) renderContent() string {
	var b strings.Builder

	for i, item := range m.filtered {
		style := m.styles.item
		isChecked := m.selected[item.GetID()]

		if i == m.cursor {
			style = m.styles.itemSelected
		} else if isChecked {
			style = m.styles.itemChecked
		}

		checkbox := "☐ "
		if isChecked {
			checkbox = "☑ "
		}

		line := checkbox + item.GetLabel()
		if m.renderExtra != nil {
			if extra := m.renderExtra(item); extra != "" {
				line += " " + extra
			}
		}

		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	return b.String()
}

func (m *MultiSelector[T]) getItemAtPosition(y int) int {
	if !m.vp.Ready {
		return -1
	}
	headerHeight := 1
	if m.filterActive || m.filterText != "" {
		headerHeight++
	}

	contentY := y - headerHeight + m.vp.Model.YOffset()
	if contentY >= 0 && contentY < len(m.filtered) {
		return contentY
	}
	return -1
}

func (m *MultiSelector[T]) ViewString() string {
	s := m.styles

	title := s.title.Render(m.title)

	var filterView string
	if m.filterActive {
		filterView = m.styles.filter.Render(m.filterInput.View()) + "\n"
	} else if m.filterText != "" {
		filterView = m.styles.filter.Render("filter: "+m.filterText) + "\n"
	}

	if !m.vp.Ready {
		return title + "\n" + filterView + LoadingMessage
	}

	return title + "\n" + filterView + m.vp.Model.View()
}

func (m *MultiSelector[T]) SetSize(width, height int) {
	viewportHeight := height - 2 - m.extraHeight
	if m.filterActive || m.filterText != "" {
		viewportHeight--
	}

	m.vp.SetSize(width, viewportHeight)
	m.updateViewport()
}

func (m *MultiSelector[T]) FilterActive() bool {
	return m.filterActive
}

func (m *MultiSelector[T]) SelectedCount() int {
	return len(m.selected)
}
