package view

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

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

	if dv.vp.Ready {
		t.Error("Expected vp.Ready to be false initially")
	}

	dv.SetSize(100, 50)

	if !dv.vp.Ready {
		t.Error("Expected vp.Ready to be true after SetSize")
	}
	if dv.width != 100 {
		t.Errorf("width = %d, want 100", dv.width)
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
	if view != LoadingMessage {
		t.Errorf("ViewString() = %q, want %q", view, LoadingMessage)
	}
}
