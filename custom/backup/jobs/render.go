package jobs

import (
	"fmt"
	"sort"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// JobRenderer renders AWS Backup jobs
// Ensure JobRenderer implements render.Navigator
var _ render.Navigator = (*JobRenderer)(nil)

type JobRenderer struct {
	render.BaseRenderer
}

// NewJobRenderer creates a new JobRenderer
func NewJobRenderer() *JobRenderer {
	return &JobRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "backup",
			Resource: "jobs",
			Cols: []render.Column{
				{Name: "JOB ID", Width: 36, Getter: getJobId},
				{Name: "STATE", Width: 12, Getter: getState},
				{Name: "RESOURCE TYPE", Width: 15, Getter: getResourceType},
				{Name: "SIZE", Width: 12, Getter: getSize},
				{Name: "PROGRESS", Width: 10, Getter: getProgress},
				{Name: "CREATED", Width: 20, Getter: getCreated},
			},
		},
	}
}

func getJobId(r dao.Resource) string {
	if j, ok := r.(*JobResource); ok {
		return j.JobId()
	}
	return ""
}

func getState(r dao.Resource) string {
	if j, ok := r.(*JobResource); ok {
		return j.State()
	}
	return ""
}

func getResourceType(r dao.Resource) string {
	if j, ok := r.(*JobResource); ok {
		return j.ResourceType()
	}
	return ""
}

func getSize(r dao.Resource) string {
	if j, ok := r.(*JobResource); ok {
		return j.BackupSizeFormatted()
	}
	return "-"
}

func getProgress(r dao.Resource) string {
	if j, ok := r.(*JobResource); ok {
		if pct := j.PercentDone(); pct != "" {
			return pct + "%"
		}
	}
	return "-"
}

func getCreated(r dao.Resource) string {
	if j, ok := r.(*JobResource); ok {
		return j.CreationDate()
	}
	return "-"
}

// RenderDetail renders detailed job information
func (r *JobRenderer) RenderDetail(resource dao.Resource) string {
	job, ok := resource.(*JobResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Backup Job", job.JobId())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Job ID", job.JobId())
	d.Field("State", job.State())
	d.Field("Backup Plan ID", job.BackupPlanId)

	// Resource
	d.Section("Resource")
	d.Field("Type", job.ResourceType())
	if arn := job.ResourceArn(); arn != "" {
		d.Field("ARN", arn)
	}

	// Backup Details
	d.Section("Backup Details")
	if vault := job.BackupVaultName(); vault != "" {
		d.Field("Vault", vault)
	}
	if size := job.BackupSizeFormatted(); size != "-" {
		d.Field("Size", size)
	}
	if pct := job.PercentDone(); pct != "" {
		d.Field("Progress", pct+"%")
	}
	if msg := job.StatusMessage(); msg != "" {
		d.Field("Status Message", msg)
	}

	// Timestamps
	d.Section("Timestamps")
	if created := job.CreationDate(); created != "" {
		d.Field("Created", created)
	}
	if completed := job.CompletionDate(); completed != "" {
		d.Field("Completed", completed)
	}
	if expected := job.ExpectedCompletionDate(); expected != "" {
		d.Field("Expected Completion", expected)
	}

	// Tags (from detail)
	if job.Detail != nil && len(job.BaseResource.Tags) > 0 {
		d.Section("Tags")
		keys := make([]string, 0, len(job.BaseResource.Tags))
		for k := range job.BaseResource.Tags {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			d.Field(k, job.BaseResource.Tags[k])
		}
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *JobRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	job, ok := resource.(*JobResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Job ID", Value: job.JobId()},
		{Label: "State", Value: job.State()},
		{Label: "Resource Type", Value: job.ResourceType()},
	}

	if vault := job.BackupVaultName(); vault != "" {
		fields = append(fields, render.SummaryField{Label: "Vault", Value: vault})
	}

	if size := job.BackupSizeFormatted(); size != "-" {
		fields = append(fields, render.SummaryField{Label: "Size", Value: size})
	}

	if pct := job.PercentDone(); pct != "" {
		fields = append(fields, render.SummaryField{Label: "Progress", Value: fmt.Sprintf("%s%%", pct)})
	}

	if created := job.CreationDate(); created != "" {
		fields = append(fields, render.SummaryField{Label: "Created", Value: created})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *JobRenderer) Navigations(resource dao.Resource) []render.Navigation {
	return nil
}
