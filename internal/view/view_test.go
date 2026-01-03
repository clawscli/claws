package view

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// mockResource for testing - shared across test files
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

// mockRenderer for testing - shared across test files
type mockRenderer struct {
	detail string
}

func (m *mockRenderer) ServiceName() string                                     { return "test" }
func (m *mockRenderer) ResourceType() string                                    { return "items" }
func (m *mockRenderer) Columns() []render.Column                                { return nil }
func (m *mockRenderer) RenderRow(r dao.Resource, cols []render.Column) []string { return nil }
func (m *mockRenderer) RenderDetail(r dao.Resource) string                      { return m.detail }
func (m *mockRenderer) RenderSummary(r dao.Resource) []render.SummaryField      { return nil }

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

// TruncateOrPadString tests

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
			got := TruncateOrPadString(tt.input, tt.width)

			// Check visual width (rune count for plain text with ellipsis)
			gotLen := len([]rune(got))
			if tt.wantLen > 0 && gotLen != tt.wantLen {
				t.Errorf("TruncateOrPadString(%q, %d) rune len = %d, want %d (got=%q)", tt.input, tt.width, gotLen, tt.wantLen, got)
			}

			if tt.wantEnd != "" && !strings.HasSuffix(got, tt.wantEnd) {
				t.Errorf("TruncateOrPadString(%q, %d) = %q, want suffix %q", tt.input, tt.width, got, tt.wantEnd)
			}
		})
	}
}
