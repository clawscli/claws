package executions

import (
	"time"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// Ensure ExecutionRenderer implements render.Navigator
var _ render.Navigator = (*ExecutionRenderer)(nil)

// ExecutionRenderer renders Step Functions executions with custom columns
type ExecutionRenderer struct {
	render.BaseRenderer
}

// NewExecutionRenderer creates a new ExecutionRenderer
func NewExecutionRenderer() render.Renderer {
	return &ExecutionRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "stepfunctions",
			Resource: "executions",
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
					Name:  "STATE MACHINE",
					Width: 30,
					Getter: func(r dao.Resource) string {
						if er, ok := r.(*ExecutionResource); ok {
							return er.StateMachineName()
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "STATUS",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if er, ok := r.(*ExecutionResource); ok {
							return er.Status()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "STARTED",
					Width: 20,
					Getter: func(r dao.Resource) string {
						if er, ok := r.(*ExecutionResource); ok {
							if er.Item.StartDate != nil {
								return er.Item.StartDate.Format("2006-01-02 15:04:05")
							}
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "DURATION",
					Width: 10,
					Getter: func(r dao.Resource) string {
						if er, ok := r.(*ExecutionResource); ok {
							if er.Item.StartDate != nil {
								end := time.Now()
								if er.Item.StopDate != nil {
									end = *er.Item.StopDate
								}
								return render.FormatDuration(end.Sub(*er.Item.StartDate))
							}
						}
						return ""
					},
					Priority: 4,
				},
			},
		},
	}
}

// RenderDetail renders detailed execution information
func (r *ExecutionRenderer) RenderDetail(resource dao.Resource) string {
	er, ok := resource.(*ExecutionResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Step Functions Execution", er.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", er.GetName())
	d.Field("ARN", er.ARN())
	d.FieldStyled("Status", er.Status(), render.StateColorer()(er.Status()))

	// State Machine
	d.Section("State Machine")
	d.Field("Name", er.StateMachineName())
	d.Field("ARN", er.StateMachineARN())

	// Timing
	d.Section("Timing")
	if er.Item.StartDate != nil {
		d.Field("Started", er.Item.StartDate.Format(time.RFC3339))
	}
	if er.Item.StopDate != nil {
		d.Field("Stopped", er.Item.StopDate.Format(time.RFC3339))
		if er.Item.StartDate != nil {
			duration := er.Item.StopDate.Sub(*er.Item.StartDate)
			d.Field("Duration", render.FormatDuration(duration))
		}
	} else if er.Item.StartDate != nil {
		d.Field("Running For", render.FormatAge(*er.Item.StartDate))
	}

	// Input/Output
	if er.Input() != "" {
		d.Section("Input")
		input := er.Input()
		if len(input) > 300 {
			input = input[:300] + "..."
		}
		d.Line(input)
	}

	if er.Output() != "" {
		d.Section("Output")
		output := er.Output()
		if len(output) > 300 {
			output = output[:300] + "..."
		}
		d.Line(output)
	}

	// Error
	if er.Error() != "" {
		d.Section("Error")
		d.Field("Error", er.Error())
		if er.Cause() != "" {
			cause := er.Cause()
			if len(cause) > 300 {
				cause = cause[:300] + "..."
			}
			d.Field("Cause", cause)
		}
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *ExecutionRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	er, ok := resource.(*ExecutionResource)
	if !ok {
		return nil
	}

	stateStyle := render.StateColorer()(er.Status())

	fields := []render.SummaryField{
		{Label: "Name", Value: er.GetName()},
		{Label: "Status", Value: er.Status(), Style: stateStyle},
		{Label: "State Machine", Value: er.StateMachineName()},
	}

	if er.Item.StartDate != nil {
		fields = append(fields, render.SummaryField{
			Label: "Started",
			Value: er.Item.StartDate.Format("2006-01-02 15:04:05"),
		})
	}

	if er.Item.StopDate != nil && er.Item.StartDate != nil {
		duration := er.Item.StopDate.Sub(*er.Item.StartDate)
		fields = append(fields, render.SummaryField{Label: "Duration", Value: render.FormatDuration(duration)})
	}

	return fields
}

// Navigations returns navigation shortcuts for executions
func (r *ExecutionRenderer) Navigations(resource dao.Resource) []render.Navigation {
	er, ok := resource.(*ExecutionResource)
	if !ok {
		return nil
	}

	var navs []render.Navigation

	// State Machine navigation
	navs = append(navs, render.Navigation{
		Key: "m", Label: "State Machine", Service: "stepfunctions", Resource: "state-machines",
		FilterField: "StateMachineArn", FilterValue: er.StateMachineARN(),
	})

	return navs
}
