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

// setAWSEnv configures AWS environment variables on the command.
// Handles both profile selection and region injection.
//
// Profile behavior depends on the credential mode:
//   - SDKDefault: preserve existing AWS_PROFILE (don't modify)
//   - EnvOnly: remove AWS_PROFILE to force IMDS/env vars
//   - NamedProfile: set AWS_PROFILE to the profile name
//
// Region behavior:
//   - If region is set in config, inject both AWS_REGION and AWS_DEFAULT_REGION
//   - If region is empty, don't modify existing region env vars
func setAWSEnv(cmd *exec.Cmd) {
	cfg := config.Global()
	sel := cfg.Selection()
	region := cfg.Region()

	// Start with existing cmd.Env or os.Environ()
	baseEnv := cmd.Env
	if baseEnv == nil {
		baseEnv = os.Environ()
	}

	// Build filtered env, removing keys we'll set
	keysToRemove := map[string]bool{}

	switch sel.Mode {
	case config.ModeEnvOnly:
		keysToRemove["AWS_PROFILE"] = true
	case config.ModeNamedProfile:
		keysToRemove["AWS_PROFILE"] = true
	}

	if region != "" {
		keysToRemove["AWS_REGION"] = true
		keysToRemove["AWS_DEFAULT_REGION"] = true
	}

	// Filter and rebuild env
	env := make([]string, 0, len(baseEnv)+3)
	for _, e := range baseEnv {
		keep := true
		for key := range keysToRemove {
			if strings.HasPrefix(e, key+"=") {
				keep = false
				break
			}
		}
		if keep {
			env = append(env, e)
		}
	}

	// Add profile-related env vars based on mode
	switch sel.Mode {
	case config.ModeNamedProfile:
		env = append(env, "AWS_PROFILE="+sel.ProfileName)
	case config.ModeEnvOnly:
		// Force CLI to ignore config files, use IMDS/env only
		env = append(env, "AWS_CONFIG_FILE=/dev/null")
		env = append(env, "AWS_SHARED_CREDENTIALS_FILE=/dev/null")
	}

	// Add region if set
	if region != "" {
		env = append(env, "AWS_REGION="+region)
		env = append(env, "AWS_DEFAULT_REGION="+region)
	}

	cmd.Env = env
}

// SimpleExec represents a simple exec command without header.
// Implements tea.ExecCommand interface.
type SimpleExec struct {
	Command string

	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

// SetStdin sets the stdin for the command
func (e *SimpleExec) SetStdin(r io.Reader) { e.stdin = r }

// SetStdout sets the stdout for the command
func (e *SimpleExec) SetStdout(w io.Writer) { e.stdout = w }

// SetStderr sets the stderr for the command
func (e *SimpleExec) SetStderr(w io.Writer) { e.stderr = w }

// Run executes the command
func (e *SimpleExec) Run() error {
	if e.Command == "" {
		return ErrEmptyCommand
	}

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

	cmd := exec.Command("/bin/sh", "-c", e.Command)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	setAWSEnv(cmd)

	return cmd.Run()
}

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
		return ErrEmptyCommand
	}

	cmd := exec.Command("/bin/sh", "-c", e.Command)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	setAWSEnv(cmd)

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

func (e *ExecWithHeader) buildHeader(_ int) string {
	profileDisplay := config.Global().Selection().DisplayName()
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

	// Context line: Profile, Region, Account
	contextParts := []string{
		labelStyle.Render("Profile: ") + valueStyle.Render(profileDisplay),
	}
	if region != "" {
		contextParts = append(contextParts, regionStyle.Render("["+region+"]"))
	}
	if accountID != "" {
		contextParts = append(contextParts, labelStyle.Render("Account: ")+valueStyle.Render(accountID))
	}
	lines = append(lines, strings.Join(contextParts, " "))

	// Hint line
	hintStyle := lipgloss.NewStyle().Foreground(ui.Current().TextDim).Italic(true)
	lines = append(lines, hintStyle.Render("Press Ctrl+D or type 'exit' to return to claws"))

	return strings.Join(lines, "\n")
}
