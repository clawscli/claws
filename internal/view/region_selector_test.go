package view

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestRegionSelectorMouseHover(t *testing.T) {
	ctx := context.Background()

	selector := NewRegionSelector(ctx)
	selector.SetSize(100, 50)

	selector.Update(regionsLoadedMsg{regions: []string{"us-east-1", "us-west-2", "eu-west-1"}})

	initialCursor := selector.selector.Cursor()

	motionMsg := tea.MouseMotionMsg{X: 10, Y: 3}
	selector.Update(motionMsg)

	t.Logf("Cursor after hover: %d (was %d)", selector.selector.Cursor(), initialCursor)
}

func TestRegionSelectorMouseClick(t *testing.T) {
	ctx := context.Background()

	selector := NewRegionSelector(ctx)
	selector.SetSize(100, 50)

	selector.Update(regionsLoadedMsg{regions: []string{"us-east-1", "us-west-2", "eu-west-1"}})

	clickMsg := tea.MouseClickMsg{X: 10, Y: 3, Button: tea.MouseLeft}
	_, cmd := selector.Update(clickMsg)

	t.Logf("Command after click: %v", cmd)
}

func TestRegionSelectorEmptyFilter(t *testing.T) {
	ctx := context.Background()

	selector := NewRegionSelector(ctx)
	selector.SetSize(100, 50)

	selector.Update(regionsLoadedMsg{regions: []string{"us-east-1", "us-west-2", "eu-west-1"}})

	selector.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	for _, r := range "zzz-nonexistent" {
		selector.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	selector.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if selector.selector.FilteredLen() != 0 {
		t.Errorf("Expected 0 filtered regions, got %d", selector.selector.FilteredLen())
	}
	if selector.selector.Cursor() != -1 {
		t.Errorf("Expected cursor -1 for empty filter, got %d", selector.selector.Cursor())
	}

	selector.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})

	if selector.selector.FilteredLen() != 3 {
		t.Errorf("Expected 3 filtered regions after clear, got %d", selector.selector.FilteredLen())
	}
	if selector.selector.Cursor() < 0 {
		t.Errorf("Expected cursor >= 0 after clear, got %d", selector.selector.Cursor())
	}
}
