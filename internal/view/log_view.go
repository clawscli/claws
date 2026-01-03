package view

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/config"
	apperrors "github.com/clawscli/claws/internal/errors"
	"github.com/clawscli/claws/internal/log"
	"github.com/clawscli/claws/internal/ui"
)

const (
	defaultLogPollInterval = 3 * time.Second
	maxLogPollInterval     = 30 * time.Second
	initialLogBufferSize   = 500
	maxLogBufferSize       = 1000
	logFetchLimit          = 100
	viewportHeaderOffset   = 4 // header(1) + status(2) + spacing(1)
)

type LogView struct {
	ctx           context.Context
	client        *cloudwatchlogs.Client
	logGroupName  string
	logStreamName string

	vp      ViewportState
	spinner spinner.Model
	styles  logViewStyles

	logs    []logEntry
	loading bool
	paused  bool
	err     error

	lastEventTime   int64
	oldestEventTime int64
	pollInterval    time.Duration
}

type logEntry struct {
	timestamp time.Time
	message   string
}

type logViewStyles struct {
	header    lipgloss.Style
	timestamp lipgloss.Style
	message   lipgloss.Style
	paused    lipgloss.Style
	error     lipgloss.Style
	dim       lipgloss.Style
}

func newLogViewStyles() logViewStyles {
	return logViewStyles{
		header:    ui.TitleStyle(),
		timestamp: ui.SecondaryStyle(),
		message:   ui.TextStyle(),
		paused:    ui.BoldWarningStyle(),
		error:     ui.DangerStyle(),
		dim:       ui.DimStyle(),
	}
}

func NewLogView(ctx context.Context, logGroupName string) *LogView {
	return &LogView{
		ctx:          ctx,
		logGroupName: logGroupName,
		spinner:      ui.NewSpinner(),
		styles:       newLogViewStyles(),
		logs:         make([]logEntry, 0, initialLogBufferSize),
		loading:      true,
		pollInterval: defaultLogPollInterval,
	}
}

func NewLogViewWithStream(ctx context.Context, logGroupName, logStreamName string, lastEventTime int64) *LogView {
	v := NewLogView(ctx, logGroupName)
	v.logStreamName = logStreamName
	if lastEventTime > 0 {
		v.lastEventTime = lastEventTime - time.Hour.Milliseconds()
	}
	return v
}

type logsLoadedMsg struct {
	entries       []logEntry
	lastEventTime int64
	err           error
	throttled     bool
	older         bool
}

type logTickMsg time.Time

func (v *LogView) Init() tea.Cmd {
	return tea.Batch(
		v.initClient,
		v.spinner.Tick,
	)
}

func (v *LogView) initClient() tea.Msg {
	if err := v.ctx.Err(); err != nil {
		return logsLoadedMsg{err: err}
	}
	cfg, err := appaws.NewConfig(v.ctx)
	if err != nil {
		return logsLoadedMsg{err: apperrors.Wrap(err, "init AWS config")}
	}
	v.client = cloudwatchlogs.NewFromConfig(cfg)
	return v.doFetchLogs(v.lastEventTime, 0, false)
}

func (v *LogView) fetchLogsCmd() tea.Cmd {
	startTime := v.lastEventTime
	return func() tea.Msg {
		return v.doFetchLogs(startTime, 0, false)
	}
}

func (v *LogView) fetchOlderLogsCmd() tea.Cmd {
	endTime := v.oldestEventTime
	if endTime == 0 {
		return nil
	}
	return func() tea.Msg {
		return v.doFetchLogs(0, endTime, true)
	}
}

func (v *LogView) doFetchLogs(startTime, endTime int64, older bool) tea.Msg {
	if err := v.ctx.Err(); err != nil {
		return logsLoadedMsg{err: err, older: older}
	}
	if v.client == nil {
		return logsLoadedMsg{
			err:   apperrors.Wrap(fmt.Errorf("CloudWatch Logs client not initialized"), "fetch logs"),
			older: older,
		}
	}

	ctx, cancel := context.WithTimeout(v.ctx, config.File().LogFetchTimeout())
	defer cancel()

	input := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName: appaws.StringPtr(v.logGroupName),
		Limit:        appaws.Int32Ptr(logFetchLimit),
	}

	if v.logStreamName != "" {
		input.LogStreamNames = []string{v.logStreamName}
	}

	if older {
		input.StartTime = appaws.Int64Ptr(endTime - time.Hour.Milliseconds())
		input.EndTime = appaws.Int64Ptr(endTime - 1)
	} else if startTime > 0 {
		input.StartTime = appaws.Int64Ptr(startTime + 1)
	} else {
		input.StartTime = appaws.Int64Ptr(time.Now().Add(-1 * time.Hour).UnixMilli())
	}

	output, err := v.client.FilterLogEvents(ctx, input)
	if err != nil {
		return v.handleFetchError(err, older)
	}

	return v.processLogEvents(output.Events, older)
}

func (v *LogView) handleFetchError(err error, older bool) logsLoadedMsg {
	var wrappedErr error
	throttled := apperrors.IsThrottling(err)

	switch {
	case apperrors.IsNotFound(err):
		if v.logStreamName != "" {
			wrappedErr = apperrors.Wrap(err, "log stream not found")
		} else {
			wrappedErr = apperrors.Wrap(err, "log group not found")
		}
	case apperrors.IsAccessDenied(err):
		wrappedErr = apperrors.Wrap(err, "access denied to CloudWatch Logs")
	default:
		wrappedErr = apperrors.Wrap(err, "filter log events")
	}

	return logsLoadedMsg{err: wrappedErr, throttled: throttled, older: older}
}

func (v *LogView) processLogEvents(events []types.FilteredLogEvent, older bool) logsLoadedMsg {
	var boundaryTime int64
	entries := make([]logEntry, 0, len(events))

	for _, event := range events {
		ts := time.UnixMilli(appaws.Int64(event.Timestamp))
		msg := appaws.Str(event.Message)
		entries = append(entries, logEntry{
			timestamp: ts,
			message:   strings.TrimSuffix(msg, "\n"),
		})

		eventTs := appaws.Int64(event.Timestamp)
		if older {
			if boundaryTime == 0 || eventTs < boundaryTime {
				boundaryTime = eventTs
			}
		} else {
			if eventTs > boundaryTime {
				boundaryTime = eventTs
			}
		}
	}

	return logsLoadedMsg{entries: entries, lastEventTime: boundaryTime, older: older}
}

func (v *LogView) tickCmd() tea.Cmd {
	return tea.Tick(v.pollInterval, func(t time.Time) tea.Msg {
		return logTickMsg(t)
	})
}

func (v *LogView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case logsLoadedMsg:
		v.loading = false
		if msg.err != nil {
			log.Warn("failed to fetch log events", "error", msg.err)
			v.err = msg.err
			if msg.throttled {
				v.pollInterval = min(v.pollInterval*2, maxLogPollInterval)
				log.Info("throttled, backing off", "interval", v.pollInterval)
				if !v.paused && !msg.older {
					return v, v.tickCmd()
				}
			}
			return v, nil
		}
		v.pollInterval = defaultLogPollInterval
		v.err = nil
		if msg.older {
			if len(msg.entries) > 0 {
				v.logs = append(msg.entries, v.logs...)
				if len(v.logs) > maxLogBufferSize {
					v.logs = v.logs[:maxLogBufferSize]
				}
				if msg.lastEventTime > 0 {
					v.oldestEventTime = msg.lastEventTime
				}
				if v.vp.Ready {
					v.updateViewportContent()
				}
			}
			return v, nil
		}
		if msg.lastEventTime > v.lastEventTime {
			v.lastEventTime = msg.lastEventTime
		}
		if len(msg.entries) > 0 {
			if v.oldestEventTime == 0 && len(msg.entries) > 0 {
				v.oldestEventTime = msg.entries[0].timestamp.UnixMilli()
			}
			v.logs = append(v.logs, msg.entries...)
			if len(v.logs) > maxLogBufferSize {
				v.logs = v.logs[len(v.logs)-maxLogBufferSize:]
			}
			if v.vp.Ready {
				v.updateViewportContent()
				v.vp.Model.GotoBottom()
			}
		}
		if !v.paused {
			return v, v.tickCmd()
		}
		return v, nil

	case logTickMsg:
		if v.paused {
			return v, nil
		}
		return v, v.fetchLogsCmd()

	case tea.KeyPressMsg:
		switch msg.String() {
		case "space":
			v.paused = !v.paused
			if !v.paused {
				return v, v.tickCmd()
			}
			return v, nil
		case "g":
			if v.vp.Ready {
				v.vp.Model.GotoTop()
			}
			return v, nil
		case "G":
			if v.vp.Ready {
				v.vp.Model.GotoBottom()
			}
			return v, nil
		case "c":
			v.logs = v.logs[:0]
			v.oldestEventTime = 0
			if v.vp.Ready {
				v.updateViewportContent()
			}
			return v, nil
		case "p":
			if v.oldestEventTime > 0 && !v.loading {
				v.loading = true
				return v, v.fetchOlderLogsCmd()
			}
			return v, nil
		}

	case spinner.TickMsg:
		if v.loading {
			var cmd tea.Cmd
			v.spinner, cmd = v.spinner.Update(msg)
			return v, cmd
		}
	}

	if v.vp.Ready {
		var cmd tea.Cmd
		v.vp.Model, cmd = v.vp.Model.Update(msg)
		return v, cmd
	}
	return v, nil
}

func (v *LogView) updateViewportContent() {
	var sb strings.Builder
	for _, entry := range v.logs {
		ts := v.styles.timestamp.Render(entry.timestamp.Format("15:04:05.000"))
		msg := v.styles.message.Render(entry.message)
		sb.WriteString(fmt.Sprintf("%s %s\n", ts, msg))
	}
	v.vp.Model.SetContent(sb.String())
}

func (v *LogView) ViewString() string {
	if !v.vp.Ready {
		return LoadingMessage
	}

	var sb strings.Builder

	title := v.logGroupName
	if v.logStreamName != "" {
		title = fmt.Sprintf("%s / %s", v.logGroupName, v.logStreamName)
	}
	sb.WriteString(v.styles.header.Render("📜 " + title))
	sb.WriteString("\n")

	if v.paused {
		sb.WriteString(v.styles.paused.Render("⏸ PAUSED"))
		sb.WriteString(" ")
	}
	sb.WriteString(v.styles.dim.Render(fmt.Sprintf("(%d lines)", len(v.logs))))
	sb.WriteString("\n\n")

	if v.loading {
		sb.WriteString(v.spinner.View())
		sb.WriteString(" Loading logs...")
		return sb.String()
	}

	if v.err != nil {
		sb.WriteString(v.styles.error.Render(fmt.Sprintf("Error: %v", v.err)))
		return sb.String()
	}

	if len(v.logs) == 0 {
		sb.WriteString(v.styles.dim.Render("No log events found in the last hour"))
		return sb.String()
	}

	sb.WriteString(v.vp.Model.View())
	return sb.String()
}

func (v *LogView) View() tea.View {
	return tea.NewView(v.ViewString())
}

func (v *LogView) SetSize(width, height int) tea.Cmd {
	viewportHeight := height - viewportHeaderOffset
	v.vp.SetSize(width, viewportHeight)
	v.updateViewportContent()
	return nil
}

func (v *LogView) StatusLine() string {
	status := "Space:pause/resume p:older g/G:top/bottom c:clear Esc:back"
	if v.paused {
		return "⏸ PAUSED • " + status
	}
	if v.pollInterval > defaultLogPollInterval {
		return fmt.Sprintf("⏳ THROTTLED (%ds) • %s", int(v.pollInterval.Seconds()), status)
	}
	return "▶ STREAMING • " + status
}
