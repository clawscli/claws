package view

import (
	"context"
	"testing"
)

func TestSettingsView_New(t *testing.T) {
	sv := NewSettingsView(context.Background())

	if sv == nil {
		t.Fatal("NewSettingsView() returned nil")
	}
}

func TestSettingsView_Init(t *testing.T) {
	sv := NewSettingsView(context.Background())

	cmd := sv.Init()
	if cmd != nil {
		t.Error("Init() should return nil")
	}
}

func TestSettingsView_StatusLine(t *testing.T) {
	sv := NewSettingsView(context.Background())

	status := sv.StatusLine()
	if status != "" {
		t.Error("StatusLine() should return empty string for modal")
	}
}

func TestSettingsView_ViewString_BeforeSetSize(t *testing.T) {
	sv := NewSettingsView(context.Background())

	view := sv.ViewString()
	if view != LoadingMessage {
		t.Errorf("ViewString() before SetSize should return LoadingMessage, got: %s", view)
	}
}

func TestSettingsView_SetSize(t *testing.T) {
	sv := NewSettingsView(context.Background())

	cmd := sv.SetSize(80, 24)
	if cmd != nil {
		t.Error("SetSize() should return nil")
	}

	if !sv.vp.Ready {
		t.Error("viewport should be ready after SetSize")
	}

	view := sv.ViewString()
	if view == LoadingMessage {
		t.Error("ViewString() after SetSize should not return LoadingMessage")
	}
}

func TestSettingsView_ThemeChanged(t *testing.T) {
	sv := NewSettingsView(context.Background())
	sv.SetSize(80, 24)

	originalStyles := sv.styles

	model, cmd := sv.Update(ThemeChangedMsg{})
	if cmd != nil {
		t.Error("Update(ThemeChangedMsg) should return nil cmd")
	}

	updated, ok := model.(*SettingsView)
	if !ok {
		t.Fatal("Update should return *SettingsView")
	}

	if &updated.styles == &originalStyles {
		t.Error("styles should be regenerated on ThemeChangedMsg")
	}
}
