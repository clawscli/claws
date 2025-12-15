package view

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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

	dv := NewDetailView(ctx, resource, nil, "ec2", "instances", nil)
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

	dv := NewDetailView(ctx, resource, nil, "ec2", "instances", nil)
	dv.SetSize(100, 50)

	// Test that "esc" string is correctly identified
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	t.Logf("Esc key string: %q", escMsg.String())

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
