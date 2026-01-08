package view

import (
	"context"
	"errors"
	"fmt"
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
	return chatStyles{
		title:        ui.TitleStyle(),
		context:      ui.DimItalicStyle(),
		userMsg:      ui.TextStyle(),
		assistantMsg: ui.SecondaryStyle(),
		toolCall:     ui.DimStyle(),
		toolError:    ui.DangerStyle(),
		thinking:     ui.DimItalicStyle(),
		input:        ui.ChatInputStyle(),
		errorMsg:     ui.DangerStyle(),
		mdBold:       ui.TitleStyle(),
		mdCode:       ui.SuccessStyle(),
		mdItalic:     ui.ItalicStyle(),
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

	messages           []chatMessage
	streamingMsg       string
	streamingThinking  string
	collapsedThinking  map[int]bool
	thinkingLineRanges map[int][2]int
	isStreaming        bool
	err                error

	// Streaming state
	pendingToolCalls []ai.ToolCall
	streamMessages   []ai.Message
	toolRound        int

	width  int
	height int
}

type chatMessage struct {
	role            ai.Role
	content         string
	thinkingContent string
	toolCall        *ai.ToolCall
	toolError       bool
}

type chatStreamMsg struct {
	event   ai.StreamEvent
	eventCh <-chan ai.StreamEvent
}

type chatToolExecuteMsg struct {
	toolCalls []ai.ToolCall
	messages  []ai.Message
	toolRound int
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
		ctx:               ctx,
		registry:          reg,
		aiCtx:             aiCtx,
		styles:            newChatStyles(),
		input:             ti,
		sessMgr:           ai.NewSessionManager(maxSessions),
		messages:          []chatMessage{},
		collapsedThinking: make(map[int]bool),
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
		ai.WithMaxTokens(config.File().GetAIMaxTokens()),
		ai.WithThinkingBudget(config.File().GetAIThinkingBudget()),
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
		return c.handleStreamEvent(msg)

	case chatToolExecuteMsg:
		return c.handleToolExecute(msg)

	case tea.MouseClickMsg:
		return c.handleMouseClick(msg)
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
		if c.isStreaming {
			return c, nil
		}

		text := strings.TrimSpace(c.input.Value())
		if text == "" {
			return c, nil
		}

		c.input.SetValue("")
		c.messages = append(c.messages, chatMessage{role: ai.RoleUser, content: text})
		c.isStreaming = true
		c.streamingMsg = ""
		c.pendingToolCalls = nil
		c.toolRound = 0
		c.err = nil
		c.updateViewport()

		return c, c.startStream(c.buildAPIMessages())
	}

	var cmd tea.Cmd
	c.input, cmd = c.input.Update(msg)
	return c, cmd
}

func (c *ChatOverlay) handleMouseClick(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if !c.vp.Ready {
		return c, nil
	}

	headerHeight := c.headerHeight()

	contentLine := msg.Y - headerHeight + c.vp.Model.YOffset()
	if contentLine < 0 {
		return c, nil
	}

	for msgIdx, lineRange := range c.thinkingLineRanges {
		if contentLine >= lineRange[0] && contentLine < lineRange[1] {
			wasCollapsed := c.collapsedThinking[msgIdx]
			c.collapsedThinking[msgIdx] = !wasCollapsed
			c.scrollToThinking(msgIdx, wasCollapsed)
			return c, nil
		}
	}
	return c, nil
}

func (c *ChatOverlay) buildAPIMessages() []ai.Message {
	var messages []ai.Message
	for _, m := range c.messages {
		if m.toolCall == nil {
			messages = append(messages, ai.Message{Role: m.role, Content: m.content})
		}
	}
	return messages
}

func (c *ChatOverlay) startStream(messages []ai.Message) tea.Cmd {
	return func() tea.Msg {
		if c.client == nil || c.executor == nil {
			return chatStreamMsg{event: ai.StreamEvent{Type: "error", Error: errors.New("client not initialized")}}
		}

		c.streamMessages = messages
		systemPrompt := c.buildSystemPrompt()

		eventCh, err := c.client.ConverseStream(c.ctx, messages, systemPrompt)
		if err != nil {
			return chatStreamMsg{event: ai.StreamEvent{Type: "error", Error: err}}
		}

		event, ok := <-eventCh
		if !ok {
			return chatStreamMsg{event: ai.StreamEvent{Type: "done"}}
		}
		return chatStreamMsg{event: event, eventCh: eventCh}
	}
}

func (c *ChatOverlay) waitForStream(eventCh <-chan ai.StreamEvent) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-eventCh
		if !ok {
			return chatStreamMsg{event: ai.StreamEvent{Type: "done"}}
		}
		return chatStreamMsg{event: event, eventCh: eventCh}
	}
}

func (c *ChatOverlay) handleStreamEvent(msg chatStreamMsg) (tea.Model, tea.Cmd) {
	event := msg.event

	switch event.Type {
	case "text":
		c.streamingMsg += event.Text
		c.updateViewport()
		return c, c.waitForStream(msg.eventCh)

	case "thinking":
		c.streamingThinking += event.Thinking
		c.updateViewport()
		return c, c.waitForStream(msg.eventCh)

	case "tool_use":
		c.pendingToolCalls = append(c.pendingToolCalls, event.ToolCalls...)
		return c, c.waitForStream(msg.eventCh)

	case "done":
		if len(c.pendingToolCalls) > 0 && c.toolRound < 5 {
			if c.streamingMsg != "" || c.streamingThinking != "" {
				c.messages = append(c.messages, chatMessage{
					role:            ai.RoleAssistant,
					content:         c.streamingMsg,
					thinkingContent: c.streamingThinking,
				})
				if c.streamingThinking != "" {
					c.collapsedThinking[len(c.messages)-1] = true
				}
				c.streamingMsg = ""
				c.streamingThinking = ""
			}
			c.updateViewport()
			return c, c.executeTools()
		}

		if c.streamingMsg != "" || c.streamingThinking != "" {
			c.messages = append(c.messages, chatMessage{
				role:            ai.RoleAssistant,
				content:         c.streamingMsg,
				thinkingContent: c.streamingThinking,
			})
			if c.streamingThinking != "" {
				c.collapsedThinking[len(c.messages)-1] = true
			}
		}
		c.streamingMsg = ""
		c.streamingThinking = ""
		c.isStreaming = false
		c.updateViewport()
		return c, nil

	case "error":
		c.err = event.Error
		c.isStreaming = false
		c.updateViewport()
		return c, nil
	}

	return c, c.waitForStream(msg.eventCh)
}

func (c *ChatOverlay) executeTools() tea.Cmd {
	toolCalls := c.pendingToolCalls
	c.pendingToolCalls = nil
	c.toolRound++

	return func() tea.Msg {
		return chatToolExecuteMsg{
			toolCalls: toolCalls,
			messages:  c.streamMessages,
			toolRound: c.toolRound,
		}
	}
}

func (c *ChatOverlay) handleToolExecute(msg chatToolExecuteMsg) (tea.Model, tea.Cmd) {
	var toolResults []ai.ToolResult
	for _, tc := range msg.toolCalls {
		tcCopy := tc
		tr := c.executor.Execute(c.ctx, tc)
		toolResults = append(toolResults, tr)
		c.messages = append(c.messages, chatMessage{
			content:   tr.Content,
			toolCall:  &tcCopy,
			toolError: tr.IsError,
		})
	}
	c.updateViewport()

	messages := c.client.AddToolResultMessages(msg.messages, msg.toolCalls, toolResults)
	c.streamMessages = messages
	c.isStreaming = true

	return c, c.startStream(messages)
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

func (c *ChatOverlay) headerHeight() int {
	lines := 2
	if c.aiCtx != nil && c.aiCtx.Service != "" {
		ctx := fmt.Sprintf("Context: %s", c.aiCtx.Service)
		if c.aiCtx.ResourceType != "" {
			ctx += "/" + c.aiCtx.ResourceType
		}
		if c.aiCtx.ResourceName != "" {
			ctx += " - " + c.aiCtx.ResourceName
		}
		rendered := c.styles.context.Render(ctx)
		lines += strings.Count(rendered, "\n") + 1
	}
	return lines
}

func (c *ChatOverlay) HasActiveInput() bool {
	return true
}

func (c *ChatOverlay) scrollToThinking(msgIdx int, wasCollapsed bool) {
	if !c.vp.Ready {
		return
	}
	content := c.renderMessages()
	c.vp.Model.SetContent(content)
	if wasCollapsed {
		if lineRange, ok := c.thinkingLineRanges[msgIdx]; ok {
			c.vp.Model.SetYOffset(lineRange[0])
		}
	}
}
