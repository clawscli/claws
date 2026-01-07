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

func TestLogViewFilterActivation(t *testing.T) {
	ctx := context.Background()
	lv := NewLogView(ctx, "/aws/test")
	lv.SetSize(80, 24)
	lv.loading = false

	if lv.filterActive {
		t.Error("Expected filterActive to be false initially")
	}

	// Activate filter with "/"
	slashMsg := tea.KeyPressMsg{Code: 0, Text: "/"}
	lv.Update(slashMsg)

	if !lv.filterActive {
		t.Error("Expected filterActive to be true after '/'")
	}

	// Deactivate with Esc
	escMsg := tea.KeyPressMsg{Code: tea.KeyEscape}
	lv.Update(escMsg)

	if lv.filterActive {
		t.Error("Expected filterActive to be false after Esc")
	}
}

func TestLogViewFilterMatching(t *testing.T) {
	ctx := context.Background()
	lv := NewLogView(ctx, "/aws/test")
	lv.SetSize(80, 24)
	lv.loading = false

	entry1 := logEntry{timestamp: time.Now(), message: "ERROR: failed to connect"}
	entry2 := logEntry{timestamp: time.Now(), message: "INFO: connection successful"}
	entry3 := logEntry{timestamp: time.Now(), message: "ERROR: timeout"}

	// Case-insensitive substring match
	lv.filterText = "error"
	if !lv.matchesFilter(entry1) {
		t.Error("Expected entry1 to match filter 'error'")
	}
	if lv.matchesFilter(entry2) {
		t.Error("Expected entry2 to not match filter 'error'")
	}
	if !lv.matchesFilter(entry3) {
		t.Error("Expected entry3 to match filter 'error'")
	}

	// Empty filter matches all
	lv.filterText = ""
	if !lv.matchesFilter(entry1) || !lv.matchesFilter(entry2) || !lv.matchesFilter(entry3) {
		t.Error("Expected all entries to match empty filter")
	}
}

func TestLogViewFilterClear(t *testing.T) {
	ctx := context.Background()
	lv := NewLogView(ctx, "/aws/test")
	lv.SetSize(80, 24)
	lv.loading = false
	lv.logs = []logEntry{
		{timestamp: time.Now(), message: "line 1"},
		{timestamp: time.Now(), message: "line 2"},
	}

	// Set filter
	lv.filterText = "test"
	lv.filterInput.SetValue("test")

	// Clear filter with "c"
	cMsg := tea.KeyPressMsg{Code: 0, Text: "c"}
	lv.Update(cMsg)

	if lv.filterText != "" {
		t.Errorf("Expected filterText to be empty after 'c', got %q", lv.filterText)
	}
	if lv.filterInput.Value() != "" {
		t.Error("Expected filterInput value to be empty after 'c'")
	}
	if len(lv.logs) != 2 {
		t.Error("Expected logs to remain after clearing filter")
	}

	// Second "c" clears buffer
	lv.Update(cMsg)
	if len(lv.logs) != 0 {
		t.Errorf("Expected logs to be cleared after second 'c', got %d logs", len(lv.logs))
	}
}

func TestLogViewFilteredCount(t *testing.T) {
	ctx := context.Background()
	lv := NewLogView(ctx, "/aws/test")
	lv.SetSize(80, 24)
	lv.loading = false
	lv.logs = []logEntry{
		{timestamp: time.Now(), message: "ERROR: test1"},
		{timestamp: time.Now(), message: "INFO: test2"},
		{timestamp: time.Now(), message: "ERROR: test3"},
		{timestamp: time.Now(), message: "WARN: test4"},
	}

	// No filter
	count := lv.getDisplayedCount()
	if count != 4 {
		t.Errorf("getDisplayedCount() = %d, want 4", count)
	}

	// Filter for "error"
	lv.filterText = "error"
	count = lv.getDisplayedCount()
	if count != 2 {
		t.Errorf("getDisplayedCount() with filter 'error' = %d, want 2", count)
	}
}

func TestLogViewHasActiveInput(t *testing.T) {
	ctx := context.Background()
	lv := NewLogView(ctx, "/aws/test")
	lv.SetSize(80, 24)

	if lv.HasActiveInput() {
		t.Error("Expected HasActiveInput to be false initially")
	}

	lv.filterActive = true
	if !lv.HasActiveInput() {
		t.Error("Expected HasActiveInput to be true when filterActive")
	}

	lv.filterActive = false
	if lv.HasActiveInput() {
		t.Error("Expected HasActiveInput to be false when filterActive is false")
	}
}

func TestLogViewFilterSetSizeRecalculation(t *testing.T) {
	ctx := context.Background()
	lv := NewLogView(ctx, "/aws/test")
	lv.SetSize(80, 24)
	lv.loading = false

	// Store original viewport height
	originalHeight := lv.vp.Model.Height()

	// Set filter
	lv.filterText = "test"

	// SetSize should adjust for filter UI
	lv.SetSize(80, 24)

	// Viewport height should be reduced by 1 line
	if lv.vp.Model.Height() != originalHeight-1 {
		t.Errorf("Expected viewport height to be %d with filter, got %d", originalHeight-1, lv.vp.Model.Height())
	}

	// Clear filter
	lv.filterText = ""
	lv.SetSize(80, 24)

	// Viewport height should be restored
	if lv.vp.Model.Height() != originalHeight {
		t.Errorf("Expected viewport height to be %d after clearing filter, got %d", originalHeight, lv.vp.Model.Height())
	}
}

func TestLogViewFilterStatusLine(t *testing.T) {
	ctx := context.Background()
	lv := NewLogView(ctx, "/aws/test")
	lv.SetSize(80, 24)

	// No filter
	status := lv.StatusLine()
	if strings.Contains(status, "🔍") {
		t.Error("Expected status line to not contain filter indicator without filter")
	}

	// With filter
	lv.filterText = "error"
	status = lv.StatusLine()
	if !strings.Contains(status, "🔍") || !strings.Contains(status, "error") {
		t.Errorf("Expected status line to contain filter indicator, got %q", status)
	}

	// Long filter truncation
	lv.filterText = "this is a very long filter text that should be truncated"
	status = lv.StatusLine()
	if !strings.Contains(status, "...") {
		t.Errorf("Expected long filter to be truncated, got %q", status)
	}

	// Filter input active
	lv.filterActive = true
	status = lv.StatusLine()
	if !strings.Contains(status, "Esc:cancel") || !strings.Contains(status, "Enter:done") {
		t.Errorf("Expected filter input status line, got %q", status)
	}
}
