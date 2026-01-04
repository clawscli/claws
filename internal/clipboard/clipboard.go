// Package clipboard provides clipboard functionality for copying resource IDs and ARNs.
// It supports both OSC52 terminal escape sequences (for SSH/tmux sessions) and native
// system clipboard via the atotto/clipboard library.
package clipboard

import (
	"encoding/base64"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"

	"github.com/clawscli/claws/internal/log"
)

// CopiedMsg is sent when a value has been successfully copied to the clipboard.
type CopiedMsg struct {
	Label string // "ID" or "ARN"
	Value string // The copied value (retained for future use: logging, undo)
}

// NoARNMsg is sent when attempting to copy an ARN for a resource that has no ARN.
type NoARNMsg struct{}

// Copy copies the given value to the clipboard and returns a tea.Cmd that sends a CopiedMsg.
// It writes to both OSC52 (terminal clipboard) and native system clipboard for maximum compatibility.
func Copy(label, value string) tea.Cmd {
	return func() tea.Msg {
		writeOSC52(value)
		if err := clipboard.WriteAll(value); err != nil {
			log.Debug("native clipboard write failed", "error", err)
		}
		return CopiedMsg{Label: label, Value: value}
	}
}

// writeOSC52 writes the value to the terminal clipboard using OSC52 escape sequences.
// It automatically detects and wraps sequences for tmux and screen terminal multiplexers.
func writeOSC52(s string) {
	encoded := base64.StdEncoding.EncodeToString([]byte(s))
	osc52 := "\x1b]52;c;" + encoded + "\x07"

	var seq string
	if os.Getenv("TMUX") != "" {
		seq = "\x1bPtmux;\x1b" + osc52 + "\x1b\\"
	} else if strings.HasPrefix(os.Getenv("TERM"), "screen") {
		seq = "\x1bP" + osc52 + "\x1b\\"
	} else {
		seq = osc52
	}
	if _, err := os.Stdout.WriteString(seq); err != nil {
		log.Debug("OSC52 clipboard write failed", "error", err)
	}
}

// CopyID copies a resource ID to the clipboard.
func CopyID(id string) tea.Cmd {
	return Copy("ID", id)
}

// CopyARN copies a resource ARN to the clipboard.
func CopyARN(arn string) tea.Cmd {
	return Copy("ARN", arn)
}

// NoARN returns a tea.Cmd that sends a NoARNMsg, indicating the resource has no ARN.
func NoARN() tea.Cmd {
	return func() tea.Msg { return NoARNMsg{} }
}
