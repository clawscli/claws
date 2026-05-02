package sanitize

import (
	"strings"
	"testing"
)

func TestLogTextRedactsCommonSecretAssignments(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		secret string
	}{
		{
			name:   "aws secret access key",
			input:  "AWS_SECRET_ACCESS_KEY=plain-secret",
			secret: "plain-secret",
		},
		{
			name:   "quoted token with spaces",
			input:  `token="plain secret with spaces"`,
			secret: "plain secret with spaces",
		},
		{
			name:   "colon secret",
			input:  "secret:plain-secret",
			secret: "plain-secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LogText(tt.input)
			if strings.Contains(got, tt.secret) {
				t.Fatalf("LogText(%q) leaked secret in %q", tt.input, got)
			}
			if !strings.Contains(got, Redacted) {
				t.Fatalf("LogText(%q) = %q, want redaction marker", tt.input, got)
			}
		})
	}
}

func TestLogTextRemovesTerminalEscapeSequences(t *testing.T) {
	got := LogText("ok \x1b[31mred\x1b[0m")
	if strings.Contains(got, "\x1b") || strings.Contains(got, "[31m") {
		t.Fatalf("LogText left terminal escape sequence in %q", got)
	}
	if !strings.Contains(got, "ok red") {
		t.Fatalf("LogText removed visible text, got %q", got)
	}
}
