package view

import (
	"context"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
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
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
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
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}

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
	browser.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if browser.cursor != 1 {
		t.Errorf("After 'l', cursor = %d, want 1", browser.cursor)
	}

	// Test navigation with 'h' (left)
	browser.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
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
	browser.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !browser.filterActive {
		t.Error("Expected filter to be active after '/'")
	}

	// Type 'ec2' in filter
	for _, r := range "ec2" {
		browser.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Should have fewer items after filtering
	if len(browser.flatItems) >= initialCount {
		t.Errorf("Expected fewer items after filter, got %d (was %d)", len(browser.flatItems), initialCount)
	}

	// Press Esc to exit filter mode
	browser.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if browser.filterActive {
		t.Error("Expected filter to be inactive after Esc")
	}

	// Press 'c' to clear filter
	browser.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
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
	browser.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

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
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
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
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
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
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	_, nav := ci.Update(enterMsg)

	if nav == nil {
		t.Error("Expected NavigateMsg for 'ec2'")
	}
}

// IsEscKey tests

func TestIsEscKey(t *testing.T) {
	tests := []struct {
		name string
		msg  tea.KeyMsg
		want bool
	}{
		{"KeyEsc", tea.KeyMsg{Type: tea.KeyEsc}, true},
		{"KeyEscape", tea.KeyMsg{Type: tea.KeyEscape}, true},
		{"raw ESC byte", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{27}}, true},
		{"Enter", tea.KeyMsg{Type: tea.KeyEnter}, false},
		{"Space", tea.KeyMsg{Type: tea.KeySpace}, false},
		{"letter a", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}, false},
		{"letter q", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}, false},
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
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
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
	qMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
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
	view := dv.View()
	if view != "Loading..." {
		t.Errorf("View() = %q, want 'Loading...'", view)
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
