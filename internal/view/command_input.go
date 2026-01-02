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

// CommandInput handles command mode input
// commandInputStyles holds cached lipgloss styles for performance
type commandInputStyles struct {
	input      lipgloss.Style
	suggestion lipgloss.Style
	highlight  lipgloss.Style
}

func newCommandInputStyles() commandInputStyles {
	t := ui.Current()
	return commandInputStyles{
		input:      lipgloss.NewStyle().Background(t.Background).Foreground(t.Text).Padding(0, 1),
		suggestion: lipgloss.NewStyle().Foreground(t.TextDim),
		highlight:  lipgloss.NewStyle().Bold(true).Foreground(t.Accent),
	}
}

// TagCompletionProvider provides tag keys and values for completion
type TagCompletionProvider interface {
	// GetTagKeys returns all unique tag keys from current resources
	GetTagKeys() []string
	// GetTagValues returns all unique values for a specific tag key
	GetTagValues(key string) []string
}

// DiffCompletionProvider provides resource names for diff command completion
type DiffCompletionProvider interface {
	// GetResourceNames returns all resource names for completion
	GetResourceNames() []string
	// GetMarkedResourceName returns the marked resource name (empty if none)
	GetMarkedResourceName() string
}

type CommandInput struct {
	ctx         context.Context
	registry    *registry.Registry
	textInput   textinput.Model
	active      bool
	width       int
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
	ti.CharLimit = 50
	ti.SetWidth(30)

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
		case "esc":
			c.Deactivate()
			return nil, nil

		case "enter":
			cmd, nav := c.executeCommand()
			c.Deactivate()
			return cmd, nav

		case "tab":
			// Cycle through suggestions
			if len(c.suggestions) > 0 {
				c.textInput.SetValue(c.suggestions[c.suggIdx])
				c.suggIdx = (c.suggIdx + 1) % len(c.suggestions)
			} else {
				// Get fresh suggestions
				c.updateSuggestions()
				if len(c.suggestions) > 0 {
					c.textInput.SetValue(c.suggestions[0])
					c.suggIdx = 1 % len(c.suggestions)
				}
			}
			return nil, nil

		case "shift+tab":
			// Cycle backward through suggestions
			if len(c.suggestions) > 0 {
				c.suggIdx = (c.suggIdx - 1 + len(c.suggestions)) % len(c.suggestions)
				c.textInput.SetValue(c.suggestions[c.suggIdx])
			}
			return nil, nil
		}
	}

	var cmd tea.Cmd
	c.textInput, cmd = c.textInput.Update(msg)

	// Update suggestions on input change
	c.updateSuggestions()

	return cmd, nil
}

func (c *CommandInput) updateSuggestions() {
	c.suggestions = c.GetSuggestions()
	c.suggIdx = 0
}

// View renders the command input
func (c *CommandInput) View() string {
	if !c.active {
		return ""
	}

	s := c.styles
	result := s.input.Render(c.textInput.View())

	// Show suggestions
	if len(c.suggestions) > 0 && c.textInput.Value() != "" {
		maxShow := 5
		if len(c.suggestions) < maxShow {
			maxShow = len(c.suggestions)
		}

		suggText := " → "
		for i := 0; i < maxShow; i++ {
			if i > 0 {
				suggText += " | "
			}
			if i == c.suggIdx%len(c.suggestions) {
				suggText += s.highlight.Render(c.suggestions[i])
			} else {
				suggText += c.suggestions[i]
			}
		}
		if len(c.suggestions) > maxShow {
			suggText += " ..."
		}
		result += s.suggestion.Render(suggText)
	}

	return result
}

// SetWidth sets the input width
func (c *CommandInput) SetWidth(width int) {
	c.width = width
	c.textInput.SetWidth(width - 4)
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

	// Empty input or home/pulse - go to dashboard
	if input == "" || input == "home" || input == "pulse" {
		dashboard := NewDashboardView(c.ctx, c.registry)
		return nil, &NavigateMsg{View: dashboard, ClearStack: true}
	}

	// Handle quit command
	if input == "q" || input == "quit" {
		return tea.Quit, nil
	}

	// Handle services/browse command - go to service browser
	if input == "services" || input == "browse" {
		browser := NewServiceBrowser(c.ctx, c.registry)
		return nil, &NavigateMsg{View: browser, ClearStack: true}
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
			// :diff <name> - compare current row with named resource
			return func() tea.Msg {
				return DiffMsg{LeftName: "", RightName: parts[0]}
			}, nil
		} else if len(parts) >= 2 {
			// :diff <name1> <name2> - compare two named resources
			return func() tea.Msg {
				return DiffMsg{LeftName: parts[0], RightName: parts[1]}
			}, nil
		}
	}

	// Parse command: service or service/resource
	parts := strings.SplitN(input, "/", 2)
	service := parts[0]
	resourceType := ""

	if len(parts) > 1 {
		resourceType = parts[1]
	}

	// Try to resolve alias first (e.g., "cfn" -> "cloudformation")
	if resolvedService, resolvedResource, ok := c.registry.ResolveAlias(service); ok {
		service = resolvedService
		if resolvedResource != "" && resourceType == "" {
			resourceType = resolvedResource
		}
	}

	// If no resource specified, use first available
	if resourceType == "" {
		resources := c.registry.ListResources(service)
		if len(resources) > 0 {
			resourceType = resources[0]
		}
	}

	// Check if service/resource exists
	if _, ok := c.registry.Get(service, resourceType); !ok {
		// Try to find partial match
		for _, svc := range c.registry.ListServices() {
			if strings.HasPrefix(svc, service) {
				service = svc
				resources := c.registry.ListResources(svc)
				if len(resources) > 0 {
					if resourceType == "" {
						resourceType = resources[0]
					} else {
						// Find matching resource
						for _, res := range resources {
							if strings.HasPrefix(res, resourceType) {
								resourceType = res
								break
							}
						}
					}
				}
				break
			}
		}
	}

	if _, ok := c.registry.Get(service, resourceType); ok {
		browser := NewResourceBrowserWithType(c.ctx, c.registry, service, resourceType)
		return nil, &NavigateMsg{View: browser}
	}

	return nil, nil
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
		if strings.HasPrefix("login", input) {
			suggestions = append(suggestions, "login")
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

		for _, svc := range c.registry.ListServices() {
			if strings.HasPrefix(svc, input) {
				suggestions = append(suggestions, svc)
			}
		}

		for _, alias := range c.registry.GetAliases() {
			if strings.HasPrefix(alias, input) {
				suggestions = append(suggestions, alias)
			}
		}

		slices.Sort(suggestions)
	}

	return suggestions
}

func (c *CommandInput) getDiffSuggestions(args string) []string {
	if c.diffProvider == nil {
		return nil
	}

	names := c.diffProvider.GetResourceNames()
	parts := strings.SplitN(args, " ", 2)

	if len(parts) == 2 {
		firstName := parts[0]
		secondPrefix := strings.ToLower(parts[1])

		var filtered []string
		for _, name := range names {
			if name != firstName {
				filtered = append(filtered, name)
			}
		}

		matched := matchNamesWithFallback(filtered, secondPrefix)
		var suggestions []string
		for _, name := range matched {
			suggestions = append(suggestions, "diff "+firstName+" "+name)
		}
		return suggestions
	}

	prefix := strings.ToLower(args)
	matched := matchNamesWithFallback(names, prefix)
	var suggestions []string
	for _, name := range matched {
		suggestions = append(suggestions, "diff "+name)
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
