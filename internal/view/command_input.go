package view

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
}

// NewCommandInput creates a new CommandInput
func NewCommandInput(ctx context.Context, reg *registry.Registry) *CommandInput {
	ti := textinput.New()
	ti.Placeholder = "service/resource"
	ti.Prompt = ":"
	ti.CharLimit = 50
	ti.Width = 30

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
	case tea.KeyMsg:
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
	c.textInput.Width = width - 4
}

// SetTagProvider sets the tag completion provider
func (c *CommandInput) SetTagProvider(provider TagCompletionProvider) {
	c.tagProvider = provider
}

func (c *CommandInput) executeCommand() (tea.Cmd, *NavigateMsg) {
	input := strings.TrimSpace(c.textInput.Value())

	// Empty input - go to service list (home)
	if input == "" {
		browser := NewServiceBrowser(c.ctx, c.registry)
		return nil, &NavigateMsg{View: browser, ClearStack: true}
	}

	// Handle sort command: :sort, :sort <column>, :sort desc <column>
	if input == "sort" || strings.HasPrefix(input, "sort ") {
		return c.parseSortCommand(input), nil
	}

	// Handle tag command: :tag <filter> - filter current view by tag
	if input == "tag" || strings.HasPrefix(input, "tag ") {
		tagFilter := ""
		if strings.HasPrefix(input, "tag ") {
			tagFilter = strings.TrimPrefix(input, "tag ")
		}
		return func() tea.Msg {
			return TagFilterMsg{Filter: tagFilter}
		}, nil
	}

	// Handle tags command: :tags, :tags <filter> - cross-service tag browser
	if input == "tags" || strings.HasPrefix(input, "tags ") {
		tagFilter := ""
		if strings.HasPrefix(input, "tags ") {
			tagFilter = strings.TrimPrefix(input, "tags ")
		}
		browser := NewTagBrowser(c.ctx, c.registry, tagFilter)
		return nil, &NavigateMsg{View: browser}
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

// parseSortCommand parses the sort command and returns a SortMsg command
// Syntax: :sort, :sort <column>, :sort desc <column>
func (c *CommandInput) parseSortCommand(input string) tea.Cmd {
	// :sort - clear sorting
	if input == "sort" {
		return func() tea.Msg {
			return SortMsg{Column: "", Ascending: true}
		}
	}

	// Parse arguments
	args := strings.TrimPrefix(input, "sort ")
	ascending := true
	column := args

	// Check for "desc" prefix
	if strings.HasPrefix(args, "desc ") {
		ascending = false
		column = strings.TrimPrefix(args, "desc ")
	} else if strings.HasPrefix(args, "asc ") {
		ascending = true
		column = strings.TrimPrefix(args, "asc ")
	}

	column = strings.TrimSpace(column)

	return func() tea.Msg {
		return SortMsg{Column: column, Ascending: ascending}
	}
}

// GetSuggestions returns command suggestions based on current input
func (c *CommandInput) GetSuggestions() []string {
	input := c.textInput.Value()
	var suggestions []string

	// Handle :tag command completion
	if strings.HasPrefix(input, "tag ") {
		return c.getTagSuggestions("tag ", strings.TrimPrefix(input, "tag "))
	}

	// Handle :tags command completion (same as :tag)
	if strings.HasPrefix(input, "tags ") {
		return c.getTagSuggestions("tags ", strings.TrimPrefix(input, "tags "))
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

		for _, svc := range c.registry.ListServices() {
			if strings.HasPrefix(svc, input) {
				suggestions = append(suggestions, svc)
			}
		}
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
