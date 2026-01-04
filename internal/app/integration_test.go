package app

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/clawscli/claws/internal/registry"
)

// TestEscKeyIntegration tests the actual esc key handling in a real bubbletea program
func TestEscKeyIntegration(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	app := New(ctx, reg, nil)

	// Create a program with custom input/output
	var out bytes.Buffer
	in := bytes.NewReader(nil)

	p := tea.NewProgram(app,
		tea.WithInput(in),
		tea.WithOutput(&out),
	)

	// Run Init
	initCmd := app.Init()
	if initCmd != nil {
		// Execute init command
		msg := initCmd()
		app.Update(msg)
	}

	// Set size
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 50})

	// Clear any warnings (AWS init may fail in CI without credentials)
	app.showWarnings = false

	t.Logf("Initial state: currentView=%T, viewStack=%d", app.currentView, len(app.viewStack))

	// Verify we start with ServiceBrowser and empty stack
	if len(app.viewStack) != 0 {
		t.Errorf("Expected empty viewStack, got %d", len(app.viewStack))
	}

	// Simulate pressing enter on a service (which would navigate to ResourceBrowser)
	// But we need to test esc handling, so let's manually set up the state

	// Create mock views for testing
	serviceBrowser := app.currentView
	resourceBrowser := &MockView{name: "ResourceBrowser"}
	detailView := &MockView{name: "DetailView"}

	// Simulate navigation: ServiceBrowser -> ResourceBrowser
	app.viewStack = append(app.viewStack, serviceBrowser)
	app.currentView = resourceBrowser
	t.Logf("After nav to ResourceBrowser: currentView=%s, viewStack=%d", app.currentView.StatusLine(), len(app.viewStack))

	// Simulate navigation: ResourceBrowser -> DetailView
	app.viewStack = append(app.viewStack, resourceBrowser)
	app.currentView = detailView
	t.Logf("After nav to DetailView: currentView=%s, viewStack=%d", app.currentView.StatusLine(), len(app.viewStack))

	if len(app.viewStack) != 2 {
		t.Errorf("Expected viewStack=2, got %d", len(app.viewStack))
	}

	// Now test esc key handling
	// Create esc key message
	escMsg := tea.KeyPressMsg{Code: tea.KeyEscape}
	t.Logf("Sending esc: msg.String()=%q, msg.Code=%d", escMsg.String(), escMsg.Code)

	// Send esc
	app.Update(escMsg)
	t.Logf("After 1st esc: currentView=%s, viewStack=%d",
		app.currentView.StatusLine(), len(app.viewStack))

	if app.currentView.StatusLine() != "ResourceBrowser" {
		t.Errorf("After 1st esc: Expected ResourceBrowser, got %s", app.currentView.StatusLine())
	}
	if len(app.viewStack) != 1 {
		t.Errorf("After 1st esc: Expected viewStack=1, got %d", len(app.viewStack))
	}

	// Send esc again
	app.Update(escMsg)
	t.Logf("After 2nd esc: currentView=%T, viewStack=%d",
		app.currentView, len(app.viewStack))

	if len(app.viewStack) != 0 {
		t.Errorf("After 2nd esc: Expected viewStack=0, got %d", len(app.viewStack))
	}

	_ = p // Silence unused variable warning
}

// TestEscRawBytes tests esc handling with raw escape byte
func TestEscRawBytes(t *testing.T) {
	// Test what tea.KeyPressMsg looks like for different escape inputs
	tests := []struct {
		name string
		msg  tea.KeyPressMsg
	}{
		{"KeyEscape", tea.KeyPressMsg{Code: tea.KeyEscape}},
		{"Runes27", tea.KeyPressMsg{Code: 27}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str := tt.msg.String()
			t.Logf("%s: String()=%q, Code=%d", tt.name, str, tt.msg.Code)
		})
	}
}

// TestRawEscapeByteHandling tests that raw escape byte (0x1b) is handled correctly
func TestRawEscapeByteHandling(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	app := New(ctx, reg, nil)
	app.width = 100
	app.height = 50

	// Set up view stack using MockView (which implements view.View)
	serviceBrowser := &MockView{name: "ServiceBrowser"}
	resourceBrowser := &MockView{name: "ResourceBrowser"}
	detailView := &MockView{name: "DetailView"}

	app.viewStack = append(app.viewStack, serviceBrowser, resourceBrowser)
	app.currentView = detailView

	// Send raw escape byte (ESC key code = 27)
	rawEscMsg := tea.KeyPressMsg{Code: 27}
	t.Logf("Sending raw escape: msg.String()=%q, msg.Code=%d", rawEscMsg.String(), rawEscMsg.Code)
	t.Logf("Before: currentView=%s, viewStack=%d", app.currentView.StatusLine(), len(app.viewStack))

	app.Update(rawEscMsg)

	t.Logf("After: currentView=%s, viewStack=%d",
		app.currentView.StatusLine(), len(app.viewStack))

	// Should have popped to ResourceBrowser
	if app.currentView.StatusLine() != "ResourceBrowser" {
		t.Errorf("Expected ResourceBrowser, got %s", app.currentView.StatusLine())
	}
	if len(app.viewStack) != 1 {
		t.Errorf("Expected viewStack=1, got %d", len(app.viewStack))
	}
}

// TestActualTeaProgram runs a real tea program and sends keys
func TestActualTeaProgram(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	app := New(ctx, reg, nil)

	// Use a pipe for input
	pr, pw := io.Pipe()

	var out bytes.Buffer

	p := tea.NewProgram(app,
		tea.WithInput(pr),
		tea.WithOutput(&out),
	)

	// Run program in background
	done := make(chan struct{})
	go func() {
		_, _ = p.Run()
		close(done)
	}()

	// Give it time to start
	time.Sleep(100 * time.Millisecond)

	// Send escape key (raw byte 0x1b)
	t.Log("Sending raw escape byte")
	_, _ = pw.Write([]byte{0x1b})

	// Give it time to process
	time.Sleep(100 * time.Millisecond)

	// Send quit
	t.Log("Sending quit")
	_, _ = pw.Write([]byte{'q'})

	// Wait for program to finish
	select {
	case <-done:
		t.Log("Program finished")
	case <-time.After(2 * time.Second):
		t.Log("Timeout waiting for program")
		p.Quit()
	}

	_ = pw.Close()
	_ = pr.Close()

	t.Logf("Output length: %d bytes", out.Len())
}
