package ui

import (
	"fmt"
	"image/color"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"sync"

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

// Preset theme names
const (
	ThemeDark       = "dark"
	ThemeLight      = "light"
	ThemeNord       = "nord"
	ThemeDracula    = "dracula"
	ThemeGruvbox    = "gruvbox"
	ThemeCatppuccin = "catppuccin"
)

// AvailableThemes returns a list of all available preset theme names
func AvailableThemes() []string {
	return []string{ThemeDark, ThemeLight, ThemeNord, ThemeDracula, ThemeGruvbox, ThemeCatppuccin}
}

type palette struct {
	primary, secondary, accent                string
	text, textBright, textDim, textMuted      string
	success, warning, danger, info, pending   string
	border, borderHighlight, bg, bgAlt        string
	selection, selectionText                  string
	tableHeader, tableHeaderText, tableBorder string
	badgeFg, badgeBg                          string
}

var presets = map[string]palette{
	ThemeDark: {
		primary: "170", secondary: "33", accent: "86",
		text: "252", textBright: "255", textDim: "247", textMuted: "244",
		success: "42", warning: "214", danger: "196", info: "33", pending: "226",
		border: "244", borderHighlight: "170", bg: "235", bgAlt: "237",
		selection: "57", selectionText: "229",
		tableHeader: "63", tableHeaderText: "229", tableBorder: "246",
		badgeFg: "16", badgeBg: "214",
	},
	ThemeLight: {
		primary: "#8839ef", secondary: "#1e66f5", accent: "#179299",
		text: "#4c4f69", textBright: "#1e1e2e", textDim: "#6c6f85", textMuted: "#9ca0b0",
		success: "#40a02b", warning: "#df8e1d", danger: "#d20f39", info: "#1e66f5", pending: "#df8e1d",
		border: "#9ca0b0", borderHighlight: "#8839ef", bg: "#eff1f5", bgAlt: "#e6e9ef",
		selection: "#8839ef", selectionText: "#eff1f5",
		tableHeader: "#7287fd", tableHeaderText: "#eff1f5", tableBorder: "#bcc0cc",
		badgeFg: "#eff1f5", badgeBg: "#df8e1d",
	},
	ThemeNord: {
		primary: "#88c0d0", secondary: "#81a1c1", accent: "#8fbcbb",
		text: "#d8dee9", textBright: "#eceff4", textDim: "#4c566a", textMuted: "#434c5e",
		success: "#a3be8c", warning: "#ebcb8b", danger: "#bf616a", info: "#5e81ac", pending: "#ebcb8b",
		border: "#4c566a", borderHighlight: "#88c0d0", bg: "#2e3440", bgAlt: "#3b4252",
		selection: "#5e81ac", selectionText: "#eceff4",
		tableHeader: "#434c5e", tableHeaderText: "#88c0d0", tableBorder: "#4c566a",
		badgeFg: "#2e3440", badgeBg: "#ebcb8b",
	},
	ThemeDracula: {
		primary: "#bd93f9", secondary: "#8be9fd", accent: "#ff79c6",
		text: "#f8f8f2", textBright: "#ffffff", textDim: "#6272a4", textMuted: "#44475a",
		success: "#50fa7b", warning: "#ffb86c", danger: "#ff5555", info: "#8be9fd", pending: "#f1fa8c",
		border: "#6272a4", borderHighlight: "#bd93f9", bg: "#282a36", bgAlt: "#44475a",
		selection: "#44475a", selectionText: "#f8f8f2",
		tableHeader: "#44475a", tableHeaderText: "#bd93f9", tableBorder: "#6272a4",
		badgeFg: "#282a36", badgeBg: "#ffb86c",
	},
	ThemeGruvbox: {
		primary: "#fe8019", secondary: "#83a598", accent: "#fabd2f",
		text: "#ebdbb2", textBright: "#fbf1c7", textDim: "#928374", textMuted: "#665c54",
		success: "#b8bb26", warning: "#fabd2f", danger: "#fb4934", info: "#83a598", pending: "#fabd2f",
		border: "#665c54", borderHighlight: "#fe8019", bg: "#282828", bgAlt: "#3c3836",
		selection: "#504945", selectionText: "#fbf1c7",
		tableHeader: "#3c3836", tableHeaderText: "#fe8019", tableBorder: "#665c54",
		badgeFg: "#282828", badgeBg: "#fabd2f",
	},
	ThemeCatppuccin: {
		primary: "#cba6f7", secondary: "#89b4fa", accent: "#f5c2e7",
		text: "#cdd6f4", textBright: "#ffffff", textDim: "#6c7086", textMuted: "#585b70",
		success: "#a6e3a1", warning: "#f9e2af", danger: "#f38ba8", info: "#89dceb", pending: "#f9e2af",
		border: "#585b70", borderHighlight: "#cba6f7", bg: "#1e1e2e", bgAlt: "#313244",
		selection: "#45475a", selectionText: "#cdd6f4",
		tableHeader: "#313244", tableHeaderText: "#cba6f7", tableBorder: "#585b70",
		badgeFg: "#1e1e2e", badgeBg: "#f9e2af",
	},
}

func buildTheme(p palette) *Theme {
	return &Theme{
		Primary:         lipgloss.Color(p.primary),
		Secondary:       lipgloss.Color(p.secondary),
		Accent:          lipgloss.Color(p.accent),
		Text:            lipgloss.Color(p.text),
		TextBright:      lipgloss.Color(p.textBright),
		TextDim:         lipgloss.Color(p.textDim),
		TextMuted:       lipgloss.Color(p.textMuted),
		Success:         lipgloss.Color(p.success),
		Warning:         lipgloss.Color(p.warning),
		Danger:          lipgloss.Color(p.danger),
		Info:            lipgloss.Color(p.info),
		Pending:         lipgloss.Color(p.pending),
		Border:          lipgloss.Color(p.border),
		BorderHighlight: lipgloss.Color(p.borderHighlight),
		Background:      lipgloss.Color(p.bg),
		BackgroundAlt:   lipgloss.Color(p.bgAlt),
		Selection:       lipgloss.Color(p.selection),
		SelectionText:   lipgloss.Color(p.selectionText),
		TableHeader:     lipgloss.Color(p.tableHeader),
		TableHeaderText: lipgloss.Color(p.tableHeaderText),
		TableBorder:     lipgloss.Color(p.tableBorder),
		BadgeForeground: lipgloss.Color(p.badgeFg),
		BadgeBackground: lipgloss.Color(p.badgeBg),
	}
}

// GetPreset returns a theme by name. Returns nil if the name is not recognized.
func GetPreset(name string) *Theme {
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		key = ThemeDark
	}
	if p, ok := presets[key]; ok {
		return buildTheme(p)
	}
	return nil
}

// DefaultTheme returns the default dark theme
func DefaultTheme() *Theme {
	return buildTheme(presets[ThemeDark])
}

// current holds the active theme
var (
	currentMu sync.RWMutex
	current   = DefaultTheme()
)

// Current returns the current active theme
func Current() *Theme {
	currentMu.RLock()
	defer currentMu.RUnlock()
	return current
}

func SetTheme(t *Theme) {
	if t != nil {
		currentMu.Lock()
		current = t
		currentMu.Unlock()
	}
}

func ApplyConfig(cfg config.ThemeConfig) {
	ApplyConfigWithOverride(cfg, "")
}

func ApplyConfigWithOverride(cfg config.ThemeConfig, cliTheme string) {
	presetName := cfg.Preset
	if cliTheme != "" {
		presetName = cliTheme
	}

	theme := GetPreset(presetName)
	if theme == nil {
		slog.Warn("unknown theme preset, using dark", "preset", presetName)
		theme = DefaultTheme()
	}

	if cliTheme != "" {
		SetTheme(theme)
		return
	}

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

func NoStyle() lipgloss.Style {
	return lipgloss.NewStyle()
}

func DimStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Current().TextDim)
}

// SuccessStyle returns a style for success states
func SuccessStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Current().Success)
}

// WarningStyle returns a style for warning states
func WarningStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Current().Warning)
}

// DangerStyle returns a style for danger/error states
func DangerStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Current().Danger)
}

func TitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(Current().Primary)
}

func SelectedStyle() lipgloss.Style {
	return lipgloss.NewStyle().Background(Current().Selection).Foreground(Current().SelectionText)
}

func TableHeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().Background(Current().TableHeader).Foreground(Current().TableHeaderText)
}

func SectionStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(Current().Secondary)
}

func HighlightStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(Current().Accent)
}

func BoldSuccessStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(Current().Success)
}

func BoldDangerStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(Current().Danger)
}

func BoldWarningStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(Current().Warning)
}

func BoldPendingStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(Current().Pending)
}

// AccentStyle returns a style for accent-colored text (non-bold)
func AccentStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Current().Accent)
}

// MutedStyle returns a style for very dim/muted text
func MutedStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Current().TextMuted)
}

// TextStyle returns a style for normal text
func TextStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Current().Text)
}

// TextBrightStyle returns a style for emphasized text
func TextBrightStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Current().TextBright)
}

// SecondaryStyle returns a style for secondary-colored text
func SecondaryStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Current().Secondary)
}

// BorderStyle returns a style for border-colored text (separators)
func BorderStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Current().Border)
}

// PrimaryStyle returns a style for primary-colored text (non-bold)
func PrimaryStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Current().Primary)
}

// InfoStyle returns a style for info states
func InfoStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Current().Info)
}

// PendingStyle returns a style for pending states
func PendingStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Current().Pending)
}

func FaintStyle() lipgloss.Style {
	return lipgloss.NewStyle().Faint(true)
}

func BoxStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Current().Border).
		Padding(0, 1)
}

func InputStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(Current().Border).
		Padding(0, 1)
}

// InputFieldStyle returns a style for input fields (filter, command input)
func InputFieldStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(Current().Background).
		Foreground(Current().Text).
		Padding(0, 1)
}

// ReadOnlyBadgeStyle returns a style for the READ-ONLY indicator badge
func ReadOnlyBadgeStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(Current().BadgeBackground).
		Foreground(Current().BadgeForeground).
		Bold(true).
		Padding(0, 1)
}

func CellStyle(width, height int) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(Current().Text).
		Width(width).
		Height(height).
		Padding(0, 1)
}

func NewSpinner() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(Current().Accent)
	return s
}
