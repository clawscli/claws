package clipboard

import (
	"encoding/base64"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"
)

type CopiedMsg struct {
	Label string
	Value string
}

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
	_, _ = os.Stdout.WriteString(seq)
}

func CopyID(id string) tea.Cmd {
	return Copy("ID", id)
}

func CopyARN(arn string) tea.Cmd {
	return Copy("ARN", arn)
}
