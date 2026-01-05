package ui

import (
	"fmt"
	"image/color"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"

	"github.com/clawscli/claws/internal/config"
)

var (
	hex6Re = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
	hex3Re = regexp.MustCompile(`^#[0-9A-Fa-f]{3}$`)
)

// ParseColor parses a color string and returns a lipgloss color.
// Accepts hex (#RGB, #RRGGBB) or ANSI 256 numbers (0-255).
// Returns nil, nil for empty strings (caller should use default).
// Returns nil, error for invalid color strings.
func ParseColor(s string) (color.Color, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}

	if strings.HasPrefix(s, "#") {
		if hex6Re.MatchString(s) {
			return lipgloss.Color(s), nil
		}
		if hex3Re.MatchString(s) {
			// Expand #RGB to #RRGGBB
			r, g, b := s[1], s[2], s[3]
			expanded := fmt.Sprintf("#%c%c%c%c%c%c", r, r, g, g, b, b)
			return lipgloss.Color(expanded), nil
		}
		return nil, fmt.Errorf("invalid hex color %q: must be #RGB or #RRGGBB", s)
	}

	n, err := strconv.Atoi(s)
	if err != nil {
		return nil, fmt.Errorf("invalid color %q: must be hex (#RGB/#RRGGBB) or ANSI number (0-255)", s)
	}
	if n < 0 || n > 255 {
		return nil, fmt.Errorf("invalid ANSI color %d: must be 0-255", n)
	}
	return lipgloss.Color(s), nil
}

// Theme defines the color scheme for the application
type Theme struct {
	// Primary colors
	Primary   color.Color // Main accent color (titles, highlights)
	Secondary color.Color // Secondary accent color
	Accent    color.Color // Navigation/links accent

	// Text colors
	Text       color.Color // Normal text
	TextBright color.Color // Bright/emphasized text
	TextDim    color.Color // Dimmed text (labels, hints)
	TextMuted  color.Color // Very dim text (separators, borders)

	// Semantic colors
	Success color.Color // Green - success states
	Warning color.Color // Yellow/Orange - warning states
	Danger  color.Color // Red - error/danger states
	Info    color.Color // Blue - info states
	Pending color.Color // Yellow - pending/in-progress states

	// UI element colors
	Border          color.Color // Border color
	BorderHighlight color.Color // Highlighted border
	Background      color.Color // Background for panels
	BackgroundAlt   color.Color // Alternative background
	Selection       color.Color // Selected item background
	SelectionText   color.Color // Selected item text

	// Table colors
	TableHeader     color.Color // Table header background
	TableHeaderText color.Color // Table header text
	TableBorder     color.Color // Table border

	// Badge colors (for READ-ONLY indicator, etc.)
	BadgeForeground color.Color // Badge text color
	BadgeBackground color.Color // Badge background color
}

func DefaultTheme() *Theme {
	return &Theme{
		Primary:         lipgloss.Color("170"),
		Secondary:       lipgloss.Color("33"),
		Accent:          lipgloss.Color("86"),
		Text:            lipgloss.Color("252"),
		TextBright:      lipgloss.Color("255"),
		TextDim:         lipgloss.Color("247"),
		TextMuted:       lipgloss.Color("244"),
		Success:         lipgloss.Color("42"),
		Warning:         lipgloss.Color("214"),
		Danger:          lipgloss.Color("196"),
		Info:            lipgloss.Color("33"),
		Pending:         lipgloss.Color("226"),
		Border:          lipgloss.Color("244"),
		BorderHighlight: lipgloss.Color("170"),
		Background:      lipgloss.Color("235"),
		BackgroundAlt:   lipgloss.Color("237"),
		Selection:       lipgloss.Color("57"),
		SelectionText:   lipgloss.Color("229"),
		TableHeader:     lipgloss.Color("63"),
		TableHeaderText: lipgloss.Color("229"),
		TableBorder:     lipgloss.Color("246"),
		BadgeForeground: lipgloss.Color("16"),
		BadgeBackground: lipgloss.Color("214"),
	}
}

// current holds the active theme
var current = DefaultTheme()

// Current returns the current active theme
func Current() *Theme {
	return current
}

func SetTheme(t *Theme) {
	if t != nil {
		current = t
	}
}

func ApplyConfig(cfg config.ThemeConfig) {
	theme := DefaultTheme()

	applyColor := func(name string, value string, target *color.Color) {
		if value == "" {
			return
		}
		c, err := ParseColor(value)
		if err != nil {
			slog.Warn("invalid theme color, using default", "field", name, "value", value, "error", err)
			return
		}
		*target = c
	}

	applyColor("primary", cfg.Primary, &theme.Primary)
	applyColor("secondary", cfg.Secondary, &theme.Secondary)
	applyColor("accent", cfg.Accent, &theme.Accent)
	applyColor("text", cfg.Text, &theme.Text)
	applyColor("text_bright", cfg.TextBright, &theme.TextBright)
	applyColor("text_dim", cfg.TextDim, &theme.TextDim)
	applyColor("text_muted", cfg.TextMuted, &theme.TextMuted)
	applyColor("success", cfg.Success, &theme.Success)
	applyColor("warning", cfg.Warning, &theme.Warning)
	applyColor("danger", cfg.Danger, &theme.Danger)
	applyColor("info", cfg.Info, &theme.Info)
	applyColor("pending", cfg.Pending, &theme.Pending)
	applyColor("border", cfg.Border, &theme.Border)
	applyColor("border_highlight", cfg.BorderHighlight, &theme.BorderHighlight)
	applyColor("background", cfg.Background, &theme.Background)
	applyColor("background_alt", cfg.BackgroundAlt, &theme.BackgroundAlt)
	applyColor("selection", cfg.Selection, &theme.Selection)
	applyColor("selection_text", cfg.SelectionText, &theme.SelectionText)
	applyColor("table_header", cfg.TableHeader, &theme.TableHeader)
	applyColor("table_header_text", cfg.TableHeaderText, &theme.TableHeaderText)
	applyColor("table_border", cfg.TableBorder, &theme.TableBorder)
	applyColor("badge_foreground", cfg.BadgeForeground, &theme.BadgeForeground)
	applyColor("badge_background", cfg.BadgeBackground, &theme.BadgeBackground)

	SetTheme(theme)
}

// Style helpers that use the current theme

// DimStyle returns a style for dimmed text
func DimStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(current.TextDim)
}

// SuccessStyle returns a style for success states
func SuccessStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(current.Success)
}

// WarningStyle returns a style for warning states
func WarningStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(current.Warning)
}

// DangerStyle returns a style for danger/error states
func DangerStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(current.Danger)
}

func TitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(current.Primary)
}

func SelectedStyle() lipgloss.Style {
	return lipgloss.NewStyle().Background(current.Selection).Foreground(current.SelectionText)
}

func TableHeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().Background(current.TableHeader).Foreground(current.TableHeaderText)
}

func SectionStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(current.Secondary)
}

func HighlightStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(current.Accent)
}

func BoldSuccessStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(current.Success)
}

func BoldDangerStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(current.Danger)
}

func BoldWarningStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(current.Warning)
}

func BoldPendingStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(current.Pending)
}

// AccentStyle returns a style for accent-colored text (non-bold)
func AccentStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(current.Accent)
}

// MutedStyle returns a style for very dim/muted text
func MutedStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(current.TextMuted)
}

// TextStyle returns a style for normal text
func TextStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(current.Text)
}

// TextBrightStyle returns a style for emphasized text
func TextBrightStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(current.TextBright)
}

// SecondaryStyle returns a style for secondary-colored text
func SecondaryStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(current.Secondary)
}

// BorderStyle returns a style for border-colored text (separators)
func BorderStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(current.Border)
}

// PrimaryStyle returns a style for primary-colored text (non-bold)
func PrimaryStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(current.Primary)
}

// InfoStyle returns a style for info states
func InfoStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(current.Info)
}

// PendingStyle returns a style for pending states
func PendingStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(current.Pending)
}

func BoxStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(current.Border).
		Padding(0, 1)
}

func InputStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(current.Border).
		Padding(0, 1)
}

// InputFieldStyle returns a style for input fields (filter, command input)
func InputFieldStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(current.Background).
		Foreground(current.Text).
		Padding(0, 1)
}

// ReadOnlyBadgeStyle returns a style for the READ-ONLY indicator badge
func ReadOnlyBadgeStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(current.BadgeBackground).
		Foreground(current.BadgeForeground).
		Bold(true).
		Padding(0, 1)
}

func NewSpinner() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(current.Accent)
	return s
}
