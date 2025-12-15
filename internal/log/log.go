// Package log provides structured logging using Go's slog package.
// It wraps slog with convenience functions and allows global configuration.
package log

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
)

var (
	mu      sync.RWMutex
	logger  *slog.Logger
	level   = new(slog.LevelVar)
	enabled = false
)

func init() {
	// Default to disabled (no-op logger)
	logger = slog.New(slog.NewTextHandler(io.Discard, nil))
}

// Enable enables logging to the specified writer (typically a file).
// By default, logging is disabled for TUI cleanliness.
func Enable(w io.Writer) {
	mu.Lock()
	defer mu.Unlock()

	level.Set(slog.LevelDebug)
	logger = slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: level,
	}))
	enabled = true
}

// EnableFile enables logging to a file at the specified path.
// Creates the file if it doesn't exist.
func EnableFile(path string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	Enable(f)
	return nil
}

// Disable disables logging (default state).
func Disable() {
	mu.Lock()
	defer mu.Unlock()

	logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	enabled = false
}

// IsEnabled returns whether logging is enabled.
func IsEnabled() bool {
	mu.RLock()
	defer mu.RUnlock()
	return enabled
}

// SetLevel sets the minimum log level.
func SetLevel(l slog.Level) {
	level.Set(l)
}

// Debug logs a debug message.
func Debug(msg string, args ...any) {
	mu.RLock()
	l := logger
	mu.RUnlock()
	l.Debug(msg, args...)
}

// Info logs an info message.
func Info(msg string, args ...any) {
	mu.RLock()
	l := logger
	mu.RUnlock()
	l.Info(msg, args...)
}

// Warn logs a warning message.
func Warn(msg string, args ...any) {
	mu.RLock()
	l := logger
	mu.RUnlock()
	l.Warn(msg, args...)
}

// Error logs an error message.
func Error(msg string, args ...any) {
	mu.RLock()
	l := logger
	mu.RUnlock()
	l.Error(msg, args...)
}

// DebugContext logs a debug message with context.
func DebugContext(ctx context.Context, msg string, args ...any) {
	mu.RLock()
	l := logger
	mu.RUnlock()
	l.DebugContext(ctx, msg, args...)
}

// InfoContext logs an info message with context.
func InfoContext(ctx context.Context, msg string, args ...any) {
	mu.RLock()
	l := logger
	mu.RUnlock()
	l.InfoContext(ctx, msg, args...)
}

// WarnContext logs a warning message with context.
func WarnContext(ctx context.Context, msg string, args ...any) {
	mu.RLock()
	l := logger
	mu.RUnlock()
	l.WarnContext(ctx, msg, args...)
}

// ErrorContext logs an error message with context.
func ErrorContext(ctx context.Context, msg string, args ...any) {
	mu.RLock()
	l := logger
	mu.RUnlock()
	l.ErrorContext(ctx, msg, args...)
}

// With returns a logger with additional attributes.
func With(args ...any) *slog.Logger {
	mu.RLock()
	l := logger
	mu.RUnlock()
	return l.With(args...)
}
