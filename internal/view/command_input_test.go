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
		{"cf", []string{"cf", "cfn"}},
		{"cfn", []string{"cfn"}},
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

func TestCommandInput_SetWidth(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	ci := NewCommandInput(ctx, reg)
	ci.SetWidth(100)
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
			name:     "empty args returns all sorted",
			provider: &mockDiffProvider{names: []string{"web-server", "db-server", "cache"}},
			args:     "",
			want:     []string{"diff cache", "diff db-server", "diff web-server"},
		},
		{
			name:     "prefix match",
			provider: &mockDiffProvider{names: []string{"web-server", "db-server", "cache"}},
			args:     "web",
			want:     []string{"diff web-server"},
		},
		{
			name:     "prefix match multiple sorted",
			provider: &mockDiffProvider{names: []string{"web-server", "web-api", "db-server"}},
			args:     "web",
			want:     []string{"diff web-api", "diff web-server"},
		},
		{
			name:     "fuzzy fallback when no prefix sorted",
			provider: &mockDiffProvider{names: []string{"web-server", "db-server", "cache"}},
			args:     "server",
			want:     []string{"diff db-server", "diff web-server"},
		},
		{
			name:     "fuzzy match pattern",
			provider: &mockDiffProvider{names: []string{"web-server", "db-server", "cache"}},
			args:     "wsr",
			want:     []string{"diff web-server"},
		},
		{
			name:     "case insensitive prefix",
			provider: &mockDiffProvider{names: []string{"Web-Server", "DB-Server", "Cache"}},
			args:     "WEB",
			want:     []string{"diff Web-Server"},
		},
		{
			name:     "case insensitive fuzzy sorted",
			provider: &mockDiffProvider{names: []string{"Web-Server", "DB-Server", "Cache"}},
			args:     "SERVER",
			want:     []string{"diff DB-Server", "diff Web-Server"},
		},
		{
			name:     "no match returns empty",
			provider: &mockDiffProvider{names: []string{"web-server", "db-server"}},
			args:     "xyz",
			want:     nil,
		},
		{
			name:     "second name completion excludes first sorted",
			provider: &mockDiffProvider{names: []string{"web-server", "db-server", "cache"}},
			args:     "web-server ",
			want:     []string{"diff web-server cache", "diff web-server db-server"},
		},
		{
			name:     "second name with prefix",
			provider: &mockDiffProvider{names: []string{"web-server", "db-server", "cache"}},
			args:     "web-server db",
			want:     []string{"diff web-server db-server"},
		},
		{
			name:     "second name fuzzy fallback",
			provider: &mockDiffProvider{names: []string{"web-server", "db-server", "cache"}},
			args:     "web-server sr",
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
