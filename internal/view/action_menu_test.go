package view

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/clawscli/claws/internal/action"
)

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
