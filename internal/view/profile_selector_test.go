package view

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestProfileSelectorMouseHover(t *testing.T) {
	ctx := context.Background()

	selector := NewProfileSelector(ctx)
	selector.SetSize(100, 50)

	// Simulate profiles loaded
	selector.profiles = []profileItem{
		{ID: "default", DisplayName: "default", IsSSO: false},
		{ID: "dev", DisplayName: "dev", IsSSO: false},
		{ID: "prod-sso", DisplayName: "prod-sso", IsSSO: true},
	}
	selector.applyFilter()
	selector.updateViewport()

	initialCursor := selector.cursor

	// Simulate mouse motion
	motionMsg := tea.MouseMotionMsg{X: 10, Y: 3}
	selector.Update(motionMsg)

	t.Logf("Cursor after hover: %d (was %d)", selector.cursor, initialCursor)
}

func TestProfileSelectorMouseClick(t *testing.T) {
	ctx := context.Background()

	selector := NewProfileSelector(ctx)
	selector.SetSize(100, 50)

	// Simulate profiles loaded
	selector.profiles = []profileItem{
		{ID: "default", DisplayName: "default", IsSSO: false},
		{ID: "dev", DisplayName: "dev", IsSSO: false},
		{ID: "prod-sso", DisplayName: "prod-sso", IsSSO: true},
	}
	selector.applyFilter()
	selector.updateViewport()

	// Simulate mouse click
	clickMsg := tea.MouseClickMsg{X: 10, Y: 3, Button: tea.MouseLeft}
	_, cmd := selector.Update(clickMsg)

	// Click might trigger profile selection toggle
	t.Logf("Command after click: %v", cmd)
}

func TestProfileSelectorEmptyFilter(t *testing.T) {
	ctx := context.Background()

	selector := NewProfileSelector(ctx)
	selector.SetSize(100, 50)

	// Simulate profiles loaded
	selector.profiles = []profileItem{
		{ID: "default", DisplayName: "default", IsSSO: false},
		{ID: "dev", DisplayName: "dev", IsSSO: false},
		{ID: "prod-sso", DisplayName: "prod-sso", IsSSO: true},
	}
	selector.applyFilter()
	selector.updateViewport()

	// Apply filter that matches nothing
	selector.filterText = "zzz-nonexistent"
	selector.applyFilter()
	selector.clampCursor()

	if len(selector.filtered) != 0 {
		t.Errorf("Expected 0 filtered profiles, got %d", len(selector.filtered))
	}
	if selector.cursor != -1 {
		t.Errorf("Expected cursor -1 for empty filter, got %d", selector.cursor)
	}

	// Clear filter - should restore profiles
	selector.filterText = ""
	selector.applyFilter()
	selector.clampCursor()

	if len(selector.filtered) != 3 {
		t.Errorf("Expected 3 filtered profiles after clear, got %d", len(selector.filtered))
	}
	if selector.cursor < 0 {
		t.Errorf("Expected cursor >= 0 after clear, got %d", selector.cursor)
	}
}

func TestProfileSelectorFilterMatching(t *testing.T) {
	ctx := context.Background()

	selector := NewProfileSelector(ctx)
	selector.SetSize(100, 50)

	// Simulate profiles loaded
	selector.profiles = []profileItem{
		{ID: "default", DisplayName: "default", IsSSO: false},
		{ID: "dev", DisplayName: "dev", IsSSO: false},
		{ID: "dev-staging", DisplayName: "dev-staging", IsSSO: false},
		{ID: "prod-sso", DisplayName: "prod-sso", IsSSO: true},
	}

	// Filter by "dev"
	selector.filterText = "dev"
	selector.applyFilter()

	if len(selector.filtered) != 2 {
		t.Errorf("Expected 2 profiles matching 'dev', got %d", len(selector.filtered))
	}

	// Filter is case-insensitive
	selector.filterText = "DEV"
	selector.applyFilter()

	if len(selector.filtered) != 2 {
		t.Errorf("Expected 2 profiles matching 'DEV' (case-insensitive), got %d", len(selector.filtered))
	}

	// Filter by "sso"
	selector.filterText = "sso"
	selector.applyFilter()

	if len(selector.filtered) != 1 {
		t.Errorf("Expected 1 profile matching 'sso', got %d", len(selector.filtered))
	}
}

func TestProfileSelectorSSODetection(t *testing.T) {
	ctx := context.Background()

	selector := NewProfileSelector(ctx)
	selector.SetSize(100, 50)

	// Simulate profiles loaded with mixed SSO status
	selector.profiles = []profileItem{
		{ID: "default", DisplayName: "default", IsSSO: false},
		{ID: "prod-sso", DisplayName: "prod-sso", IsSSO: true},
	}
	selector.applyFilter()
	selector.updateViewport()

	// Find SSO profile
	var ssoProfile *profileItem
	for i := range selector.profiles {
		if selector.profiles[i].IsSSO {
			ssoProfile = &selector.profiles[i]
			break
		}
	}

	if ssoProfile == nil {
		t.Fatal("Expected to find SSO profile")
	}
	if ssoProfile.ID != "prod-sso" {
		t.Errorf("Expected SSO profile 'prod-sso', got %q", ssoProfile.ID)
	}

	// Verify non-SSO profile
	var nonSSOProfile *profileItem
	for i := range selector.profiles {
		if !selector.profiles[i].IsSSO {
			nonSSOProfile = &selector.profiles[i]
			break
		}
	}

	if nonSSOProfile == nil {
		t.Fatal("Expected to find non-SSO profile")
	}
	if nonSSOProfile.IsSSO {
		t.Error("Expected non-SSO profile to have IsSSO=false")
	}
}

func TestProfileSelectorToggle(t *testing.T) {
	ctx := context.Background()

	selector := NewProfileSelector(ctx)
	selector.SetSize(100, 50)

	// Simulate profiles loaded
	selector.profiles = []profileItem{
		{ID: "default", DisplayName: "default", IsSSO: false},
		{ID: "dev", DisplayName: "dev", IsSSO: false},
	}
	selector.applyFilter()
	selector.cursor = 0
	selector.updateViewport()

	// Initially no selection
	selector.selected = make(map[string]bool)

	// Toggle first profile
	selector.toggleCurrent()
	if !selector.selected["default"] {
		t.Error("Expected 'default' to be selected after toggle")
	}

	// Toggle again to deselect
	selector.toggleCurrent()
	if selector.selected["default"] {
		t.Error("Expected 'default' to be deselected after second toggle")
	}

	// Select multiple
	selector.cursor = 0
	selector.toggleCurrent()
	selector.cursor = 1
	selector.toggleCurrent()

	if !selector.selected["default"] || !selector.selected["dev"] {
		t.Error("Expected both profiles to be selected")
	}
}
