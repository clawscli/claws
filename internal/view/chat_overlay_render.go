package view

import (
	"fmt"
	"regexp"
	"strings"

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

	for _, msg := range c.messages {
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
