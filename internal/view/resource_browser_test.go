package view

import (
	"context"
	"strings"
	"sync"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/clawscli/claws/internal/config"
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

func TestResourceBrowserSetInitialFilter(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	browser := NewResourceBrowser(ctx, reg, "ec2")
	browser.SetInitialFilter("bastion")

	if browser.FilterText() != "bastion" {
		t.Errorf("FilterText() = %q, want %q", browser.FilterText(), "bastion")
	}
	if browser.filterInput.Value() != "bastion" {
		t.Errorf("filterInput.Value() = %q, want %q", browser.filterInput.Value(), "bastion")
	}

	browser.resources = []dao.Resource{
		&mockResource{id: "i-1", name: "prod-bastion"},
		&mockResource{id: "i-2", name: "web-server"},
		&mockResource{id: "i-3", name: "dev-bastion"},
	}
	browser.applyFilter()

	if len(browser.filtered) != 2 {
		t.Fatalf("got %d resources, want 2", len(browser.filtered))
	}
	for _, want := range []string{"i-1", "i-3"} {
		found := false
		for _, res := range browser.filtered {
			if res.GetID() == want {
				found = true
			}
		}
		if !found {
			t.Errorf("expected %q in filtered results", want)
		}
	}
}

func TestResourceBrowserSetInitialTagFilter(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	browser := NewResourceBrowser(ctx, reg, "ec2")
	browser.SetInitialTagFilter("Role=bastion")

	browser.resources = []dao.Resource{
		&mockResource{id: "i-1", name: "host-1", tags: map[string]string{"Role": "bastion"}},
		&mockResource{id: "i-2", name: "host-2", tags: map[string]string{"Role": "web"}},
		&mockResource{id: "i-3", name: "host-3", tags: map[string]string{"Role": "bastion"}},
	}
	browser.applyFilter()

	if len(browser.filtered) != 2 {
		t.Fatalf("got %d resources, want 2", len(browser.filtered))
	}
	for i, want := range []string{"i-1", "i-3"} {
		if browser.filtered[i].GetID() != want {
			t.Errorf("filtered[%d].GetID() = %q, want %q", i, browser.filtered[i].GetID(), want)
		}
	}
}

func TestResourceBrowserFilterIndicators(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	tests := []struct {
		name        string
		filterText  string
		tagFilter   string
		wantContain []string
		wantAbsent  []string
	}{
		{
			name:       "no filters shows nothing",
			wantAbsent: []string{"filter:", "tag:"},
		},
		{
			name:        "fuzzy filter only",
			filterText:  "web",
			wantContain: []string{"filter: web"},
			wantAbsent:  []string{"tag:"},
		},
		{
			name:        "tag filter only",
			tagFilter:   "Role=bastion",
			wantContain: []string{"tag: Role=bastion"},
			wantAbsent:  []string{"filter:"},
		},
		{
			name:        "both filters",
			filterText:  "web",
			tagFilter:   "Env=prod",
			wantContain: []string{"filter: web", "tag: Env=prod", "·"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			browser := NewResourceBrowser(ctx, reg, "ec2")
			browser.SetSize(100, 50)
			browser.loading = false
			browser.resources = []dao.Resource{
				&mockResource{id: "i-1", name: "web-prod", tags: map[string]string{"Role": "bastion", "Env": "prod"}},
			}
			browser.filterText = tt.filterText
			browser.tagFilterText = tt.tagFilter
			browser.applyFilter()
			browser.buildTable()

			out := browser.ViewString()
			for _, want := range tt.wantContain {
				if !strings.Contains(out, want) {
					t.Errorf("view should contain %q, got:\n%s", want, out)
				}
			}
			for _, absent := range tt.wantAbsent {
				if strings.Contains(out, absent) {
					t.Errorf("view should not contain %q, got:\n%s", absent, out)
				}
			}
		})
	}
}

func TestResourceBrowserClearFilterClearsAll(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	browser := NewResourceBrowser(ctx, reg, "ec2")
	browser.filterText = "web"
	browser.filterInput.SetValue("web")
	browser.tagFilterText = "Role=bastion"
	browser.fieldFilter = "VpcId"
	browser.fieldFilterValue = "vpc-123"

	browser.handleClearFilter()

	if browser.filterText != "" {
		t.Errorf("filterText = %q, want empty", browser.filterText)
	}
	if browser.filterInput.Value() != "" {
		t.Errorf("filterInput.Value() = %q, want empty", browser.filterInput.Value())
	}
	if browser.tagFilterText != "" {
		t.Errorf("tagFilterText = %q, want empty", browser.tagFilterText)
	}
	if browser.fieldFilter != "" {
		t.Errorf("fieldFilter = %q, want empty", browser.fieldFilter)
	}
	if browser.fieldFilterValue != "" {
		t.Errorf("fieldFilterValue = %q, want empty", browser.fieldFilterValue)
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

	initialCursor := browser.Cursor()

	// Simulate mouse motion
	motionMsg := tea.MouseMotionMsg{X: 30, Y: 10}
	browser.Update(motionMsg)

	t.Logf("Cursor after hover: %d (was %d)", browser.Cursor(), initialCursor)
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
	browser.SetCursor(0)
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
	browser.SetCursor(0)
	browser.Update(mMsg)
	browser.SetCursor(1)
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

	browser.SetCursor(0)
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
	browser.SetCursor(0)
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
	browser.SetCursor(0)
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
	browser.SetCursor(0)
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

	browser.SetCursor(0)
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
	browser.SetCursor(0)
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

	browser.SetCursor(0)
	browser.Update(tea.KeyPressMsg{Code: 'm'})

	if browser.markedResource == nil {
		t.Fatal("Expected resource to be marked")
	}

	browser.SetCursor(1)
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

func TestHandleLoadNextPageWithMultiProfileTokens(t *testing.T) {
	browser := NewResourceBrowser(context.Background(), registry.New(), "ec2")
	browser.loading = false
	browser.hasMorePages = true
	browser.nextMultiPageTokens = map[profileRegionKey]string{
		{Profile: "dev", Region: "us-east-1"}: "token-1",
	}

	_, cmd := browser.handleLoadNextPage()

	if cmd == nil {
		t.Fatal("handleLoadNextPage() returned nil cmd, want loadNextPage cmd")
	}
	if !browser.isLoadingMore {
		t.Fatal("handleLoadNextPage() did not set isLoadingMore")
	}
}

func TestShouldLoadNextPageWithMultiProfileTokens(t *testing.T) {
	browser := NewResourceBrowser(context.Background(), registry.New(), "ec2")
	browser.loading = false
	browser.hasMorePages = true
	browser.nextMultiPageTokens = map[profileRegionKey]string{
		{Profile: "dev", Region: "us-east-1"}: "token-1",
	}
	for i := 0; i < 20; i++ {
		browser.filtered = append(browser.filtered, &mockResource{id: "item"})
	}
	browser.tc.SetCursor(10, len(browser.filtered))

	if !browser.shouldLoadNextPage() {
		t.Fatal("shouldLoadNextPage() = false, want true near bottom with nextMultiPageTokens")
	}

	browser.isLoadingMore = true
	if browser.shouldLoadNextPage() {
		t.Fatal("shouldLoadNextPage() = true while isLoadingMore, want false")
	}
}

func TestHandleNextPageLoadedUpdatesMultiProfileTokens(t *testing.T) {
	browser := NewResourceBrowser(context.Background(), registry.New(), "ec2")
	browser.isLoadingMore = true
	browser.resources = []dao.Resource{&mockResource{id: "existing"}}
	browser.filtered = browser.resources
	remaining := map[profileRegionKey]string{
		{Profile: "prod", Region: "ap-northeast-1"}: "token-2",
	}

	browser.handleNextPageLoaded(nextPageLoadedMsg{
		resources:           []dao.Resource{&mockResource{id: "next"}},
		nextMultiPageTokens: remaining,
		hasMorePages:        true,
	})

	if browser.isLoadingMore {
		t.Fatal("handleNextPageLoaded() left isLoadingMore true")
	}
	if len(browser.resources) != 2 {
		t.Fatalf("resources len = %d, want 2", len(browser.resources))
	}
	if browser.nextMultiPageTokens[profileRegionKey{Profile: "prod", Region: "ap-northeast-1"}] != "token-2" {
		t.Fatalf("nextMultiPageTokens not updated: %#v", browser.nextMultiPageTokens)
	}
	if !browser.hasMorePages {
		t.Fatal("hasMorePages = false, want true")
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

func TestFetchMultiProfileResourcesSkipsPairsWithoutNextToken(t *testing.T) {
	recorder := &recordingPaginatedDAO{BaseDAO: dao.NewBaseDAO("svc", "items")}
	reg := registry.New()
	reg.RegisterCustom("svc", "items", registry.Entry{
		DAOFactory: func(context.Context) (dao.DAO, error) {
			return recorder, nil
		},
	})

	profiles := []config.ProfileSelection{config.NamedProfile("p1"), config.NamedProfile("p2")}
	regions := []string{"us-east-1", "us-west-2"}
	config.Global().SetAccountIDs(map[string]string{"p1": "111111111111", "p2": "222222222222"})

	browser := &ResourceBrowser{
		ctx:          context.Background(),
		registry:     reg,
		service:      "svc",
		resourceType: "items",
		pageSize:     10,
	}

	result := browser.fetchMultiProfileResources(profiles, regions, map[profileRegionKey]string{
		{Profile: "p1", Region: "us-east-1"}: "next-p1-r1",
	})

	if len(result.errors) != 0 {
		t.Fatalf("unexpected errors: %v", result.errors)
	}
	if got := recorder.tokens(); len(got) != 1 || got[0] != "next-p1-r1" {
		t.Fatalf("ListPage tokens = %v, want only [next-p1-r1]", got)
	}
	if len(result.resources) != 1 {
		t.Fatalf("resources = %d, want 1", len(result.resources))
	}
}

type recordingPaginatedDAO struct {
	dao.BaseDAO
	mu         sync.Mutex
	pageTokens []string
}

func (d *recordingPaginatedDAO) List(context.Context) ([]dao.Resource, error) {
	return nil, nil
}

func (d *recordingPaginatedDAO) Get(context.Context, string) (dao.Resource, error) {
	return nil, nil
}

func (d *recordingPaginatedDAO) Delete(context.Context, string) error {
	return nil
}

func (d *recordingPaginatedDAO) ListPage(_ context.Context, _ int, pageToken string) ([]dao.Resource, string, error) {
	d.mu.Lock()
	d.pageTokens = append(d.pageTokens, pageToken)
	d.mu.Unlock()
	return []dao.Resource{&mockResource{id: pageToken, name: pageToken}}, "", nil
}

func (d *recordingPaginatedDAO) tokens() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.pageTokens...)
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
	browser.SetCursor(0)

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
	browser.SetCursor(0)

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
	browser.SetCursor(0)

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
