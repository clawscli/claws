package view

import (
	"context"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
)

func TestResourceBrowserFilterPaste(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	browser := NewResourceBrowser(ctx, reg, "ec2")
	browser.SetSize(100, 50)
	browser.renderer = &mockRenderer{detail: "test"}
	browser.resources = []dao.Resource{
		&mockResource{id: "i-0123456789abcdef0", name: "web-server"},
		&mockResource{id: "i-0fedcba9876543210", name: "db-server"},
	}
	browser.applyFilter()
	browser.buildTable()

	// Paste while the filter is inactive should be ignored
	browser.Update(tea.PasteMsg{Content: "i-0123456789abcdef0"})
	if browser.filterText != "" {
		t.Fatalf("Expected paste to be ignored when filter is inactive, got filterText %q", browser.filterText)
	}

	browser.filterActive = true
	browser.filterInput.Focus()

	// Bracketed paste (Ctrl+Shift+V / tmux paste) arrives as tea.PasteMsg
	browser.Update(tea.PasteMsg{Content: "i-0123456789abcdef0"})

	if browser.filterText != "i-0123456789abcdef0" {
		t.Errorf("filterText = %q, want %q", browser.filterText, "i-0123456789abcdef0")
	}
	if len(browser.filtered) != 1 || browser.filtered[0].GetID() != "i-0123456789abcdef0" {
		t.Errorf("Expected filter to narrow to the pasted instance ID, got %d resources", len(browser.filtered))
	}

	// Ctrl+V must produce the textinput's clipboard-read command
	_, cmd := browser.Update(tea.KeyPressMsg{Code: 'v', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Error("Expected ctrl+v to return the clipboard paste command")
	}
}

func TestServiceBrowserFilterPaste(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()
	reg.RegisterCustom("ec2", "instances", registry.Entry{})
	reg.RegisterCustom("s3", "buckets", registry.Entry{})

	browser := NewServiceBrowser(ctx, reg)
	browser.Update(browser.Init()())

	browser.filterActive = true
	browser.filterInput.Focus()

	browser.Update(tea.PasteMsg{Content: "ec2"})

	if browser.filterText != "ec2" {
		t.Errorf("filterText = %q, want %q", browser.filterText, "ec2")
	}
	if len(browser.flatItems) != 1 {
		t.Errorf("Expected 1 service after paste, got %d", len(browser.flatItems))
	}
}

func TestLogViewFilterPaste(t *testing.T) {
	ctx := context.Background()
	lv := NewLogView(ctx, "/aws/test")
	lv.SetSize(80, 24)
	lv.loading = false
	lv.logs = []logEntry{
		{timestamp: time.Now(), message: "error in handler"},
		{timestamp: time.Now(), message: "request completed"},
	}

	lv.filterActive = true
	lv.filterInput.Focus()

	lv.Update(tea.PasteMsg{Content: "error"})

	if lv.filterText != "error" {
		t.Errorf("filterText = %q, want %q", lv.filterText, "error")
	}
}

func TestTagSearchViewFilterPaste(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	v := NewTagSearchView(ctx, reg, "")
	v.SetSize(100, 50)
	v.resources = []taggedARN{
		{RawARN: "arn:aws:ec2:us-east-1:123456789012:instance/i-0123456789abcdef0"},
		{RawARN: "arn:aws:s3:::my-bucket"},
	}
	v.applyFilter()
	v.buildTable()

	v.filterActive = true
	v.filterInput.Focus()

	v.Update(tea.PasteMsg{Content: "i-0123456789abcdef0"})

	if v.filterText != "i-0123456789abcdef0" {
		t.Errorf("filterText = %q, want %q", v.filterText, "i-0123456789abcdef0")
	}
	if len(v.filtered) != 1 {
		t.Errorf("Expected 1 resource after paste, got %d", len(v.filtered))
	}
}

type selectorMockItem struct {
	id    string
	label string
}

func (s selectorMockItem) GetID() string    { return s.id }
func (s selectorMockItem) GetLabel() string { return s.label }

func TestMultiSelectorFilterPaste(t *testing.T) {
	m := NewMultiSelector[selectorMockItem]("Select", nil)
	m.SetItems([]selectorMockItem{
		{id: "i-1", label: "web-server"},
		{id: "i-2", label: "db-server"},
	})

	m.filterActive = true
	m.filterInput.Focus()

	_, result := m.HandleUpdate(tea.PasteMsg{Content: "db"})

	if result != KeyHandled {
		t.Errorf("Expected paste to be handled, got %v", result)
	}
	if m.filterText != "db" {
		t.Errorf("filterText = %q, want %q", m.filterText, "db")
	}
	if len(m.filtered) != 1 || m.filtered[0].GetID() != "i-2" {
		t.Errorf("Expected filter to narrow to the pasted text, got %d items", len(m.filtered))
	}
}
