package logstreams

import (
	"time"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// LogStreamRenderer renders CloudWatch Log Streams
// Ensure LogStreamRenderer implements render.Navigator
var _ render.Navigator = (*LogStreamRenderer)(nil)

type LogStreamRenderer struct {
	render.BaseRenderer
}

// NewLogStreamRenderer creates a new LogStreamRenderer
func NewLogStreamRenderer() render.Renderer {
	return &LogStreamRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "cloudwatch",
			Resource: "log-streams",
			Cols: []render.Column{
				{Name: "STREAM NAME", Width: 50, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "LAST EVENT", Width: 20, Getter: getLastEvent},
				{Name: "AGE", Width: 10, Getter: getAge},
			},
		},
	}
}

func getLastEvent(r dao.Resource) string {
	if ls, ok := dao.UnwrapResource(r).(*LogStreamResource); ok {
		lastEvent := ls.LastEventTimestamp()
		if lastEvent > 0 {
			t := time.UnixMilli(lastEvent)
			return render.FormatAge(t) + " ago"
		}
	}
	return "-"
}

func getAge(r dao.Resource) string {
	if ls, ok := dao.UnwrapResource(r).(*LogStreamResource); ok {
		creationTime := ls.CreationTime()
		if creationTime > 0 {
			t := time.UnixMilli(creationTime)
			return render.FormatAge(t)
		}
	}
	return "-"
}

// RenderDetail renders detailed log stream information
func (r *LogStreamRenderer) RenderDetail(resource dao.Resource) string {
	ls, ok := dao.UnwrapResource(resource).(*LogStreamResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("CloudWatch Log Stream", ls.LogStreamName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Stream Name", ls.LogStreamName())
	d.Field("Log Group", ls.LogGroupName())
	d.Field("ARN", ls.GetARN())

	// Timestamps
	d.Section("Timestamps")
	if firstEvent := ls.FirstEventTimestamp(); firstEvent > 0 {
		t := time.UnixMilli(firstEvent)
		d.Field("First Event", t.Format("2006-01-02 15:04:05"))
	}
	if lastEvent := ls.LastEventTimestamp(); lastEvent > 0 {
		t := time.UnixMilli(lastEvent)
		d.Field("Last Event", t.Format("2006-01-02 15:04:05"))
		d.Field("Time Since Last Event", time.Since(t).Truncate(time.Second).String())
	}
	if lastIngestion := ls.LastIngestionTime(); lastIngestion > 0 {
		t := time.UnixMilli(lastIngestion)
		d.Field("Last Ingestion", t.Format("2006-01-02 15:04:05"))
	}
	if creationTime := ls.CreationTime(); creationTime > 0 {
		t := time.UnixMilli(creationTime)
		d.Field("Created", t.Format("2006-01-02 15:04:05"))
		d.Field("Age", time.Since(t).Truncate(time.Second).String())
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *LogStreamRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	ls, ok := dao.UnwrapResource(resource).(*LogStreamResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Stream Name", Value: ls.LogStreamName()},
		{Label: "Log Group", Value: ls.LogGroupName()},
	}

	if lastEvent := ls.LastEventTimestamp(); lastEvent > 0 {
		t := time.UnixMilli(lastEvent)
		fields = append(fields, render.SummaryField{Label: "Last Event", Value: t.Format("2006-01-02 15:04:05")})
	}

	if creationTime := ls.CreationTime(); creationTime > 0 {
		t := time.UnixMilli(creationTime)
		fields = append(fields, render.SummaryField{Label: "Created", Value: t.Format("2006-01-02 15:04:05")})
	}

	return fields
}

func (r *LogStreamRenderer) Navigations(resource dao.Resource) []render.Navigation {
	ls, ok := dao.UnwrapResource(resource).(*LogStreamResource)
	if !ok {
		return nil
	}

	return []render.Navigation{
		{
			Key:      "t",
			Label:    "Tail",
			ViewType: render.ViewTypeLogView,
		},
		{
			Key:         "g",
			Label:       "Log Group",
			Service:     "cloudwatch",
			Resource:    "log-groups",
			FilterField: "LogGroupPrefix",
			FilterValue: ls.LogGroupName(),
		},
	}
}
