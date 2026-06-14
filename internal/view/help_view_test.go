package view

import (
	"strings"
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

func TestHelpView_TagaddHelp(t *testing.T) {
	hv := NewHelpView()
	content := hv.renderContent()

	for _, want := range []string{":tagadd key=val", "Append an AND tag filter", ":tagadd Role=web"} {
		if !strings.Contains(content, want) {
			t.Fatalf("help content should contain %q", want)
		}
	}
}
