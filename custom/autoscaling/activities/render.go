package activities

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// ActivityRenderer renders Auto Scaling activities
// Ensure ActivityRenderer implements render.Navigator
var _ render.Navigator = (*ActivityRenderer)(nil)

type ActivityRenderer struct {
	render.BaseRenderer
}

// NewActivityRenderer creates a new ActivityRenderer
func NewActivityRenderer() *ActivityRenderer {
	return &ActivityRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "autoscaling",
			Resource: "activities",
			Cols: []render.Column{
				{Name: "STATUS", Width: 15, Getter: getStatus},
				{Name: "DESCRIPTION", Width: 40, Getter: getDescription},
				{Name: "PROGRESS", Width: 10, Getter: getProgress},
				{Name: "STARTED", Width: 20, Getter: getStarted},
				{Name: "DURATION", Width: 10, Getter: getDuration},
			},
		},
	}
}

func getStatus(r dao.Resource) string {
	if a, ok := r.(*ActivityResource); ok {
		return a.StatusCode()
	}
	return ""
}

func getDescription(r dao.Resource) string {
	if a, ok := r.(*ActivityResource); ok {
		desc := a.Description()
		if len(desc) > 40 {
			return desc[:37] + "..."
		}
		return desc
	}
	return ""
}

func getProgress(r dao.Resource) string {
	if a, ok := r.(*ActivityResource); ok {
		return fmt.Sprintf("%d%%", a.Progress())
	}
	return ""
}

func getStarted(r dao.Resource) string {
	if a, ok := r.(*ActivityResource); ok {
		return a.StartTime()
	}
	return "-"
}

func getDuration(r dao.Resource) string {
	if a, ok := r.(*ActivityResource); ok {
		return a.Duration()
	}
	return "-"
}

// RenderDetail renders detailed activity information
func (r *ActivityRenderer) RenderDetail(resource dao.Resource) string {
	activity, ok := resource.(*ActivityResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Scaling Activity", activity.StatusCode())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Activity ID", activity.ActivityId())
	d.Field("Auto Scaling Group", activity.ASGName())
	d.Field("Status", activity.StatusCode())
	d.Field("Progress", fmt.Sprintf("%d%%", activity.Progress()))

	// Description
	d.Section("Description")
	if desc := activity.Description(); desc != "" {
		d.Field("Description", desc)
	}

	// Cause
	if cause := activity.Cause(); cause != "" {
		d.Section("Cause")
		d.Field("Cause", cause)
	}

	// Status Message
	if msg := activity.StatusMessage(); msg != "" {
		d.Section("Status Message")
		d.Field("Message", msg)
	}

	// Details
	if details := activity.Details(); details != "" {
		d.Section("Details")
		d.Field("Details", details)
	}

	// Timestamps
	d.Section("Timestamps")
	if started := activity.StartTime(); started != "" {
		d.Field("Started", started)
	}
	if ended := activity.EndTime(); ended != "" {
		d.Field("Ended", ended)
	}
	if dur := activity.Duration(); dur != "" {
		d.Field("Duration", dur)
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *ActivityRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	activity, ok := resource.(*ActivityResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Activity ID", Value: activity.ActivityId()},
		{Label: "ASG", Value: activity.ASGName()},
		{Label: "Status", Value: activity.StatusCode()},
		{Label: "Progress", Value: fmt.Sprintf("%d%%", activity.Progress())},
	}

	if started := activity.StartTime(); started != "" {
		fields = append(fields, render.SummaryField{Label: "Started", Value: started})
	}

	if dur := activity.Duration(); dur != "" {
		fields = append(fields, render.SummaryField{Label: "Duration", Value: dur})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *ActivityRenderer) Navigations(resource dao.Resource) []render.Navigation {
	return nil
}
