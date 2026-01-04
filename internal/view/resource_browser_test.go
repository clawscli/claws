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

func TestFetchParallelBasic(t *testing.T) {
	ctx := context.Background()
	keys := []string{"a", "b", "c"}

	fetch := func(_ context.Context, k string) ([]dao.Resource, string, error) {
		return []dao.Resource{&mockResource{id: k + "-1"}}, "", nil
	}
	formatError := func(k string, err error) string {
		return k + ": " + err.Error()
	}

	result := fetchParallel(ctx, keys, fetch, formatError)

	if len(result.resources) != 3 {
		t.Errorf("got %d resources, want 3", len(result.resources))
	}
	if len(result.errors) != 0 {
		t.Errorf("got %d errors, want 0", len(result.errors))
	}
}

func TestFetchParallelWithPageTokens(t *testing.T) {
	ctx := context.Background()
	keys := []string{"region-1", "region-2"}

	fetch := func(_ context.Context, k string) ([]dao.Resource, string, error) {
		if k == "region-1" {
			return []dao.Resource{&mockResource{id: "r1-item"}}, "next-token-1", nil
		}
		return []dao.Resource{&mockResource{id: "r2-item"}}, "", nil
	}
	formatError := func(k string, err error) string { return k + ": " + err.Error() }

	result := fetchParallel(ctx, keys, fetch, formatError)

	if len(result.resources) != 2 {
		t.Errorf("got %d resources, want 2", len(result.resources))
	}
	if len(result.pageTokens) != 1 {
		t.Errorf("got %d page tokens, want 1", len(result.pageTokens))
	}
	if result.pageTokens["region-1"] != "next-token-1" {
		t.Errorf("got token %q, want %q", result.pageTokens["region-1"], "next-token-1")
	}
}

func TestFetchParallelPartialErrors(t *testing.T) {
	ctx := context.Background()
	keys := []string{"ok", "fail", "ok2"}

	fetch := func(_ context.Context, k string) ([]dao.Resource, string, error) {
		if k == "fail" {
			return nil, "", context.DeadlineExceeded
		}
		return []dao.Resource{&mockResource{id: k}}, "", nil
	}
	formatError := func(k string, err error) string { return k + ": " + err.Error() }

	result := fetchParallel(ctx, keys, fetch, formatError)

	if len(result.resources) != 2 {
		t.Errorf("got %d resources, want 2", len(result.resources))
	}
	if len(result.errors) != 1 {
		t.Errorf("got %d errors, want 1", len(result.errors))
	}
	if !strings.Contains(result.errors[0], "fail") {
		t.Errorf("error should mention 'fail', got: %s", result.errors[0])
	}
}

func TestFetchParallelEmptyKeys(t *testing.T) {
	ctx := context.Background()
	var keys []string

	fetch := func(_ context.Context, k string) ([]dao.Resource, string, error) {
		t.Error("fetch should not be called for empty keys")
		return nil, "", nil
	}
	formatError := func(k string, err error) string { return "" }

	result := fetchParallel(ctx, keys, fetch, formatError)

	if len(result.resources) != 0 {
		t.Errorf("got %d resources, want 0", len(result.resources))
	}
	if len(result.errors) != 0 {
		t.Errorf("got %d errors, want 0", len(result.errors))
	}
}

func TestFetchParallelPreservesKeyOrder(t *testing.T) {
	ctx := context.Background()
	keys := []string{"z", "a", "m"}

	fetch := func(_ context.Context, k string) ([]dao.Resource, string, error) {
		return []dao.Resource{&mockResource{id: k}}, k + "-token", nil
	}
	formatError := func(k string, err error) string { return "" }

	result := fetchParallel(ctx, keys, fetch, formatError)

	if len(result.resources) != 3 {
		t.Fatalf("got %d resources, want 3", len(result.resources))
	}
	for i, key := range keys {
		if result.resources[i].GetID() != key {
			t.Errorf("resources[%d].GetID() = %q, want %q", i, result.resources[i].GetID(), key)
		}
	}
}

func TestFetchParallelAllErrors(t *testing.T) {
	ctx := context.Background()
	keys := []string{"fail1", "fail2"}

	fetch := func(_ context.Context, k string) ([]dao.Resource, string, error) {
		return nil, "", context.DeadlineExceeded
	}
	formatError := func(k string, err error) string { return k + ": timeout" }

	result := fetchParallel(ctx, keys, fetch, formatError)

	if len(result.resources) != 0 {
		t.Errorf("got %d resources, want 0", len(result.resources))
	}
	if len(result.errors) != 2 {
		t.Errorf("got %d errors, want 2", len(result.errors))
	}
}

func TestResourceBrowserCopyID(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	browser := NewResourceBrowser(ctx, reg, "ec2")
	browser.SetSize(100, 50)
	browser.renderer = &mockRenderer{detail: "test"}

	browser.resources = []dao.Resource{
		&mockResource{id: "i-1234567890abcdef0", name: "instance-1", arn: "arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0"},
	}
	browser.applyFilter()
	browser.buildTable()
	browser.table.SetCursor(0)

	_, cmd := browser.Update(tea.KeyPressMsg{Code: 'y'})
	if cmd == nil {
		t.Fatal("Expected cmd from 'y' key press")
	}

	msg := cmd()
	if msg == nil {
		t.Fatal("Expected message from clipboard command")
	}
}

func TestResourceBrowserCopyARN(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	browser := NewResourceBrowser(ctx, reg, "ec2")
	browser.SetSize(100, 50)
	browser.renderer = &mockRenderer{detail: "test"}

	browser.resources = []dao.Resource{
		&mockResource{id: "i-1234567890abcdef0", name: "instance-1", arn: "arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0"},
	}
	browser.applyFilter()
	browser.buildTable()
	browser.table.SetCursor(0)

	_, cmd := browser.Update(tea.KeyPressMsg{Code: 'Y'})
	if cmd == nil {
		t.Fatal("Expected cmd from 'Y' key press")
	}
}

func TestResourceBrowserCopyARNNoARN(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	browser := NewResourceBrowser(ctx, reg, "ec2")
	browser.SetSize(100, 50)
	browser.renderer = &mockRenderer{detail: "test"}

	browser.resources = []dao.Resource{
		&mockResource{id: "resource-1", name: "no-arn-resource", arn: ""},
	}
	browser.applyFilter()
	browser.buildTable()
	browser.table.SetCursor(0)

	_, cmd := browser.Update(tea.KeyPressMsg{Code: 'Y'})
	if cmd == nil {
		t.Fatal("Expected cmd from 'Y' key press for NoARN")
	}
}

func TestResourceBrowserCopyEmptyList(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	browser := NewResourceBrowser(ctx, reg, "ec2")
	browser.SetSize(100, 50)
	browser.resources = []dao.Resource{}
	browser.applyFilter()
	browser.buildTable()

	_, cmdY := browser.Update(tea.KeyPressMsg{Code: 'y'})
	if cmdY != nil {
		t.Error("Expected nil cmd for 'y' on empty list")
	}

	_, cmdShiftY := browser.Update(tea.KeyPressMsg{Code: 'Y'})
	if cmdShiftY != nil {
		t.Error("Expected nil cmd for 'Y' on empty list")
	}
}
