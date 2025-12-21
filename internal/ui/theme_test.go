package ui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestDefaultTheme(t *testing.T) {
	theme := DefaultTheme()

	if theme == nil {
		t.Fatal("DefaultTheme() returned nil")
	}

	// Check that primary colors are set
	if theme.Primary == "" {
		t.Error("Primary color should not be empty")
	}
	if theme.Secondary == "" {
		t.Error("Secondary color should not be empty")
	}
	if theme.Accent == "" {
		t.Error("Accent color should not be empty")
	}

	// Check semantic colors
	if theme.Success == "" {
		t.Error("Success color should not be empty")
	}
	if theme.Warning == "" {
		t.Error("Warning color should not be empty")
	}
	if theme.Danger == "" {
		t.Error("Danger color should not be empty")
	}
}

func TestCurrent(t *testing.T) {
	theme := Current()

	if theme == nil {
		t.Fatal("Current() returned nil")
	}

	// Current should return the same as DefaultTheme initially
	defaultTheme := DefaultTheme()
	if theme.Primary != defaultTheme.Primary {
		t.Errorf("Current().Primary = %v, want %v", theme.Primary, defaultTheme.Primary)
	}
}

func TestDimStyle(t *testing.T) {
	style := DimStyle()

	// Should have foreground color set
	fg := style.GetForeground()
	if fg == nil {
		t.Error("DimStyle() should have foreground color")
	}

	// Render should work without panic
	result := style.Render("test")
	if result == "" {
		t.Error("DimStyle().Render() should produce output")
	}
}

func TestSuccessStyle(t *testing.T) {
	style := SuccessStyle()

	// Should have foreground color set
	fg := style.GetForeground()
	if fg == nil {
		t.Error("SuccessStyle() should have foreground color")
	}
}

func TestWarningStyle(t *testing.T) {
	style := WarningStyle()

	// Should have foreground color set
	fg := style.GetForeground()
	if fg == nil {
		t.Error("WarningStyle() should have foreground color")
	}
}

func TestDangerStyle(t *testing.T) {
	style := DangerStyle()

	// Should have foreground color set
	fg := style.GetForeground()
	if fg == nil {
		t.Error("DangerStyle() should have foreground color")
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

	// Test all text colors are set
	textColors := []struct {
		name  string
		color lipgloss.Color
	}{
		{"Text", theme.Text},
		{"TextBright", theme.TextBright},
		{"TextDim", theme.TextDim},
		{"TextMuted", theme.TextMuted},
	}

	for _, tc := range textColors {
		if tc.color == "" {
			t.Errorf("%s color should not be empty", tc.name)
		}
	}

	// Test UI element colors
	uiColors := []struct {
		name  string
		color lipgloss.Color
	}{
		{"Border", theme.Border},
		{"BorderHighlight", theme.BorderHighlight},
		{"Background", theme.Background},
		{"BackgroundAlt", theme.BackgroundAlt},
		{"Selection", theme.Selection},
		{"SelectionText", theme.SelectionText},
	}

	for _, tc := range uiColors {
		if tc.color == "" {
			t.Errorf("%s color should not be empty", tc.name)
		}
	}

	// Test table colors
	tableColors := []struct {
		name  string
		color lipgloss.Color
	}{
		{"TableHeader", theme.TableHeader},
		{"TableHeaderText", theme.TableHeaderText},
		{"TableBorder", theme.TableBorder},
	}

	for _, tc := range tableColors {
		if tc.color == "" {
			t.Errorf("%s color should not be empty", tc.name)
		}
	}
}
