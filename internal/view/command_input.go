package view

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/config"
	navmsg "github.com/clawscli/claws/internal/msg"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/ui"
)

const (
	commandInputWidth1 = 15
	commandInputWidth2 = 30
	commandInputWidth3 = 60
	commandInputWidth4 = 90
)

// CommandInput handles command mode input
// commandInputStyles holds cached lipgloss styles for performance
type commandInputStyles struct {
	input      lipgloss.Style
	suggestion lipgloss.Style
	highlight  lipgloss.Style
	alias      lipgloss.Style
}

func newCommandInputStyles() commandInputStyles {
	return commandInputStyles{
		input:      ui.InputFieldStyle(),
		suggestion: ui.DimStyle(),
		highlight:  ui.HighlightStyle(),
		alias:      ui.TextStyle(),
	}
}

// TagCompletionProvider provides tag keys and values for completion
type TagCompletionProvider interface {
	// GetTagKeys returns all unique tag keys from current resources
	GetTagKeys() []string
	// GetTagValues returns all unique values for a specific tag key
	GetTagValues(key string) []string
}

// DiffCompletionProvider provides resource IDs for diff command completion
type DiffCompletionProvider interface {
	// GetResourceIDs returns all resource IDs for completion
	GetResourceIDs() []string
	// GetMarkedResourceID returns the marked resource ID (empty if none)
	GetMarkedResourceID() string
}

type CommandInput struct {
	ctx         context.Context
	registry    *registry.Registry
	textInput   textinput.Model
	active      bool
	suggestions []string
	suggIdx     int
	styles      commandInputStyles

	// Tag completion
	tagProvider TagCompletionProvider
	// Diff completion
	diffProvider DiffCompletionProvider
}

// NewCommandInput creates a new CommandInput
func NewCommandInput(ctx context.Context, reg *registry.Registry) *CommandInput {
	ti := textinput.New()
	ti.Placeholder = "service/resource"
	ti.Prompt = ":"
	ti.CharLimit = 150
	ti.SetWidth(commandInputWidth1)
	ti.SetStyles(ui.TextInputStyles())

	return &CommandInput{
		ctx:       ctx,
		registry:  reg,
		textInput: ti,
		styles:    newCommandInputStyles(),
	}
}

// Activate activates command mode
func (c *CommandInput) Activate() tea.Cmd {
	c.active = true
	c.textInput.SetValue("")
	c.textInput.Focus()
	c.suggestions = nil
	c.suggIdx = 0
	return textinput.Blink
}

func (c *CommandInput) ReloadStyles() {
	c.styles = newCommandInputStyles()
	c.textInput.SetStyles(ui.TextInputStyles())
}

// Deactivate deactivates command mode
func (c *CommandInput) Deactivate() {
	c.active = false
	c.textInput.Blur()
	c.suggestions = nil
}

// IsActive returns whether command mode is active
func (c *CommandInput) IsActive() bool {
	return c.active
}

// Update handles input updates
func (c *CommandInput) Update(msg tea.Msg) (tea.Cmd, *NavigateMsg) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			c.Deactivate()
			return nil, nil

		case "enter":
			cmd, nav := c.executeCommand()
			c.Deactivate()
			return cmd, nav

		case "tab":
			// Bash-style completion: common prefix first, then cycle
			if len(c.suggestions) == 0 {
				c.updateSuggestions()
			}
			if len(c.suggestions) > 0 {
				current := c.textInput.Value()
				prefix := commonPrefix(c.suggestions)

				if len(prefix) > len(current) {
					// Expand to common prefix
					c.textInput.Reset()
					c.textInput.SetValue(prefix)
					c.suggIdx = 0
				} else {
					// Common prefix = current input, cycle through suggestions
					c.textInput.Reset()
					c.textInput.SetValue(c.suggestions[c.suggIdx])
					c.suggIdx = (c.suggIdx + 1) % len(c.suggestions)
				}
				c.updateWidth()
			}
			return nil, nil

		case "shift+tab":
			// Bash-style completion: common prefix first, then cycle backward
			if len(c.suggestions) == 0 {
				c.updateSuggestions()
			}
			if len(c.suggestions) > 0 {
				current := c.textInput.Value()
				prefix := commonPrefix(c.suggestions)

				if len(prefix) > len(current) {
					// Expand to common prefix
					c.textInput.Reset()
					c.textInput.SetValue(prefix)
					c.suggIdx = 0
				} else {
					// Common prefix = current input, cycle backward
					c.suggIdx = (c.suggIdx - 1 + len(c.suggestions)) % len(c.suggestions)
					c.textInput.Reset()
					c.textInput.SetValue(c.suggestions[c.suggIdx])
				}
				c.updateWidth()
			}
			return nil, nil
		}
	}

	var cmd tea.Cmd
	c.textInput, cmd = c.textInput.Update(msg)

	// Update suggestions on input change
	c.updateSuggestions()

	c.updateWidth()

	return cmd, nil
}

func (c *CommandInput) updateSuggestions() {
	c.suggestions = c.GetSuggestions()
	c.suggIdx = 0
}

// currentThreshold returns the width threshold for the given input length
func currentThreshold(inputLen int) int {
	switch {
	case inputLen >= commandInputWidth3:
		return commandInputWidth4
	case inputLen >= commandInputWidth2:
		return commandInputWidth3
	case inputLen >= commandInputWidth1:
		return commandInputWidth2
	default:
		return commandInputWidth1
	}
}

// updateWidth adjusts input width based on current input length (4-stage: 15 → 30 → 60 → 90)
func (c *CommandInput) updateWidth() {
	newWidth := currentThreshold(len(c.textInput.Value()))
	c.textInput.SetWidth(newWidth)
	// Re-set value to reset display offset after width change
	c.textInput.SetValue(c.textInput.Value())
}

// renderInputWithSuggestion renders textinput with fish-style inline suggestion
func (c *CommandInput) renderInputWithSuggestion(s commandInputStyles, input string) string {
	baseView := c.textInput.View()

	// No suggestion for empty input
	if input == "" {
		return s.input.Render(baseView)
	}

	// Find first suggestion that extends current input
	var suffix string
	for _, sugg := range c.suggestions {
		if strings.HasPrefix(sugg, input) && len(sugg) > len(input) {
			suffix = sugg[len(input):]
			break
		}
	}

	if suffix == "" {
		return s.input.Render(baseView)
	}

	// Calculate remaining space within threshold
	threshold := currentThreshold(len(input))
	remaining := threshold - len(input)
	if remaining <= 0 {
		return s.input.Render(baseView)
	}

	// Truncate suffix to fit within threshold
	if len(suffix) > remaining {
		suffix = suffix[:remaining]
	}

	// Remove trailing padding from textinput so suggestion appears right after cursor
	trimmedView := strings.TrimRight(baseView, " ")

	// Restore padding after suffix to maintain original width
	removedPadding := len(baseView) - len(trimmedView)
	paddingNeeded := removedPadding - len(suffix)
	if paddingNeeded < 0 {
		paddingNeeded = 0
	}

	// Render: trimmed input + dim suffix + padding (within same input box style)
	return s.input.Render(trimmedView + s.suggestion.Render(suffix) + strings.Repeat(" ", paddingNeeded))
}

// View renders the command input
func (c *CommandInput) View() string {
	if !c.active {
		return ""
	}

	s := c.styles
	input := c.textInput.Value()

	// Fish-style inline suggestion: show completion suffix within threshold
	inputView := c.renderInputWithSuggestion(s, input)

	// Calculate where Enter will navigate to (alias resolution or prefix match)
	destination := c.resolveDestination(input)

	// Build view: destination (highlighted) + other suggestions (dim)
	var destView, suggView string

	if destination != "" {
		destView = s.alias.Render(" → " + s.highlight.Render(destination))
	}

	// Show other suggestions (white, no highlight)
	if len(c.suggestions) > 0 && input != "" {
		maxShow := 5
		shown := 0
		var parts []string

		for _, sugg := range c.suggestions {
			if sugg == destination {
				continue // Skip duplicate
			}
			if shown >= maxShow {
				break
			}
			parts = append(parts, sugg)
			shown++
		}

		if len(parts) > 0 {
			var suggText string
			if destView != "" {
				suggText = " | " + strings.Join(parts, " | ")
			} else {
				suggText = " → " + strings.Join(parts, " | ")
			}
			if len(c.suggestions) > maxShow+1 {
				suggText += " ..."
			}
			suggView = s.alias.Render(suggText)
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, inputView, destView, suggView)
}

// resolveDestination returns where Enter will navigate to for the given input.
// It uses the same logic as executeCommand: alias resolution, then prefix match.
func (c *CommandInput) resolveDestination(input string) string {
	if input == "" {
		return ""
	}

	// Skip non-navigation commands
	if strings.HasPrefix(input, "tag ") || strings.HasPrefix(input, "tags ") ||
		strings.HasPrefix(input, "diff ") || strings.HasPrefix(input, "sort ") ||
		strings.HasPrefix(input, "theme ") || strings.HasPrefix(input, "autosave ") ||
		strings.HasPrefix(input, "login ") {
		return ""
	}

	// Try alias resolution first
	if service, resource, ok := c.registry.ResolveAlias(input); ok {
		if resource != "" {
			return service + "/" + resource
		}
		return service
	}

	// If input contains "/", try ParseServiceResource for full path
	if strings.Contains(input, "/") {
		parts := strings.SplitN(input, "/", 2)
		resourcePart := ""
		if len(parts) > 1 {
			resourcePart = parts[1]
		}
		if service, resourceType, err := c.registry.ParseServiceResource(input); err == nil {
			// Only show resource if user typed something after "/"
			if resourcePart != "" && resourceType != "" {
				return service + "/" + resourceType
			}
			return service
		}
	}

	// Fallback: prefix match on service/alias
	if svc, res, ok := c.resolvePrefixMatch(input); ok {
		if res != "" {
			return svc + "/" + res
		}
		return svc
	}

	return ""
}

// resolvePrefixMatch tries prefix match on services, aliases, and resources.
// Returns resolved service/resource. Empty strings if no match found.
func (c *CommandInput) resolvePrefixMatch(input string) (service, resource string, ok bool) {
	parts := strings.SplitN(input, "/", 2)
	servicePart := parts[0]
	resourcePart := ""
	if len(parts) > 1 {
		resourcePart = parts[1]
	}

	// Try prefix match on service name
	var matchedService string
	for _, svc := range c.registry.ListServices() {
		if strings.HasPrefix(svc, servicePart) {
			matchedService = svc
			break
		}
	}

	// Try prefix match on alias if no service matched
	var aliasResource string
	if matchedService == "" {
		for _, alias := range c.registry.GetAliases() {
			if strings.HasPrefix(alias, servicePart) {
				// Resolve alias to service (and resource if alias includes it)
				if resolved, res, resolveOK := c.registry.ResolveAlias(alias); resolveOK {
					matchedService = resolved
					aliasResource = res
					break
				}
			}
		}
	}

	if matchedService == "" {
		return "", "", false
	}

	// If no resource part specified, use alias resource (if any) or let caller use default
	if resourcePart == "" {
		return matchedService, aliasResource, true
	}

	// Try prefix match on resource name (sorted, so first match = alphabetically first)
	for _, res := range c.registry.ListResources(matchedService) {
		if strings.HasPrefix(res, resourcePart) {
			return matchedService, res, true
		}
	}

	// No matching resource found
	return "", "", false
}

// SetTagProvider sets the tag completion provider
func (c *CommandInput) SetTagProvider(provider TagCompletionProvider) {
	c.tagProvider = provider
}

// SetDiffProvider sets the diff completion provider
func (c *CommandInput) SetDiffProvider(provider DiffCompletionProvider) {
	c.diffProvider = provider
}

func (c *CommandInput) executeCommand() (tea.Cmd, *NavigateMsg) {
	input := strings.TrimSpace(c.textInput.Value())

	// Empty input or home - go to service browser (new default home)
	if input == "" || input == "home" {
		browser := NewServiceBrowser(c.ctx, c.registry)
		return nil, &NavigateMsg{View: browser, ClearStack: false}
	}

	// Handle pulse command - go to dashboard
	if input == "pulse" {
		dashboard := NewDashboardView(c.ctx, c.registry)
		return nil, &NavigateMsg{View: dashboard, ClearStack: false}
	}

	// Handle quit command
	if input == "q" || input == "quit" {
		return tea.Quit, nil
	}

	// Handle clear-history command - clear navigation stack
	if input == "clear-history" {
		return func() tea.Msg {
			return ClearHistoryMsg{}
		}, nil
	}

	// Handle dashboard command - explicitly open dashboard
	if input == "dashboard" {
		dashboard := NewDashboardView(c.ctx, c.registry)
		return nil, &NavigateMsg{View: dashboard, ClearStack: false}
	}

	// Handle services/browse command - go to service browser
	if input == "services" || input == "browse" {
		browser := NewServiceBrowser(c.ctx, c.registry)
		return nil, &NavigateMsg{View: browser, ClearStack: false}
	}

	// Handle settings command - show settings modal
	if input == "settings" {
		return func() tea.Msg {
			return ShowModalMsg{
				Modal: &Modal{
					Content: NewSettingsView(c.ctx),
					Width:   ModalWidthSettings,
				},
			}
		}, nil
	}

	// Handle sort command: :sort (clear) or :sort <column> (sort by column)
	if input == "sort" {
		return func() tea.Msg {
			return SortMsg{Column: "", Ascending: true}
		}, nil
	}
	if suffix, ok := strings.CutPrefix(input, "sort "); ok {
		return c.parseSortArgs(suffix), nil
	}

	// Handle login command: :login (default) or :login <profile>
	if input == "login" {
		return c.executeLogin("claws-login"), nil
	}
	if suffix, ok := strings.CutPrefix(input, "login "); ok {
		profileName := strings.TrimSpace(suffix)
		if profileName == "" {
			return c.executeLogin("claws-login"), nil
		}
		if !config.IsValidProfileName(profileName) {
			return func() tea.Msg {
				return ErrorMsg{Err: fmt.Errorf("invalid profile name: %q", profileName)}
			}, nil
		}
		return c.executeLogin(profileName), nil
	}

	// Handle tag command: :tag (clear) or :tag <filter> (filter by tag)
	if input == "tag" {
		return func() tea.Msg {
			return TagFilterMsg{Filter: ""}
		}, nil
	}
	if tagFilter, ok := strings.CutPrefix(input, "tag "); ok {
		return func() tea.Msg {
			return TagFilterMsg{Filter: tagFilter}
		}, nil
	}

	// Handle tags command: :tags (all) or :tags <filter> (cross-service tag search)
	if input == "tags" {
		browser := NewTagSearchView(c.ctx, c.registry, "")
		return nil, &NavigateMsg{View: browser}
	}
	if tagFilter, ok := strings.CutPrefix(input, "tags "); ok {
		browser := NewTagSearchView(c.ctx, c.registry, tagFilter)
		return nil, &NavigateMsg{View: browser}
	}

	// Handle diff command: :diff <name> or :diff <name1> <name2>
	if suffix, ok := strings.CutPrefix(input, "diff "); ok {
		parts := strings.Fields(suffix)
		if len(parts) == 1 {
			return func() tea.Msg {
				return DiffMsg{LeftID: "", RightID: parts[0]}
			}, nil
		} else if len(parts) >= 2 {
			return func() tea.Msg {
				return DiffMsg{LeftID: parts[0], RightID: parts[1]}
			}, nil
		}
	}

	if suffix, ok := strings.CutPrefix(input, "theme "); ok {
		themeName := strings.TrimSpace(suffix)
		if themeName != "" {
			return func() tea.Msg {
				return ThemeChangeMsg{Name: themeName}
			}, nil
		}
	}

	if suffix, ok := strings.CutPrefix(input, "autosave "); ok {
		switch strings.TrimSpace(suffix) {
		case "on":
			return func() tea.Msg {
				return PersistenceChangeMsg{Enabled: true}
			}, nil
		case "off":
			return func() tea.Msg {
				return PersistenceChangeMsg{Enabled: false}
			}, nil
		}
	}

	// Try ParseServiceResource first (handles aliases, defaults, validation)
	service, resourceType, err := c.registry.ParseServiceResource(input)
	if err == nil {
		browser := NewResourceBrowserWithType(c.ctx, c.registry, service, resourceType)
		return nil, &NavigateMsg{View: browser}
	}

	// Fallback: prefix matching for partial input
	if svc, res, ok := c.resolvePrefixMatch(input); ok {
		if res == "" {
			res = c.registry.DefaultResource(svc)
		}
		browser := NewResourceBrowserWithType(c.ctx, c.registry, svc, res)
		return nil, &NavigateMsg{View: browser}
	}

	return func() tea.Msg {
		return ErrorMsg{Err: fmt.Errorf("unknown command: %s", input)}
	}, nil
}

func (c *CommandInput) parseSortArgs(args string) tea.Cmd {
	ascending := true
	column := args

	if col, ok := strings.CutPrefix(args, "desc "); ok {
		ascending = false
		column = col
	} else if col, ok := strings.CutPrefix(args, "asc "); ok {
		column = col
	}

	return func() tea.Msg {
		return SortMsg{Column: strings.TrimSpace(column), Ascending: ascending}
	}
}

func (c *CommandInput) executeLogin(profileName string) tea.Cmd {
	exec := &action.SimpleExec{
		Command:    fmt.Sprintf("aws login --remote --profile %s", profileName),
		ActionName: action.ActionNameLogin,
		SkipAWSEnv: true,
	}
	return tea.Exec(exec, func(err error) tea.Msg {
		if err != nil {
			return ErrorMsg{Err: err}
		}
		sel := config.NamedProfile(profileName)
		config.Global().SetSelections([]config.ProfileSelection{sel})
		return navmsg.ProfilesChangedMsg{Selections: []config.ProfileSelection{sel}}
	})
}

// GetSuggestions returns command suggestions based on current input
func (c *CommandInput) GetSuggestions() []string {
	input := c.textInput.Value()
	var suggestions []string

	// Handle :tag command completion
	if suffix, ok := strings.CutPrefix(input, "tag "); ok {
		return c.getTagSuggestions("tag ", suffix)
	}

	// Handle :tags command completion (same as :tag)
	if suffix, ok := strings.CutPrefix(input, "tags "); ok {
		return c.getTagSuggestions("tags ", suffix)
	}

	// Handle :diff command completion
	if suffix, ok := strings.CutPrefix(input, "diff "); ok {
		return c.getDiffSuggestions(suffix)
	}

	if suffix, ok := strings.CutPrefix(input, "theme "); ok {
		return c.getThemeSuggestions(suffix)
	}

	if suffix, ok := strings.CutPrefix(input, "autosave "); ok {
		return c.getAutosaveSuggestions(suffix)
	}

	if strings.Contains(input, "/") {
		// Suggest resources
		parts := strings.SplitN(input, "/", 2)
		service := parts[0]
		prefix := ""
		if len(parts) > 1 {
			prefix = parts[1]
		}

		for _, res := range c.registry.ListResources(service) {
			if strings.HasPrefix(res, prefix) {
				suggestions = append(suggestions, service+"/"+res)
			}
		}
	} else {
		// Suggest services and special commands
		// Add navigation commands
		if strings.HasPrefix("quit", input) {
			suggestions = append(suggestions, "quit")
		}
		if strings.HasPrefix("home", input) {
			suggestions = append(suggestions, "home")
		}
		if strings.HasPrefix("services", input) {
			suggestions = append(suggestions, "services")
		}
		if strings.HasPrefix("dashboard", input) {
			suggestions = append(suggestions, "dashboard")
		}
		if strings.HasPrefix("login", input) {
			suggestions = append(suggestions, "login")
		}
		if strings.HasPrefix("clear-history", input) {
			suggestions = append(suggestions, "clear-history")
		}

		// Add "tag" command (current view filter)
		if strings.HasPrefix("tag", input) && !strings.HasPrefix("tags", input) {
			suggestions = append(suggestions, "tag")
		}

		// Add "tags" command (cross-service browser)
		if strings.HasPrefix("tags", input) {
			suggestions = append(suggestions, "tags")
		}

		// Add "sort" command
		if strings.HasPrefix("sort", input) {
			suggestions = append(suggestions, "sort")
		}

		// Add "diff" command
		if strings.HasPrefix("diff", input) && c.diffProvider != nil {
			suggestions = append(suggestions, "diff")
		}

		if strings.HasPrefix("theme", input) {
			suggestions = append(suggestions, "theme")
		}

		if strings.HasPrefix("autosave", input) {
			suggestions = append(suggestions, "autosave")
		}

		if strings.HasPrefix("settings", input) {
			suggestions = append(suggestions, "settings")
		}

		for _, svc := range c.registry.ListServices() {
			// Skip if input exactly matches service (already fully typed)
			if svc != input && strings.HasPrefix(svc, input) {
				suggestions = append(suggestions, svc)
			}
		}

		for _, alias := range c.registry.GetAliases() {
			// Skip if input exactly matches alias (already fully typed)
			if alias != input && strings.HasPrefix(alias, input) {
				suggestions = append(suggestions, alias)
			}
		}

		slices.Sort(suggestions)
	}

	return suggestions
}

func (c *CommandInput) getThemeSuggestions(prefix string) []string {
	themes := ui.AvailableThemes()
	prefix = strings.ToLower(strings.TrimSpace(prefix))

	var suggestions []string
	for _, t := range themes {
		if prefix == "" || strings.HasPrefix(t, prefix) {
			suggestions = append(suggestions, "theme "+t)
		}
	}
	return suggestions
}

func (c *CommandInput) getAutosaveSuggestions(prefix string) []string {
	prefix = strings.ToLower(strings.TrimSpace(prefix))
	options := []string{"on", "off"}

	var suggestions []string
	for _, opt := range options {
		if prefix == "" || strings.HasPrefix(opt, prefix) {
			suggestions = append(suggestions, "autosave "+opt)
		}
	}
	return suggestions
}

func (c *CommandInput) getDiffSuggestions(args string) []string {
	if c.diffProvider == nil {
		return nil
	}

	ids := c.diffProvider.GetResourceIDs()
	parts := strings.SplitN(args, " ", 2)

	if len(parts) == 2 {
		firstID := parts[0]
		secondPrefix := strings.ToLower(parts[1])

		var filtered []string
		for _, id := range ids {
			if id != firstID {
				filtered = append(filtered, id)
			}
		}

		matched := matchNamesWithFallback(filtered, secondPrefix)
		var suggestions []string
		for _, id := range matched {
			suggestions = append(suggestions, "diff "+firstID+" "+id)
		}
		return suggestions
	}

	prefix := strings.ToLower(args)
	matched := matchNamesWithFallback(ids, prefix)
	var suggestions []string
	for _, id := range matched {
		suggestions = append(suggestions, "diff "+id)
	}
	return suggestions
}

// getTagSuggestions returns tag key/value suggestions with command prefix
func (c *CommandInput) getTagSuggestions(cmdPrefix, tagPart string) []string {
	if c.tagProvider == nil {
		return nil
	}

	var suggestions []string

	// Check if we're completing a value (after = or ~)
	if strings.Contains(tagPart, "=") {
		parts := strings.SplitN(tagPart, "=", 2)
		key := parts[0]
		valuePrefix := strings.ToLower(parts[1])

		for _, val := range c.tagProvider.GetTagValues(key) {
			if valuePrefix == "" || strings.HasPrefix(strings.ToLower(val), valuePrefix) {
				suggestions = append(suggestions, cmdPrefix+key+"="+val)
			}
		}
	} else if strings.Contains(tagPart, "~") {
		parts := strings.SplitN(tagPart, "~", 2)
		key := parts[0]
		valuePrefix := strings.ToLower(parts[1])

		for _, val := range c.tagProvider.GetTagValues(key) {
			if valuePrefix == "" || strings.HasPrefix(strings.ToLower(val), valuePrefix) {
				suggestions = append(suggestions, cmdPrefix+key+"~"+val)
			}
		}
	} else {
		// Completing a key
		keyPrefix := strings.ToLower(tagPart)
		for _, key := range c.tagProvider.GetTagKeys() {
			if keyPrefix == "" || strings.HasPrefix(strings.ToLower(key), keyPrefix) {
				suggestions = append(suggestions, cmdPrefix+key)
			}
		}
	}

	return suggestions
}

// commonPrefix returns the longest common prefix of all suggestions.
// Returns empty string if suggestions is empty.
func commonPrefix(suggestions []string) string {
	if len(suggestions) == 0 {
		return ""
	}
	if len(suggestions) == 1 {
		return suggestions[0]
	}

	prefix := suggestions[0]
	for _, s := range suggestions[1:] {
		for len(prefix) > 0 && !strings.HasPrefix(s, prefix) {
			prefix = prefix[:len(prefix)-1]
		}
		if prefix == "" {
			return ""
		}
	}
	return prefix
}
