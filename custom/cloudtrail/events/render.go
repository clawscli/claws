package events

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// EventRenderer renders CloudTrail events.
type EventRenderer struct {
	render.BaseRenderer
}

// NewEventRenderer creates a new EventRenderer.
func NewEventRenderer() render.Renderer {
	return &EventRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "cloudtrail",
			Resource: "events",
			Cols: []render.Column{
				{Name: "EVENT TIME", Width: 20, Getter: getEventTime},
				{Name: "EVENT NAME", Width: 35, Getter: getEventName},
				{Name: "EVENT SOURCE", Width: 30, Getter: getEventSource},
				{Name: "USERNAME", Width: 25, Getter: getUsername},
				{Name: "READ ONLY", Width: 10, Getter: getReadOnly},
			},
		},
	}
}

func getEventTime(r dao.Resource) string {
	event, ok := r.(*EventResource)
	if !ok {
		return ""
	}
	if t := event.EventTime(); t != nil {
		return t.Format("2006-01-02 15:04:05")
	}
	return ""
}

func getEventName(r dao.Resource) string {
	event, ok := r.(*EventResource)
	if !ok {
		return ""
	}
	return event.EventName()
}

func getEventSource(r dao.Resource) string {
	event, ok := r.(*EventResource)
	if !ok {
		return ""
	}
	return event.EventSource()
}

func getUsername(r dao.Resource) string {
	event, ok := r.(*EventResource)
	if !ok {
		return ""
	}
	return event.Username()
}

func getReadOnly(r dao.Resource) string {
	event, ok := r.(*EventResource)
	if !ok {
		return ""
	}
	return event.ReadOnly()
}

// RenderDetail renders the detail view for a CloudTrail event.
func (r *EventRenderer) RenderDetail(resource dao.Resource) string {
	event, ok := resource.(*EventResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("CloudTrail Event", event.EventName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Event ID", event.EventId())
	d.Field("Event Name", event.EventName())
	d.Field("Event Source", event.EventSource())
	if t := event.EventTime(); t != nil {
		d.Field("Event Time", t.Format("2006-01-02 15:04:05"))
	}

	// Identity
	d.Section("Identity")
	if username := event.Username(); username != "" {
		d.Field("Username", username)
	}
	if accessKey := event.AccessKeyId(); accessKey != "" {
		d.Field("Access Key ID", accessKey)
	}

	// Event Type
	d.Section("Event Type")
	if readOnly := event.ReadOnly(); readOnly != "" {
		d.Field("Read Only", readOnly)
	}

	// Affected Resources
	if resources := event.Resources(); len(resources) > 0 {
		d.Section("Affected Resources")
		for i, res := range resources {
			if i >= 10 {
				d.Field("", fmt.Sprintf("... and %d more", len(resources)-10))
				break
			}
			resourceType := ""
			resourceName := ""
			if res.ResourceType != nil {
				resourceType = *res.ResourceType
			}
			if res.ResourceName != nil {
				resourceName = *res.ResourceName
			}
			d.Field(resourceType, resourceName)
		}
	}

	// Raw Event (at bottom for readability)
	if rawEvent := event.CloudTrailEvent(); rawEvent != "" {
		d.Section("Raw Event")
		d.Line(prettyJSON(rawEvent))
	}

	return d.String()
}

// prettyJSON formats JSON string with indentation
func prettyJSON(s string) string {
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(s), "", "  "); err != nil {
		return s
	}
	return buf.String()
}

// RenderSummary renders summary fields for a CloudTrail event.
func (r *EventRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	event, ok := resource.(*EventResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Event ID", Value: event.EventId()},
		{Label: "Event Name", Value: event.EventName()},
		{Label: "Event Source", Value: event.EventSource()},
	}

	if username := event.Username(); username != "" {
		fields = append(fields, render.SummaryField{Label: "Username", Value: username})
	}

	if t := event.EventTime(); t != nil {
		fields = append(fields, render.SummaryField{Label: "Event Time", Value: t.Format("2006-01-02 15:04:05")})
	}

	return fields
}
