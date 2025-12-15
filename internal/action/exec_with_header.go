package action

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/ui"
	"golang.org/x/term"
)

// ExecWithHeader represents an exec command that should run with a fixed header
// Implements tea.ExecCommand interface
type ExecWithHeader struct {
	Command  string
	Resource dao.Resource
	Service  string
	ResType  string

	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

// SetStdin sets the stdin for the command
func (e *ExecWithHeader) SetStdin(r io.Reader) {
	e.stdin = r
}

// SetStdout sets the stdout for the command
func (e *ExecWithHeader) SetStdout(w io.Writer) {
	e.stdout = w
}

// SetStderr sets the stderr for the command
func (e *ExecWithHeader) SetStderr(w io.Writer) {
	e.stderr = w
}

// Run executes the command with a fixed header at the top
func (e *ExecWithHeader) Run() error {
	// Use provided or default stdin/stdout/stderr
	stdin := e.stdin
	stdout := e.stdout
	stderr := e.stderr
	if stdin == nil {
		stdin = os.Stdin
	}
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}

	// Get terminal size (try stdout first, then fallback)
	width, height := 80, 24
	if f, ok := stdout.(*os.File); ok {
		if w, h, err := term.GetSize(int(f.Fd())); err == nil {
			width, height = w, h
		}
	}

	// Build header content
	header := e.buildHeader(width)
	headerLines := strings.Count(header, "\n") + 1

	// Clear screen and move to top
	_, _ = fmt.Fprint(stdout, "\x1b[2J\x1b[H")

	// Print header
	_, _ = fmt.Fprint(stdout, header)

	// Print separator
	separatorStyle := lipgloss.NewStyle().Foreground(ui.Current().TextDim)
	_, _ = fmt.Fprintln(stdout, separatorStyle.Render(strings.Repeat("─", width)))
	headerLines++

	// Set scroll region to exclude header (1-indexed)
	// ESC [ top ; bottom r - Set scrolling region
	scrollTop := headerLines + 1
	scrollBottom := height
	_, _ = fmt.Fprintf(stdout, "\x1b[%d;%dr", scrollTop, scrollBottom)

	// Move cursor to scroll region
	_, _ = fmt.Fprintf(stdout, "\x1b[%d;1H", scrollTop)

	// Prepare command - run through shell to support quoting and pipes
	if e.Command == "" {
		return fmt.Errorf("empty command")
	}

	cmd := exec.Command("/bin/sh", "-c", e.Command)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	// Run the command
	err := cmd.Run()

	// Reset scroll region
	_, _ = fmt.Fprint(stdout, "\x1b[r")

	// Move to bottom and clear
	_, _ = fmt.Fprintf(stdout, "\x1b[%d;1H", height)

	// If command failed, show error and wait for keypress
	if err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Bold(true)
		_, _ = fmt.Fprintln(stdout)
		_, _ = fmt.Fprintln(stdout, errorStyle.Render("Command failed: ")+err.Error())
		_, _ = fmt.Fprintln(stdout)
		_, _ = fmt.Fprint(stdout, "Press Enter to continue...")

		// Wait for Enter key
		buf := make([]byte, 1)
		if f, ok := stdin.(*os.File); ok {
			_, _ = f.Read(buf)
		}
	}

	return err
}

func (e *ExecWithHeader) buildHeader(width int) string {
	region := config.Global().Region()
	accountID := config.Global().AccountID()

	// Styles
	t := ui.Current()

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Primary)

	labelStyle := lipgloss.NewStyle().
		Foreground(t.TextDim)

	valueStyle := lipgloss.NewStyle().
		Foreground(t.TextBright)

	regionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Secondary)

	var lines []string

	// Title line
	title := fmt.Sprintf("%s/%s", e.Service, e.ResType)
	lines = append(lines, titleStyle.Render(title))

	// Resource info
	resourceLine := labelStyle.Render("Resource: ") + valueStyle.Render(e.Resource.GetName())
	if id := e.Resource.GetID(); id != e.Resource.GetName() {
		resourceLine += labelStyle.Render(" (") + valueStyle.Render(id) + labelStyle.Render(")")
	}
	lines = append(lines, resourceLine)

	// Context line
	contextParts := []string{}
	if region != "" {
		contextParts = append(contextParts, regionStyle.Render("["+region+"]"))
	}
	if accountID != "" {
		contextParts = append(contextParts, labelStyle.Render("Account: ")+valueStyle.Render(accountID))
	}
	if len(contextParts) > 0 {
		lines = append(lines, strings.Join(contextParts, " "))
	}

	// Hint line
	hintStyle := lipgloss.NewStyle().Foreground(ui.Current().TextDim).Italic(true)
	lines = append(lines, hintStyle.Render("Press Ctrl+D or type 'exit' to return to claws"))

	return strings.Join(lines, "\n")
}
