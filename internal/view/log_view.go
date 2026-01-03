package view

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/ui"
)

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
		pollInterval: 3 * time.Second,
	}
}

func NewLogViewWithStream(ctx context.Context, logGroupName, logStreamName string) *LogView {
	v := NewLogView(ctx, logGroupName)
	v.logStreamName = logStreamName
	return v
}

type logsLoadedMsg struct {
	entries []logEntry
	err     error
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
		return logsLoadedMsg{err: fmt.Errorf("init AWS config: %w", err)}
	}
	v.client = cloudwatchlogs.NewFromConfig(cfg)
	return v.fetchLogs()
}

func (v *LogView) fetchLogs() tea.Msg {
	if v.client == nil {
		return logsLoadedMsg{err: fmt.Errorf("client not initialized")}
	}

	input := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName: aws.String(v.logGroupName),
		Limit:        aws.Int32(100),
	}

	if v.logStreamName != "" {
		input.LogStreamNames = []string{v.logStreamName}
	}

	if v.lastEventTime > 0 {
		input.StartTime = aws.Int64(v.lastEventTime + 1)
	} else {
		input.StartTime = aws.Int64(time.Now().Add(-1 * time.Hour).UnixMilli())
	}

	output, err := v.client.FilterLogEvents(v.ctx, input)
	if err != nil {
		return logsLoadedMsg{err: fmt.Errorf("filter log events: %w", err)}
	}

	entries := make([]logEntry, 0, len(output.Events))
	for _, event := range output.Events {
		ts := time.UnixMilli(aws.ToInt64(event.Timestamp))
		msg := aws.ToString(event.Message)
		entries = append(entries, logEntry{
			timestamp: ts,
			message:   strings.TrimSuffix(msg, "\n"),
		})
		if aws.ToInt64(event.Timestamp) > v.lastEventTime {
			v.lastEventTime = aws.ToInt64(event.Timestamp)
		}
	}

	return logsLoadedMsg{entries: entries}
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
		if len(msg.entries) > 0 {
			v.logs = append(v.logs, msg.entries...)
			if len(v.logs) > 1000 {
				v.logs = v.logs[len(v.logs)-1000:]
			}
			v.updateViewportContent()
			v.viewport.GotoBottom()
		}
		if !v.paused {
			return v, v.tickCmd()
		}
		return v, nil

	case logTickMsg:
		if v.paused {
			return v, nil
		}
		return v, func() tea.Msg { return v.fetchLogs() }

	case tea.KeyPressMsg:
		switch msg.String() {
		case "space":
			v.paused = !v.paused
			if !v.paused {
				return v, v.tickCmd()
			}
			return v, nil
		case "g":
			v.viewport.GotoTop()
			return v, nil
		case "G":
			v.viewport.GotoBottom()
			return v, nil
		case "c":
			v.logs = v.logs[:0]
			v.updateViewportContent()
			return v, nil
		}

	case spinner.TickMsg:
		if v.loading {
			var cmd tea.Cmd
			v.spinner, cmd = v.spinner.Update(msg)
			return v, cmd
		}
	}

	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	return v, cmd
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
