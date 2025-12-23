package ui

import (
	"image/color"
	"testing"
)

func TestDefaultTheme(t *testing.T) {
	theme := DefaultTheme()

	if theme == nil {
		t.Fatal("DefaultTheme() returned nil")
	}

	// Check that primary colors are set (not nil)
	if theme.Primary == nil {
		t.Error("Primary color should not be nil")
	}
	if theme.Secondary == nil {
		t.Error("Secondary color should not be nil")
	}
	if theme.Accent == nil {
		t.Error("Accent color should not be nil")
	}

	// Check semantic colors
	if theme.Success == nil {
		t.Error("Success color should not be nil")
	}
	if theme.Warning == nil {
		t.Error("Warning color should not be nil")
	}
	if theme.Danger == nil {
		t.Error("Danger color should not be nil")
	}
}

func TestCurrent(t *testing.T) {
	theme := Current()

	if theme == nil {
		t.Fatal("Current() returned nil")
	}

	// Current should return the same as DefaultTheme initially
	defaultTheme := DefaultTheme()
	if !colorsEqual(theme.Primary, defaultTheme.Primary) {
		t.Errorf("Current().Primary should equal DefaultTheme().Primary")
	}
}

// colorsEqual compares two colors for equality
func colorsEqual(a, b color.Color) bool {
	if a == nil || b == nil {
		return a == b
	}
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}

func TestDimStyle(t *testing.T) {
	style := DimStyle()

	// Just verify it doesn't panic and produces output
	rendered := style.Render("test")
	if rendered == "" {
		t.Error("DimStyle().Render() should produce output")
	}
}

func TestSuccessStyle(t *testing.T) {
	style := SuccessStyle()

	rendered := style.Render("success")
	if rendered == "" {
		t.Error("SuccessStyle().Render() should produce output")
	}
}

func TestWarningStyle(t *testing.T) {
	style := WarningStyle()

	rendered := style.Render("warning")
	if rendered == "" {
		t.Error("WarningStyle().Render() should produce output")
	}
}

func TestDangerStyle(t *testing.T) {
	style := DangerStyle()

	rendered := style.Render("danger")
	if rendered == "" {
		t.Error("DangerStyle().Render() should produce output")
	}
}

func TestNewSpinner(t *testing.T) {
	s := NewSpinner()

	// Spinner should be initialized
	if s.Spinner.Frames == nil {
		t.Error("NewSpinner() should have spinner frames")
	}

	// Should use Dot spinner (has specific frame count)
	// spinner.Dot has 10 frames
	if len(s.Spinner.Frames) == 0 {
		t.Error("NewSpinner() should have non-empty frames")
	}

	// View should produce output
	view := s.View()
	if view == "" {
		t.Error("NewSpinner().View() should produce output")
	}
}

func TestThemeFields(t *testing.T) {
	theme := DefaultTheme()

	// Test all text colors are set (not nil)
	textColors := []struct {
		name  string
		color color.Color
	}{
		{"Text", theme.Text},
		{"TextBright", theme.TextBright},
		{"TextDim", theme.TextDim},
		{"TextMuted", theme.TextMuted},
	}

	for _, tc := range textColors {
		if tc.color == nil {
			t.Errorf("%s color should not be nil", tc.name)
		}
	}

	// Test UI element colors
	uiColors := []struct {
		name  string
		color color.Color
	}{
		{"Border", theme.Border},
		{"BorderHighlight", theme.BorderHighlight},
		{"Background", theme.Background},
		{"BackgroundAlt", theme.BackgroundAlt},
		{"Selection", theme.Selection},
		{"SelectionText", theme.SelectionText},
	}

	for _, tc := range uiColors {
		if tc.color == nil {
			t.Errorf("%s color should not be nil", tc.name)
		}
	}

	// Test table colors
	tableColors := []struct {
		name  string
		color color.Color
	}{
		{"TableHeader", theme.TableHeader},
		{"TableHeaderText", theme.TableHeaderText},
		{"TableBorder", theme.TableBorder},
	}

	for _, tc := range tableColors {
		if tc.color == nil {
			t.Errorf("%s color should not be nil", tc.name)
		}
	}
}
