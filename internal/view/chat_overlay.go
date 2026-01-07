package view

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/clawscli/claws/internal/ai"
	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/ui"
)

type chatStyles struct {
	title        lipgloss.Style
	context      lipgloss.Style
	userMsg      lipgloss.Style
	assistantMsg lipgloss.Style
	thinking     lipgloss.Style
	input        lipgloss.Style
}

func newChatStyles() chatStyles {
	t := ui.Current()
	return chatStyles{
		title:        lipgloss.NewStyle().Bold(true).Foreground(t.Primary),
		context:      lipgloss.NewStyle().Foreground(t.TextDim).Italic(true),
		userMsg:      lipgloss.NewStyle().Foreground(t.Text),
		assistantMsg: lipgloss.NewStyle().Foreground(t.Secondary),
		thinking:     lipgloss.NewStyle().Foreground(t.TextDim).Italic(true),
		input:        lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(t.Border).Padding(0, 1),
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
	role    ai.Role
	content string
}

type chatStreamMsg struct {
	event ai.StreamEvent
}

type chatDoneMsg struct {
	err error
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
		return chatInitMsg{err: fmt.Errorf("init tool executor: %w", err)}
	}

	client, err := ai.NewClient(
		c.ctx,
		ai.WithModel(config.File().GetAIModel()),
		ai.WithTools(executor.Tools()),
	)
	if err != nil {
		return chatInitMsg{err: fmt.Errorf("init ai client: %w", err)}
	}

	session, err := c.sessMgr.NewSession(c.aiCtx)
	if err != nil {
		return chatInitMsg{err: fmt.Errorf("create session: %w", err)}
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
		if msg.err != nil {
			c.err = msg.err
			c.isThinking = false
		}
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
		if c.client == nil {
			return chatDoneMsg{err: fmt.Errorf("client not initialized")}
		}

		messages := make([]ai.Message, len(c.messages))
		for i, m := range c.messages {
			messages[i] = ai.Message{Role: m.role, Content: m.content}
		}

		systemPrompt := c.buildSystemPrompt()
		result, err := c.client.Converse(c.ctx, messages, systemPrompt)
		if err != nil {
			return chatDoneMsg{err: err}
		}

		for len(result.ToolCalls) > 0 {
			var toolResults []ai.ToolResult
			for _, tc := range result.ToolCalls {
				tr := c.executor.Execute(c.ctx, tc)
				toolResults = append(toolResults, tr)
			}

			messages = c.client.AddToolResultMessages(messages, result.ToolCalls, toolResults)

			result, err = c.client.Converse(c.ctx, messages, systemPrompt)
			if err != nil {
				return chatDoneMsg{err: err}
			}
		}

		return chatStreamMsg{event: ai.StreamEvent{Type: "done", Text: result.Message}}
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
	prompt := `You are an AI assistant helping users explore and understand their AWS resources.
You have access to tools that can list services, query resources, get resource details, and tail CloudWatch logs.
Be concise but helpful. When showing resources, summarize key information.
If the user asks about logs for a resource like Lambda, infer the log group name (e.g., /aws/lambda/{function-name}).`

	if c.aiCtx != nil {
		if c.aiCtx.Service != "" {
			prompt += fmt.Sprintf("\n\nCurrent context: Service=%s", c.aiCtx.Service)
			if c.aiCtx.ResourceType != "" {
				prompt += fmt.Sprintf(", ResourceType=%s", c.aiCtx.ResourceType)
			}
			if c.aiCtx.ResourceID != "" {
				prompt += fmt.Sprintf(", ResourceID=%s", c.aiCtx.ResourceID)
			}
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

	for _, msg := range c.messages {
		switch msg.role {
		case ai.RoleUser:
			sb.WriteString(c.styles.userMsg.Render("You: " + msg.content))
		case ai.RoleAssistant:
			sb.WriteString(c.styles.assistantMsg.Render("AI: " + msg.content))
		}
		sb.WriteString("\n\n")
	}

	if c.streamingMsg != "" {
		sb.WriteString(c.styles.assistantMsg.Render("AI: " + c.streamingMsg))
		sb.WriteString("\n")
	}

	if c.isThinking && c.streamingMsg == "" {
		sb.WriteString(c.styles.thinking.Render("Thinking..."))
		sb.WriteString("\n")
	}

	if c.err != nil {
		sb.WriteString(lipgloss.NewStyle().Foreground(ui.Current().Danger).Render("Error: " + c.err.Error()))
		sb.WriteString("\n")
	}

	return sb.String()
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
