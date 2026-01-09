package view

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/clawscli/claws/internal/ai"
	"github.com/clawscli/claws/internal/config"
	apperrors "github.com/clawscli/claws/internal/errors"
	"github.com/clawscli/claws/internal/log"
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
	collapsedToolCalls map[int]bool
	thinkingLineRanges map[int][2]int
	toolCallLineRanges map[int][2]int
	isStreaming        bool
	err                error

	// Streaming state - accumulates ContentBlocks for the current assistant turn
	pendingToolUses    []*ai.ToolUseContent
	currentReasoning   string
	reasoningSignature string
	streamMessages     []ai.Message
	toolRound          int
	toolCallCount      int // Counts tool calls within current query (reset per query)

	width  int
	height int

	showingHistory bool
	sessionHistory *SessionHistory

	statusMsg     string
	statusMsgTime time.Time

	contextExpanded bool

	// Stream cancellation - prevents goroutine leaks when overlay closes mid-stream
	streamCancel   context.CancelFunc
	streamCancelMu sync.Mutex
}

// chatMessage is a UI-level message for display purposes.
// It stores extracted text/thinking for rendering.
type chatMessage struct {
	role            ai.Role
	content         string
	thinkingContent string
	toolUse         *ai.ToolUseContent
	toolResult      *ai.ToolResultContent
	toolError       bool
}

type chatStreamMsg struct {
	event   ai.StreamEvent
	eventCh <-chan ai.StreamEvent
}

type chatToolExecuteMsg struct {
	// The assistant message with ToolUse blocks that triggered this execution
	assistantBlocks []ai.ContentBlock
	toolUses        []*ai.ToolUseContent
	messages        []ai.Message
	toolRound       int
}

type chatInitMsg struct {
	client   *ai.Client
	executor *ai.ToolExecutor
	session  *ai.Session
	err      error
}

func NewChatOverlay(ctx context.Context, reg *registry.Registry, aiCtx *ai.Context) *ChatOverlay {
	cfg := config.File()

	ti := textinput.New()
	ti.Placeholder = "Ask about AWS resources..."
	ti.Focus()
	ti.CharLimit = 500

	return &ChatOverlay{
		ctx:                ctx,
		registry:           reg,
		aiCtx:              aiCtx,
		styles:             newChatStyles(),
		input:              ti,
		sessMgr:            ai.NewSessionManager(cfg.GetAIMaxSessions(), cfg.GetAISaveSessions()),
		messages:           []chatMessage{},
		collapsedThinking:  make(map[int]bool),
		collapsedToolCalls: make(map[int]bool),
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
	if c.showingHistory {
		return c.handleHistoryUpdate(msg)
	}

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
		var vpCmd tea.Cmd
		c.vp.Model, vpCmd = c.vp.Model.Update(msg)
		cmds = append(cmds, vpCmd)
	}

	var inputCmd tea.Cmd
	c.input, inputCmd = c.input.Update(msg)
	cmds = append(cmds, inputCmd)

	return c, tea.Batch(cmds...)
}

func (c *ChatOverlay) cancelStream() {
	c.streamCancelMu.Lock()
	defer c.streamCancelMu.Unlock()
	if c.streamCancel != nil {
		c.streamCancel()
		c.streamCancel = nil
	}
}

func (c *ChatOverlay) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if IsEscKey(msg) {
		c.cancelStream()
		return c, func() tea.Msg { return HideModalMsg{} }
	}

	switch msg.String() {
	case "ctrl+c":
		c.cancelStream()
		return c, func() tea.Msg { return HideModalMsg{} }
	case "ctrl+h":
		return c.showHistory()
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
		c.streamingThinking = ""
		c.pendingToolUses = nil
		c.currentReasoning = ""
		c.reasoningSignature = ""
		c.toolRound = 0
		c.toolCallCount = 0 // Reset per-query tool call counter
		c.err = nil
		c.updateViewport()

		userMsg := ai.NewUserMessage(text)
		c.streamMessages = append(c.streamMessages, userMsg)
		if c.session != nil {
			if err := c.sessMgr.AddMessage(c.session, userMsg); err != nil {
				log.Warn("failed to save user message", "error", err)
				c.statusMsg = "Failed to save message"
				c.statusMsgTime = time.Now()
			}
		}
		return c, c.startStream(c.streamMessages)
	}

	var kpCmd tea.Cmd
	c.input, kpCmd = c.input.Update(msg)
	return c, kpCmd
}

func (c *ChatOverlay) handleMouseClick(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if c.aiCtx != nil && c.aiCtx.Service != "" && msg.Y == 1 {
		c.contextExpanded = !c.contextExpanded
		c.updateViewport()
		return c, nil
	}

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
			c.scrollToCollapsible(lineRange[0], wasCollapsed)
			return c, nil
		}
	}

	for msgIdx, lineRange := range c.toolCallLineRanges {
		if contentLine >= lineRange[0] && contentLine < lineRange[1] {
			wasCollapsed := c.collapsedToolCalls[msgIdx]
			c.collapsedToolCalls[msgIdx] = !wasCollapsed
			c.scrollToCollapsible(lineRange[0], wasCollapsed)
			return c, nil
		}
	}

	return c, nil
}

func (c *ChatOverlay) startStream(messages []ai.Message) tea.Cmd {
	c.cancelStream()
	streamCtx, cancel := context.WithCancel(c.ctx)

	c.streamCancelMu.Lock()
	c.streamCancel = cancel
	c.streamCancelMu.Unlock()

	return func() tea.Msg {
		if c.client == nil || c.executor == nil {
			return chatStreamMsg{event: ai.StreamEvent{Type: "error", Error: errors.New("client not initialized")}}
		}

		systemPrompt := c.buildSystemPrompt()

		eventCh, err := c.client.ConverseStream(streamCtx, messages, systemPrompt)
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
		if event.Thinking != nil {
			c.streamingThinking += event.Thinking.Text
		}
		c.updateViewport()
		return c, c.waitForStream(msg.eventCh)

	case "thinking_complete":
		// Capture the complete thinking with signature for API replay
		if event.Thinking != nil {
			c.currentReasoning = event.Thinking.Text
			c.reasoningSignature = event.Thinking.Signature
		}
		return c, c.waitForStream(msg.eventCh)

	case "tool_use":
		if event.ToolUse != nil {
			c.pendingToolUses = append(c.pendingToolUses, event.ToolUse)
		}
		return c, c.waitForStream(msg.eventCh)

	case "done":
		return c.handleStreamDone(msg.eventCh)

	case "error":
		c.err = event.Error
		c.isStreaming = false
		c.updateViewport()
		return c, nil
	}

	return c, c.waitForStream(msg.eventCh)
}

func (c *ChatOverlay) handleStreamDone(_ <-chan ai.StreamEvent) (tea.Model, tea.Cmd) {
	// Build the assistant's ContentBlocks from accumulated state
	var assistantBlocks []ai.ContentBlock

	// Add reasoning block if present
	if c.currentReasoning != "" {
		assistantBlocks = append(assistantBlocks, ai.ContentBlock{
			Reasoning:          c.currentReasoning,
			ReasoningSignature: c.reasoningSignature,
		})
	}

	// Add text block if present
	if c.streamingMsg != "" {
		assistantBlocks = append(assistantBlocks, ai.ContentBlock{Text: c.streamingMsg})
	}

	// Add tool use blocks
	for _, tu := range c.pendingToolUses {
		assistantBlocks = append(assistantBlocks, ai.ContentBlock{ToolUse: tu})
	}

	// Save to UI messages for display
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

	// If there are tool uses, execute them
	if len(c.pendingToolUses) > 0 && c.toolRound < config.File().GetAIMaxToolRounds() {
		c.updateViewport()

		// Save assistant message with tool uses to session
		if c.session != nil && len(assistantBlocks) > 0 {
			assistantMsg := ai.Message{
				Role:    ai.RoleAssistant,
				Content: assistantBlocks,
			}
			c.streamMessages = append(c.streamMessages, assistantMsg)
			if err := c.sessMgr.AddMessage(c.session, assistantMsg); err != nil {
				log.Warn("failed to save assistant message with tool uses", "error", err)
			}
		}

		// Clear streaming state before tool execution
		toolUses := c.pendingToolUses
		c.pendingToolUses = nil
		c.streamingMsg = ""
		c.streamingThinking = ""
		c.currentReasoning = ""
		c.reasoningSignature = ""
		c.toolRound++

		return c, func() tea.Msg {
			return chatToolExecuteMsg{
				assistantBlocks: assistantBlocks,
				toolUses:        toolUses,
				messages:        c.streamMessages,
				toolRound:       c.toolRound,
			}
		}
	}

	// No tool uses or max rounds reached - done
	if len(assistantBlocks) > 0 {
		assistantMsg := ai.Message{
			Role:    ai.RoleAssistant,
			Content: assistantBlocks,
		}
		c.streamMessages = append(c.streamMessages, assistantMsg)
		if c.session != nil {
			if err := c.sessMgr.AddMessage(c.session, assistantMsg); err != nil {
				log.Warn("failed to save assistant message", "error", err)
				c.statusMsg = "Failed to save message"
				c.statusMsgTime = time.Now()
			}
		}
	}

	if len(c.pendingToolUses) > 0 && c.toolRound >= config.File().GetAIMaxToolRounds() {
		c.messages = append(c.messages, chatMessage{
			role:    ai.RoleAssistant,
			content: "(tool limit reached)",
		})
	}

	c.streamingMsg = ""
	c.streamingThinking = ""
	c.currentReasoning = ""
	c.reasoningSignature = ""
	c.pendingToolUses = nil
	c.isStreaming = false
	c.updateViewport()
	return c, nil
}

func (c *ChatOverlay) handleToolExecute(msg chatToolExecuteMsg) (tea.Model, tea.Cmd) {
	maxCalls := config.File().GetAIMaxToolCallsPerQuery()

	// Execute each tool and collect results
	var toolResults []ai.ToolResultContent
	for _, tu := range msg.toolUses {
		// Check tool call limit before executing each tool
		if c.toolCallCount >= maxCalls {
			c.err = fmt.Errorf("Tool call limit reached (%d calls). Start new query to continue.", maxCalls)
			c.isStreaming = false
			c.updateViewport()
			return c, nil
		}

		result := c.executor.Execute(c.ctx, tu)
		toolResults = append(toolResults, result)
		c.toolCallCount++

		c.messages = append(c.messages, chatMessage{
			content:    result.Content,
			toolUse:    tu,
			toolResult: &result,
			toolError:  result.IsError,
		})
		c.collapsedToolCalls[len(c.messages)-1] = true
	}
	c.updateViewport()

	// Build the new messages to send to API:
	// 1. Previous messages (including assistant message with tool uses from handleStreamDone)
	// 2. User message with tool results

	messages := make([]ai.Message, len(msg.messages), len(msg.messages)+1)
	copy(messages, msg.messages)

	// Add user message with tool results
	var resultBlocks []ai.ContentBlock
	for _, tr := range toolResults {
		resultBlocks = append(resultBlocks, ai.ContentBlock{ToolResult: &tr})
	}
	messages = append(messages, ai.Message{
		Role:    ai.RoleUser,
		Content: resultBlocks,
	})

	c.streamMessages = messages
	c.isStreaming = true

	// Save tool result message to session
	if c.session != nil {
		toolResultMsg := messages[len(messages)-1] // Last message with tool results
		if err := c.sessMgr.AddMessage(c.session, toolResultMsg); err != nil {
			log.Warn("failed to save tool result message", "error", err)
		}
	}

	return c, c.startStream(messages)
}

func (c *ChatOverlay) View() tea.View {
	return tea.NewView(c.ViewString())
}

func (c *ChatOverlay) ViewString() string {
	if c.showingHistory && c.sessionHistory != nil {
		return c.sessionHistory.ViewString()
	}

	var sb strings.Builder

	title := c.styles.title.Render("AI Chat")
	hint := c.styles.context.Render("Ctrl+h: history")
	titleWidth := lipgloss.Width(title)
	hintWidth := lipgloss.Width(hint)
	padding := c.width - titleWidth - hintWidth
	if padding < 1 {
		padding = 1
	}
	sb.WriteString(title + strings.Repeat(" ", padding) + hint)
	sb.WriteString("\n")

	if c.aiCtx != nil && c.aiCtx.Service != "" {
		indicator := "▶"
		if c.contextExpanded {
			indicator = "▼"
		}
		ctx := fmt.Sprintf("Context: %s", c.aiCtx.Service)
		if c.aiCtx.ResourceType != "" {
			ctx += "/" + c.aiCtx.ResourceType
		}
		if c.aiCtx.ResourceName != "" {
			ctx += " - " + c.aiCtx.ResourceName
		}
		ctx += " [" + indicator + "]"
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
	if c.statusMsg != "" && time.Since(c.statusMsgTime) < 3*time.Second {
		return c.statusMsg
	}
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

func (c *ChatOverlay) scrollToCollapsible(startLine int, wasCollapsed bool) {
	if !c.vp.Ready {
		return
	}
	content := c.renderMessages()
	c.vp.Model.SetContent(content)
	if wasCollapsed {
		c.vp.Model.SetYOffset(startLine)
	}
}

func (c *ChatOverlay) showHistory() (tea.Model, tea.Cmd) {
	sessions, _ := c.sessMgr.ListSessions()
	currentID := ""
	if c.session != nil {
		currentID = c.session.ID
	}
	c.sessionHistory = NewSessionHistory(sessions, currentID)
	c.sessionHistory.SetSize(c.width, c.height)
	c.showingHistory = true
	return c, nil
}

func (c *ChatOverlay) handleHistoryUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case SessionSelectedMsg:
		c.showingHistory = false
		c.sessionHistory = nil
		if msg.Session != nil {
			return c.loadSession(msg.Session)
		}
		return c, nil

	case NewSessionMsg:
		c.showingHistory = false
		c.sessionHistory = nil
		return c.newSession()

	case CloseHistoryMsg:
		c.showingHistory = false
		c.sessionHistory = nil
		return c, nil
	}

	if c.sessionHistory != nil {
		model, cmd := c.sessionHistory.Update(msg)
		if sh, ok := model.(*SessionHistory); ok {
			c.sessionHistory = sh
		}
		return c, cmd
	}
	return c, nil
}

func (c *ChatOverlay) loadSession(sess *ai.Session) (tea.Model, tea.Cmd) {
	if sess == nil {
		return c, nil
	}

	c.cancelStream()
	if c.isStreaming {
		c.isStreaming = false
		c.streamingMsg = ""
		c.streamingThinking = ""
		c.pendingToolUses = nil
		c.currentReasoning = ""
		c.reasoningSignature = ""
	}

	c.session = sess
	c.messages = []chatMessage{}
	c.streamMessages = []ai.Message{}
	c.collapsedThinking = make(map[int]bool)
	c.collapsedToolCalls = make(map[int]bool)
	c.toolCallCount = 0 // Reset per-query counter

	for _, msg := range sess.Messages {
		cm := chatMessage{role: msg.Role}
		for _, block := range msg.Content {
			if block.Text != "" {
				cm.content = block.Text
			}
			if block.Reasoning != "" {
				cm.thinkingContent = block.Reasoning
			}
		}
		c.messages = append(c.messages, cm)
		c.streamMessages = append(c.streamMessages, msg)
	}

	c.updateViewport()
	return c, nil
}

func (c *ChatOverlay) newSession() (tea.Model, tea.Cmd) {
	session, err := c.sessMgr.NewSession(c.aiCtx)
	if err != nil {
		c.err = err
		return c, nil
	}
	c.session = session
	c.messages = []chatMessage{}
	c.streamMessages = []ai.Message{}
	c.collapsedThinking = make(map[int]bool)
	c.collapsedToolCalls = make(map[int]bool)
	c.toolCallCount = 0 // Reset per-query counter
	c.updateViewport()
	return c, nil
}
