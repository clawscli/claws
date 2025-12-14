package jobs

import (
	"fmt"
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// JobRenderer renders Glue jobs.
// Ensure JobRenderer implements render.Navigator
var _ render.Navigator = (*JobRenderer)(nil)

type JobRenderer struct {
	render.BaseRenderer
}

// NewJobRenderer creates a new JobRenderer.
func NewJobRenderer() render.Renderer {
	return &JobRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "glue",
			Resource: "jobs",
			Cols: []render.Column{
				{Name: "JOB NAME", Width: 40, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "GLUE VER", Width: 10, Getter: getGlueVersion},
				{Name: "WORKER TYPE", Width: 12, Getter: getWorkerType},
				{Name: "WORKERS", Width: 9, Getter: getWorkers},
				{Name: "TIMEOUT", Width: 9, Getter: getTimeout},
				{Name: "MODIFIED", Width: 20, Getter: getModified},
			},
		},
	}
}

func getGlueVersion(r dao.Resource) string {
	job, ok := r.(*JobResource)
	if !ok {
		return ""
	}
	return job.GlueVersion()
}

func getWorkerType(r dao.Resource) string {
	job, ok := r.(*JobResource)
	if !ok {
		return ""
	}
	return job.WorkerType()
}

func getWorkers(r dao.Resource) string {
	job, ok := r.(*JobResource)
	if !ok {
		return ""
	}
	workers := job.NumberOfWorkers()
	if workers > 0 {
		return fmt.Sprintf("%d", workers)
	}
	return ""
}

func getTimeout(r dao.Resource) string {
	job, ok := r.(*JobResource)
	if !ok {
		return ""
	}
	timeout := job.Timeout()
	if timeout > 0 {
		return fmt.Sprintf("%dm", timeout)
	}
	return ""
}

func getModified(r dao.Resource) string {
	job, ok := r.(*JobResource)
	if !ok {
		return ""
	}
	if t := job.LastModifiedOn(); t != nil {
		return t.Format("2006-01-02 15:04")
	}
	return ""
}

// RenderDetail renders the detail view for a Glue job.
func (r *JobRenderer) RenderDetail(resource dao.Resource) string {
	job, ok := resource.(*JobResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Glue Job", job.Name())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Job Name", job.Name())
	if desc := job.Description(); desc != "" {
		d.Field("Description", desc)
	}
	d.Field("IAM Role", job.Role())

	// Runtime Configuration
	d.Section("Runtime Configuration")
	d.Field("Glue Version", job.GlueVersion())
	d.Field("Worker Type", job.WorkerType())
	if workers := job.NumberOfWorkers(); workers > 0 {
		d.Field("Number of Workers", fmt.Sprintf("%d", workers))
	}
	if execClass := job.ExecutionClass(); execClass != "" {
		d.Field("Execution Class", execClass)
	}

	// Execution Settings
	d.Section("Execution Settings")
	if timeout := job.Timeout(); timeout > 0 {
		d.Field("Timeout", fmt.Sprintf("%d minutes", timeout))
	}
	if retries := job.MaxRetries(); retries > 0 {
		d.Field("Max Retries", fmt.Sprintf("%d", retries))
	}

	// Command
	if cmd := job.Command(); cmd != nil {
		d.Section("Job Command")
		if cmd.Name != nil {
			d.Field("Command Name", *cmd.Name)
		}
		if cmd.ScriptLocation != nil {
			d.Field("Script Location", *cmd.ScriptLocation)
		}
		if cmd.PythonVersion != nil {
			d.Field("Python Version", *cmd.PythonVersion)
		}
	}

	// Job Mode
	if mode := job.JobMode(); mode != "" {
		d.Field("Job Mode", mode)
	}

	// Connections
	if conns := job.Connections(); len(conns) > 0 {
		d.Section("Connections")
		d.Field("Connections", strings.Join(conns, ", "))
	}

	// Security
	if secConfig := job.SecurityConfiguration(); secConfig != "" {
		d.Section("Security")
		d.Field("Security Configuration", secConfig)
	}

	// Default Arguments
	if args := job.DefaultArguments(); len(args) > 0 {
		d.Section("Default Arguments")
		for k, v := range args {
			d.Field(k, v)
		}
	}

	// Timestamps
	d.Section("Timestamps")
	if t := job.CreatedOn(); t != nil {
		d.Field("Created", t.Format("2006-01-02 15:04:05"))
	}
	if t := job.LastModifiedOn(); t != nil {
		d.Field("Last Modified", t.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary renders summary fields for a Glue job.
func (r *JobRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	job, ok := resource.(*JobResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Job Name", Value: job.Name()},
		{Label: "Glue Version", Value: job.GlueVersion()},
		{Label: "Worker Type", Value: job.WorkerType()},
	}

	if workers := job.NumberOfWorkers(); workers > 0 {
		fields = append(fields, render.SummaryField{Label: "Workers", Value: fmt.Sprintf("%d", workers)})
	}

	if t := job.LastModifiedOn(); t != nil {
		fields = append(fields, render.SummaryField{Label: "Last Modified", Value: t.Format("2006-01-02 15:04:05")})
	}

	return fields
}

// Navigations returns available navigations from a Glue job.
func (r *JobRenderer) Navigations(resource dao.Resource) []render.Navigation {
	job, ok := resource.(*JobResource)
	if !ok {
		return nil
	}
	return []render.Navigation{
		{
			Key:         "r",
			Label:       "Runs",
			Service:     "glue",
			Resource:    "job-runs",
			FilterField: "JobName",
			FilterValue: job.Name(),
		},
	}
}
