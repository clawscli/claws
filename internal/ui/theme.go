package ui

import (
	"image/color"

	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"
)

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
}

// DefaultTheme returns the default dark theme
func DefaultTheme() *Theme {
	return &Theme{
		// Primary colors
		Primary:   lipgloss.Color("170"), // Pink/Magenta
		Secondary: lipgloss.Color("33"),  // Blue
		Accent:    lipgloss.Color("86"),  // Cyan

		// Text colors
		Text:       lipgloss.Color("252"), // Light gray
		TextBright: lipgloss.Color("255"), // White
		TextDim:    lipgloss.Color("247"), // Medium gray
		TextMuted:  lipgloss.Color("244"), // Darker gray

		// Semantic colors
		Success: lipgloss.Color("42"),  // Green
		Warning: lipgloss.Color("214"), // Orange
		Danger:  lipgloss.Color("196"), // Red
		Info:    lipgloss.Color("33"),  // Blue
		Pending: lipgloss.Color("226"), // Yellow

		// UI element colors
		Border:          lipgloss.Color("244"), // Gray border
		BorderHighlight: lipgloss.Color("170"), // Pink highlight
		Background:      lipgloss.Color("235"), // Dark background
		BackgroundAlt:   lipgloss.Color("237"), // Slightly lighter
		Selection:       lipgloss.Color("57"),  // Purple selection
		SelectionText:   lipgloss.Color("229"), // Light yellow

		// Table colors
		TableHeader:     lipgloss.Color("63"),  // Purple header
		TableHeaderText: lipgloss.Color("229"), // Light yellow
		TableBorder:     lipgloss.Color("246"), // Gray border
	}
}

// current holds the active theme
var current = DefaultTheme()

// Current returns the current active theme
func Current() *Theme {
	return current
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
		Background(current.Warning).
		Foreground(lipgloss.Color("#000000")). // TODO: extract to theme in #96
		Bold(true).
		Padding(0, 1)
}

func NewSpinner() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(current.Accent)
	return s
}
