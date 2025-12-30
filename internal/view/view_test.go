package view

import (
	"context"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
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

func TestDetailViewEsc(t *testing.T) {
	// Create a mock resource
	resource := &mockResource{id: "i-123", name: "test-instance"}
	ctx := context.Background()

	dv := NewDetailView(ctx, resource, nil, "ec2", "instances", nil, nil)
	dv.SetSize(100, 50) // Initialize viewport

	// Send esc to DetailView
	escMsg := tea.KeyPressMsg{Code: tea.KeyEscape}
	model, cmd := dv.Update(escMsg)

	// DetailView should NOT handle esc (returns same model, nil cmd)
	if model != dv {
		t.Error("Expected same model to be returned")
	}
	if cmd != nil {
		t.Error("Expected nil cmd (DetailView doesn't handle esc)")
	}
}

func TestDetailViewEscString(t *testing.T) {
	// Test with string-based esc check
	resource := &mockResource{id: "i-123", name: "test-instance"}
	ctx := context.Background()

	dv := NewDetailView(ctx, resource, nil, "ec2", "instances", nil, nil)
	dv.SetSize(100, 50)

	// Test that "esc" string is correctly identified
	escMsg := tea.KeyPressMsg{Code: tea.KeyEscape}

	if escMsg.String() != "esc" {
		t.Errorf("Expected esc key String() to be 'esc', got %q", escMsg.String())
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

// mockResource for testing
type mockResource struct {
	id   string
	name string
	tags map[string]string
}

func (m *mockResource) GetID() string              { return m.id }
func (m *mockResource) GetName() string            { return m.name }
func (m *mockResource) GetARN() string             { return "" }
func (m *mockResource) GetTags() map[string]string { return m.tags }
func (m *mockResource) Raw() any                   { return nil }

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

// ServiceBrowser tests

func TestServiceBrowserNavigation(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	// Register some test services
	reg.RegisterCustom("ec2", "instances", registry.Entry{})
	reg.RegisterCustom("s3", "buckets", registry.Entry{})
	reg.RegisterCustom("lambda", "functions", registry.Entry{})
	reg.RegisterCustom("iam", "roles", registry.Entry{})

	browser := NewServiceBrowser(ctx, reg)

	// Initialize to load services
	browser.Update(browser.Init()())

	// Check initial state
	if browser.cursor != 0 {
		t.Errorf("Initial cursor = %d, want 0", browser.cursor)
	}

	// Test navigation with 'l' (right)
	browser.Update(tea.KeyPressMsg{Code: 'l'})
	if browser.cursor != 1 {
		t.Errorf("After 'l', cursor = %d, want 1", browser.cursor)
	}

	// Test navigation with 'h' (left)
	browser.Update(tea.KeyPressMsg{Code: 'h'})
	if browser.cursor != 0 {
		t.Errorf("After 'h', cursor = %d, want 0", browser.cursor)
	}
}

func TestServiceBrowserFilter(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	// Register test services
	reg.RegisterCustom("ec2", "instances", registry.Entry{})
	reg.RegisterCustom("s3", "buckets", registry.Entry{})
	reg.RegisterCustom("lambda", "functions", registry.Entry{})

	browser := NewServiceBrowser(ctx, reg)
	browser.Update(browser.Init()())

	initialCount := len(browser.flatItems)
	if initialCount == 0 {
		t.Fatal("No services loaded")
	}

	// Activate filter mode
	browser.Update(tea.KeyPressMsg{Text: "/", Code: '/'})
	if !browser.filterActive {
		t.Error("Expected filter to be active after '/'")
	}

	// Type 'ec2' in filter
	for _, r := range "ec2" {
		browser.Update(tea.KeyPressMsg{Text: string(r), Code: r})
	}

	// Should have fewer items after filtering
	if len(browser.flatItems) >= initialCount {
		t.Errorf("Expected fewer items after filter, got %d (was %d)", len(browser.flatItems), initialCount)
	}

	// Press Esc to exit filter mode
	browser.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if browser.filterActive {
		t.Error("Expected filter to be inactive after Esc")
	}

	// Press 'c' to clear filter
	browser.Update(tea.KeyPressMsg{Code: 'c'})
	if len(browser.flatItems) != initialCount {
		t.Errorf("After clear, items = %d, want %d", len(browser.flatItems), initialCount)
	}
}

func TestServiceBrowserHasActiveInput(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	browser := NewServiceBrowser(ctx, reg)

	// Check ServiceBrowser implements InputCapture
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

func TestServiceBrowserCategoryNavigation(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	// Register services in different categories
	reg.RegisterCustom("ec2", "instances", registry.Entry{})    // Compute
	reg.RegisterCustom("lambda", "functions", registry.Entry{}) // Compute
	reg.RegisterCustom("s3", "buckets", registry.Entry{})       // Storage
	reg.RegisterCustom("iam", "roles", registry.Entry{})        // Security

	browser := NewServiceBrowser(ctx, reg)
	browser.Update(browser.Init()())

	initialCursor := browser.cursor
	initialCat := -1
	if len(browser.flatItems) > 0 {
		initialCat = browser.flatItems[browser.cursor].categoryIdx
	}

	// Test 'j' moves to next category
	browser.Update(tea.KeyPressMsg{Code: 'j'})

	if len(browser.flatItems) > 1 && browser.cursor > 0 {
		newCat := browser.flatItems[browser.cursor].categoryIdx
		if newCat == initialCat && browser.cursor != initialCursor {
			// If still in same category, cursor should have moved
			t.Log("Moved within category or wrapped")
		}
	}
}

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		str     string
		pattern string
		want    bool
	}{
		{"AgentCoreStackdev", "agecrstdev", true},
		{"AgentCoreStackdev", "agent", true},
		{"AgentCoreStackdev", "acd", true},
		{"AgentCoreStackdev", "xyz", false},
		{"AgentCoreStackdev", "deva", false}, // order matters
		{"i-1234567890abcdef0", "i1234", true},
		{"i-1234567890abcdef0", "abcdef", true},
		{"production", "prod", true},
		{"production", "pdn", true},
		{"", "a", false},
		{"abc", "", true}, // empty pattern matches everything
	}

	for _, tt := range tests {
		t.Run(tt.str+"_"+tt.pattern, func(t *testing.T) {
			got := fuzzyMatch(tt.str, tt.pattern)
			if got != tt.want {
				t.Errorf("fuzzyMatch(%q, %q) = %v, want %v", tt.str, tt.pattern, got, tt.want)
			}
		})
	}
}

// CommandInput tests

func TestCommandInput_NewAndBasics(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	ci := NewCommandInput(ctx, reg)

	// Initially should not be active
	if ci.IsActive() {
		t.Error("Expected IsActive() to be false initially")
	}

	// View should be empty when not active
	if ci.View() != "" {
		t.Error("Expected empty View() when not active")
	}
}

func TestCommandInput_ActivateDeactivate(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	ci := NewCommandInput(ctx, reg)

	// Activate
	ci.Activate()
	if !ci.IsActive() {
		t.Error("Expected IsActive() to be true after Activate()")
	}

	// Deactivate
	ci.Deactivate()
	if ci.IsActive() {
		t.Error("Expected IsActive() to be false after Deactivate()")
	}
}

func TestCommandInput_GetSuggestions(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	// Register some services
	reg.RegisterCustom("ec2", "instances", registry.Entry{})
	reg.RegisterCustom("ec2", "volumes", registry.Entry{})
	reg.RegisterCustom("s3", "buckets", registry.Entry{})
	reg.RegisterCustom("lambda", "functions", registry.Entry{})

	ci := NewCommandInput(ctx, reg)
	ci.Activate()

	// Test service suggestions
	ci.textInput.SetValue("e")
	suggestions := ci.GetSuggestions()
	found := false
	for _, s := range suggestions {
		if s == "ec2" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'ec2' in suggestions for 'e'")
	}

	// Test resource suggestions
	ci.textInput.SetValue("ec2/")
	suggestions = ci.GetSuggestions()
	if len(suggestions) == 0 {
		t.Error("Expected suggestions for 'ec2/'")
	}

	// Test tags suggestion
	ci.textInput.SetValue("ta")
	suggestions = ci.GetSuggestions()
	foundTags := false
	for _, s := range suggestions {
		if s == "tags" {
			foundTags = true
			break
		}
	}
	if !foundTags {
		t.Error("Expected 'tags' in suggestions for 'ta'")
	}
}

func TestCommandInput_SetWidth(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	ci := NewCommandInput(ctx, reg)
	ci.SetWidth(100)

	if ci.width != 100 {
		t.Errorf("width = %d, want 100", ci.width)
	}
}

func TestCommandInput_Update_Esc(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	ci := NewCommandInput(ctx, reg)
	ci.Activate()

	// Send esc
	escMsg := tea.KeyPressMsg{Code: tea.KeyEscape}
	ci.Update(escMsg)

	if ci.IsActive() {
		t.Error("Expected IsActive() to be false after esc")
	}
}

func TestCommandInput_Update_Enter_Empty(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	ci := NewCommandInput(ctx, reg)
	ci.Activate()

	// Send enter with empty input (should navigate to service list)
	enterMsg := tea.KeyPressMsg{Code: tea.KeyEnter}
	_, nav := ci.Update(enterMsg)

	if nav == nil {
		t.Error("Expected NavigateMsg for empty enter")
	}
	if nav != nil && !nav.ClearStack {
		t.Error("Expected ClearStack=true for home navigation")
	}
}

func TestCommandInput_Update_Enter_Service(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	reg.RegisterCustom("ec2", "instances", registry.Entry{})

	ci := NewCommandInput(ctx, reg)
	ci.Activate()
	ci.textInput.SetValue("ec2")

	// Send enter
	enterMsg := tea.KeyPressMsg{Code: tea.KeyEnter}
	_, nav := ci.Update(enterMsg)

	if nav == nil {
		t.Error("Expected NavigateMsg for 'ec2'")
	}
}

// IsEscKey tests

func TestIsEscKey(t *testing.T) {
	tests := []struct {
		name string
		msg  tea.KeyPressMsg
		want bool
	}{
		{"KeyEscape", tea.KeyPressMsg{Code: tea.KeyEscape}, true},
		{"raw ESC byte", tea.KeyPressMsg{Code: 27}, true},
		{"Enter", tea.KeyPressMsg{Code: tea.KeyEnter}, false},
		{"Space", tea.KeyPressMsg{Code: tea.KeySpace}, false},
		{"letter a", tea.KeyPressMsg{Code: 'a'}, false},
		{"letter q", tea.KeyPressMsg{Code: 'q'}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsEscKey(tt.msg)
			if got != tt.want {
				t.Errorf("IsEscKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

// DetailView async refresh tests

func TestDetailViewRefreshError(t *testing.T) {
	resource := &mockResource{id: "i-123", name: "test-instance"}
	ctx := context.Background()

	dv := NewDetailView(ctx, resource, nil, "ec2", "instances", nil, nil)
	dv.SetSize(100, 50)

	// Simulate refresh error
	errMsg := detailRefreshMsg{
		resource: resource,
		err:      fmt.Errorf("access denied"),
	}

	dv.Update(errMsg)

	// Check that error is stored
	if dv.refreshErr == nil {
		t.Error("Expected refreshErr to be set after error message")
	}

	// Check status line contains error indicator
	status := dv.StatusLine()
	if !strings.Contains(status, "refresh failed") {
		t.Errorf("StatusLine() = %q, want to contain 'refresh failed'", status)
	}
}

func TestDetailViewRefreshSuccess(t *testing.T) {
	resource := &mockResource{id: "i-123", name: "test-instance"}
	ctx := context.Background()

	dv := NewDetailView(ctx, resource, nil, "ec2", "instances", nil, nil)
	dv.SetSize(100, 50)

	// Set an initial error
	dv.refreshErr = fmt.Errorf("previous error")

	// Simulate successful refresh
	newResource := &mockResource{id: "i-123", name: "updated-instance"}
	successMsg := detailRefreshMsg{
		resource: newResource,
		err:      nil,
	}

	dv.Update(successMsg)

	// Error should be cleared
	if dv.refreshErr != nil {
		t.Error("Expected refreshErr to be nil after successful refresh")
	}

	// Resource should be updated
	if dv.resource.GetName() != "updated-instance" {
		t.Errorf("resource.GetName() = %q, want 'updated-instance'", dv.resource.GetName())
	}
}

// mockDAO for testing
type mockDAO struct {
	dao.BaseDAO
	supportsGet bool
	getErr      error
}

func (m *mockDAO) List(ctx context.Context) ([]dao.Resource, error) {
	return nil, nil
}

func (m *mockDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return &mockResource{id: id, name: "fetched"}, nil
}

func (m *mockDAO) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockDAO) Supports(op dao.Operation) bool {
	if op == dao.OpGet {
		return m.supportsGet
	}
	return true
}

func TestDetailViewInitWithSupportsGet(t *testing.T) {
	resource := &mockResource{id: "i-123", name: "test"}
	ctx := context.Background()

	// DAO that supports Get
	daoWithGet := &mockDAO{supportsGet: true}
	dv := NewDetailView(ctx, resource, nil, "ec2", "instances", nil, daoWithGet)

	cmd := dv.Init()
	if cmd == nil {
		t.Error("Expected Init() to return command when DAO supports Get")
	}
	if !dv.refreshing {
		t.Error("Expected refreshing to be true when DAO supports Get")
	}
}

func TestDetailViewInitWithoutSupportsGet(t *testing.T) {
	resource := &mockResource{id: "i-123", name: "test"}
	ctx := context.Background()

	// DAO that doesn't support Get
	daoWithoutGet := &mockDAO{supportsGet: false}
	dv := NewDetailView(ctx, resource, nil, "ec2", "instances", nil, daoWithoutGet)

	cmd := dv.Init()
	if cmd != nil {
		t.Error("Expected Init() to return nil when DAO doesn't support Get")
	}
	if dv.refreshing {
		t.Error("Expected refreshing to be false when DAO doesn't support Get")
	}
}

// HelpView tests

func TestHelpView_New(t *testing.T) {
	hv := NewHelpView()

	if hv == nil {
		t.Fatal("NewHelpView() returned nil")
	}
}

func TestHelpView_StatusLine(t *testing.T) {
	hv := NewHelpView()

	status := hv.StatusLine()
	if status == "" {
		t.Error("StatusLine() should not be empty")
	}
}

// truncateOrPad tests

func TestTruncateOrPad(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		width   int
		wantLen int // expected visual width (0 means skip check)
		wantEnd string
	}{
		{
			name:    "exact width",
			input:   "hello",
			width:   5,
			wantLen: 5,
		},
		{
			name:    "needs padding",
			input:   "hi",
			width:   5,
			wantLen: 5,
			wantEnd: "   ", // 3 spaces padding
		},
		{
			name:    "needs truncation",
			input:   "hello world",
			width:   5,
			wantLen: 5,
			wantEnd: "…",
		},
		{
			name:    "zero width",
			input:   "hello",
			width:   0,
			wantLen: 0,
		},
		{
			name:    "negative width",
			input:   "hello",
			width:   -1,
			wantLen: 0,
		},
		{
			name:    "empty string padded",
			input:   "",
			width:   5,
			wantLen: 5,
		},
		{
			name:    "width 1 truncation",
			input:   "hello",
			width:   1,
			wantLen: 1,
			wantEnd: "…",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateOrPad(tt.input, tt.width)

			// Check visual width (rune count for plain text with ellipsis)
			gotLen := len([]rune(got))
			if tt.wantLen > 0 && gotLen != tt.wantLen {
				t.Errorf("truncateOrPad(%q, %d) rune len = %d, want %d (got=%q)", tt.input, tt.width, gotLen, tt.wantLen, got)
			}

			if tt.wantEnd != "" && !strings.HasSuffix(got, tt.wantEnd) {
				t.Errorf("truncateOrPad(%q, %d) = %q, want suffix %q", tt.input, tt.width, got, tt.wantEnd)
			}
		})
	}
}

// DiffView tests

func TestDiffView_New(t *testing.T) {
	ctx := context.Background()
	left := &mockResource{id: "i-111", name: "instance-a"}
	right := &mockResource{id: "i-222", name: "instance-b"}

	dv := NewDiffView(ctx, left, right, nil, "ec2", "instances")

	if dv == nil {
		t.Fatal("NewDiffView() returned nil")
	}
	if dv.left.GetID() != "i-111" {
		t.Errorf("left.GetID() = %q, want %q", dv.left.GetID(), "i-111")
	}
	if dv.right.GetID() != "i-222" {
		t.Errorf("right.GetID() = %q, want %q", dv.right.GetID(), "i-222")
	}
}

func TestDiffView_StatusLine(t *testing.T) {
	ctx := context.Background()
	left := &mockResource{id: "i-111", name: "instance-a"}
	right := &mockResource{id: "i-222", name: "instance-b"}

	dv := NewDiffView(ctx, left, right, nil, "ec2", "instances")

	status := dv.StatusLine()
	if !strings.Contains(status, "instance-a") {
		t.Errorf("StatusLine() = %q, want to contain 'instance-a'", status)
	}
	if !strings.Contains(status, "instance-b") {
		t.Errorf("StatusLine() = %q, want to contain 'instance-b'", status)
	}
}

func TestDiffView_SetSize(t *testing.T) {
	ctx := context.Background()
	left := &mockResource{id: "i-111", name: "instance-a"}
	right := &mockResource{id: "i-222", name: "instance-b"}

	dv := NewDiffView(ctx, left, right, nil, "ec2", "instances")

	// Initially not ready
	if dv.ready {
		t.Error("Expected ready to be false initially")
	}

	// SetSize should initialize viewport
	dv.SetSize(100, 50)

	if !dv.ready {
		t.Error("Expected ready to be true after SetSize")
	}
	if dv.width != 100 {
		t.Errorf("width = %d, want 100", dv.width)
	}
	if dv.height != 50 {
		t.Errorf("height = %d, want 50", dv.height)
	}
}

func TestDiffView_Update_Esc(t *testing.T) {
	ctx := context.Background()
	left := &mockResource{id: "i-111", name: "instance-a"}
	right := &mockResource{id: "i-222", name: "instance-b"}

	dv := NewDiffView(ctx, left, right, nil, "ec2", "instances")
	dv.SetSize(100, 50)

	// Send esc - should return nil cmd (let app handle back navigation)
	escMsg := tea.KeyPressMsg{Code: tea.KeyEscape}
	model, cmd := dv.Update(escMsg)

	if model != dv {
		t.Error("Expected same model to be returned")
	}
	if cmd != nil {
		t.Error("Expected nil cmd (DiffView doesn't handle esc)")
	}
}

func TestDiffView_Update_Q(t *testing.T) {
	ctx := context.Background()
	left := &mockResource{id: "i-111", name: "instance-a"}
	right := &mockResource{id: "i-222", name: "instance-b"}

	dv := NewDiffView(ctx, left, right, nil, "ec2", "instances")
	dv.SetSize(100, 50)

	// Send 'q' - should also return nil cmd
	qMsg := tea.KeyPressMsg{Code: 'q'}
	model, cmd := dv.Update(qMsg)

	if model != dv {
		t.Error("Expected same model to be returned")
	}
	if cmd != nil {
		t.Error("Expected nil cmd for 'q' key")
	}
}

func TestDiffView_View_NotReady(t *testing.T) {
	ctx := context.Background()
	left := &mockResource{id: "i-111", name: "instance-a"}
	right := &mockResource{id: "i-222", name: "instance-b"}

	dv := NewDiffView(ctx, left, right, nil, "ec2", "instances")

	// Without SetSize, should show loading
	view := dv.ViewString()
	if view != "Loading..." {
		t.Errorf("ViewString() = %q, want 'Loading...'", view)
	}
}

// mockRenderer for testing renderContent with Loading replacement
type mockRenderer struct {
	detail string
}

func (m *mockRenderer) ServiceName() string                                     { return "test" }
func (m *mockRenderer) ResourceType() string                                    { return "items" }
func (m *mockRenderer) Columns() []render.Column                                { return nil }
func (m *mockRenderer) RenderRow(r dao.Resource, cols []render.Column) []string { return nil }
func (m *mockRenderer) RenderDetail(r dao.Resource) string                      { return m.detail }
func (m *mockRenderer) RenderSummary(r dao.Resource) []render.SummaryField      { return nil }

func TestDetailViewLoadingPlaceholderReplacement(t *testing.T) {
	ctx := context.Background()
	resource := &mockResource{id: "test-1", name: "test-resource"}

	tests := []struct {
		name            string
		detail          string
		refreshing      bool
		wantContains    []string
		wantNotContains []string
	}{
		{
			name:            "refreshing replaces NotConfigured at line end",
			detail:          "Status: " + render.NotConfigured + "\n",
			refreshing:      true,
			wantContains:    []string{"Loading..."},
			wantNotContains: []string{render.NotConfigured},
		},
		{
			name:            "refreshing replaces Empty at line end",
			detail:          "Items: " + render.Empty + "\n",
			refreshing:      true,
			wantContains:    []string{"Loading..."},
			wantNotContains: []string{render.Empty},
		},
		{
			name:            "refreshing replaces NoValue at line end",
			detail:          "Comment: " + render.NoValue + "\n",
			refreshing:      true,
			wantContains:    []string{"Loading..."},
			wantNotContains: []string{render.NoValue},
		},
		{
			name:            "refreshing replaces placeholder at EOF without newline",
			detail:          "Status: " + render.NotConfigured,
			refreshing:      true,
			wantContains:    []string{"Loading..."},
			wantNotContains: []string{render.NotConfigured},
		},
		{
			name:            "refreshing does NOT replace placeholder in middle of text",
			detail:          "Name: Not configured server\n",
			refreshing:      true,
			wantContains:    []string{"Not configured server"}, // Should remain
			wantNotContains: []string{},
		},
		{
			name:            "refreshing does NOT replace NoValue in middle of text",
			detail:          "ID: i-1234567890abcdef0\n",
			refreshing:      true,
			wantContains:    []string{"i-1234567890abcdef0"}, // Hyphens should remain
			wantNotContains: []string{},
		},
		{
			name:            "refreshing replaces multiple different placeholders",
			detail:          "Status: " + render.NotConfigured + "\nItems: " + render.Empty + "\nComment: " + render.NoValue + "\n",
			refreshing:      true,
			wantContains:    []string{"Loading..."},
			wantNotContains: []string{render.NotConfigured, render.Empty, render.NoValue},
		},
		{
			name:            "refreshing replaces multiple same placeholders",
			detail:          "Status: " + render.NotConfigured + "\nEncryption: " + render.NotConfigured + "\n",
			refreshing:      true,
			wantContains:    []string{"Loading..."},
			wantNotContains: []string{render.NotConfigured},
		},
		{
			name:            "refreshing replaces consecutive placeholders",
			detail:          "Status: " + render.NotConfigured + "\n" + render.NoValue + "\n",
			refreshing:      true,
			wantContains:    []string{"Loading..."},
			wantNotContains: []string{render.NotConfigured, render.NoValue},
		},
		{
			name:            "not refreshing keeps placeholders",
			detail:          "Status: " + render.NotConfigured + "\nItems: " + render.Empty + "\n",
			refreshing:      false,
			wantContains:    []string{render.NotConfigured, render.Empty},
			wantNotContains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := &mockRenderer{detail: tt.detail}
			dv := NewDetailView(ctx, resource, renderer, "test", "items", nil, nil)
			dv.refreshing = tt.refreshing
			dv.SetSize(100, 50)

			// Get the viewport content
			content := dv.viewport.View()

			for _, want := range tt.wantContains {
				if !strings.Contains(content, want) {
					t.Errorf("content should contain %q, got:\n%s", want, content)
				}
			}

			for _, notWant := range tt.wantNotContains {
				if strings.Contains(content, notWant) {
					t.Errorf("content should not contain %q, got:\n%s", notWant, content)
				}
			}
		})
	}
}

// Mouse interaction tests

func TestServiceBrowserMouseHover(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	reg.RegisterCustom("ec2", "instances", registry.Entry{})
	reg.RegisterCustom("s3", "buckets", registry.Entry{})

	browser := NewServiceBrowser(ctx, reg)
	browser.Update(browser.Init()())
	browser.SetSize(100, 50)

	initialCursor := browser.cursor

	// Simulate mouse motion - exact position depends on layout
	// Just verify it doesn't crash and cursor can change
	motionMsg := tea.MouseMotionMsg{X: 30, Y: 5}
	browser.Update(motionMsg)

	// Cursor may or may not change depending on position
	// Main test is that it doesn't panic
	t.Logf("Cursor after hover: %d (was %d)", browser.cursor, initialCursor)
}

func TestServiceBrowserMouseClick(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	reg.RegisterCustom("ec2", "instances", registry.Entry{})
	reg.RegisterCustom("s3", "buckets", registry.Entry{})

	browser := NewServiceBrowser(ctx, reg)
	browser.Update(browser.Init()())
	browser.SetSize(100, 50)

	// Simulate mouse click
	clickMsg := tea.MouseClickMsg{X: 30, Y: 5, Button: tea.MouseLeft}
	_, cmd := browser.Update(clickMsg)

	// Click might trigger navigation or do nothing depending on position
	t.Logf("Command after click: %v", cmd)
}

func TestServiceBrowserMouseWheel(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	reg.RegisterCustom("ec2", "instances", registry.Entry{})
	reg.RegisterCustom("s3", "buckets", registry.Entry{})

	browser := NewServiceBrowser(ctx, reg)
	browser.Update(browser.Init()())
	browser.SetSize(100, 50)

	// Simulate mouse wheel
	wheelMsg := tea.MouseWheelMsg{X: 30, Y: 5, Button: tea.MouseWheelDown}
	browser.Update(wheelMsg)

	// Should not panic
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

func TestActionMenuMouseHover(t *testing.T) {
	ctx := context.Background()
	resource := &mockResource{id: "i-123", name: "test"}

	menu := NewActionMenu(ctx, resource, "ec2", "instances")

	initialCursor := menu.cursor

	// Simulate mouse motion
	motionMsg := tea.MouseMotionMsg{X: 10, Y: 5}
	menu.Update(motionMsg)

	t.Logf("Cursor after hover: %d (was %d)", menu.cursor, initialCursor)
}

// ConfirmDangerous state machine tests

func TestActionMenuConfirmDangerousCorrectToken(t *testing.T) {
	ctx := context.Background()
	resource := &mockResource{id: "i-12345", name: "test-instance"}

	menu := NewActionMenu(ctx, resource, "test", "items")

	// Manually set up dangerous confirm state (normally triggered by action selection)
	menu.dangerous.active = true
	menu.confirmIdx = 0
	menu.dangerous.token = "i-12345" // Default: uses GetID()
	menu.dangerous.input = ""

	// Type the correct suffix (last 6 chars of "i-12345" = "-12345")
	suffix := action.ConfirmSuffix("i-12345")
	for _, r := range suffix {
		msg := tea.KeyPressMsg{Text: string(r), Code: r}
		menu.Update(msg)
	}

	if menu.dangerous.input != suffix {
		t.Errorf("dangerousInput = %q, want %q", menu.dangerous.input, suffix)
	}

	// Press enter - should accept since input matches suffix
	enterMsg := tea.KeyPressMsg{Code: tea.KeyEnter}
	menu.Update(enterMsg)

	// Confirm state should be cleared on successful match
	if menu.dangerous.active {
		t.Error("Expected dangerousConfirm to be false after correct token + enter")
	}
	if menu.dangerous.input != "" {
		t.Errorf("Expected dangerousInput to be cleared, got %q", menu.dangerous.input)
	}
	if menu.dangerous.token != "" {
		t.Errorf("Expected confirmToken to be cleared, got %q", menu.dangerous.token)
	}
}

func TestActionMenuConfirmDangerousWrongToken(t *testing.T) {
	ctx := context.Background()
	resource := &mockResource{id: "i-12345", name: "test-instance"}

	menu := NewActionMenu(ctx, resource, "test", "items")

	// Set up dangerous confirm state
	menu.dangerous.active = true
	menu.confirmIdx = 0
	menu.dangerous.token = "i-12345"
	menu.dangerous.input = ""

	// Type wrong token
	for _, r := range "wrong" {
		msg := tea.KeyPressMsg{Text: string(r), Code: r}
		menu.Update(msg)
	}

	if menu.dangerous.input != "wrong" {
		t.Errorf("dangerousInput = %q, want %q", menu.dangerous.input, "wrong")
	}

	// Press enter - should NOT accept since input doesn't match token
	enterMsg := tea.KeyPressMsg{Code: tea.KeyEnter}
	menu.Update(enterMsg)

	// Confirm state should remain (not cleared)
	if !menu.dangerous.active {
		t.Error("Expected dangerousConfirm to remain true after wrong token + enter")
	}
	if menu.dangerous.input != "wrong" {
		t.Errorf("Expected dangerousInput to remain %q, got %q", "wrong", menu.dangerous.input)
	}
}

func TestActionMenuConfirmDangerousEscCancels(t *testing.T) {
	ctx := context.Background()
	resource := &mockResource{id: "i-12345", name: "test-instance"}

	menu := NewActionMenu(ctx, resource, "test", "items")

	// Set up dangerous confirm state with partial input
	menu.dangerous.active = true
	menu.confirmIdx = 0
	menu.dangerous.token = "i-12345"
	menu.dangerous.input = "i-123"

	// Press esc - should cancel
	escMsg := tea.KeyPressMsg{Code: tea.KeyEscape}
	menu.Update(escMsg)

	// Confirm state should be cleared
	if menu.dangerous.active {
		t.Error("Expected dangerousConfirm to be false after esc")
	}
	if menu.dangerous.input != "" {
		t.Errorf("Expected dangerousInput to be cleared, got %q", menu.dangerous.input)
	}
	if menu.dangerous.token != "" {
		t.Errorf("Expected confirmToken to be cleared, got %q", menu.dangerous.token)
	}
}

func TestActionMenuConfirmDangerousBackspaceString(t *testing.T) {
	ctx := context.Background()
	resource := &mockResource{id: "i-12345", name: "test-instance"}

	menu := NewActionMenu(ctx, resource, "test", "items")

	// Set up dangerous confirm state with input
	menu.dangerous.active = true
	menu.confirmIdx = 0
	menu.dangerous.token = "i-12345"
	menu.dangerous.input = "i-123"

	// Test backspace via msg.String() == "backspace"
	// This handles terminals that send backspace as a string
	backspaceMsg := tea.KeyPressMsg{Text: "backspace"}
	menu.Update(backspaceMsg)

	if menu.dangerous.input != "i-12" {
		t.Errorf("After string backspace: dangerousInput = %q, want %q", menu.dangerous.input, "i-12")
	}
}

func TestActionMenuConfirmDangerousBackspaceKeyCode(t *testing.T) {
	ctx := context.Background()
	resource := &mockResource{id: "i-12345", name: "test-instance"}

	menu := NewActionMenu(ctx, resource, "test", "items")

	// Set up dangerous confirm state with input
	menu.dangerous.active = true
	menu.confirmIdx = 0
	menu.dangerous.token = "i-12345"
	menu.dangerous.input = "i-123"

	// Test backspace via msg.Code == tea.KeyBackspace
	// This handles terminals that send backspace as a key code
	backspaceMsg := tea.KeyPressMsg{Code: tea.KeyBackspace}
	menu.Update(backspaceMsg)

	if menu.dangerous.input != "i-12" {
		t.Errorf("After keycode backspace: dangerousInput = %q, want %q", menu.dangerous.input, "i-12")
	}
}

func TestActionMenuConfirmDangerousBackspaceEmpty(t *testing.T) {
	ctx := context.Background()
	resource := &mockResource{id: "i-12345", name: "test-instance"}

	menu := NewActionMenu(ctx, resource, "test", "items")

	// Set up dangerous confirm state with empty input
	menu.dangerous.active = true
	menu.confirmIdx = 0
	menu.dangerous.token = "i-12345"
	menu.dangerous.input = ""

	// Backspace on empty input should be safe (not panic)
	backspaceMsg := tea.KeyPressMsg{Code: tea.KeyBackspace}
	menu.Update(backspaceMsg)

	if menu.dangerous.input != "" {
		t.Errorf("After backspace on empty: dangerousInput = %q, want empty", menu.dangerous.input)
	}

	// Also test string backspace on empty
	backspaceStrMsg := tea.KeyPressMsg{Text: "backspace"}
	menu.Update(backspaceStrMsg)

	if menu.dangerous.input != "" {
		t.Errorf("After string backspace on empty: dangerousInput = %q, want empty", menu.dangerous.input)
	}
}

func TestActionMenuConfirmDangerousHasActiveInput(t *testing.T) {
	ctx := context.Background()
	resource := &mockResource{id: "i-12345", name: "test-instance"}

	menu := NewActionMenu(ctx, resource, "test", "items")

	// Initially no active input
	if menu.HasActiveInput() {
		t.Error("Expected HasActiveInput() to be false initially")
	}

	// Enter dangerous confirm mode
	menu.dangerous.active = true

	// Now should have active input
	if !menu.HasActiveInput() {
		t.Error("Expected HasActiveInput() to be true when dangerousConfirm is active")
	}
}

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

// Mark/Unmark and Diff behavior tests

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

// TagSearchView tests

func TestFormatTags(t *testing.T) {
	tests := []struct {
		name   string
		tags   map[string]string
		maxLen int
		want   string
	}{
		{
			name:   "nil tags",
			tags:   nil,
			maxLen: 50,
			want:   "",
		},
		{
			name:   "single tag",
			tags:   map[string]string{"Env": "prod"},
			maxLen: 50,
			want:   "Env=prod",
		},
		{
			name:   "sorted output",
			tags:   map[string]string{"Z": "1", "A": "2"},
			maxLen: 50,
			want:   "A=2, Z=1",
		},
		{
			name:   "truncation",
			tags:   map[string]string{"VeryLongKey": "VeryLongValue"},
			maxLen: 10,
			want:   "VeryLongK…",
		},
		{
			name:   "control char sanitization",
			tags:   map[string]string{"Key": "val\x00ue\x1b[31m"},
			maxLen: 50,
			want:   "Key=value[31m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTags(tt.tags, tt.maxLen)
			if got != tt.want {
				t.Errorf("formatTags() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSanitizeTagValue(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"normal", "normal"},
		{"with\x00null", "withnull"},
		{"with\ttab", "withtab"},
		{"ansi\x1b[31mred", "ansi[31mred"},
		{"del\x7fchar", "delchar"},
		{"日本語", "日本語"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeTagValue(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeTagValue(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTagSearchView_parseTagFilters(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	tests := []struct {
		name       string
		tagFilter  string
		wantKey    string
		wantValues []string
		wantNil    bool
	}{
		{
			name:      "empty filter returns nil",
			tagFilter: "",
			wantNil:   true,
		},
		{
			name:       "key only",
			tagFilter:  "Environment",
			wantKey:    "Environment",
			wantValues: nil,
		},
		{
			name:       "key=value",
			tagFilter:  "Environment=production",
			wantKey:    "Environment",
			wantValues: []string{"production"},
		},
		{
			name:       "key with equals in value",
			tagFilter:  "Config=key=value",
			wantKey:    "Config",
			wantValues: []string{"key=value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewTagSearchView(ctx, reg, tt.tagFilter)
			filters := v.parseTagFilters()

			if tt.wantNil {
				if filters != nil {
					t.Errorf("parseTagFilters() = %v, want nil", filters)
				}
				return
			}

			if len(filters) != 1 {
				t.Fatalf("parseTagFilters() returned %d filters, want 1", len(filters))
			}

			filter := filters[0]
			if *filter.Key != tt.wantKey {
				t.Errorf("filter.Key = %q, want %q", *filter.Key, tt.wantKey)
			}

			if tt.wantValues == nil {
				if filter.Values != nil {
					t.Errorf("filter.Values = %v, want nil", filter.Values)
				}
			} else {
				if len(filter.Values) != len(tt.wantValues) {
					t.Errorf("filter.Values = %v, want %v", filter.Values, tt.wantValues)
				}
				for i, v := range tt.wantValues {
					if filter.Values[i] != v {
						t.Errorf("filter.Values[%d] = %q, want %q", i, filter.Values[i], v)
					}
				}
			}
		})
	}
}

func TestTagSearchView_applyFilter(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	v := NewTagSearchView(ctx, reg, "")
	v.resources = []taggedARN{
		{
			RawARN: "arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0",
			Region: "us-east-1",
			ARN:    &aws.ARN{Service: "ec2", ResourceType: "instance", ResourceID: "i-1234567890abcdef0"},
			Tags:   map[string]string{"Name": "web-server", "Environment": "production"},
		},
		{
			RawARN: "arn:aws:s3:::my-bucket",
			Region: "us-east-1",
			ARN:    &aws.ARN{Service: "s3", ResourceType: "", ResourceID: "my-bucket"},
			Tags:   map[string]string{"Name": "storage", "Environment": "development"},
		},
		{
			RawARN: "arn:aws:lambda:us-west-2:123456789012:function:my-func",
			Region: "us-west-2",
			ARN:    &aws.ARN{Service: "lambda", ResourceType: "function", ResourceID: "my-func"},
			Tags:   map[string]string{"Name": "processor"},
		},
	}

	tests := []struct {
		name       string
		filterText string
		wantCount  int
		wantIDs    []string
	}{
		{
			name:       "empty filter returns all",
			filterText: "",
			wantCount:  3,
		},
		{
			name:       "filter by service",
			filterText: "ec2",
			wantCount:  1,
			wantIDs:    []string{"i-1234567890abcdef0"},
		},
		{
			name:       "filter by region",
			filterText: "us-west-2",
			wantCount:  1,
			wantIDs:    []string{"my-func"},
		},
		{
			name:       "filter by tag value",
			filterText: "production",
			wantCount:  1,
			wantIDs:    []string{"i-1234567890abcdef0"},
		},
		{
			name:       "filter by tag key",
			filterText: "environment",
			wantCount:  2,
		},
		{
			name:       "filter by resource ID",
			filterText: "bucket",
			wantCount:  1,
			wantIDs:    []string{"my-bucket"},
		},
		{
			name:       "no match",
			filterText: "nonexistent",
			wantCount:  0,
		},
		{
			name:       "case insensitive",
			filterText: "LAMBDA",
			wantCount:  1,
			wantIDs:    []string{"my-func"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v.filterText = tt.filterText
			v.applyFilter()

			if len(v.filtered) != tt.wantCount {
				t.Errorf("applyFilter() filtered %d resources, want %d", len(v.filtered), tt.wantCount)
			}

			if tt.wantIDs != nil {
				for i, wantID := range tt.wantIDs {
					if i < len(v.filtered) && v.filtered[i].ARN.ResourceID != wantID {
						t.Errorf("filtered[%d].ResourceID = %q, want %q", i, v.filtered[i].ARN.ResourceID, wantID)
					}
				}
			}
		})
	}
}

// mockDiffProvider for testing getDiffSuggestions
type mockDiffProvider struct {
	names      []string
	markedName string
}

func (m *mockDiffProvider) GetResourceNames() []string {
	return m.names
}

func (m *mockDiffProvider) GetMarkedResourceName() string {
	return m.markedName
}

func TestCommandInput_getDiffSuggestions(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	tests := []struct {
		name     string
		provider *mockDiffProvider
		args     string
		want     []string
	}{
		{
			name:     "nil provider",
			provider: nil,
			args:     "",
			want:     nil,
		},
		{
			name:     "empty args returns all",
			provider: &mockDiffProvider{names: []string{"web-server", "db-server", "cache"}},
			args:     "",
			want:     []string{"diff web-server", "diff db-server", "diff cache"},
		},
		{
			name:     "first name prefix filter",
			provider: &mockDiffProvider{names: []string{"web-server", "db-server", "cache"}},
			args:     "server",
			want:     []string{"diff web-server", "diff db-server"},
		},
		{
			name:     "case insensitive match",
			provider: &mockDiffProvider{names: []string{"Web-Server", "DB-Server", "Cache"}},
			args:     "SERVER",
			want:     []string{"diff Web-Server", "diff DB-Server"},
		},
		{
			name:     "no match returns empty",
			provider: &mockDiffProvider{names: []string{"web-server", "db-server"}},
			args:     "xyz",
			want:     nil,
		},
		{
			name:     "second name completion excludes first",
			provider: &mockDiffProvider{names: []string{"web-server", "db-server", "cache"}},
			args:     "web-server ",
			want:     []string{"diff web-server db-server", "diff web-server cache"},
		},
		{
			name:     "second name with prefix",
			provider: &mockDiffProvider{names: []string{"web-server", "db-server", "cache"}},
			args:     "web-server db",
			want:     []string{"diff web-server db-server"},
		},
		{
			name:     "second name no match",
			provider: &mockDiffProvider{names: []string{"web-server", "db-server"}},
			args:     "web-server xyz",
			want:     nil,
		},
		{
			name:     "empty names list",
			provider: &mockDiffProvider{names: []string{}},
			args:     "",
			want:     nil,
		},
		{
			name:     "single resource for first",
			provider: &mockDiffProvider{names: []string{"only-one"}},
			args:     "",
			want:     []string{"diff only-one"},
		},
		{
			name:     "single resource for second - no suggestions",
			provider: &mockDiffProvider{names: []string{"only-one"}},
			args:     "only-one ",
			want:     nil, // can't diff with self
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ci := NewCommandInput(ctx, reg)
			if tt.provider != nil {
				ci.SetDiffProvider(tt.provider)
			}

			got := ci.getDiffSuggestions(tt.args)

			// Check length
			if len(got) != len(tt.want) {
				t.Errorf("getDiffSuggestions(%q) returned %d items, want %d\ngot:  %v\nwant: %v",
					tt.args, len(got), len(tt.want), got, tt.want)
				return
			}

			// Check each item
			for i, want := range tt.want {
				if got[i] != want {
					t.Errorf("getDiffSuggestions(%q)[%d] = %q, want %q", tt.args, i, got[i], want)
				}
			}
		})
	}
}
