package clipboard

import (
	"encoding/base64"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"

	"github.com/clawscli/claws/internal/log"
)

type CopiedMsg struct {
	Label string
	Value string // retained for future use (logging, undo)
}

type NoARNMsg struct{}

func Copy(label, value string) tea.Cmd {
	return func() tea.Msg {
		writeOSC52(value)
		_ = clipboard.WriteAll(value)
		return CopiedMsg{Label: label, Value: value}
	}
}

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

func CopyID(id string) tea.Cmd {
	return Copy("ID", id)
}

func CopyARN(arn string) tea.Cmd {
	return Copy("ARN", arn)
}

func NoARN() tea.Cmd {
	return func() tea.Msg { return NoARNMsg{} }
}
