package statemachines

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// Ensure StateMachineRenderer implements render.Navigator
var _ render.Navigator = (*StateMachineRenderer)(nil)

// StateMachineRenderer renders Step Functions state machines with custom columns
type StateMachineRenderer struct {
	render.BaseRenderer
}

// NewStateMachineRenderer creates a new StateMachineRenderer
func NewStateMachineRenderer() render.Renderer {
	return &StateMachineRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "sfn",
			Resource: "state-machines",
			Cols: []render.Column{
				{
					Name:  "NAME",
					Width: 40,
					Getter: func(r dao.Resource) string {
						return r.GetName()
					},
					Priority: 0,
				},
				{
					Name:  "TYPE",
					Width: 10,
					Getter: func(r dao.Resource) string {
						if sr, ok := r.(*StateMachineResource); ok {
							return sr.Type()
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "STATUS",
					Width: 10,
					Getter: func(r dao.Resource) string {
						if sr, ok := r.(*StateMachineResource); ok {
							return sr.Status()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "AGE",
					Width: 8,
					Getter: func(r dao.Resource) string {
						if sr, ok := r.(*StateMachineResource); ok {
							if sr.Item.CreationDate != nil {
								return render.FormatAge(*sr.Item.CreationDate)
							}
						}
						return ""
					},
					Priority: 3,
				},
			},
		},
	}
}

// RenderDetail renders detailed state machine information
func (r *StateMachineRenderer) RenderDetail(resource dao.Resource) string {
	sr, ok := resource.(*StateMachineResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Step Functions State Machine", sr.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", sr.GetName())
	d.Field("ARN", sr.ARN())
	d.FieldStyled("Status", sr.Status(), render.StateColorer()(sr.Status()))
	d.Field("Type", sr.Type())
	if sr.Item.CreationDate != nil {
		d.Field("Created", sr.Item.CreationDate.Format(time.RFC3339))
		d.Field("Age", render.FormatAge(*sr.Item.CreationDate))
	}

	// Description
	if sr.Detail != nil && sr.Detail.Description != nil && *sr.Detail.Description != "" {
		d.Field("Description", *sr.Detail.Description)
	}

	// Revision
	if sr.Detail != nil && sr.Detail.RevisionId != nil {
		d.Field("Revision ID", *sr.Detail.RevisionId)
	}

	// IAM Role
	if sr.RoleARN() != "" {
		d.Section("IAM Role")
		d.Field("Role Name", sr.RoleName())
		d.Field("Role ARN", sr.RoleARN())
	}

	// Logging Configuration
	if sr.Detail != nil && sr.Detail.LoggingConfiguration != nil {
		lc := sr.Detail.LoggingConfiguration
		d.Section("Logging")
		d.Field("Level", string(lc.Level))
		d.Field("Include Execution Data", formatBool(lc.IncludeExecutionData))
		if len(lc.Destinations) > 0 {
			for _, dest := range lc.Destinations {
				if dest.CloudWatchLogsLogGroup != nil && dest.CloudWatchLogsLogGroup.LogGroupArn != nil {
					d.Field("Log Group", *dest.CloudWatchLogsLogGroup.LogGroupArn)
				}
			}
		}
	}

	// Tracing Configuration
	if sr.Detail != nil && sr.Detail.TracingConfiguration != nil {
		tc := sr.Detail.TracingConfiguration
		d.Section("Tracing (X-Ray)")
		if tc.Enabled {
			d.FieldStyled("X-Ray Tracing", "Enabled", render.SuccessStyle())
		} else {
			d.Field("X-Ray Tracing", "Disabled")
		}
	}

	// Definition (pretty printed)
	if sr.Definition() != "" {
		d.Section("Definition")
		d.Line(prettyJSON(sr.Definition()))
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *StateMachineRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	sr, ok := resource.(*StateMachineResource)
	if !ok {
		return nil
	}

	stateStyle := render.StateColorer()(sr.Status())

	fields := []render.SummaryField{
		{Label: "Name", Value: sr.GetName()},
		{Label: "ARN", Value: sr.ARN()},
		{Label: "Status", Value: sr.Status(), Style: stateStyle},
		{Label: "Type", Value: sr.Type()},
	}

	if sr.RoleName() != "" {
		fields = append(fields, render.SummaryField{Label: "Role", Value: sr.RoleName()})
	}

	if sr.Item.CreationDate != nil {
		fields = append(fields, render.SummaryField{
			Label: "Created",
			Value: sr.Item.CreationDate.Format("2006-01-02 15:04") + " (" + render.FormatAge(*sr.Item.CreationDate) + ")",
		})
	}

	return fields
}

// Navigations returns navigation shortcuts for state machines
func (r *StateMachineRenderer) Navigations(resource dao.Resource) []render.Navigation {
	sr, ok := resource.(*StateMachineResource)
	if !ok {
		return nil
	}

	var navs []render.Navigation

	// Executions navigation
	navs = append(navs, render.Navigation{
		Key: "e", Label: "Executions", Service: "sfn", Resource: "executions",
		FilterField: "StateMachineName", FilterValue: sr.GetName(),
	})

	// IAM Role navigation
	if sr.RoleName() != "" {
		navs = append(navs, render.Navigation{
			Key: "r", Label: "Role", Service: "iam", Resource: "roles",
			FilterField: "RoleName", FilterValue: sr.RoleName(),
		})
	}

	return navs
}

func formatBool(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

// prettyJSON formats JSON string with indentation
func prettyJSON(s string) string {
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(s), "", "  "); err != nil {
		return s // return original if not valid JSON
	}
	return buf.String()
}
