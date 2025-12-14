package jobruns

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// JobRunRenderer renders Glue job runs.
type JobRunRenderer struct {
	render.BaseRenderer
}

// NewJobRunRenderer creates a new JobRunRenderer.
func NewJobRunRenderer() render.Renderer {
	return &JobRunRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "glue",
			Resource: "job-runs",
			Cols: []render.Column{
				{Name: "RUN ID", Width: 38, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "STATE", Width: 12, Getter: getState},
				{Name: "STARTED", Width: 18, Getter: getStarted},
				{Name: "DURATION", Width: 12, Getter: getDuration},
				{Name: "WORKERS", Width: 10, Getter: getWorkers},
			},
		},
	}
}

func getState(r dao.Resource) string {
	run, ok := r.(*JobRunResource)
	if !ok {
		return ""
	}
	return run.JobRunState()
}

func getStarted(r dao.Resource) string {
	run, ok := r.(*JobRunResource)
	if !ok {
		return ""
	}
	if t := run.StartedOn(); t != nil {
		return t.Format("2006-01-02 15:04")
	}
	return ""
}

func getDuration(r dao.Resource) string {
	run, ok := r.(*JobRunResource)
	if !ok {
		return ""
	}
	secs := run.ExecutionTime()
	if secs > 0 {
		if secs >= 3600 {
			return fmt.Sprintf("%dh%dm", secs/3600, (secs%3600)/60)
		}
		if secs >= 60 {
			return fmt.Sprintf("%dm%ds", secs/60, secs%60)
		}
		return fmt.Sprintf("%ds", secs)
	}
	return ""
}

func getWorkers(r dao.Resource) string {
	run, ok := r.(*JobRunResource)
	if !ok {
		return ""
	}
	if n := run.NumberOfWorkers(); n > 0 {
		return fmt.Sprintf("%d", n)
	}
	return ""
}

// RenderDetail renders the detail view for a Glue job run.
func (r *JobRunRenderer) RenderDetail(resource dao.Resource) string {
	run, ok := resource.(*JobRunResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Glue Job Run", run.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Run ID", run.GetID())
	d.Field("Job Name", run.JobName())
	d.Field("State", run.JobRunState())
	d.Field("Attempt", fmt.Sprintf("%d", run.Attempt()))

	// Execution
	d.Section("Execution")
	if t := run.StartedOn(); t != nil {
		d.Field("Started", t.Format("2006-01-02 15:04:05"))
	}
	if t := run.CompletedOn(); t != nil {
		d.Field("Completed", t.Format("2006-01-02 15:04:05"))
	}
	if secs := run.ExecutionTime(); secs > 0 {
		d.Field("Execution Time", fmt.Sprintf("%d seconds", secs))
	}

	// Resources
	d.Section("Resources")
	if wt := run.WorkerType(); wt != "" {
		d.Field("Worker Type", wt)
	}
	if n := run.NumberOfWorkers(); n > 0 {
		d.Field("Number of Workers", fmt.Sprintf("%d", n))
	}
	if gv := run.GlueVersion(); gv != "" {
		d.Field("Glue Version", gv)
	}

	// Error
	if errMsg := run.ErrorMessage(); errMsg != "" {
		d.Section("Error")
		d.Field("Message", errMsg)
	}

	return d.String()
}

// RenderSummary renders summary fields for a Glue job run.
func (r *JobRunRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	run, ok := resource.(*JobRunResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Run ID", Value: run.GetID()},
		{Label: "State", Value: run.JobRunState()},
		{Label: "Job Name", Value: run.JobName()},
	}

	if secs := run.ExecutionTime(); secs > 0 {
		fields = append(fields, render.SummaryField{Label: "Duration", Value: fmt.Sprintf("%ds", secs)})
	}

	return fields
}
