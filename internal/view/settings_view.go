package view

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/ui"
)

const (
	noneValue              = "(none)"
	settingsSeparatorInset = 10
	settingsLabelWidth     = 18
)

// settingsViewStyles holds cached lipgloss styles for performance.
type settingsViewStyles struct {
	title     lipgloss.Style
	separator lipgloss.Style
	text      lipgloss.Style
}

func newSettingsViewStyles() settingsViewStyles {
	return settingsViewStyles{
		title:     ui.TitleStyle(),
		separator: ui.DimStyle(),
		text:      ui.TextStyle(),
	}
}

func wrapSettingsValue(value string, maxWidth int) string {
	if lipgloss.Width(value) <= maxWidth || maxWidth <= 0 {
		return value
	}

	var lines []string
	indent := strings.Repeat(" ", settingsLabelWidth)

	words := strings.Split(value, ", ")
	var currentLine string

	for i, word := range words {
		sep := ""
		if i > 0 {
			sep = ", "
		}
		candidate := currentLine + sep + word
		if lipgloss.Width(candidate) > maxWidth && currentLine != "" {
			lines = append(lines, currentLine)
			currentLine = word
		} else {
			currentLine = candidate
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return strings.Join(lines, "\n"+indent)
}

type SettingsView struct {
	ctx          context.Context
	vp           ViewportState
	screenWidth  int
	screenHeight int
	styles       settingsViewStyles
}

func NewSettingsView(ctx context.Context) *SettingsView {
	return &SettingsView{
		ctx:    ctx,
		styles: newSettingsViewStyles(),
	}
}

func (v *SettingsView) Init() tea.Cmd {
	return nil
}

func (v *SettingsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case ThemeChangedMsg:
		v.styles = newSettingsViewStyles()
		if v.vp.Ready {
			v.vp.Model.SetContent(v.buildContent())
		}
		return v, nil
	}

	if !v.vp.Ready {
		return v, nil
	}

	var cmd tea.Cmd
	v.vp.Model, cmd = v.vp.Model.Update(msg)
	return v, cmd
}

func (v *SettingsView) View() tea.View {
	return tea.NewView(v.ViewString())
}

func (v *SettingsView) ViewString() string {
	if !v.vp.Ready {
		return LoadingMessage
	}
	return v.vp.Model.View()
}

func (v *SettingsView) StatusLine() string {
	return ""
}

func (v *SettingsView) SetSize(w, h int) tea.Cmd {
	v.screenWidth, v.screenHeight = w, h
	content := v.buildContent()
	v.vp.SetSize(w, h)
	v.vp.Model.SetContent(content)
	return nil
}

func (v *SettingsView) buildContent() string {
	var sb strings.Builder
	cfg := config.File()
	globalCfg := config.Global()

	separatorWidth := max(0, ModalWidthSettings-settingsSeparatorInset)
	separator := v.styles.separator.Render("  " + strings.Repeat("─", separatorWidth))

	valueWidth := ModalWidthSettings - settingsLabelWidth - 2

	sb.WriteString(v.styles.title.Render("Config File"))
	sb.WriteString("\n\n")
	configPath := config.GetConfigPath()
	if configPath != "" {
		sb.WriteString(fmt.Sprintf("  Path          %s\n", wrapSettingsValue(configPath, valueWidth)))
		sb.WriteString("  Type          custom\n")
	} else {
		sb.WriteString("  Path          ~/.config/claws/config.yaml (default)\n")
		sb.WriteString("  Type          default\n")
	}
	sb.WriteString("\n")
	sb.WriteString(separator)
	sb.WriteString("\n\n")

	// Section 2: Runtime
	sb.WriteString(v.styles.title.Render("Runtime"))
	sb.WriteString("\n\n")

	regionStr := strings.Join(globalCfg.Regions(), ", ")
	if regionStr == "" {
		regionStr = noneValue
	}
	sb.WriteString(fmt.Sprintf("  Regions       %s\n", wrapSettingsValue(regionStr, valueWidth)))

	profileStr := v.formatProfiles(globalCfg.Selections())
	sb.WriteString(fmt.Sprintf("  Profiles      %s\n", wrapSettingsValue(profileStr, valueWidth)))

	// Read-only
	readOnly := "no"
	if globalCfg.ReadOnly() {
		readOnly = "yes"
	}
	sb.WriteString(fmt.Sprintf("  Read-only     %s\n", readOnly))

	compactHeader := "no"
	if globalCfg.CompactHeader() {
		compactHeader = "yes"
	}
	sb.WriteString(fmt.Sprintf("  Compact       %s\n", compactHeader))

	sb.WriteString("\n")
	sb.WriteString(separator)
	sb.WriteString("\n\n")

	// Section 3: Theme
	sb.WriteString(v.styles.title.Render("Theme"))
	sb.WriteString("\n\n")

	theme := cfg.GetTheme()
	preset := theme.Preset
	if preset == "" {
		preset = noneValue
	}
	sb.WriteString(fmt.Sprintf("  Preset        %s\n", preset))

	// Show overridden colors
	overrides := v.getThemeOverrides(theme)
	if len(overrides) > 0 {
		sb.WriteString("\n")
		for _, override := range overrides {
			sb.WriteString(fmt.Sprintf("  + %s\n", override))
		}
	}

	sb.WriteString("\n")
	sb.WriteString(separator)
	sb.WriteString("\n\n")

	// Section 5: Timeouts
	sb.WriteString(v.styles.title.Render("Timeouts"))
	sb.WriteString("\n\n")
	sb.WriteString(fmt.Sprintf("  AWS Init      %s\n", cfg.Timeouts.AWSInit.Duration().String()))
	sb.WriteString(fmt.Sprintf("  Multi-region  %s\n", cfg.Timeouts.MultiRegionFetch.Duration().String()))
	sb.WriteString(fmt.Sprintf("  Tag search    %s\n", cfg.Timeouts.TagSearch.Duration().String()))
	sb.WriteString(fmt.Sprintf("  Metrics load  %s\n", cfg.Timeouts.MetricsLoad.Duration().String()))
	sb.WriteString(fmt.Sprintf("  Log fetch     %s\n", cfg.Timeouts.LogFetch.Duration().String()))
	sb.WriteString(fmt.Sprintf("  Docs search   %s\n", cfg.Timeouts.DocsSearch.Duration().String()))
	sb.WriteString("\n")
	sb.WriteString(separator)
	sb.WriteString("\n\n")

	// Section 6: Concurrency
	sb.WriteString(v.styles.title.Render("Concurrency"))
	sb.WriteString("\n\n")
	sb.WriteString(fmt.Sprintf("  Max fetches   %d\n", cfg.Concurrency.MaxFetches))
	sb.WriteString("\n")
	sb.WriteString(separator)
	sb.WriteString("\n\n")

	// Section 7: CloudWatch
	sb.WriteString(v.styles.title.Render("CloudWatch"))
	sb.WriteString("\n\n")
	sb.WriteString(fmt.Sprintf("  Metrics window  %s\n", cfg.CloudWatch.Window.Duration().String()))
	sb.WriteString("\n")
	sb.WriteString(separator)
	sb.WriteString("\n\n")

	// Section 8: Navigation
	sb.WriteString(v.styles.title.Render("Navigation"))
	sb.WriteString("\n\n")
	sb.WriteString(fmt.Sprintf("  Max stack size  %d\n", cfg.Navigation.MaxStackSize))
	sb.WriteString("\n")
	sb.WriteString(separator)
	sb.WriteString("\n\n")

	// Section 9: Autosave
	sb.WriteString(v.styles.title.Render("Autosave"))
	sb.WriteString("\n\n")
	enabled := "no"
	if cfg.Autosave.Enabled {
		enabled = "yes"
	}
	sb.WriteString(fmt.Sprintf("  Enabled       %s\n", enabled))
	sb.WriteString("\n")
	sb.WriteString(separator)
	sb.WriteString("\n\n")

	// Section 10: AI
	sb.WriteString(v.styles.title.Render("AI"))
	sb.WriteString("\n\n")

	aiProfile := cfg.AI.Profile
	if aiProfile == "" {
		aiProfile = "(default)"
	}
	sb.WriteString(fmt.Sprintf("  Profile       %s\n", aiProfile))

	aiRegion := cfg.AI.Region
	if aiRegion == "" {
		aiRegion = "(default)"
	}
	sb.WriteString(fmt.Sprintf("  Region        %s\n", aiRegion))

	sb.WriteString(fmt.Sprintf("  Model         %s\n", wrapSettingsValue(cfg.GetAIModel(), valueWidth)))
	sb.WriteString(fmt.Sprintf("  Max sessions  %d\n", cfg.GetAIMaxSessions()))
	sb.WriteString(fmt.Sprintf("  Max tokens    %d\n", cfg.GetAIMaxTokens()))
	sb.WriteString(fmt.Sprintf("  Thinking budget  %d\n", cfg.GetAIThinkingBudget()))
	sb.WriteString(fmt.Sprintf("  Max tool rounds  %d\n", cfg.GetAIMaxToolRounds()))
	sb.WriteString(fmt.Sprintf("  Max tool calls   %d\n", cfg.GetAIMaxToolCallsPerQuery()))

	saveSessions := "no"
	if cfg.GetAISaveSessions() {
		saveSessions = "yes"
	}
	sb.WriteString(fmt.Sprintf("  Save sessions    %s\n", saveSessions))

	return v.styles.text.Render(sb.String())
}

func (v *SettingsView) getThemeOverrides(theme config.ThemeConfig) []string {
	var overrides []string

	if theme.Primary != "" {
		overrides = append(overrides, fmt.Sprintf("Primary: %s", theme.Primary))
	}
	if theme.Secondary != "" {
		overrides = append(overrides, fmt.Sprintf("Secondary: %s", theme.Secondary))
	}
	if theme.Accent != "" {
		overrides = append(overrides, fmt.Sprintf("Accent: %s", theme.Accent))
	}
	if theme.Text != "" {
		overrides = append(overrides, fmt.Sprintf("Text: %s", theme.Text))
	}
	if theme.TextBright != "" {
		overrides = append(overrides, fmt.Sprintf("TextBright: %s", theme.TextBright))
	}
	if theme.TextDim != "" {
		overrides = append(overrides, fmt.Sprintf("TextDim: %s", theme.TextDim))
	}
	if theme.TextMuted != "" {
		overrides = append(overrides, fmt.Sprintf("TextMuted: %s", theme.TextMuted))
	}
	if theme.Success != "" {
		overrides = append(overrides, fmt.Sprintf("Success: %s", theme.Success))
	}
	if theme.Warning != "" {
		overrides = append(overrides, fmt.Sprintf("Warning: %s", theme.Warning))
	}
	if theme.Danger != "" {
		overrides = append(overrides, fmt.Sprintf("Danger: %s", theme.Danger))
	}
	if theme.Info != "" {
		overrides = append(overrides, fmt.Sprintf("Info: %s", theme.Info))
	}
	if theme.Pending != "" {
		overrides = append(overrides, fmt.Sprintf("Pending: %s", theme.Pending))
	}
	if theme.Border != "" {
		overrides = append(overrides, fmt.Sprintf("Border: %s", theme.Border))
	}
	if theme.BorderHighlight != "" {
		overrides = append(overrides, fmt.Sprintf("BorderHighlight: %s", theme.BorderHighlight))
	}
	if theme.Background != "" {
		overrides = append(overrides, fmt.Sprintf("Background: %s", theme.Background))
	}
	if theme.BackgroundAlt != "" {
		overrides = append(overrides, fmt.Sprintf("BackgroundAlt: %s", theme.BackgroundAlt))
	}
	if theme.Selection != "" {
		overrides = append(overrides, fmt.Sprintf("Selection: %s", theme.Selection))
	}
	if theme.SelectionText != "" {
		overrides = append(overrides, fmt.Sprintf("SelectionText: %s", theme.SelectionText))
	}
	if theme.TableHeader != "" {
		overrides = append(overrides, fmt.Sprintf("TableHeader: %s", theme.TableHeader))
	}
	if theme.TableHeaderText != "" {
		overrides = append(overrides, fmt.Sprintf("TableHeaderText: %s", theme.TableHeaderText))
	}
	if theme.TableBorder != "" {
		overrides = append(overrides, fmt.Sprintf("TableBorder: %s", theme.TableBorder))
	}
	if theme.BadgeForeground != "" {
		overrides = append(overrides, fmt.Sprintf("BadgeForeground: %s", theme.BadgeForeground))
	}
	if theme.BadgeBackground != "" {
		overrides = append(overrides, fmt.Sprintf("BadgeBackground: %s", theme.BadgeBackground))
	}

	return overrides
}

func (v *SettingsView) formatProfiles(selections []config.ProfileSelection) string {
	if len(selections) == 0 {
		return noneValue
	}

	names := make([]string, len(selections))
	for i, sel := range selections {
		names[i] = sel.DisplayName()
	}
	return strings.Join(names, ", ")
}
