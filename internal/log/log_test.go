package log

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestEnableDisable(t *testing.T) {
	// Start disabled
	if IsEnabled() {
		t.Error("expected logging to be disabled by default")
	}

	// Enable
	var buf bytes.Buffer
	Enable(&buf)
	if !IsEnabled() {
		t.Error("expected logging to be enabled after Enable()")
	}

	// Log something
	Info("test message", "key", "value")
	if !strings.Contains(buf.String(), "test message") {
		t.Errorf("expected log to contain 'test message', got: %s", buf.String())
	}

	// Disable
	Disable()
	if IsEnabled() {
		t.Error("expected logging to be disabled after Disable()")
	}
}

func TestLogLevels(t *testing.T) {
	var buf bytes.Buffer
	Enable(&buf)
	defer Disable()

	Debug("debug msg")
	Info("info msg")
	Warn("warn msg")
	Error("error msg")

	output := buf.String()
	if !strings.Contains(output, "debug msg") {
		t.Error("expected debug message in output")
	}
	if !strings.Contains(output, "info msg") {
		t.Error("expected info message in output")
	}
	if !strings.Contains(output, "warn msg") {
		t.Error("expected warn message in output")
	}
	if !strings.Contains(output, "error msg") {
		t.Error("expected error message in output")
	}
}

func TestSetLevel(t *testing.T) {
	var buf bytes.Buffer
	Enable(&buf)
	defer Disable()

	// Set level to Warn
	SetLevel(slog.LevelWarn)

	Debug("debug msg")
	Info("info msg")
	Warn("warn msg")
	Error("error msg")

	output := buf.String()
	if strings.Contains(output, "debug msg") {
		t.Error("debug message should not appear at Warn level")
	}
	if strings.Contains(output, "info msg") {
		t.Error("info message should not appear at Warn level")
	}
	if !strings.Contains(output, "warn msg") {
		t.Error("expected warn message in output")
	}
	if !strings.Contains(output, "error msg") {
		t.Error("expected error message in output")
	}
}

func TestWith(t *testing.T) {
	var buf bytes.Buffer
	Enable(&buf)
	defer Disable()

	subLogger := With("service", "test")
	subLogger.Info("sublogger message")

	output := buf.String()
	if !strings.Contains(output, "service=test") {
		t.Error("expected 'service=test' in output")
	}
}
