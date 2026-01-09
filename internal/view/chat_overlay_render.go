package view

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/mattn/go-runewidth"

	"github.com/clawscli/claws/internal/ai"
)

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
	lineNum := 0
	c.thinkingLineRanges = make(map[int][2]int)
	c.toolCallLineRanges = make(map[int][2]int)

	if c.contextExpanded && c.aiCtx != nil {
		params := c.renderContextParams()
		for _, line := range strings.Split(strings.TrimSuffix(params, "\n"), "\n") {
			sb.WriteString(c.styles.context.Render(line))
			sb.WriteString("\n")
			lineNum++
		}
		sb.WriteString("\n")
		lineNum++
	}

	for i, msg := range c.messages {
		if msg.toolUse != nil {
			startLine := lineNum
			toolStr := c.renderToolCall(i, msg.toolUse, msg.toolError, w)
			sb.WriteString(toolStr)
			lineNum += strings.Count(toolStr, "\n")
			c.toolCallLineRanges[i] = [2]int{startLine, lineNum}
		} else {
			switch msg.role {
			case ai.RoleUser:
				userText := c.styles.userMsg.Render(wrapText("You: "+msg.content, w))
				sb.WriteString(userText)
				sb.WriteString("\n")
				lineNum += strings.Count(userText, "\n") + 1
			case ai.RoleAssistant:
				if msg.thinkingContent != "" {
					startLine := lineNum
					thinkingStr := c.renderThinking(i, msg.thinkingContent, w)
					sb.WriteString(thinkingStr)
					lineNum += strings.Count(thinkingStr, "\n")
					c.thinkingLineRanges[i] = [2]int{startLine, lineNum}
					if msg.content != "" {
						sb.WriteString("\n")
						lineNum++
					}
				}
				if msg.content != "" {
					rendered := c.renderMarkdown(msg.content, w)
					contentStr := c.styles.assistantMsg.Render("AI: ") + "\n" + rendered
					sb.WriteString(contentStr)
					sb.WriteString("\n")
					lineNum += strings.Count(contentStr, "\n") + 1
				}
			}
		}
		sb.WriteString("\n")
		lineNum++
	}

	if c.streamingThinking != "" {
		sb.WriteString(c.styles.thinking.Render("💭 ▶ Thinking..."))
		sb.WriteString("\n")
		if c.streamingMsg != "" {
			sb.WriteString("\n")
		}
	}
	if c.streamingMsg != "" {
		sb.WriteString(c.styles.assistantMsg.Render("AI: "))
		sb.WriteString("\n")
		sb.WriteString(wrapText(c.streamingMsg, w))
		sb.WriteString("\n")
	} else if c.isStreaming && c.streamingThinking == "" {
		sb.WriteString(c.styles.thinking.Render("⏳ Waiting..."))
		sb.WriteString("\n")
	}

	if c.err != nil {
		sb.WriteString(c.styles.errorMsg.Render(wrapText("Error: "+c.err.Error(), w)))
		sb.WriteString("\n")
	}

	return sb.String()
}

func (c *ChatOverlay) renderThinking(idx int, content string, width int) string {
	collapsed := c.collapsedThinking[idx]
	var sb strings.Builder

	if collapsed {
		sb.WriteString(c.styles.thinking.Render("💭 ▶ [click to expand]"))
		sb.WriteString("\n")
	} else {
		sb.WriteString(c.styles.thinking.Render("💭 ▼ Thinking:"))
		sb.WriteString("\n")
		wrapped := wrapText(content, width-2)
		for _, line := range strings.Split(wrapped, "\n") {
			sb.WriteString(c.styles.thinking.Render("  " + line))
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func (c *ChatOverlay) renderToolCall(idx int, tu *ai.ToolUseContent, isError bool, width int) string {
	collapsed := c.collapsedToolCalls[idx]
	style := c.styles.toolCall
	if isError {
		style = c.styles.toolError
	}

	var sb strings.Builder
	paramCount := len(tu.Input)

	if collapsed {
		summary := fmt.Sprintf("🔧 %s ▶ [%d params]", tu.Name, paramCount)
		sb.WriteString(style.Render(wrapText(summary, width)))
		sb.WriteString("\n")
	} else {
		header := fmt.Sprintf("🔧 %s ▼", tu.Name)
		sb.WriteString(style.Render(header))
		sb.WriteString("\n")

		keys := make([]string, 0, len(tu.Input))
		for k := range tu.Input {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			v := tu.Input[k]
			line := fmt.Sprintf("  %s: %v", k, v)
			sb.WriteString(style.Render(wrapText(line, width)))
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func (c *ChatOverlay) wrapWidth() int {
	if c.width > 4 {
		return c.width - 4
	}
	return 76
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

func runeWidth(r rune) int {
	return runewidth.RuneWidth(r)
}
