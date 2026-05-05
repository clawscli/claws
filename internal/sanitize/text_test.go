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

func TestSensitiveTextRedactsValueOnlySecretPatterns(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		secret string
	}{
		{
			name:   "uri credentials",
			input:  "postgres://app:super-secret-password@db.example.com:5432/app",
			secret: "super-secret-password",
		},
		{
			name:   "bearer token",
			input:  "Authorization: Bearer abcdefghijklmnop",
			secret: "abcdefghijklmnop",
		},
		{
			name:   "basic token",
			input:  "Authorization: Basic dXNlcjpwYXNz",
			secret: "dXNlcjpwYXNz",
		},
		{
			name:   "jwt",
			input:  "jwt eyJhbGciOiJIUzI1NiJ9.payload.signature",
			secret: "eyJhbGciOiJIUzI1NiJ9.payload.signature",
		},
		{
			name:   "aws access key id",
			input:  "caller AKIAIOSFODNN7EXAMPLE",
			secret: "AKIAIOSFODNN7EXAMPLE",
		},
		{
			name:   "pem block",
			input:  "-----BEGIN PRIVATE KEY-----\nplain-private-key\n-----END PRIVATE KEY-----",
			secret: "plain-private-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SensitiveText(tt.input)
			if strings.Contains(got, tt.secret) {
				t.Fatalf("SensitiveText(%q) leaked secret in %q", tt.input, got)
			}
			if !strings.Contains(got, Redacted) {
				t.Fatalf("SensitiveText(%q) = %q, want redaction marker", tt.input, got)
			}
		})
	}
}

func TestSensitiveTextPreservesBasicDocumentationPhrase(t *testing.T) {
	input := "basic authentication for CloudFront"
	got := SensitiveText(input)
	if got != input {
		t.Fatalf("SensitiveText(%q) = %q, want unchanged documentation phrase", input, got)
	}
}
