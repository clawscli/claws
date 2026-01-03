package view

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"

	appaws "github.com/clawscli/claws/internal/aws"
	apperrors "github.com/clawscli/claws/internal/errors"
	"github.com/clawscli/claws/internal/ui"
)

const defaultLogPollInterval = 3 * time.Second

type LogView struct {
	ctx           context.Context
	client        *cloudwatchlogs.Client
	logGroupName  string
	logStreamName string

	viewport viewport.Model
	spinner  spinner.Model
	styles   logViewStyles

	logs          []logEntry
	loading       bool
	paused        bool
	err           error
	ready         bool
	width, height int

	lastEventTime int64
	pollInterval  time.Duration
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
	t := ui.Current()
	return logViewStyles{
		header:    lipgloss.NewStyle().Bold(true).Foreground(t.Primary).MarginBottom(1),
		timestamp: lipgloss.NewStyle().Foreground(t.Secondary),
		message:   lipgloss.NewStyle().Foreground(t.Text),
		paused:    lipgloss.NewStyle().Bold(true).Foreground(t.Warning),
		error:     lipgloss.NewStyle().Foreground(t.Danger),
		dim:       lipgloss.NewStyle().Foreground(t.TextDim),
	}
}

func NewLogView(ctx context.Context, logGroupName string) *LogView {
	s := spinner.New()
	s.Spinner = spinner.Dot

	return &LogView{
		ctx:          ctx,
		logGroupName: logGroupName,
		spinner:      s,
		styles:       newLogViewStyles(),
		logs:         make([]logEntry, 0, 500),
		loading:      true,
		pollInterval: defaultLogPollInterval,
	}
}

func NewLogViewWithStream(ctx context.Context, logGroupName, logStreamName string) *LogView {
	v := NewLogView(ctx, logGroupName)
	v.logStreamName = logStreamName
	return v
}

type logsLoadedMsg struct {
	entries       []logEntry
	lastEventTime int64
	err           error
}

type logTickMsg time.Time

func (v *LogView) Init() tea.Cmd {
	return tea.Batch(
		v.initClient,
		v.spinner.Tick,
	)
}

func (v *LogView) initClient() tea.Msg {
	cfg, err := appaws.NewConfig(v.ctx)
	if err != nil {
		return logsLoadedMsg{err: apperrors.Wrap(err, "init AWS config")}
	}
	v.client = cloudwatchlogs.NewFromConfig(cfg)
	return v.fetchLogs(v.lastEventTime)
}

// fetchLogsCmd captures lastEventTime to avoid data race in the command goroutine.
func (v *LogView) fetchLogsCmd() tea.Cmd {
	startTime := v.lastEventTime
	return func() tea.Msg {
		return v.fetchLogs(startTime)
	}
}

func (v *LogView) fetchLogs(startTime int64) tea.Msg {
	if v.client == nil {
		return logsLoadedMsg{err: errors.New("client not initialized")}
	}

	input := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName: appaws.StringPtr(v.logGroupName),
		Limit:        appaws.Int32Ptr(100),
	}

	if v.logStreamName != "" {
		input.LogStreamNames = []string{v.logStreamName}
	}

	if startTime > 0 {
		input.StartTime = appaws.Int64Ptr(startTime + 1)
	} else {
		input.StartTime = appaws.Int64Ptr(time.Now().Add(-1 * time.Hour).UnixMilli())
	}

	output, err := v.client.FilterLogEvents(v.ctx, input)
	if err != nil {
		return logsLoadedMsg{err: apperrors.Wrap(err, "filter log events")}
	}

	var maxEventTime int64
	entries := make([]logEntry, 0, len(output.Events))
	for _, event := range output.Events {
		ts := time.UnixMilli(appaws.Int64(event.Timestamp))
		msg := appaws.Str(event.Message)
		entries = append(entries, logEntry{
			timestamp: ts,
			message:   strings.TrimSuffix(msg, "\n"),
		})
		if eventTs := appaws.Int64(event.Timestamp); eventTs > maxEventTime {
			maxEventTime = eventTs
		}
	}

	return logsLoadedMsg{entries: entries, lastEventTime: maxEventTime}
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
			v.err = msg.err
			return v, nil
		}
		if msg.lastEventTime > v.lastEventTime {
			v.lastEventTime = msg.lastEventTime
		}
		if len(msg.entries) > 0 {
			v.logs = append(v.logs, msg.entries...)
			if len(v.logs) > 1000 {
				v.logs = v.logs[len(v.logs)-1000:]
			}
			if v.ready {
				v.updateViewportContent()
				v.viewport.GotoBottom()
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
			if v.ready {
				v.viewport.GotoTop()
			}
			return v, nil
		case "G":
			if v.ready {
				v.viewport.GotoBottom()
			}
			return v, nil
		case "c":
			v.logs = v.logs[:0]
			if v.ready {
				v.updateViewportContent()
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

	if v.ready {
		var cmd tea.Cmd
		v.viewport, cmd = v.viewport.Update(msg)
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
	v.viewport.SetContent(sb.String())
}

func (v *LogView) ViewString() string {
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

	if !v.ready {
		return sb.String()
	}

	sb.WriteString(v.viewport.View())
	return sb.String()
}

func (v *LogView) View() tea.View {
	return tea.NewView(v.ViewString())
}

func (v *LogView) SetSize(width, height int) tea.Cmd {
	v.width = width
	v.height = height
	viewportHeight := height - 4
	v.viewport = viewport.New(viewport.WithWidth(width), viewport.WithHeight(viewportHeight))
	v.ready = true
	v.updateViewportContent()
	return nil
}

func (v *LogView) StatusLine() string {
	status := "Space:pause/resume g/G:top/bottom c:clear Esc:back"
	if v.paused {
		return "⏸ PAUSED • " + status
	}
	return "▶ STREAMING • " + status
}

func (v *LogView) HasActiveInput() bool {
	return false
}
