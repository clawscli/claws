package sanitize

import (
	"regexp"
	"strings"
	"unicode"
)

const Redacted = "[REDACTED]"

var sensitiveAssignmentPattern = regexp.MustCompile(`(?i)(^|[^A-Za-z0-9_])((?:aws[_-]?)?secret[_-]?access[_-]?key|password|passwd|pwd|secret|token|api[_-]?key|access[_-]?key(?:[_-]?id)?|credential)(\s*[:=]\s*)("[^"]*"|'[^']*'|[^\s,;]+)`)
var uriCredentialPattern = regexp.MustCompile(`(?i)\b([a-z][a-z0-9+.-]*://)([^/\s:@]+):([^@\s/]+)@`)
var bearerCredentialPattern = regexp.MustCompile(`(?i)\bbearer\s+[A-Za-z0-9._~+/=-]{16,}`)
var basicCredentialPattern = regexp.MustCompile(`\b[Bb]asic\s+[A-Za-z0-9+/=]*[A-Z0-9+/=][A-Za-z0-9+/=]{7,}`)
var jwtPattern = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]*\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`)
var awsAccessKeyPattern = regexp.MustCompile(`\b(?:AKIA|ASIA)[A-Z0-9]{16}\b`)
var pemBlockPattern = regexp.MustCompile(`(?s)-----BEGIN [A-Z0-9 ]+-----.*?-----END [A-Z0-9 ]+-----`)
var ansiEscapePattern = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]|\x1b\][^\x07]*(\x07|\x1b\\)|\x1b[@-Z\\-_]`)

// TerminalText removes ANSI escape sequences and control characters that can alter terminal state.
func TerminalText(s string) string {
	s = ansiEscapePattern.ReplaceAllString(s, "")
	return strings.Map(func(r rune) rune {
		if r == '\t' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}

// SensitiveText redacts common key=value or key:value secret assignments.
func SensitiveText(s string) string {
	s = sensitiveAssignmentPattern.ReplaceAllString(s, `${1}${2}${3}`+Redacted)
	s = uriCredentialPattern.ReplaceAllString(s, `${1}`+Redacted+`@`)
	s = bearerCredentialPattern.ReplaceAllString(s, `Bearer `+Redacted)
	s = basicCredentialPattern.ReplaceAllString(s, `Basic `+Redacted)
	s = jwtPattern.ReplaceAllString(s, Redacted)
	s = awsAccessKeyPattern.ReplaceAllString(s, Redacted)
	s = pemBlockPattern.ReplaceAllString(s, Redacted)
	return s
}

// LogText prepares untrusted log text for display or AI output.
func LogText(s string) string {
	return SensitiveText(TerminalText(s))
}
