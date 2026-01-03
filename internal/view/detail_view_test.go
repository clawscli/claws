package view

import (
	"context"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

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
			wantContains:    []string{LoadingMessage},
			wantNotContains: []string{render.NotConfigured},
		},
		{
			name:            "refreshing replaces Empty at line end",
			detail:          "Items: " + render.Empty + "\n",
			refreshing:      true,
			wantContains:    []string{LoadingMessage},
			wantNotContains: []string{render.Empty},
		},
		{
			name:            "refreshing replaces NoValue at line end",
			detail:          "Comment: " + render.NoValue + "\n",
			refreshing:      true,
			wantContains:    []string{LoadingMessage},
			wantNotContains: []string{render.NoValue},
		},
		{
			name:            "refreshing replaces placeholder at EOF without newline",
			detail:          "Status: " + render.NotConfigured,
			refreshing:      true,
			wantContains:    []string{LoadingMessage},
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
			wantContains:    []string{LoadingMessage},
			wantNotContains: []string{render.NotConfigured, render.Empty, render.NoValue},
		},
		{
			name:            "refreshing replaces multiple same placeholders",
			detail:          "Status: " + render.NotConfigured + "\nEncryption: " + render.NotConfigured + "\n",
			refreshing:      true,
			wantContains:    []string{LoadingMessage},
			wantNotContains: []string{render.NotConfigured},
		},
		{
			name:            "refreshing replaces consecutive placeholders",
			detail:          "Status: " + render.NotConfigured + "\n" + render.NoValue + "\n",
			refreshing:      true,
			wantContains:    []string{LoadingMessage},
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

			content := dv.vp.Model.View()

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
