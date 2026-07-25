package clipboard

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func withClipboardReader(t *testing.T, nativeReader func() (string, error)) {
	t.Helper()
	originalNativeRead := nativeClipboardRead
	nativeClipboardRead = nativeReader
	t.Cleanup(func() {
		nativeClipboardRead = originalNativeRead
	})
}

type failingStringWriter struct{}

func (failingStringWriter) WriteString(string) (int, error) {
	return 0, errors.New("terminal clipboard unavailable")
}

func withClipboardWriters(t *testing.T, terminalWriter interface{ WriteString(string) (int, error) }, nativeWriter func(string) error) {
	t.Helper()
	originalTerminalWriter := terminalClipboardWriter
	originalNativeWrite := nativeClipboardWrite
	terminalClipboardWriter = terminalWriter
	nativeClipboardWrite = nativeWriter
	t.Cleanup(func() {
		terminalClipboardWriter = originalTerminalWriter
		nativeClipboardWrite = originalNativeWrite
	})
}

func TestCopiedMsg(t *testing.T) {
	msg := CopiedMsg{Label: "ID", Value: "i-1234567890abcdef0"}
	if msg.Label != "ID" {
		t.Errorf("expected Label 'ID', got %q", msg.Label)
	}
	if msg.Value != "i-1234567890abcdef0" {
		t.Errorf("expected Value 'i-1234567890abcdef0', got %q", msg.Value)
	}
}

func TestCopy(t *testing.T) {
	var terminal strings.Builder
	withClipboardWriters(t, &terminal, func(string) error { return nil })

	cmd := Copy("TestLabel", "TestValue")
	if cmd == nil {
		t.Fatal("Copy should return a non-nil command")
	}

	msg := cmd()
	copiedMsg, ok := msg.(CopiedMsg)
	if !ok {
		t.Fatalf("expected CopiedMsg, got %T", msg)
	}
	if copiedMsg.Label != "TestLabel" {
		t.Errorf("expected Label 'TestLabel', got %q", copiedMsg.Label)
	}
	if copiedMsg.Value != "TestValue" {
		t.Errorf("expected Value 'TestValue', got %q", copiedMsg.Value)
	}
}

func TestCopyReturnsCopiedWhenNativeClipboardFailsButOSC52Succeeds(t *testing.T) {
	var terminal strings.Builder
	withClipboardWriters(t, &terminal, func(string) error { return errors.New("native clipboard unavailable") })

	msg := Copy("ID", "i-123")()
	if _, ok := msg.(CopiedMsg); !ok {
		t.Fatalf("expected CopiedMsg when OSC52 succeeds, got %T", msg)
	}
	if terminal.String() == "" {
		t.Fatal("expected OSC52 data to be written")
	}
}

func TestCopyReturnsFailureWhenAllClipboardWritesFail(t *testing.T) {
	withClipboardWriters(t, failingStringWriter{}, func(string) error { return errors.New("native clipboard unavailable") })

	msg := Copy("ID", "i-123")()
	failedMsg, ok := msg.(CopyFailedMsg)
	if !ok {
		t.Fatalf("expected CopyFailedMsg, got %T", msg)
	}
	if failedMsg.Label != "ID" || failedMsg.Value != "i-123" {
		t.Fatalf("failure msg = %+v, want label/value preserved", failedMsg)
	}
	if failedMsg.Err == nil {
		t.Fatal("expected failure error")
	}
}

func TestCopyID(t *testing.T) {
	var terminal strings.Builder
	withClipboardWriters(t, &terminal, func(string) error { return nil })

	cmd := CopyID("i-1234567890abcdef0")
	if cmd == nil {
		t.Fatal("CopyID should return a non-nil command")
	}

	msg := cmd()
	copiedMsg, ok := msg.(CopiedMsg)
	if !ok {
		t.Fatalf("expected CopiedMsg, got %T", msg)
	}
	if copiedMsg.Label != "ID" {
		t.Errorf("expected Label 'ID', got %q", copiedMsg.Label)
	}
	if copiedMsg.Value != "i-1234567890abcdef0" {
		t.Errorf("expected Value 'i-1234567890abcdef0', got %q", copiedMsg.Value)
	}
}

func TestCopyARN(t *testing.T) {
	var terminal strings.Builder
	withClipboardWriters(t, &terminal, func(string) error { return nil })

	arn := "arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0"
	cmd := CopyARN(arn)
	if cmd == nil {
		t.Fatal("CopyARN should return a non-nil command")
	}

	msg := cmd()
	copiedMsg, ok := msg.(CopiedMsg)
	if !ok {
		t.Fatalf("expected CopiedMsg, got %T", msg)
	}
	if copiedMsg.Label != "ARN" {
		t.Errorf("expected Label 'ARN', got %q", copiedMsg.Label)
	}
	if copiedMsg.Value != arn {
		t.Errorf("expected Value %q, got %q", arn, copiedMsg.Value)
	}
}

func TestPasteReturnsPasteMsg(t *testing.T) {
	withClipboardReader(t, func() (string, error) { return "i-1234567890abcdef0", nil })

	cmd := Paste()
	if cmd == nil {
		t.Fatal("Paste should return a non-nil command")
	}

	msg := cmd()
	pasteMsg, ok := msg.(tea.PasteMsg)
	if !ok {
		t.Fatalf("expected tea.PasteMsg, got %T", msg)
	}
	if pasteMsg.Content != "i-1234567890abcdef0" {
		t.Errorf("expected Content 'i-1234567890abcdef0', got %q", pasteMsg.Content)
	}
}

func TestPasteReturnsNilWhenClipboardUnavailable(t *testing.T) {
	withClipboardReader(t, func() (string, error) { return "", errors.New("native clipboard unavailable") })

	msg := Paste()()
	if msg != nil {
		t.Fatalf("expected nil msg when clipboard read fails, got %T", msg)
	}
}

func TestNoARN(t *testing.T) {
	cmd := NoARN()
	if cmd == nil {
		t.Fatal("NoARN should return a non-nil command")
	}

	msg := cmd()
	if _, ok := msg.(NoARNMsg); !ok {
		t.Errorf("expected NoARNMsg, got %T", msg)
	}
}
