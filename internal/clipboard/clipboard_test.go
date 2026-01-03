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
		t.Error("Copy should return a non-nil command")
	}
}

func TestCopyID(t *testing.T) {
	cmd := CopyID("i-1234567890abcdef0")
	if cmd == nil {
		t.Error("CopyID should return a non-nil command")
	}
}

func TestCopyARN(t *testing.T) {
	cmd := CopyARN("arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0")
	if cmd == nil {
		t.Error("CopyARN should return a non-nil command")
	}
}

func TestNoARN(t *testing.T) {
	cmd := NoARN()
	if cmd == nil {
		t.Error("NoARN should return a non-nil command")
	}
	msg := cmd()
	if _, ok := msg.(NoARNMsg); !ok {
		t.Errorf("expected NoARNMsg, got %T", msg)
	}
}
