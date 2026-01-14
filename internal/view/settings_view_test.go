package view

import (
	"context"
	"strings"
	"testing"

	"github.com/clawscli/claws/internal/config"
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

	contentBefore := sv.vp.Model.View()

	model, cmd := sv.Update(ThemeChangedMsg{})
	if cmd != nil {
		t.Error("Update(ThemeChangedMsg) should return nil cmd")
	}

	updated, ok := model.(*SettingsView)
	if !ok {
		t.Fatal("Update should return *SettingsView")
	}

	contentAfter := updated.vp.Model.View()
	if contentAfter == "" {
		t.Error("content should be regenerated on ThemeChangedMsg")
	}

	if contentBefore != contentAfter {
		t.Log("content was regenerated (styles may have changed)")
	}
}

func TestSettingsView_BuildContent(t *testing.T) {
	sv := NewSettingsView(context.Background())
	sv.SetSize(80, 24)

	content := sv.buildContent()

	expectedSections := []string{
		"Config File",
		"Runtime",
		"Theme",
		"Timeouts",
		"Concurrency",
		"CloudWatch",
		"Navigation",
		"Autosave",
		"AI",
	}

	for _, section := range expectedSections {
		if !strings.Contains(content, section) {
			t.Errorf("buildContent() should contain section %q", section)
		}
	}

	expectedFields := []string{
		"Compact",
		"Read-only",
		"Regions",
		"Profiles",
	}

	for _, field := range expectedFields {
		if !strings.Contains(content, field) {
			t.Errorf("buildContent() should contain field %q", field)
		}
	}
}

func TestSettingsView_GetThemeOverrides_Empty(t *testing.T) {
	sv := NewSettingsView(context.Background())

	overrides := sv.getThemeOverrides(config.ThemeConfig{})

	if len(overrides) != 0 {
		t.Errorf("getThemeOverrides() with empty config should return empty slice, got %d items", len(overrides))
	}
}

func TestSettingsView_GetThemeOverrides_WithOverrides(t *testing.T) {
	sv := NewSettingsView(context.Background())

	theme := config.ThemeConfig{
		Primary:   "#ff0000",
		Secondary: "#00ff00",
	}
	overrides := sv.getThemeOverrides(theme)

	if len(overrides) != 2 {
		t.Errorf("getThemeOverrides() should return 2 overrides, got %d", len(overrides))
	}

	if !strings.Contains(overrides[0], "Primary") {
		t.Errorf("first override should contain 'Primary', got %q", overrides[0])
	}
}

func TestSettingsView_FormatProfiles_Empty(t *testing.T) {
	sv := NewSettingsView(context.Background())

	result := sv.formatProfiles(nil)

	if result != noneValue {
		t.Errorf("formatProfiles(nil) should return %q, got %q", noneValue, result)
	}

	result = sv.formatProfiles([]config.ProfileSelection{})
	if result != noneValue {
		t.Errorf("formatProfiles([]) should return %q, got %q", noneValue, result)
	}
}
