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

	// Simulate regions loaded
	selector.regions = []string{"us-east-1", "us-west-2", "eu-west-1"}
	selector.applyFilter()
	selector.updateViewport()

	initialCursor := selector.cursor

	// Simulate mouse motion
	motionMsg := tea.MouseMotionMsg{X: 10, Y: 3}
	selector.Update(motionMsg)

	t.Logf("Cursor after hover: %d (was %d)", selector.cursor, initialCursor)
}

func TestRegionSelectorMouseClick(t *testing.T) {
	ctx := context.Background()

	selector := NewRegionSelector(ctx)
	selector.SetSize(100, 50)

	// Simulate regions loaded
	selector.regions = []string{"us-east-1", "us-west-2", "eu-west-1"}
	selector.applyFilter()
	selector.updateViewport()

	// Simulate mouse click
	clickMsg := tea.MouseClickMsg{X: 10, Y: 3, Button: tea.MouseLeft}
	_, cmd := selector.Update(clickMsg)

	// Click might trigger region selection
	t.Logf("Command after click: %v", cmd)
}

func TestRegionSelectorEmptyFilter(t *testing.T) {
	ctx := context.Background()

	selector := NewRegionSelector(ctx)
	selector.SetSize(100, 50)

	// Simulate regions loaded
	selector.regions = []string{"us-east-1", "us-west-2", "eu-west-1"}
	selector.applyFilter()
	selector.updateViewport()

	// Apply filter that matches nothing
	selector.filterText = "zzz-nonexistent"
	selector.applyFilter()
	selector.clampCursor()

	if len(selector.filtered) != 0 {
		t.Errorf("Expected 0 filtered regions, got %d", len(selector.filtered))
	}
	if selector.cursor != -1 {
		t.Errorf("Expected cursor -1 for empty filter, got %d", selector.cursor)
	}

	// Clear filter - should restore regions
	selector.filterText = ""
	selector.applyFilter()
	selector.clampCursor()

	if len(selector.filtered) != 3 {
		t.Errorf("Expected 3 filtered regions after clear, got %d", len(selector.filtered))
	}
	if selector.cursor < 0 {
		t.Errorf("Expected cursor >= 0 after clear, got %d", selector.cursor)
	}
}
