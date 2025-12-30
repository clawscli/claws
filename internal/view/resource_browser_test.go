package view

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
)

func TestResourceBrowserFilterEsc(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	browser := NewResourceBrowser(ctx, reg, "ec2")

	// Simulate filter being active
	browser.filterActive = true
	browser.filterInput.Focus()

	// Verify HasActiveInput returns true
	if !browser.HasActiveInput() {
		t.Error("Expected HasActiveInput() to be true when filter is active")
	}

	// Send esc
	escMsg := tea.KeyPressMsg{Code: tea.KeyEscape}
	browser.Update(escMsg)

	// Filter should now be inactive
	if browser.filterActive {
		t.Error("Expected filterActive to be false after esc")
	}

	// HasActiveInput should now return false
	if browser.HasActiveInput() {
		t.Error("Expected HasActiveInput() to be false after esc")
	}
}

func TestResourceBrowserInputCapture(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	browser := NewResourceBrowser(ctx, reg, "ec2")

	// Check that ResourceBrowser implements InputCapture
	var _ InputCapture = browser

	// Initially no active input
	if browser.HasActiveInput() {
		t.Error("Expected HasActiveInput() to be false initially")
	}

	// Activate filter
	browser.filterActive = true
	if !browser.HasActiveInput() {
		t.Error("Expected HasActiveInput() to be true when filter is active")
	}
}

func TestResourceBrowserTagFilter(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	browser := NewResourceBrowser(ctx, reg, "ec2")

	// Set up test resources with tags
	browser.resources = []dao.Resource{
		&mockResource{id: "i-1", name: "web-prod", tags: map[string]string{"Environment": "production", "Team": "web"}},
		&mockResource{id: "i-2", name: "web-dev", tags: map[string]string{"Environment": "development", "Team": "web"}},
		&mockResource{id: "i-3", name: "api-prod", tags: map[string]string{"Environment": "production", "Team": "api"}},
		&mockResource{id: "i-4", name: "no-tags", tags: nil},
	}

	tests := []struct {
		name      string
		tagFilter string
		wantCount int
		wantIDs   []string
	}{
		{
			name:      "exact match",
			tagFilter: "Environment=production",
			wantCount: 2,
			wantIDs:   []string{"i-1", "i-3"},
		},
		{
			name:      "key exists",
			tagFilter: "Team",
			wantCount: 3,
			wantIDs:   []string{"i-1", "i-2", "i-3"},
		},
		{
			name:      "partial match",
			tagFilter: "Environment~prod",
			wantCount: 2,
			wantIDs:   []string{"i-1", "i-3"},
		},
		{
			name:      "partial match case insensitive",
			tagFilter: "Environment~PROD",
			wantCount: 2,
			wantIDs:   []string{"i-1", "i-3"},
		},
		{
			name:      "no match",
			tagFilter: "Environment=staging",
			wantCount: 0,
			wantIDs:   []string{},
		},
		{
			name:      "non-existent key",
			tagFilter: "NonExistent",
			wantCount: 0,
			wantIDs:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use tagFilterText (from :tag command) instead of filterText
			browser.tagFilterText = tt.tagFilter
			browser.filterText = "" // Clear text filter
			browser.applyFilter()

			if len(browser.filtered) != tt.wantCount {
				t.Errorf("got %d resources, want %d", len(browser.filtered), tt.wantCount)
			}

			for i, wantID := range tt.wantIDs {
				if i < len(browser.filtered) && browser.filtered[i].GetID() != wantID {
					t.Errorf("filtered[%d].GetID() = %q, want %q", i, browser.filtered[i].GetID(), wantID)
				}
			}

			// Clean up for next test
			browser.tagFilterText = ""
		})
	}
}

func TestResourceBrowserMouseHover(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	browser := NewResourceBrowser(ctx, reg, "ec2")
	browser.SetSize(100, 50)

	// Add some test resources
	browser.resources = []dao.Resource{
		&mockResource{id: "i-1", name: "instance-1"},
		&mockResource{id: "i-2", name: "instance-2"},
	}
	browser.applyFilter()
	browser.buildTable()

	initialCursor := browser.table.Cursor()

	// Simulate mouse motion
	motionMsg := tea.MouseMotionMsg{X: 30, Y: 10}
	browser.Update(motionMsg)

	t.Logf("Cursor after hover: %d (was %d)", browser.table.Cursor(), initialCursor)
}

func TestResourceBrowserMouseClick(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	browser := NewResourceBrowser(ctx, reg, "ec2")
	browser.SetSize(100, 50)

	// Add some test resources
	browser.resources = []dao.Resource{
		&mockResource{id: "i-1", name: "instance-1"},
		&mockResource{id: "i-2", name: "instance-2"},
	}
	browser.applyFilter()
	browser.buildTable()

	// Simulate mouse click
	clickMsg := tea.MouseClickMsg{X: 30, Y: 10, Button: tea.MouseLeft}
	_, cmd := browser.Update(clickMsg)

	t.Logf("Command after click: %v", cmd)
}

func TestResourceBrowserMarkUnmark(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	browser := NewResourceBrowser(ctx, reg, "ec2")
	browser.SetSize(100, 50)
	browser.renderer = &mockRenderer{detail: "test"}

	browser.resources = []dao.Resource{
		&mockResource{id: "i-1", name: "instance-1"},
		&mockResource{id: "i-2", name: "instance-2"},
	}
	browser.applyFilter()
	browser.buildTable()

	// Initially no mark
	if browser.markedResource != nil {
		t.Error("Expected no marked resource initially")
	}

	// Mark first resource
	browser.table.SetCursor(0)
	mMsg := tea.KeyPressMsg{Code: 'm'}
	browser.Update(mMsg)

	if browser.markedResource == nil {
		t.Fatal("Expected resource to be marked after 'm'")
	}
	if browser.markedResource.GetID() != "i-1" {
		t.Errorf("Expected marked resource i-1, got %s", browser.markedResource.GetID())
	}

	// Mark same resource again (should unmark)
	browser.Update(mMsg)

	if browser.markedResource != nil {
		t.Error("Expected mark to be cleared when marking same resource")
	}

	// Mark first, then mark second (should replace)
	browser.table.SetCursor(0)
	browser.Update(mMsg)
	browser.table.SetCursor(1)
	browser.Update(mMsg)

	if browser.markedResource == nil {
		t.Fatal("Expected resource to be marked")
	}
	if browser.markedResource.GetID() != "i-2" {
		t.Errorf("Expected marked resource i-2, got %s", browser.markedResource.GetID())
	}
}

func TestResourceBrowserMarkClearedOnResourceTypeSwitch(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	reg.RegisterCustom("ec2", "instances", registry.Entry{})
	reg.RegisterCustom("ec2", "volumes", registry.Entry{})

	browser := NewResourceBrowserWithType(ctx, reg, "ec2", "instances")
	browser.SetSize(100, 50)
	browser.renderer = &mockRenderer{detail: "test"}

	browser.resources = []dao.Resource{
		&mockResource{id: "i-1", name: "instance-1"},
	}
	browser.applyFilter()
	browser.buildTable()

	browser.table.SetCursor(0)
	mMsg := tea.KeyPressMsg{Code: 'm'}
	browser.Update(mMsg)

	if browser.markedResource == nil {
		t.Fatal("Expected resource to be marked")
	}

	// Switch resource type with Tab
	browser.cycleResourceType(1)

	if browser.markedResource != nil {
		t.Error("Expected mark to be cleared after Tab (cycleResourceType)")
	}

	browser.resourceType = "instances"
	browser.renderer = &mockRenderer{detail: "test"}
	browser.resources = []dao.Resource{
		&mockResource{id: "i-1", name: "instance-1"},
	}
	browser.applyFilter()
	browser.buildTable()
	browser.table.SetCursor(0)
	browser.Update(mMsg)

	if browser.markedResource == nil {
		t.Fatal("Expected resource to be marked again")
	}

	// Switch with number key (simulated via direct resourceType change + clear)
	// The actual key handling clears markedResource, so we test that path
	numMsg := tea.KeyPressMsg{Code: '2'}
	browser.Update(numMsg)

	if browser.markedResource != nil {
		t.Error("Expected mark to be cleared after number key switch")
	}
}

func TestResourceBrowserMarkClearedOnFilter(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	browser := NewResourceBrowser(ctx, reg, "ec2")
	browser.SetSize(100, 50)
	browser.renderer = &mockRenderer{detail: "test"}

	browser.resources = []dao.Resource{
		&mockResource{id: "i-1", name: "web-server"},
		&mockResource{id: "i-2", name: "db-server"},
	}
	browser.applyFilter()
	browser.buildTable()

	// Mark the first resource
	browser.table.SetCursor(0)
	mMsg := tea.KeyPressMsg{Code: 'm'}
	browser.Update(mMsg)

	if browser.markedResource == nil {
		t.Fatal("Expected resource to be marked")
	}

	// Apply filter that excludes marked resource
	browser.filterText = "db"
	browser.applyFilter()
	browser.buildTable()

	// Mark should be cleared when marked resource is filtered out
	if browser.markedResource != nil {
		t.Error("Expected mark to be cleared when marked resource is filtered out")
	}
}

func TestResourceBrowserDiffHintVisibility(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	browser := NewResourceBrowser(ctx, reg, "ec2")
	browser.SetSize(100, 50)
	browser.renderer = &mockRenderer{detail: "test"}

	browser.resources = []dao.Resource{
		&mockResource{id: "i-1", name: "web-server"},
		&mockResource{id: "i-2", name: "db-server"},
	}
	browser.applyFilter()
	browser.buildTable()

	// No mark: should show "d:describe"
	status := browser.StatusLine()
	if !strings.Contains(status, "d:describe") {
		t.Errorf("Expected 'd:describe' in status line without mark, got: %s", status)
	}
	if strings.Contains(status, "d:diff") {
		t.Errorf("Unexpected 'd:diff' in status line without mark, got: %s", status)
	}

	// Mark a resource: should show "d:diff"
	browser.table.SetCursor(0)
	mMsg := tea.KeyPressMsg{Code: 'm'}
	browser.Update(mMsg)

	status = browser.StatusLine()
	if !strings.Contains(status, "d:diff") {
		t.Errorf("Expected 'd:diff' in status line with mark, got: %s", status)
	}
}

func TestResourceBrowserMarkColumnRendering(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	browser := NewResourceBrowser(ctx, reg, "ec2")
	browser.SetSize(100, 50)
	browser.renderer = &mockRenderer{detail: "test"}
	browser.loading = false

	browser.resources = []dao.Resource{
		&mockResource{id: "i-1", name: "instance-1"},
		&mockResource{id: "i-2", name: "instance-2"},
	}
	browser.applyFilter()
	browser.buildTable()

	view := browser.ViewString()
	if view == "" {
		t.Error("Expected non-empty view")
	}

	browser.table.SetCursor(0)
	mMsg := tea.KeyPressMsg{Code: 'm'}
	browser.Update(mMsg)

	view = browser.ViewString()
	if !strings.Contains(view, "◆") {
		t.Errorf("Expected mark indicator '◆' in view, got: %s", view)
	}
}

func TestResourceBrowserEscClearsMark(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	browser := NewResourceBrowser(ctx, reg, "ec2")
	browser.SetSize(100, 50)
	browser.renderer = &mockRenderer{detail: "test"}

	browser.resources = []dao.Resource{
		&mockResource{id: "i-1", name: "instance-1"},
	}
	browser.applyFilter()
	browser.buildTable()

	// Mark a resource
	browser.table.SetCursor(0)
	mMsg := tea.KeyPressMsg{Code: 'm'}
	browser.Update(mMsg)

	if browser.markedResource == nil {
		t.Fatal("Expected resource to be marked")
	}

	// Press Esc - should clear mark and consume key
	escMsg := tea.KeyPressMsg{Code: tea.KeyEscape}
	_, cmd := browser.Update(escMsg)

	if browser.markedResource != nil {
		t.Error("Expected mark to be cleared after Esc")
	}
	if cmd != nil {
		t.Error("Expected nil cmd (Esc consumed by mark clear)")
	}
}

func TestResourceBrowserDiffNavigation(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	browser := NewResourceBrowser(ctx, reg, "ec2")
	browser.SetSize(100, 50)
	browser.renderer = &mockRenderer{detail: "test"}
	browser.loading = false

	browser.resources = []dao.Resource{
		&mockResource{id: "i-1", name: "instance-1"},
		&mockResource{id: "i-2", name: "instance-2"},
	}
	browser.applyFilter()
	browser.buildTable()

	browser.table.SetCursor(0)
	browser.Update(tea.KeyPressMsg{Code: 'm'})

	if browser.markedResource == nil {
		t.Fatal("Expected resource to be marked")
	}

	browser.table.SetCursor(1)
	_, cmd := browser.Update(tea.KeyPressMsg{Code: 'd'})

	if cmd == nil {
		t.Fatal("Expected cmd from 'd' press with mark set")
	}

	msg := cmd()
	navMsg, ok := msg.(NavigateMsg)
	if !ok {
		t.Fatalf("Expected NavigateMsg, got %T", msg)
	}

	if _, isDiff := navMsg.View.(*DiffView); !isDiff {
		t.Errorf("Expected DiffView, got %T", navMsg.View)
	}
}
