package app

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/view"
)

// MockView is a simple view for testing
type MockView struct {
	name        string
	hasInput    bool
	escReceived bool
}

func (m *MockView) Init() tea.Cmd                     { return nil }
func (m *MockView) View() string                      { return m.name }
func (m *MockView) SetSize(width, height int) tea.Cmd { return nil }
func (m *MockView) StatusLine() string                { return m.name }
func (m *MockView) HasActiveInput() bool              { return m.hasInput }
func (m *MockView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "esc" {
		m.escReceived = true
		m.hasInput = false // Close input on esc
	}
	return m, nil
}

func TestEscInDetailView(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	app := New(ctx, reg)
	app.width = 100
	app.height = 50

	// Set up view stack: ServiceBrowser -> ResourceBrowser -> DetailView
	serviceBrowser := &MockView{name: "ServiceBrowser"}
	resourceBrowser := &MockView{name: "ResourceBrowser"}
	detailView := &MockView{name: "DetailView"}

	app.viewStack = []view.View{serviceBrowser, resourceBrowser}
	app.currentView = detailView

	// Press esc
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	t.Logf("Before esc: currentView=%s, viewStack=%d", app.currentView.StatusLine(), len(app.viewStack))

	app.Update(escMsg)

	t.Logf("After esc: currentView=%s, viewStack=%d", app.currentView.StatusLine(), len(app.viewStack))

	// Should now be on ResourceBrowser
	if app.currentView.StatusLine() != "ResourceBrowser" {
		t.Errorf("Expected currentView to be ResourceBrowser, got %s", app.currentView.StatusLine())
	}

	// ViewStack should have 1 item (ServiceBrowser)
	if len(app.viewStack) != 1 {
		t.Errorf("Expected viewStack length 1, got %d", len(app.viewStack))
	}

	// Press esc again
	app.Update(escMsg)

	t.Logf("After 2nd esc: currentView=%s, viewStack=%d", app.currentView.StatusLine(), len(app.viewStack))

	// Should now be on ServiceBrowser
	if app.currentView.StatusLine() != "ServiceBrowser" {
		t.Errorf("Expected currentView to be ServiceBrowser, got %s", app.currentView.StatusLine())
	}

	// ViewStack should be empty
	if len(app.viewStack) != 0 {
		t.Errorf("Expected viewStack length 0, got %d", len(app.viewStack))
	}
}

func TestEscInFilterMode(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	app := New(ctx, reg)
	app.width = 100
	app.height = 50

	// Set up view with active filter
	resourceBrowser := &MockView{name: "ResourceBrowser", hasInput: true}
	serviceBrowser := &MockView{name: "ServiceBrowser"}

	app.viewStack = []view.View{serviceBrowser}
	app.currentView = resourceBrowser

	// Press esc - should close filter, NOT go back
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	t.Logf("Before esc: currentView=%s, hasInput=%v, viewStack=%d",
		app.currentView.StatusLine(), resourceBrowser.hasInput, len(app.viewStack))

	app.Update(escMsg)

	t.Logf("After esc: currentView=%s, escReceived=%v, hasInput=%v, viewStack=%d",
		app.currentView.StatusLine(), resourceBrowser.escReceived, resourceBrowser.hasInput, len(app.viewStack))

	// Should still be on ResourceBrowser
	if app.currentView.StatusLine() != "ResourceBrowser" {
		t.Errorf("Expected currentView to be ResourceBrowser, got %s", app.currentView.StatusLine())
	}

	// Esc should have been received by the view
	if !resourceBrowser.escReceived {
		t.Errorf("Expected escReceived to be true")
	}

	// hasInput should now be false
	if resourceBrowser.hasInput {
		t.Errorf("Expected hasInput to be false after esc")
	}

	// ViewStack should still have 1 item
	if len(app.viewStack) != 1 {
		t.Errorf("Expected viewStack length 1, got %d", len(app.viewStack))
	}
}

func TestNavigationFlow(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	app := New(ctx, reg)
	app.width = 100
	app.height = 50

	// Start with ServiceBrowser
	serviceBrowser := &MockView{name: "ServiceBrowser"}
	app.currentView = serviceBrowser
	app.viewStack = nil

	t.Logf("Initial: currentView=%s, viewStack=%d", app.currentView.StatusLine(), len(app.viewStack))

	// Simulate navigating to ResourceBrowser
	resourceBrowser := &MockView{name: "ResourceBrowser"}
	navMsg1 := view.NavigateMsg{View: resourceBrowser}
	app.Update(navMsg1)

	t.Logf("After nav to ResourceBrowser: currentView=%s, viewStack=%d",
		app.currentView.StatusLine(), len(app.viewStack))

	if app.currentView.StatusLine() != "ResourceBrowser" {
		t.Errorf("Expected currentView to be ResourceBrowser, got %s", app.currentView.StatusLine())
	}
	if len(app.viewStack) != 1 {
		t.Errorf("Expected viewStack length 1, got %d", len(app.viewStack))
	}

	// Simulate navigating to DetailView
	detailView := &MockView{name: "DetailView"}
	navMsg2 := view.NavigateMsg{View: detailView}
	app.Update(navMsg2)

	t.Logf("After nav to DetailView: currentView=%s, viewStack=%d",
		app.currentView.StatusLine(), len(app.viewStack))

	if app.currentView.StatusLine() != "DetailView" {
		t.Errorf("Expected currentView to be DetailView, got %s", app.currentView.StatusLine())
	}
	if len(app.viewStack) != 2 {
		t.Errorf("Expected viewStack length 2, got %d", len(app.viewStack))
	}

	// Press esc - should go back to ResourceBrowser
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	app.Update(escMsg)

	t.Logf("After 1st esc: currentView=%s, viewStack=%d",
		app.currentView.StatusLine(), len(app.viewStack))

	if app.currentView.StatusLine() != "ResourceBrowser" {
		t.Errorf("Expected currentView to be ResourceBrowser, got %s", app.currentView.StatusLine())
	}
	if len(app.viewStack) != 1 {
		t.Errorf("Expected viewStack length 1, got %d", len(app.viewStack))
	}

	// Press esc again - should go back to ServiceBrowser
	app.Update(escMsg)

	t.Logf("After 2nd esc: currentView=%s, viewStack=%d",
		app.currentView.StatusLine(), len(app.viewStack))

	if app.currentView.StatusLine() != "ServiceBrowser" {
		t.Errorf("Expected currentView to be ServiceBrowser, got %s", app.currentView.StatusLine())
	}
	if len(app.viewStack) != 0 {
		t.Errorf("Expected viewStack length 0, got %d", len(app.viewStack))
	}
}
