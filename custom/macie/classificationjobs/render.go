package classificationjobs

import (
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// ClassificationJobRenderer renders Macie classification jobs.
type ClassificationJobRenderer struct {
	render.BaseRenderer
}

// NewClassificationJobRenderer creates a new ClassificationJobRenderer.
func NewClassificationJobRenderer() render.Renderer {
	return &ClassificationJobRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "macie",
			Resource: "classification-jobs",
			Cols: []render.Column{
				{Name: "JOB ID", Width: 35, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "NAME", Width: 25, Getter: getName},
				{Name: "STATUS", Width: 12, Getter: getStatus},
				{Name: "TYPE", Width: 12, Getter: getJobType},
				{Name: "CREATED", Width: 15, Getter: getCreated},
			},
		},
	}
}

func getName(r dao.Resource) string {
	job, ok := r.(*ClassificationJobResource)
	if !ok {
		return ""
	}
	return job.Name()
}

func getStatus(r dao.Resource) string {
	job, ok := r.(*ClassificationJobResource)
	if !ok {
		return ""
	}
	return job.Status()
}

func getJobType(r dao.Resource) string {
	job, ok := r.(*ClassificationJobResource)
	if !ok {
		return ""
	}
	return job.JobType()
}

func getCreated(r dao.Resource) string {
	job, ok := r.(*ClassificationJobResource)
	if !ok {
		return ""
	}
	if t := job.CreatedAt(); t != nil {
		return render.FormatAge(*t)
	}
	return ""
}

// RenderDetail renders the detail view for a classification job.
func (r *ClassificationJobRenderer) RenderDetail(resource dao.Resource) string {
	job, ok := resource.(*ClassificationJobResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Macie Classification Job", job.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Job ID", job.GetID())
	d.Field("Name", job.Name())
	d.Field("Status", job.Status())
	d.Field("Type", job.JobType())

	// Timestamps
	d.Section("Timestamps")
	if t := job.CreatedAt(); t != nil {
		d.Field("Created", t.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary renders summary fields for a classification job.
func (r *ClassificationJobRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	job, ok := resource.(*ClassificationJobResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Job ID", Value: job.GetID()},
		{Label: "Name", Value: job.Name()},
		{Label: "Status", Value: job.Status()},
	}
}
