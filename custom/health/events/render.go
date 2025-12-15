package events

import (
	"time"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// EventRenderer renders AWS Health events.
type EventRenderer struct {
	render.BaseRenderer
}

// NewEventRenderer creates a new EventRenderer.
func NewEventRenderer() render.Renderer {
	return &EventRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "health",
			Resource: "events",
			Cols: []render.Column{
				{Name: "SERVICE", Width: 20, Getter: getService},
				{Name: "EVENT TYPE", Width: 40, Getter: getEventType},
				{Name: "CATEGORY", Width: 15, Getter: getCategory},
				{Name: "STATUS", Width: 10, Getter: getStatus},
				{Name: "REGION", Width: 15, Getter: getRegion},
				{Name: "STARTED", Width: 20, Getter: getStarted},
			},
		},
	}
}

func getService(r dao.Resource) string {
	event, ok := r.(*EventResource)
	if !ok {
		return ""
	}
	return event.Service()
}

func getEventType(r dao.Resource) string {
	event, ok := r.(*EventResource)
	if !ok {
		return ""
	}
	return event.EventTypeCode()
}

func getCategory(r dao.Resource) string {
	event, ok := r.(*EventResource)
	if !ok {
		return ""
	}
	return event.EventTypeCategory()
}

func getStatus(r dao.Resource) string {
	event, ok := r.(*EventResource)
	if !ok {
		return ""
	}
	return event.StatusCode()
}

func getRegion(r dao.Resource) string {
	event, ok := r.(*EventResource)
	if !ok {
		return ""
	}
	return event.Region()
}

func getStarted(r dao.Resource) string {
	event, ok := r.(*EventResource)
	if !ok {
		return ""
	}
	if t := event.StartTime(); t != nil {
		return t.Format("2006-01-02 15:04")
	}
	return ""
}

// RenderDetail renders the detail view for a Health event.
func (r *EventRenderer) RenderDetail(resource dao.Resource) string {
	event, ok := resource.(*EventResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("AWS Health Event", event.EventTypeCode())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Event ARN", event.GetARN())
	d.Field("Service", event.Service())
	d.Field("Event Type", event.EventTypeCode())
	d.Field("Category", event.EventTypeCategory())
	d.Field("Status", event.StatusCode())

	// Location
	d.Section("Location")
	if region := event.Region(); region != "" {
		d.Field("Region", region)
	}
	if az := event.AvailabilityZone(); az != "" {
		d.Field("Availability Zone", az)
	}

	// Scope
	if scope := event.EventScopeCode(); scope != "" {
		d.Section("Scope")
		d.Field("Event Scope", scope)
	}

	// Timing
	d.Section("Timing")
	if t := event.StartTime(); t != nil {
		d.Field("Start Time", t.Format("2006-01-02 15:04:05"))
		d.Field("Duration", time.Since(*t).Truncate(time.Second).String())
	}
	if t := event.EndTime(); t != nil {
		d.Field("End Time", t.Format("2006-01-02 15:04:05"))
	}
	if t := event.LastUpdatedTime(); t != nil {
		d.Field("Last Updated", t.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary renders summary fields for a Health event.
func (r *EventRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	event, ok := resource.(*EventResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Event ARN", Value: event.GetARN()},
		{Label: "Service", Value: event.Service()},
		{Label: "Event Type", Value: event.EventTypeCode()},
		{Label: "Category", Value: event.EventTypeCategory()},
		{Label: "Status", Value: event.StatusCode()},
	}

	if region := event.Region(); region != "" {
		fields = append(fields, render.SummaryField{Label: "Region", Value: region})
	}

	if t := event.StartTime(); t != nil {
		fields = append(fields, render.SummaryField{Label: "Started", Value: t.Format("2006-01-02 15:04:05")})
	}

	return fields
}
