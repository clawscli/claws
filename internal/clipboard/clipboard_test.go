package clipboard

import (
	"testing"
)

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

func TestCopyID(t *testing.T) {
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
