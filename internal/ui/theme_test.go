package ui

import (
	"image/color"
	"testing"

	"github.com/clawscli/claws/internal/config"
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

func TestTitleStyle(t *testing.T) {
	style := TitleStyle()
	rendered := style.Render("title")
	if rendered == "" {
		t.Error("TitleStyle().Render() should produce output")
	}
}

func TestSelectedStyle(t *testing.T) {
	style := SelectedStyle()
	rendered := style.Render("selected")
	if rendered == "" {
		t.Error("SelectedStyle().Render() should produce output")
	}
}

func TestTableHeaderStyle(t *testing.T) {
	style := TableHeaderStyle()
	rendered := style.Render("header")
	if rendered == "" {
		t.Error("TableHeaderStyle().Render() should produce output")
	}
}

func TestSectionStyle(t *testing.T) {
	style := SectionStyle()
	rendered := style.Render("section")
	if rendered == "" {
		t.Error("SectionStyle().Render() should produce output")
	}
}

func TestHighlightStyle(t *testing.T) {
	style := HighlightStyle()
	rendered := style.Render("highlight")
	if rendered == "" {
		t.Error("HighlightStyle().Render() should produce output")
	}
}

func TestBoldSuccessStyle(t *testing.T) {
	style := BoldSuccessStyle()
	rendered := style.Render("bold success")
	if rendered == "" {
		t.Error("BoldSuccessStyle().Render() should produce output")
	}
}

func TestBoldDangerStyle(t *testing.T) {
	style := BoldDangerStyle()
	rendered := style.Render("bold danger")
	if rendered == "" {
		t.Error("BoldDangerStyle().Render() should produce output")
	}
}

func TestBoldWarningStyle(t *testing.T) {
	style := BoldWarningStyle()
	rendered := style.Render("bold warning")
	if rendered == "" {
		t.Error("BoldWarningStyle().Render() should produce output")
	}
}

func TestBoldPendingStyle(t *testing.T) {
	style := BoldPendingStyle()
	rendered := style.Render("bold pending")
	if rendered == "" {
		t.Error("BoldPendingStyle().Render() should produce output")
	}
}

func TestAccentStyle(t *testing.T) {
	style := AccentStyle()
	rendered := style.Render("accent")
	if rendered == "" {
		t.Error("AccentStyle().Render() should produce output")
	}
}

func TestMutedStyle(t *testing.T) {
	style := MutedStyle()
	rendered := style.Render("muted")
	if rendered == "" {
		t.Error("MutedStyle().Render() should produce output")
	}
}

func TestTextStyle(t *testing.T) {
	style := TextStyle()
	rendered := style.Render("text")
	if rendered == "" {
		t.Error("TextStyle().Render() should produce output")
	}
}

func TestTextBrightStyle(t *testing.T) {
	style := TextBrightStyle()
	rendered := style.Render("bright")
	if rendered == "" {
		t.Error("TextBrightStyle().Render() should produce output")
	}
}

func TestSecondaryStyle(t *testing.T) {
	style := SecondaryStyle()
	rendered := style.Render("secondary")
	if rendered == "" {
		t.Error("SecondaryStyle().Render() should produce output")
	}
}

func TestBorderStyle(t *testing.T) {
	style := BorderStyle()
	rendered := style.Render("border")
	if rendered == "" {
		t.Error("BorderStyle().Render() should produce output")
	}
}

func TestPrimaryStyle(t *testing.T) {
	style := PrimaryStyle()
	rendered := style.Render("primary")
	if rendered == "" {
		t.Error("PrimaryStyle().Render() should produce output")
	}
}

func TestInfoStyle(t *testing.T) {
	style := InfoStyle()
	rendered := style.Render("info")
	if rendered == "" {
		t.Error("InfoStyle().Render() should produce output")
	}
}

func TestPendingStyle(t *testing.T) {
	style := PendingStyle()
	rendered := style.Render("pending")
	if rendered == "" {
		t.Error("PendingStyle().Render() should produce output")
	}
}

func TestBoxStyle(t *testing.T) {
	style := BoxStyle()
	rendered := style.Render("box content")
	if rendered == "" {
		t.Error("BoxStyle().Render() should produce output")
	}
}

func TestInputStyle(t *testing.T) {
	style := InputStyle()
	rendered := style.Render("input content")
	if rendered == "" {
		t.Error("InputStyle().Render() should produce output")
	}
}

func TestInputFieldStyle(t *testing.T) {
	style := InputFieldStyle()
	rendered := style.Render("filter text")
	if rendered == "" {
		t.Error("InputFieldStyle().Render() should produce output")
	}
}

func TestReadOnlyBadgeStyle(t *testing.T) {
	style := ReadOnlyBadgeStyle()
	rendered := style.Render("READ-ONLY")
	if rendered == "" {
		t.Error("ReadOnlyBadgeStyle().Render() should produce output")
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

	// Test badge colors
	badgeColors := []struct {
		name  string
		color color.Color
	}{
		{"BadgeForeground", theme.BadgeForeground},
		{"BadgeBackground", theme.BadgeBackground},
	}

	for _, tc := range badgeColors {
		if tc.color == nil {
			t.Errorf("%s color should not be nil", tc.name)
		}
	}
}

func TestParseColor(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantNil bool
		wantErr bool
	}{
		{"empty string", "", true, false},
		{"whitespace only", "   ", true, false},
		{"valid hex 6", "#ff5733", false, false},
		{"valid hex 6 upper", "#FF5733", false, false},
		{"valid hex 3", "#f00", false, false},
		{"valid hex 3 upper", "#F00", false, false},
		{"valid ANSI 0", "0", false, false},
		{"valid ANSI 170", "170", false, false},
		{"valid ANSI 255", "255", false, false},
		{"invalid hex short", "#ff", false, true},
		{"invalid hex long", "#ff57331", false, true},
		{"invalid hex chars", "#gggggg", false, true},
		{"invalid ANSI negative", "-1", false, true},
		{"invalid ANSI over 255", "256", false, true},
		{"invalid string", "red", false, true},
		{"invalid mixed", "ff5733", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := ParseColor(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseColor(%q) expected error, got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseColor(%q) unexpected error: %v", tt.input, err)
				return
			}

			if tt.wantNil && c != nil {
				t.Errorf("ParseColor(%q) expected nil, got %v", tt.input, c)
			}
			if !tt.wantNil && c == nil {
				t.Errorf("ParseColor(%q) expected color, got nil", tt.input)
			}
		})
	}
}

func TestParseColorHex3Expansion(t *testing.T) {
	c, err := ParseColor("#f00")
	if err != nil {
		t.Fatalf("ParseColor(#f00) unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("ParseColor(#f00) returned nil")
	}

	r, g, b, _ := c.RGBA()
	if r>>8 != 0xff || g>>8 != 0 || b>>8 != 0 {
		t.Errorf("ParseColor(#f00) expected red, got R=%d G=%d B=%d", r>>8, g>>8, b>>8)
	}
}

func TestSetTheme(t *testing.T) {
	original := Current()

	newTheme := DefaultTheme()
	newTheme.Primary = nil

	SetTheme(newTheme)
	if Current() != newTheme {
		t.Error("SetTheme did not set the theme")
	}

	SetTheme(nil)
	if Current() != newTheme {
		t.Error("SetTheme(nil) should not change theme")
	}

	SetTheme(original)
}

func TestApplyConfig(t *testing.T) {
	original := Current()
	defer SetTheme(original)

	cfg := config.ThemeConfig{
		Primary: "#ff0000",
		Success: "42",
		Danger:  "#f00",
	}

	ApplyConfig(cfg)

	theme := Current()

	r, g, b, _ := theme.Primary.RGBA()
	if r>>8 != 0xff || g>>8 != 0 || b>>8 != 0 {
		t.Errorf("Primary expected red, got R=%d G=%d B=%d", r>>8, g>>8, b>>8)
	}

	if theme.Secondary == nil {
		t.Error("Secondary should use default, not nil")
	}
}

func TestApplyConfigInvalidColor(t *testing.T) {
	original := Current()
	defer SetTheme(original)

	defaultTheme := DefaultTheme()

	cfg := config.ThemeConfig{
		Primary: "invalid",
		Success: "#gggggg",
	}

	ApplyConfig(cfg)

	theme := Current()

	if !colorsEqual(theme.Primary, defaultTheme.Primary) {
		t.Error("Invalid primary should fallback to default")
	}
	if !colorsEqual(theme.Success, defaultTheme.Success) {
		t.Error("Invalid success should fallback to default")
	}
}

func TestApplyConfigEmpty(t *testing.T) {
	original := Current()
	defer SetTheme(original)

	defaultTheme := DefaultTheme()

	ApplyConfig(config.ThemeConfig{})

	theme := Current()

	if !colorsEqual(theme.Primary, defaultTheme.Primary) {
		t.Error("Empty config should use default primary")
	}
	if !colorsEqual(theme.Success, defaultTheme.Success) {
		t.Error("Empty config should use default success")
	}
}

func TestAvailableThemes(t *testing.T) {
	themes := AvailableThemes()
	if len(themes) != 6 {
		t.Errorf("Expected 6 themes, got %d", len(themes))
	}

	expected := []string{"dark", "light", "nord", "dracula", "gruvbox", "catppuccin"}
	for i, name := range expected {
		if themes[i] != name {
			t.Errorf("Expected themes[%d] = %q, got %q", i, name, themes[i])
		}
	}
}

func TestGetPreset(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantNil bool
	}{
		{"empty uses dark", "", false},
		{"dark", "dark", false},
		{"light", "light", false},
		{"nord", "nord", false},
		{"dracula", "dracula", false},
		{"gruvbox", "gruvbox", false},
		{"catppuccin", "catppuccin", false},
		{"case insensitive", "NORD", false},
		{"with spaces", "  dark  ", false},
		{"unknown", "unknown-theme", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			theme := GetPreset(tt.input)
			if tt.wantNil && theme != nil {
				t.Errorf("GetPreset(%q) expected nil, got theme", tt.input)
			}
			if !tt.wantNil && theme == nil {
				t.Errorf("GetPreset(%q) expected theme, got nil", tt.input)
			}
		})
	}
}

func TestGetPresetColors(t *testing.T) {
	presets := AvailableThemes()
	for _, name := range presets {
		t.Run(name, func(t *testing.T) {
			theme := GetPreset(name)
			if theme == nil {
				t.Fatalf("GetPreset(%q) returned nil", name)
			}

			if theme.Primary == nil {
				t.Error("Primary should not be nil")
			}
			if theme.Text == nil {
				t.Error("Text should not be nil")
			}
			if theme.Success == nil {
				t.Error("Success should not be nil")
			}
			if theme.Selection == nil {
				t.Error("Selection should not be nil")
			}
		})
	}
}

func TestApplyConfigWithPreset(t *testing.T) {
	original := Current()
	defer SetTheme(original)

	cfg := config.ThemeConfig{Preset: "nord"}
	ApplyConfig(cfg)

	theme := Current()
	nordTheme := GetPreset("nord")

	if !colorsEqual(theme.Primary, nordTheme.Primary) {
		t.Error("Preset nord should apply nord primary color")
	}
}

func TestApplyConfigWithPresetAndOverride(t *testing.T) {
	original := Current()
	defer SetTheme(original)

	cfg := config.ThemeConfig{
		Preset:  "nord",
		Primary: "#ff0000",
	}
	ApplyConfig(cfg)

	theme := Current()

	r, g, b, _ := theme.Primary.RGBA()
	if r>>8 != 0xff || g>>8 != 0 || b>>8 != 0 {
		t.Errorf("Primary should be overridden to red, got R=%d G=%d B=%d", r>>8, g>>8, b>>8)
	}
}

func TestApplyConfigWithOverride(t *testing.T) {
	original := Current()
	defer SetTheme(original)

	cfg := config.ThemeConfig{Preset: "nord"}
	ApplyConfigWithOverride(cfg, "dracula")

	theme := Current()
	draculaTheme := GetPreset("dracula")

	if !colorsEqual(theme.Primary, draculaTheme.Primary) {
		t.Error("CLI override should use dracula, not nord")
	}
}

func TestThemeConcurrentAccess(t *testing.T) {
	original := Current()
	defer SetTheme(original)

	themes := []*Theme{
		GetPreset("dark"),
		GetPreset("light"),
		GetPreset("nord"),
		GetPreset("dracula"),
	}

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				SetTheme(themes[j%len(themes)])
				_ = Current()
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// If we get here without race detector panic, the test passes
	if Current() == nil {
		t.Error("Current() should not return nil")
	}
}
