package jobs

import (
	"fmt"
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// JobRenderer renders Batch jobs.
type JobRenderer struct {
	render.BaseRenderer
}

// NewJobRenderer creates a new JobRenderer.
func NewJobRenderer() render.Renderer {
	return &JobRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "batch",
			Resource: "jobs",
			Cols: []render.Column{
				{Name: "JOB ID", Width: 35, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "NAME", Width: 30, Getter: getName},
				{Name: "STATUS", Width: 12, Getter: getStatus},
				{Name: "CREATED", Width: 15, Getter: getCreated},
			},
		},
	}
}

func getName(r dao.Resource) string {
	job, ok := r.(*JobResource)
	if !ok {
		return ""
	}
	return job.Name()
}

func getStatus(r dao.Resource) string {
	job, ok := r.(*JobResource)
	if !ok {
		return ""
	}
	return job.Status()
}

func getCreated(r dao.Resource) string {
	job, ok := r.(*JobResource)
	if !ok {
		return ""
	}
	if t := job.CreatedAt(); t != nil {
		return render.FormatAge(*t)
	}
	return ""
}

// RenderDetail renders the detail view for a job.
func (r *JobRenderer) RenderDetail(resource dao.Resource) string {
	job, ok := resource.(*JobResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Batch Job", job.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Job ID", job.GetID())
	d.Field("Name", job.Name())
	d.Field("ARN", job.GetARN())
	d.Field("Status", job.Status())

	if job.JobDefinition != "" {
		d.Field("Job Definition", job.JobDefinition)
	}
	if job.GetJobQueue() != "" {
		d.Field("Job Queue", job.GetJobQueue())
	}
	if job.StatusReason != "" {
		d.Field("Status Reason", job.StatusReason)
	}
	if caps := job.GetPlatformCapabilities(); len(caps) > 0 {
		d.Field("Platform", strings.Join(caps, ", "))
	}

	// Container Details
	if container := job.GetContainer(); container != nil {
		d.Section("Container")
		if container.Image != nil {
			d.Field("Image", *container.Image)
		}
		if container.ExitCode != nil {
			d.Field("Exit Code", fmt.Sprintf("%d", *container.ExitCode))
		}
		if container.Reason != nil {
			d.Field("Reason", *container.Reason)
		}
		if container.LogStreamName != nil {
			d.Field("Log Stream", *container.LogStreamName)
		}
		if container.Vcpus != nil {
			d.Field("vCPUs", fmt.Sprintf("%d", *container.Vcpus))
		}
		if container.Memory != nil {
			d.Field("Memory (MiB)", fmt.Sprintf("%d", *container.Memory))
		}
		if len(container.Command) > 0 {
			d.Field("Command", strings.Join(container.Command, " "))
		}
	}

	// Retry Strategy
	if retry := job.GetRetryStrategy(); retry != nil && retry.Attempts != nil && *retry.Attempts > 0 {
		d.Section("Retry Strategy")
		d.Field("Max Attempts", fmt.Sprintf("%d", *retry.Attempts))
	}

	// Timeout
	if timeout := job.GetTimeout(); timeout != nil && timeout.AttemptDurationSeconds != nil {
		d.Section("Timeout")
		d.Field("Attempt Duration", fmt.Sprintf("%ds", *timeout.AttemptDurationSeconds))
	}

	// Dependencies
	if deps := job.GetDependsOn(); len(deps) > 0 {
		d.Section("Dependencies")
		for _, dep := range deps {
			if dep.JobId != nil {
				d.Field("Depends On", *dep.JobId)
			}
		}
	}

	// Parameters
	if params := job.GetParameters(); len(params) > 0 {
		d.Section("Parameters")
		for k, v := range params {
			d.Field(k, v)
		}
	}

	// Attempts
	if attempts := job.GetAttempts(); len(attempts) > 0 {
		d.Section("Attempts")
		for i, attempt := range attempts {
			label := fmt.Sprintf("Attempt %d", i+1)
			if attempt.Container != nil && attempt.Container.ExitCode != nil {
				d.Field(label+" Exit Code", fmt.Sprintf("%d", *attempt.Container.ExitCode))
			}
			if attempt.Container != nil && attempt.Container.Reason != nil {
				d.Field(label+" Reason", *attempt.Container.Reason)
			}
			if attempt.StartedAt != nil {
				d.Field(label+" Started", fmt.Sprintf("%d", *attempt.StartedAt))
			}
			if attempt.StoppedAt != nil {
				d.Field(label+" Stopped", fmt.Sprintf("%d", *attempt.StoppedAt))
			}
		}
	}

	// Timestamps
	d.Section("Timestamps")
	if t := job.CreatedAt(); t != nil {
		d.Field("Created", t.Format("2006-01-02 15:04:05"))
	}
	if t := job.StartedAt(); t != nil {
		d.Field("Started", t.Format("2006-01-02 15:04:05"))
	}
	if t := job.StoppedAt(); t != nil {
		d.Field("Stopped", t.Format("2006-01-02 15:04:05"))
	}

	// Tags
	if tags := job.GetTags(); len(tags) > 0 {
		d.Section("Tags")
		for k, v := range tags {
			d.Field(k, v)
		}
	}

	return d.String()
}

// RenderSummary renders summary fields for a job.
func (r *JobRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	job, ok := resource.(*JobResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Job ID", Value: job.GetID()},
		{Label: "Name", Value: job.Name()},
		{Label: "Status", Value: job.Status()},
	}
}
