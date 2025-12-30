package view

import (
	"testing"
)

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
