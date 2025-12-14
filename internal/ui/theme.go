package ui

import "github.com/charmbracelet/lipgloss"

// Theme defines the color scheme for the application
type Theme struct {
	// Primary colors
	Primary   lipgloss.Color // Main accent color (titles, highlights)
	Secondary lipgloss.Color // Secondary accent color
	Accent    lipgloss.Color // Navigation/links accent

	// Text colors
	Text       lipgloss.Color // Normal text
	TextBright lipgloss.Color // Bright/emphasized text
	TextDim    lipgloss.Color // Dimmed text (labels, hints)
	TextMuted  lipgloss.Color // Very dim text (separators, borders)

	// Semantic colors
	Success lipgloss.Color // Green - success states
	Warning lipgloss.Color // Yellow/Orange - warning states
	Danger  lipgloss.Color // Red - error/danger states
	Info    lipgloss.Color // Blue - info states
	Pending lipgloss.Color // Yellow - pending/in-progress states

	// UI element colors
	Border          lipgloss.Color // Border color
	BorderHighlight lipgloss.Color // Highlighted border
	Background      lipgloss.Color // Background for panels
	BackgroundAlt   lipgloss.Color // Alternative background
	Selection       lipgloss.Color // Selected item background
	SelectionText   lipgloss.Color // Selected item text

	// Table colors
	TableHeader     lipgloss.Color // Table header background
	TableHeaderText lipgloss.Color // Table header text
	TableBorder     lipgloss.Color // Table border
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
