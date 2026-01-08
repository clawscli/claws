package view

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/clawscli/claws/internal/ai"
	"github.com/clawscli/claws/internal/config"
	apperrors "github.com/clawscli/claws/internal/errors"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/ui"
)

type chatStyles struct {
	title        lipgloss.Style
	context      lipgloss.Style
	userMsg      lipgloss.Style
	assistantMsg lipgloss.Style
	toolCall     lipgloss.Style
	toolError    lipgloss.Style
	thinking     lipgloss.Style
	input        lipgloss.Style
	errorMsg     lipgloss.Style
	mdBold       lipgloss.Style
	mdCode       lipgloss.Style
	mdItalic     lipgloss.Style
}

func newChatStyles() chatStyles {
	t := ui.Current()
	return chatStyles{
		title:        ui.TitleStyle(),
		context:      lipgloss.NewStyle().Foreground(t.TextDim).Italic(true),
		userMsg:      ui.TextStyle(),
		assistantMsg: ui.SecondaryStyle(),
		toolCall:     ui.DimStyle(),
		toolError:    ui.DangerStyle(),
		thinking:     lipgloss.NewStyle().Foreground(t.TextDim).Italic(true),
		input:        lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(t.Border).Padding(0, 1),
		errorMsg:     ui.DangerStyle(),
		mdBold:       ui.TitleStyle(),
		mdCode:       ui.SuccessStyle(),
		mdItalic:     lipgloss.NewStyle().Italic(true),
	}
}

type ChatOverlay struct {
	ctx      context.Context
	registry *registry.Registry
	aiCtx    *ai.Context
	styles   chatStyles

	client   *ai.Client
	executor *ai.ToolExecutor
	session  *ai.Session
	sessMgr  *ai.SessionManager

	input textinput.Model
	vp    ViewportState

	messages     []chatMessage
	streamingMsg string
	isThinking   bool
	err          error

	width  int
	height int
}

type chatMessage struct {
	role      ai.Role
	content   string
	toolCall  *ai.ToolCall
	toolError bool
}

type chatStreamMsg struct {
	event ai.StreamEvent
}

type chatDoneMsg struct {
	response  string
	toolCalls []chatMessage
	err       error
}

type chatInitMsg struct {
	client   *ai.Client
	executor *ai.ToolExecutor
	session  *ai.Session
	err      error
}

func NewChatOverlay(ctx context.Context, reg *registry.Registry, aiCtx *ai.Context) *ChatOverlay {
	maxSessions := config.File().GetAIMaxSessions()

	ti := textinput.New()
	ti.Placeholder = "Ask about AWS resources..."
	ti.Focus()
	ti.CharLimit = 500

	return &ChatOverlay{
		ctx:      ctx,
		registry: reg,
		aiCtx:    aiCtx,
		styles:   newChatStyles(),
		input:    ti,
		sessMgr:  ai.NewSessionManager(maxSessions),
		messages: []chatMessage{},
	}
}

func (c *ChatOverlay) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		c.initClient,
	)
}

func (c *ChatOverlay) initClient() tea.Msg {
	executor, err := ai.NewToolExecutor(c.ctx, c.registry)
	if err != nil {
		return chatInitMsg{err: apperrors.Wrap(err, "init tool executor")}
	}

	client, err := ai.NewClient(
		c.ctx,
		ai.WithModel(config.File().GetAIModel()),
		ai.WithTools(executor.Tools()),
	)
	if err != nil {
		return chatInitMsg{err: apperrors.Wrap(err, "init ai client")}
	}

	session, err := c.sessMgr.NewSession(c.aiCtx)
	if err != nil {
		return chatInitMsg{err: apperrors.Wrap(err, "create session")}
	}

	return chatInitMsg{client: client, executor: executor, session: session}
}

func (c *ChatOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case chatInitMsg:
		if msg.err != nil {
			c.err = msg.err
		} else {
			c.client = msg.client
			c.executor = msg.executor
			c.session = msg.session
		}
		return c, nil

	case tea.KeyPressMsg:
		return c.handleKeyPress(msg)

	case chatStreamMsg:
		return c.handleStreamEvent(msg.event)

	case chatDoneMsg:
		c.isThinking = false
		if msg.err != nil {
			c.err = msg.err
		}
		c.messages = append(c.messages, msg.toolCalls...)
		if msg.response != "" {
			c.messages = append(c.messages, chatMessage{role: ai.RoleAssistant, content: msg.response})
		}
		c.updateViewport()
		return c, nil
	}

	var cmds []tea.Cmd

	if c.vp.Ready {
		var cmd tea.Cmd
		c.vp.Model, cmd = c.vp.Model.Update(msg)
		cmds = append(cmds, cmd)
	}

	var cmd tea.Cmd
	c.input, cmd = c.input.Update(msg)
	cmds = append(cmds, cmd)

	return c, tea.Batch(cmds...)
}

func (c *ChatOverlay) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if IsEscKey(msg) {
		return c, func() tea.Msg { return HideModalMsg{} }
	}

	switch msg.String() {
	case "ctrl+c":
		return c, func() tea.Msg { return HideModalMsg{} }
	case "enter":
		if c.isThinking {
			return c, nil
		}

		text := strings.TrimSpace(c.input.Value())
		if text == "" {
			return c, nil
		}

		c.input.SetValue("")
		c.messages = append(c.messages, chatMessage{role: ai.RoleUser, content: text})
		c.isThinking = true
		c.streamingMsg = ""
		c.err = nil
		c.updateViewport()

		return c, c.sendMessage()
	}

	var cmd tea.Cmd
	c.input, cmd = c.input.Update(msg)
	return c, cmd
}

func (c *ChatOverlay) sendMessage() tea.Cmd {
	return func() tea.Msg {
		if c.client == nil || c.executor == nil {
			return chatDoneMsg{err: fmt.Errorf("client not initialized")}
		}

		var messages []ai.Message
		for _, m := range c.messages {
			// Only send user/assistant messages to API (skip tool call display messages)
			if m.toolCall == nil {
				messages = append(messages, ai.Message{Role: m.role, Content: m.content})
			}
		}

		var toolCallMsgs []chatMessage
		systemPrompt := c.buildSystemPrompt()
		result, err := c.client.Converse(c.ctx, messages, systemPrompt)
		if err != nil {
			return chatDoneMsg{err: err}
		}

		const maxToolRounds = 5
		for round := 0; len(result.ToolCalls) > 0 && round < maxToolRounds; round++ {
			var toolResults []ai.ToolResult
			for _, tc := range result.ToolCalls {
				tcCopy := tc
				tr := c.executor.Execute(c.ctx, tc)
				toolResults = append(toolResults, tr)
				toolCallMsgs = append(toolCallMsgs, chatMessage{
					content:   tr.Content,
					toolCall:  &tcCopy,
					toolError: tr.IsError,
				})
			}

			messages = c.client.AddToolResultMessages(messages, result.ToolCalls, toolResults)

			result, err = c.client.Converse(c.ctx, messages, systemPrompt)
			if err != nil {
				return chatDoneMsg{err: err, toolCalls: toolCallMsgs}
			}
		}

		return chatDoneMsg{response: result.Message, toolCalls: toolCallMsgs}
	}
}

func (c *ChatOverlay) handleStreamEvent(event ai.StreamEvent) (tea.Model, tea.Cmd) {
	switch event.Type {
	case "text":
		c.streamingMsg += event.Text
		c.updateViewport()

	case "done":
		if event.Text != "" {
			c.messages = append(c.messages, chatMessage{role: ai.RoleAssistant, content: event.Text})
		} else if c.streamingMsg != "" {
			c.messages = append(c.messages, chatMessage{role: ai.RoleAssistant, content: c.streamingMsg})
		}
		c.streamingMsg = ""
		c.isThinking = false
		c.updateViewport()

	case "error":
		c.err = event.Error
		c.isThinking = false
	}

	return c, nil
}

func (c *ChatOverlay) buildSystemPrompt() string {
	services := c.registry.ListServices()
	serviceList := strings.Join(services, ", ")

	prompt := fmt.Sprintf(`You are an AWS resource assistant in claws TUI.

<available_services>
%s
</available_services>

<tool_usage>
When a user asks about AWS resources, you MUST call the appropriate tool. Do not just describe what you would do - actually call the tool.
Use ONLY the service names listed in available_services above. Do not guess or use similar names.
All resource tools require a region parameter.

Available tools:
- list_resources(service): Lists resource types for a service
- query_resources(service, resource_type, region): Lists actual resources
- get_resource_detail(service, resource_type, region, id): Gets resource details
- tail_logs(service, resource_type, region, id, cluster?): Fetches CloudWatch logs for a resource
  - Supported: lambda/functions, ecs/services, ecs/tasks, ecs/task-definitions, codebuild/projects, codebuild/builds, cloudtrail/trails, apigateway/stages, apigateway/stages-v2, stepfunctions/state-machines
  - cluster parameter required for ecs/services and ecs/tasks
</tool_usage>

<examples>
query_resources(service="ec2", resource_type="instances", region="us-east-1")
query_resources(service="lambda", resource_type="functions", region="us-west-2")
get_resource_detail(service="lambda", resource_type="functions", region="us-west-2", id="my-function")
tail_logs(service="lambda", resource_type="functions", region="us-east-1", id="my-func")
tail_logs(service="ecs", resource_type="tasks", region="us-east-1", id="my-task", cluster="my-cluster")
</examples>

<response_format>
Be concise. Use markdown for formatting.
</response_format>`, serviceList)

	if c.aiCtx != nil {
		if len(c.aiCtx.Regions) > 0 {
			prompt += fmt.Sprintf("\n\n<current_regions>%s</current_regions>", strings.Join(c.aiCtx.Regions, ", "))
		}
		if c.aiCtx.Service != "" {
			prompt += fmt.Sprintf("\n<current_context>service=%s", c.aiCtx.Service)
			if c.aiCtx.ResourceType != "" {
				prompt += ", resource_type=" + c.aiCtx.ResourceType
			}
			if c.aiCtx.ResourceRegion != "" {
				prompt += ", region=" + c.aiCtx.ResourceRegion
			}
			if c.aiCtx.ResourceID != "" {
				prompt += ", id=" + c.aiCtx.ResourceID
			}
			if c.aiCtx.Cluster != "" {
				prompt += ", cluster=" + c.aiCtx.Cluster
			}
			prompt += "</current_context>"
			prompt += "\nUse these values when querying this resource."
		}
	}

	return prompt
}

func (c *ChatOverlay) updateViewport() {
	if !c.vp.Ready {
		return
	}
	content := c.renderMessages()
	c.vp.Model.SetContent(content)
	c.vp.Model.GotoBottom()
}

func (c *ChatOverlay) renderMessages() string {
	var sb strings.Builder
	w := c.wrapWidth()

	for _, msg := range c.messages {
		// Tool call messages (identified by toolCall != nil)
		if msg.toolCall != nil {
			toolInfo := fmt.Sprintf("🔧 %s(%s)", msg.toolCall.Name, formatToolInput(msg.toolCall.Input))
			style := c.styles.toolCall
			if msg.toolError {
				style = c.styles.toolError
			}
			for _, line := range strings.Split(wrapText(toolInfo, w), "\n") {
				sb.WriteString(style.Render(line))
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
			continue
		}

		// User/Assistant messages
		switch msg.role {
		case ai.RoleUser:
			sb.WriteString(c.styles.userMsg.Render(wrapText("You: "+msg.content, w)))
		case ai.RoleAssistant:
			rendered := c.renderMarkdown(msg.content, w)
			sb.WriteString(c.styles.assistantMsg.Render("AI: ") + "\n" + rendered)
		}
		sb.WriteString("\n\n")
	}

	if c.streamingMsg != "" {
		sb.WriteString(c.styles.assistantMsg.Render(wrapText("AI: "+c.streamingMsg, w)))
		sb.WriteString("\n")
	}

	if c.isThinking && c.streamingMsg == "" {
		sb.WriteString(c.styles.thinking.Render("Thinking..."))
		sb.WriteString("\n")
	}

	if c.err != nil {
		sb.WriteString(c.styles.errorMsg.Render(wrapText("Error: "+c.err.Error(), w)))
		sb.WriteString("\n")
	}

	return sb.String()
}

func (c *ChatOverlay) wrapWidth() int {
	if c.width > 4 {
		return c.width - 4
	}
	return 76
}

func wrapText(text string, width int) string {
	if width <= 0 {
		width = 76
	}
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		lines = append(lines, wrapLine(line, width)...)
	}
	return strings.Join(lines, "\n")
}

func wrapLine(line string, width int) []string {
	if len(line) == 0 {
		return []string{""}
	}
	runes := []rune(line)
	lineWidth := 0
	for _, r := range runes {
		lineWidth += runeWidth(r)
	}
	if lineWidth <= width {
		return []string{line}
	}

	var lines []string
	var current []rune
	currentWidth := 0

	for _, r := range runes {
		rw := runeWidth(r)
		if currentWidth+rw > width && len(current) > 0 {
			lines = append(lines, string(current))
			current = nil
			currentWidth = 0
		}
		current = append(current, r)
		currentWidth += rw
	}
	if len(current) > 0 {
		lines = append(lines, string(current))
	}
	return lines
}

func formatToolInput(input map[string]any) string {
	if len(input) == 0 {
		return ""
	}
	var parts []string
	for k, v := range input {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	return strings.Join(parts, ", ")
}

func runeWidth(r rune) int {
	if r >= 0x1100 &&
		(r <= 0x115F || r == 0x2329 || r == 0x232A ||
			(r >= 0x2E80 && r <= 0xA4CF && r != 0x303F) ||
			(r >= 0xAC00 && r <= 0xD7A3) ||
			(r >= 0xF900 && r <= 0xFAFF) ||
			(r >= 0xFE10 && r <= 0xFE1F) ||
			(r >= 0xFE30 && r <= 0xFE6F) ||
			(r >= 0xFF00 && r <= 0xFF60) ||
			(r >= 0xFFE0 && r <= 0xFFE6) ||
			(r >= 0x20000 && r <= 0x2FFFD) ||
			(r >= 0x30000 && r <= 0x3FFFD)) {
		return 2
	}
	return 1
}

var (
	mdBold   = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	mdItalic = regexp.MustCompile(`\*([^*]+)\*`)
	mdCode   = regexp.MustCompile("`([^`]+)`")
)

func (c *ChatOverlay) renderMarkdown(text string, width int) string {
	wrapped := wrapText(text, width)

	wrapped = mdBold.ReplaceAllStringFunc(wrapped, func(m string) string {
		inner := mdBold.FindStringSubmatch(m)[1]
		return c.styles.mdBold.Render(inner)
	})
	wrapped = mdCode.ReplaceAllStringFunc(wrapped, func(m string) string {
		inner := mdCode.FindStringSubmatch(m)[1]
		return c.styles.mdCode.Render(inner)
	})
	wrapped = mdItalic.ReplaceAllStringFunc(wrapped, func(m string) string {
		inner := mdItalic.FindStringSubmatch(m)[1]
		return c.styles.mdItalic.Render(inner)
	})

	return wrapped
}

func (c *ChatOverlay) View() tea.View {
	return tea.NewView(c.ViewString())
}

func (c *ChatOverlay) ViewString() string {
	var sb strings.Builder

	title := c.styles.title.Render("AI Chat")
	sb.WriteString(title)
	sb.WriteString("\n")

	if c.aiCtx != nil && c.aiCtx.Service != "" {
		ctx := fmt.Sprintf("Context: %s", c.aiCtx.Service)
		if c.aiCtx.ResourceType != "" {
			ctx += "/" + c.aiCtx.ResourceType
		}
		if c.aiCtx.ResourceName != "" {
			ctx += " - " + c.aiCtx.ResourceName
		}
		sb.WriteString(c.styles.context.Render(ctx))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	if c.vp.Ready {
		sb.WriteString(c.vp.Model.View())
	} else {
		sb.WriteString(c.renderMessages())
	}

	sb.WriteString("\n")
	sb.WriteString(c.styles.input.Render(c.input.View()))

	return sb.String()
}

func (c *ChatOverlay) SetSize(width, height int) tea.Cmd {
	c.width = width
	c.height = height

	vpHeight := height - 8
	if vpHeight < 5 {
		vpHeight = 5
	}

	c.vp.SetSize(width, vpHeight)
	c.updateViewport()

	return nil
}

func (c *ChatOverlay) StatusLine() string {
	return "AI Chat | Enter: send | Esc: close"
}

func (c *ChatOverlay) HasActiveInput() bool {
	return true
}
