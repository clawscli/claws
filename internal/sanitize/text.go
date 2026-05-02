package sanitize

import (
	"regexp"
	"strings"
	"unicode"
)

const Redacted = "[REDACTED]"

var sensitiveAssignmentPattern = regexp.MustCompile(`(?i)(^|[^A-Za-z0-9_])((?:aws[_-]?)?secret[_-]?access[_-]?key|password|passwd|pwd|secret|token|api[_-]?key|access[_-]?key(?:[_-]?id)?|credential)(\s*[:=]\s*)("[^"]*"|'[^']*'|[^\s,;]+)`)
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
	return sensitiveAssignmentPattern.ReplaceAllString(s, `${1}${2}${3}`+Redacted)
}

// LogText prepares untrusted log text for display or AI output.
func LogText(s string) string {
	return SensitiveText(TerminalText(s))
}
