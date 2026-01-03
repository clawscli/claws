package events

import (
	"strings"
	"time"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
	"github.com/clawscli/claws/internal/ui"
)

// EventRenderer renders CloudFormation stack events
type EventRenderer struct {
	render.BaseRenderer
}

// NewEventRenderer creates a new EventRenderer
func NewEventRenderer() render.Renderer {
	return &EventRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "cloudformation",
			Resource: "events",
			Cols: []render.Column{
				{
					Name:  "TIMESTAMP",
					Width: 20,
					Getter: func(r dao.Resource) string {
						if er, ok := r.(*EventResource); ok {
							if er.Item.Timestamp != nil {
								return er.Item.Timestamp.Format("01-02 15:04:05")
							}
						}
						return ""
					},
					Priority: 0,
				},
				{
					Name:  "LOGICAL ID",
					Width: 30,
					Getter: func(r dao.Resource) string {
						return r.GetName()
					},
					Priority: 1,
				},
				{
					Name:  "STATUS",
					Width: 24,
					Getter: func(r dao.Resource) string {
						if er, ok := r.(*EventResource); ok {
							return er.ResourceStatus()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "TYPE",
					Width: 30,
					Getter: func(r dao.Resource) string {
						if er, ok := r.(*EventResource); ok {
							return er.ResourceType()
						}
						return ""
					},
					Priority: 3,
				},
			},
		},
	}
}

// RenderDetail renders detailed event information
func (r *EventRenderer) RenderDetail(resource dao.Resource) string {
	er, ok := resource.(*EventResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Stack Event", er.GetName())

	d.Section("Event Information")
	d.Field("Event ID", er.GetID())
	d.Field("Logical Resource ID", er.GetName())
	d.FieldStyled("Status", er.ResourceStatus(), cfnResourceStatusColorer(er.ResourceStatus()))
	d.Field("Resource Type", er.ResourceType())

	if er.Item.Timestamp != nil {
		d.Field("Timestamp", er.Item.Timestamp.Format(time.RFC3339))
	}

	d.FieldIf("Physical Resource ID", er.Item.PhysicalResourceId)

	if er.StatusReason() != "" {
		d.Section("Status Reason")
		d.Line("  " + er.StatusReason())
	}

	d.FieldIf("Stack Name", er.Item.StackName)
	d.FieldIf("Client Request Token", er.Item.ClientRequestToken)

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *EventRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	er, ok := resource.(*EventResource)
	if !ok {
		return nil
	}

	fields := []render.SummaryField{
		{Label: "Logical ID", Value: er.GetName()},
		{Label: "Status", Value: er.ResourceStatus(), Style: cfnResourceStatusColorer(er.ResourceStatus())},
		{Label: "Type", Value: er.ResourceType()},
	}

	if er.Item.Timestamp != nil {
		fields = append(fields, render.SummaryField{
			Label: "Time",
			Value: er.Item.Timestamp.Format("2006-01-02 15:04:05"),
		})
	}

	if reason := er.StatusReason(); reason != "" {
		if len(reason) > 80 {
			reason = reason[:77] + "..."
		}
		fields = append(fields, render.SummaryField{Label: "Reason", Value: reason})
	}

	if er.Item.PhysicalResourceId != nil && *er.Item.PhysicalResourceId != "" {
		fields = append(fields, render.SummaryField{Label: "Physical ID", Value: *er.Item.PhysicalResourceId})
	}

	return fields
}

// cfnResourceStatusColorer returns a style for CloudFormation resource status
func cfnResourceStatusColorer(status string) render.Style {
	switch {
	case strings.HasSuffix(status, "_COMPLETE") && !strings.Contains(status, "ROLLBACK") && !strings.Contains(status, "DELETE"):
		return ui.SuccessStyle()
	case strings.Contains(status, "IN_PROGRESS"):
		return ui.WarningStyle()
	case strings.Contains(status, "FAILED") || strings.Contains(status, "ROLLBACK"):
		return ui.DangerStyle()
	case strings.Contains(status, "DELETE_COMPLETE") || strings.Contains(status, "SKIPPED"):
		return ui.DimStyle()
	default:
		return render.DefaultStyle()
	}
}
