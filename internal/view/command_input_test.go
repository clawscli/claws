package view

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/clawscli/claws/internal/registry"
)

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

func TestCommandInput_GetSuggestions_Aliases(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	reg.RegisterCustom("costexplorer", "costs", registry.Entry{})
	reg.RegisterCustom("cloudformation", "stacks", registry.Entry{})

	ci := NewCommandInput(ctx, reg)
	ci.Activate()

	tests := []struct {
		input    string
		expected []string
	}{
		{"cost", []string{"costexplorer", "cost-explorer"}},
		{"cf", []string{"cfn"}}, // "cf" excluded (exact match)
		{"cfn", []string{}},     // "cfn" excluded (exact match)
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			ci.textInput.SetValue(tt.input)
			suggestions := ci.GetSuggestions()

			for _, exp := range tt.expected {
				found := false
				for _, s := range suggestions {
					if s == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected %q in suggestions for %q, got %v", exp, tt.input, suggestions)
				}
			}
		})
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
	if nav != nil && nav.ClearStack {
		t.Error("Expected ClearStack=false (preserves navigation stack)")
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

func TestCommandInput_QuitCommand(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	tests := []struct {
		input    string
		wantQuit bool
	}{
		{"q", true},
		{"quit", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			ci := NewCommandInput(ctx, reg)
			ci.Activate()
			ci.textInput.SetValue(tt.input)

			cmd, nav := ci.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

			if cmd == nil {
				t.Error("Expected tea.Quit command")
			}
			if nav != nil {
				t.Error("Expected nil NavigateMsg for quit")
			}
		})
	}
}

// mockDiffProvider for testing getDiffSuggestions
type mockDiffProvider struct {
	ids      []string
	markedID string
}

func (m *mockDiffProvider) GetResourceIDs() []string {
	return m.ids
}

func (m *mockDiffProvider) GetMarkedResourceID() string {
	return m.markedID
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
			name:     "empty args returns all sorted",
			provider: &mockDiffProvider{ids: []string{"web-server", "db-server", "cache"}},
			args:     "",
			want:     []string{"diff cache", "diff db-server", "diff web-server"},
		},
		{
			name:     "prefix match",
			provider: &mockDiffProvider{ids: []string{"web-server", "db-server", "cache"}},
			args:     "web",
			want:     []string{"diff web-server"},
		},
		{
			name:     "prefix match multiple sorted",
			provider: &mockDiffProvider{ids: []string{"web-server", "web-api", "db-server"}},
			args:     "web",
			want:     []string{"diff web-api", "diff web-server"},
		},
		{
			name:     "fuzzy fallback when no prefix sorted",
			provider: &mockDiffProvider{ids: []string{"web-server", "db-server", "cache"}},
			args:     "server",
			want:     []string{"diff db-server", "diff web-server"},
		},
		{
			name:     "fuzzy match pattern",
			provider: &mockDiffProvider{ids: []string{"web-server", "db-server", "cache"}},
			args:     "wsr",
			want:     []string{"diff web-server"},
		},
		{
			name:     "case insensitive prefix",
			provider: &mockDiffProvider{ids: []string{"Web-Server", "DB-Server", "Cache"}},
			args:     "WEB",
			want:     []string{"diff Web-Server"},
		},
		{
			name:     "case insensitive fuzzy sorted",
			provider: &mockDiffProvider{ids: []string{"Web-Server", "DB-Server", "Cache"}},
			args:     "SERVER",
			want:     []string{"diff DB-Server", "diff Web-Server"},
		},
		{
			name:     "no match returns empty",
			provider: &mockDiffProvider{ids: []string{"web-server", "db-server"}},
			args:     "xyz",
			want:     nil,
		},
		{
			name:     "second name completion excludes first sorted",
			provider: &mockDiffProvider{ids: []string{"web-server", "db-server", "cache"}},
			args:     "web-server ",
			want:     []string{"diff web-server cache", "diff web-server db-server"},
		},
		{
			name:     "second name with prefix",
			provider: &mockDiffProvider{ids: []string{"web-server", "db-server", "cache"}},
			args:     "web-server db",
			want:     []string{"diff web-server db-server"},
		},
		{
			name:     "second name fuzzy fallback",
			provider: &mockDiffProvider{ids: []string{"web-server", "db-server", "cache"}},
			args:     "web-server sr",
			want:     []string{"diff web-server db-server"},
		},
		{
			name:     "second name no match",
			provider: &mockDiffProvider{ids: []string{"web-server", "db-server"}},
			args:     "web-server xyz",
			want:     nil,
		},
		{
			name:     "empty names list",
			provider: &mockDiffProvider{ids: []string{}},
			args:     "",
			want:     nil,
		},
		{
			name:     "single resource for first",
			provider: &mockDiffProvider{ids: []string{"only-one"}},
			args:     "",
			want:     []string{"diff only-one"},
		},
		{
			name:     "single resource for second - no suggestions",
			provider: &mockDiffProvider{ids: []string{"only-one"}},
			args:     "only-one ",
			want:     nil,
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

func TestCommandInput_DiffTabCompletion(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	ci := NewCommandInput(ctx, reg)
	ci.SetDiffProvider(&mockDiffProvider{ids: []string{"i-123", "i-456", "i-789"}})
	ci.Activate()

	// Type "diff "
	ci.textInput.SetValue("diff ")
	ci.updateSuggestions()

	// Verify suggestions are generated
	if len(ci.suggestions) != 3 {
		t.Fatalf("Expected 3 suggestions, got %d: %v", len(ci.suggestions), ci.suggestions)
	}

	// Verify suggestions have correct format
	expected := []string{"diff i-123", "diff i-456", "diff i-789"}
	for i, want := range expected {
		if ci.suggestions[i] != want {
			t.Errorf("suggestions[%d] = %q, want %q", i, ci.suggestions[i], want)
		}
	}

	// Press Tab - bash-style: first expand to common prefix "diff i-"
	ci.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	got := ci.textInput.Value()
	if got != "diff i-" {
		t.Errorf("After 1st Tab, textInput.Value() = %q, want %q (common prefix)", got, "diff i-")
	}

	// Press Tab again - now cycle to first suggestion
	ci.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	got = ci.textInput.Value()
	if got != "diff i-123" {
		t.Errorf("After 2nd Tab, textInput.Value() = %q, want %q", got, "diff i-123")
	}

	// Press Tab again - cycle to second suggestion
	ci.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	got = ci.textInput.Value()
	if got != "diff i-456" {
		t.Errorf("After 3rd Tab, textInput.Value() = %q, want %q", got, "diff i-456")
	}

	// Check View() contains "diff"
	view := ci.View()
	if !contains(view, "diff") {
		t.Errorf("View() should contain 'diff', got: %q", view)
	}
}

func TestCommandInput_DiffTabCompletion_RealKeyInput(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	ci := NewCommandInput(ctx, reg)
	ci.SetDiffProvider(&mockDiffProvider{ids: []string{"i-123", "i-456", "i-789"}})
	ci.Activate()

	// Type "diff " character by character (simulating real input)
	for _, r := range "diff " {
		ci.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}

	t.Logf("After typing 'diff ': Value=%q, suggestions=%v", ci.textInput.Value(), ci.suggestions)

	// Press Tab - bash-style: first expand to common prefix "diff i-"
	ci.Update(tea.KeyPressMsg{Code: tea.KeyTab})

	got := ci.textInput.Value()
	if got != "diff i-" {
		t.Errorf("After 1st Tab, textInput.Value() = %q, want %q (common prefix)", got, "diff i-")
	}

	// Press Tab again - now cycle to first suggestion
	ci.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	got = ci.textInput.Value()
	if got != "diff i-123" {
		t.Errorf("After 2nd Tab, textInput.Value() = %q, want %q", got, "diff i-123")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestCommandInput_ClearHistoryCommand(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	ci := NewCommandInput(ctx, reg)
	ci.Activate()
	ci.textInput.SetValue("clear-history")

	cmd, nav := ci.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	// Should return a command (not quit)
	if cmd == nil {
		t.Error("Expected command for clear-history")
	}

	// Should not return NavigateMsg
	if nav != nil {
		t.Error("Expected nil NavigateMsg for clear-history")
	}

	// Execute the command to verify it returns ClearHistoryMsg
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(ClearHistoryMsg); !ok {
			t.Errorf("Expected ClearHistoryMsg, got %T", msg)
		}
	}
}

func TestCommandInput_DashboardCommand(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	tests := []struct {
		input          string
		wantNavigate   bool
		wantClearStack bool
	}{
		{"pulse", true, false},
		{"dashboard", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			ci := NewCommandInput(ctx, reg)
			ci.Activate()
			ci.textInput.SetValue(tt.input)

			_, nav := ci.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

			if tt.wantNavigate && nav == nil {
				t.Errorf("Expected NavigateMsg for %q", tt.input)
			}
			if nav != nil && nav.ClearStack != tt.wantClearStack {
				t.Errorf("%q: ClearStack = %v, want %v", tt.input, nav.ClearStack, tt.wantClearStack)
			}
		})
	}
}

func TestCommandInput_ServicesCommand(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	tests := []string{"services", "browse", "home", ""}

	for _, input := range tests {
		t.Run("input="+input, func(t *testing.T) {
			ci := NewCommandInput(ctx, reg)
			ci.Activate()
			ci.textInput.SetValue(input)

			_, nav := ci.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

			if nav == nil {
				t.Errorf("Expected NavigateMsg for %q", input)
			}
			if nav != nil && nav.ClearStack {
				t.Errorf("%q: ClearStack = true, want false (preserves stack)", input)
			}
		})
	}
}

func TestCommandInput_CtrlCExit(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	ci := NewCommandInput(ctx, reg)
	ci.Activate()

	if !ci.IsActive() {
		t.Fatal("Expected command input to be active")
	}

	// Press Ctrl+C
	ci.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})

	if ci.IsActive() {
		t.Error("Expected Ctrl+C to deactivate command input")
	}
}

func TestCommandInput_AliasResolutionInView(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	ci := NewCommandInput(ctx, reg)
	ci.Activate()

	// "sq" is an alias for "service-quotas"
	ci.textInput.SetValue("sq")
	ci.updateSuggestions()

	view := ci.View()

	// View should contain the resolved alias
	if !contains(view, "service-quotas") {
		t.Errorf("View should contain resolved alias 'service-quotas', got: %q", view)
	}
}

func TestCommandInput_DynamicWidth(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	ci := NewCommandInput(ctx, reg)
	ci.Activate()

	// Short input - default width
	ci.textInput.SetValue("ec2")
	ci.Update(tea.KeyPressMsg{Code: '2', Text: "2"})

	// Long input - expanded width
	longInput := "diff i-0123456789abcdef0 i-fedcba9876543210"
	ci.textInput.SetValue(longInput)
	ci.Update(tea.KeyPressMsg{Code: '0', Text: "0"})

	// Just verify no panic with long input
	view := ci.View()
	if view == "" {
		t.Error("Expected non-empty view for long input")
	}
}

func TestCommonPrefix(t *testing.T) {
	tests := []struct {
		name        string
		suggestions []string
		want        string
	}{
		{"empty", []string{}, ""},
		{"single", []string{"ec2"}, "ec2"},
		{"exact match", []string{"ec2", "ec2"}, "ec2"},
		{"common prefix", []string{"saaa", "saab", "saba"}, "sa"},
		{"no common", []string{"abc", "xyz"}, ""},
		{"full prefix", []string{"ec2", "ec2/instances"}, "ec2"},
		{"different lengths", []string{"cloudformation", "cloudfront", "cloudwatch"}, "cloud"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := commonPrefix(tt.suggestions)
			if got != tt.want {
				t.Errorf("commonPrefix(%v) = %q, want %q", tt.suggestions, got, tt.want)
			}
		})
	}
}

func TestCommandInput_BashStyleTabCompletion(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	ci := NewCommandInput(ctx, reg)
	ci.Activate()

	// Type "cloud" - multiple matches with common prefix "cloud"
	ci.textInput.SetValue("cloud")
	ci.updateSuggestions()

	// Should have multiple suggestions starting with "cloud"
	if len(ci.suggestions) < 2 {
		t.Skipf("Need multiple cloud* services for this test, got %d", len(ci.suggestions))
	}

	// First Tab: should expand to common prefix (might be "cloud" itself if that's the max)
	ci.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	afterFirstTab := ci.textInput.Value()

	// Common prefix should be >= original input
	if len(afterFirstTab) < len("cloud") {
		t.Errorf("After first Tab, value %q is shorter than input 'cloud'", afterFirstTab)
	}

	// If common prefix == input, second Tab should cycle to first suggestion
	if afterFirstTab == "cloud" {
		ci.Update(tea.KeyPressMsg{Code: tea.KeyTab})
		afterSecondTab := ci.textInput.Value()
		if afterSecondTab == "cloud" {
			t.Errorf("After second Tab, value should have cycled to a suggestion")
		}
	}
}

func TestCommandInput_TabCompletionSingleMatch(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	ci := NewCommandInput(ctx, reg)
	ci.Activate()

	// Type something that matches only one service
	ci.textInput.SetValue("bedroc")
	ci.updateSuggestions()

	if len(ci.suggestions) != 1 {
		t.Skipf("Expected exactly 1 suggestion for 'bedroc', got %d: %v", len(ci.suggestions), ci.suggestions)
	}

	// Tab should complete directly to the single match
	ci.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	got := ci.textInput.Value()
	if got != "bedrock" {
		t.Errorf("After Tab with single match, got %q, want 'bedrock'", got)
	}
}

func TestCommandInput_ResourcePrefixMatching(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	// Register ec2 with multiple resources
	reg.RegisterCustom("ec2", "instances", registry.Entry{})
	reg.RegisterCustom("ec2", "images", registry.Entry{})
	reg.RegisterCustom("ec2", "volumes", registry.Entry{})

	ci := NewCommandInput(ctx, reg)
	ci.Activate()

	tests := []struct {
		input    string
		wantDest string
	}{
		{"ec2/in", "ec2/instances"},        // prefix match "in" -> "instances"
		{"ec2/im", "ec2/images"},           // prefix match "im" -> "images"
		{"ec2/vol", "ec2/volumes"},         // prefix match "vol" -> "volumes"
		{"ec2/instances", "ec2/instances"}, // exact match
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ci.resolveDestination(tt.input)
			if got != tt.wantDest {
				t.Errorf("resolveDestination(%q) = %q, want %q", tt.input, got, tt.wantDest)
			}
		})
	}
}

func TestCommandInput_AliasResourcePreservation(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	// Register cloudwatch with log-groups (the "logs" alias points here)
	reg.RegisterCustom("cloudwatch", "log-groups", registry.Entry{})
	reg.RegisterCustom("cloudwatch", "metrics", registry.Entry{})

	ci := NewCommandInput(ctx, reg)
	ci.Activate()

	// "logs" alias resolves to "cloudwatch/log-groups"
	dest := ci.resolveDestination("logs")
	if dest != "cloudwatch/log-groups" {
		t.Errorf("resolveDestination('logs') = %q, want 'cloudwatch/log-groups'", dest)
	}

	// Prefix match on alias should also work
	dest = ci.resolveDestination("log")
	if dest != "cloudwatch/log-groups" {
		t.Errorf("resolveDestination('log') = %q, want 'cloudwatch/log-groups'", dest)
	}
}
