package view

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

func TestNewLogView(t *testing.T) {
	ctx := context.Background()
	lv := NewLogView(ctx, "/aws/lambda/my-function")

	if lv.logGroupName != "/aws/lambda/my-function" {
		t.Errorf("logGroupName = %q, want %q", lv.logGroupName, "/aws/lambda/my-function")
	}
	if lv.logStreamName != "" {
		t.Errorf("logStreamName = %q, want empty", lv.logStreamName)
	}
	if !lv.loading {
		t.Error("Expected loading to be true initially")
	}
	if lv.paused {
		t.Error("Expected paused to be false initially")
	}
	if lv.pollInterval != defaultLogPollInterval {
		t.Errorf("pollInterval = %v, want %v", lv.pollInterval, defaultLogPollInterval)
	}
}

func TestNewLogViewWithStream(t *testing.T) {
	ctx := context.Background()
	lv := NewLogViewWithStream(ctx, "/aws/lambda/my-function", "2024/01/01/[$LATEST]abc123", 0)

	if lv.logGroupName != "/aws/lambda/my-function" {
		t.Errorf("logGroupName = %q, want %q", lv.logGroupName, "/aws/lambda/my-function")
	}
	if lv.logStreamName != "2024/01/01/[$LATEST]abc123" {
		t.Errorf("logStreamName = %q, want %q", lv.logStreamName, "2024/01/01/[$LATEST]abc123")
	}
}

func TestLogViewLogsLoadedSuccess(t *testing.T) {
	ctx := context.Background()
	lv := NewLogView(ctx, "/aws/test")
	lv.SetSize(80, 24)

	entries := []logEntry{
		{timestamp: time.Now(), message: "log line 1"},
		{timestamp: time.Now(), message: "log line 2"},
	}
	msg := logsLoadedMsg{entries: entries, lastEventTime: 1234567890}

	lv.Update(msg)

	if lv.loading {
		t.Error("Expected loading to be false after logsLoadedMsg")
	}
	if len(lv.logs) != 2 {
		t.Errorf("len(logs) = %d, want 2", len(lv.logs))
	}
	if lv.lastEventTime != 1234567890 {
		t.Errorf("lastEventTime = %d, want 1234567890", lv.lastEventTime)
	}
	if lv.err != nil {
		t.Errorf("err = %v, want nil", lv.err)
	}
}

func TestLogViewLogsLoadedError(t *testing.T) {
	ctx := context.Background()
	lv := NewLogView(ctx, "/aws/test")
	lv.SetSize(80, 24)

	msg := logsLoadedMsg{err: fmt.Errorf("access denied")}

	lv.Update(msg)

	if lv.loading {
		t.Error("Expected loading to be false after error")
	}
	if lv.err == nil {
		t.Error("Expected err to be set after error message")
	}
}

func TestLogViewBufferTrimming(t *testing.T) {
	ctx := context.Background()
	lv := NewLogView(ctx, "/aws/test")
	lv.SetSize(80, 24)
	lv.loading = false

	for i := 0; i < 999; i++ {
		lv.logs = append(lv.logs, logEntry{
			timestamp: time.Now(),
			message:   fmt.Sprintf("line %d", i),
		})
	}

	newEntries := make([]logEntry, 10)
	for i := 0; i < 10; i++ {
		newEntries[i] = logEntry{timestamp: time.Now(), message: fmt.Sprintf("new line %d", i)}
	}
	msg := logsLoadedMsg{entries: newEntries, lastEventTime: 1}

	lv.Update(msg)

	if len(lv.logs) != 1000 {
		t.Errorf("len(logs) = %d, want 1000 (buffer should trim to max)", len(lv.logs))
	}

	if !strings.Contains(lv.logs[0].message, "line 9") {
		t.Errorf("first log = %q, expected oldest kept entry 'line 9'", lv.logs[0].message)
	}
}

func TestLogViewPauseToggle(t *testing.T) {
	ctx := context.Background()
	lv := NewLogView(ctx, "/aws/test")
	lv.SetSize(80, 24)
	lv.loading = false

	if lv.paused {
		t.Error("Expected paused to be false initially")
	}

	spaceMsg := tea.KeyPressMsg{Code: tea.KeySpace}
	lv.Update(spaceMsg)

	if !lv.paused {
		t.Error("Expected paused to be true after first space")
	}

	lv.Update(spaceMsg)

	if lv.paused {
		t.Error("Expected paused to be false after second space")
	}
}

func TestLogViewClearLogs(t *testing.T) {
	ctx := context.Background()
	lv := NewLogView(ctx, "/aws/test")
	lv.SetSize(80, 24)
	lv.loading = false
	lv.logs = []logEntry{
		{timestamp: time.Now(), message: "line 1"},
		{timestamp: time.Now(), message: "line 2"},
	}

	cMsg := tea.KeyPressMsg{Code: 0, Text: "c"}
	lv.Update(cMsg)

	if len(lv.logs) != 0 {
		t.Errorf("len(logs) = %d, want 0 after clear", len(lv.logs))
	}
}

func TestLogViewTickWhenPaused(t *testing.T) {
	ctx := context.Background()
	lv := NewLogView(ctx, "/aws/test")
	lv.SetSize(80, 24)
	lv.loading = false
	lv.paused = true

	tickMsg := logTickMsg(time.Now())
	_, cmd := lv.Update(tickMsg)

	if cmd != nil {
		t.Error("Expected nil cmd when paused (no fetch should be triggered)")
	}
}

func TestLogViewStatusLine(t *testing.T) {
	ctx := context.Background()
	lv := NewLogView(ctx, "/aws/test")

	streamingStatus := lv.StatusLine()
	if !strings.Contains(streamingStatus, "STREAMING") {
		t.Errorf("StatusLine() = %q, want to contain 'STREAMING'", streamingStatus)
	}

	lv.paused = true
	pausedStatus := lv.StatusLine()
	if !strings.Contains(pausedStatus, "PAUSED") {
		t.Errorf("StatusLine() = %q, want to contain 'PAUSED'", pausedStatus)
	}
}

func TestLogViewViewStringStates(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		setup       func(*LogView)
		wantContain string
	}{
		{
			name:        "loading state",
			setup:       func(lv *LogView) { lv.loading = true },
			wantContain: "Loading",
		},
		{
			name: "error state",
			setup: func(lv *LogView) {
				lv.loading = false
				lv.err = fmt.Errorf("test error")
			},
			wantContain: "Error",
		},
		{
			name: "empty state",
			setup: func(lv *LogView) {
				lv.loading = false
			},
			wantContain: "No log events",
		},
		{
			name: "paused state",
			setup: func(lv *LogView) {
				lv.loading = false
				lv.paused = true
			},
			wantContain: "PAUSED",
		},
		{
			name: "with stream name in title",
			setup: func(lv *LogView) {
				lv.loading = false
				lv.logStreamName = "my-stream"
			},
			wantContain: "my-stream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lv := NewLogView(ctx, "/aws/test")
			lv.SetSize(80, 24)
			tt.setup(lv)

			view := lv.ViewString()
			if !strings.Contains(view, tt.wantContain) {
				t.Errorf("ViewString() = %q, want to contain %q", view, tt.wantContain)
			}
		})
	}
}

func TestLogViewSetSize(t *testing.T) {
	ctx := context.Background()
	lv := NewLogView(ctx, "/aws/test")

	cmd := lv.SetSize(120, 40)

	if cmd != nil {
		t.Error("Expected SetSize to return nil cmd")
	}
	if !lv.vp.Ready {
		t.Error("Expected vp.Ready to be true after SetSize")
	}
}

func TestLogViewGotoTopBottom(t *testing.T) {
	ctx := context.Background()
	lv := NewLogView(ctx, "/aws/test")
	lv.SetSize(80, 24)
	lv.loading = false

	for i := 0; i < 50; i++ {
		lv.logs = append(lv.logs, logEntry{
			timestamp: time.Now(),
			message:   fmt.Sprintf("line %d", i),
		})
	}
	lv.updateViewportContent()

	gMsg := tea.KeyPressMsg{Code: 0, Text: "g"}
	lv.Update(gMsg)

	GMsg := tea.KeyPressMsg{Code: 0, Text: "G"}
	lv.Update(GMsg)
}
