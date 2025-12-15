package app

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"
)

// TestPTYEscHandling runs the actual app in a PTY and tests esc handling
func TestPTYEscHandling(t *testing.T) {
	// Build the app first
	buildCmd := exec.Command("go", "build", "-o", "/tmp/claws-test", "../../cmd/claws")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build: %v\n%s", err, out)
	}
	defer func() { _ = os.Remove("/tmp/claws-test") }()

	// Run the app in a PTY
	cmd := exec.Command("/tmp/claws-test")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		t.Fatalf("Failed to start PTY: %v", err)
	}
	defer func() { _ = ptmx.Close() }()
	defer func() { _ = cmd.Process.Kill() }()

	// Set PTY size
	_ = pty.Setsize(ptmx, &pty.Winsize{Rows: 50, Cols: 100})

	// Read output in background
	outputChan := make(chan string, 100)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				return
			}
			if n > 0 {
				outputChan <- string(buf[:n])
			}
		}
	}()

	// Helper to read all available output
	readOutput := func(timeout time.Duration) string {
		var result strings.Builder
		deadline := time.After(timeout)
		for {
			select {
			case s := <-outputChan:
				result.WriteString(s)
			case <-deadline:
				return result.String()
			}
		}
	}

	// Helper to find stack info in output
	findStack := func(output string) string {
		idx := strings.Index(output, "[stack:")
		if idx == -1 {
			return "not found"
		}
		end := strings.Index(output[idx:], "]")
		if end == -1 {
			return output[idx:]
		}
		return output[idx : idx+end+1]
	}

	// Wait for initial queries
	t.Log("Waiting for initial terminal queries...")
	time.Sleep(100 * time.Millisecond)
	initialQueries := readOutput(100 * time.Millisecond)
	t.Logf("Initial queries: %q", initialQueries)

	// Respond to cursor position query
	t.Log("Sending cursor position response...")
	_, _ = ptmx.Write([]byte("\x1b[1;1R"))
	time.Sleep(300 * time.Millisecond)

	// Now wait for UI render
	initialOutput := readOutput(500 * time.Millisecond)
	t.Logf("Initial output (last 500 chars): ...%s", lastN(initialOutput, 500))
	t.Logf("Initial stack: %s", findStack(initialOutput))

	// Navigate: press 'j' to move down, then 'enter' to select EC2
	t.Log("Pressing 'j' then 'enter' to navigate to EC2...")
	_, _ = ptmx.Write([]byte("j"))
	time.Sleep(100 * time.Millisecond)
	_, _ = ptmx.Write([]byte("\r")) // enter
	time.Sleep(500 * time.Millisecond)

	output1 := readOutput(200 * time.Millisecond)
	t.Logf("After enter (last 500 chars): ...%s", lastN(output1, 500))
	t.Logf("Stack after enter: %s", findStack(output1))

	// Now press 'enter' again to go to detail view
	t.Log("Pressing 'enter' to go to detail view...")
	_, _ = ptmx.Write([]byte("\r"))
	time.Sleep(500 * time.Millisecond)

	output2 := readOutput(200 * time.Millisecond)
	t.Logf("After second enter (last 500 chars): ...%s", lastN(output2, 500))
	t.Logf("Stack after second enter: %s", findStack(output2))

	// Now test ESC key
	t.Log("Pressing ESC...")
	_, _ = ptmx.Write([]byte{0x1b}) // ESC byte
	time.Sleep(300 * time.Millisecond)

	output3 := readOutput(200 * time.Millisecond)
	t.Logf("After ESC (last 500 chars): ...%s", lastN(output3, 500))
	t.Logf("Stack after ESC: %s", findStack(output3))

	// Check what key was received
	keyIdx := strings.Index(output3, "key=")
	if keyIdx != -1 {
		keyEnd := strings.Index(output3[keyIdx:], "]")
		if keyEnd != -1 {
			t.Logf("Key info: %s", output3[keyIdx:keyIdx+keyEnd])
		}
	}

	// Press ESC again
	t.Log("Pressing ESC again...")
	_, _ = ptmx.Write([]byte{0x1b})
	time.Sleep(300 * time.Millisecond)

	output4 := readOutput(200 * time.Millisecond)
	t.Logf("After second ESC (last 500 chars): ...%s", lastN(output4, 500))
	t.Logf("Stack after second ESC: %s", findStack(output4))

	// Quit
	t.Log("Pressing 'q' to quit...")
	_, _ = ptmx.Write([]byte("q"))
	time.Sleep(100 * time.Millisecond)

	_ = cmd.Process.Signal(os.Interrupt)
	_ = cmd.Wait()
}

func lastN(s string, n int) string {
	// Remove ANSI escape codes for cleaner output
	s = stripANSI(s)
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

func stripANSI(s string) string {
	var result bytes.Buffer
	inEscape := false
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			inEscape = true
			continue
		}
		if inEscape {
			if (s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z') {
				inEscape = false
			}
			continue
		}
		result.WriteByte(s[i])
	}
	return result.String()
}
