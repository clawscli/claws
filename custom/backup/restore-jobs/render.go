package restorejobs

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// Ensure RestoreJobRenderer implements render.Navigator
var _ render.Navigator = (*RestoreJobRenderer)(nil)

// RestoreJobRenderer renders AWS Backup restore jobs
type RestoreJobRenderer struct {
	render.BaseRenderer
}

// NewRestoreJobRenderer creates a new RestoreJobRenderer
func NewRestoreJobRenderer() *RestoreJobRenderer {
	return &RestoreJobRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "backup",
			Resource: "restore-jobs",
			Cols: []render.Column{
				{Name: "JOB ID", Width: 36, Getter: getJobId},
				{Name: "STATUS", Width: 12, Getter: getStatus},
				{Name: "RESOURCE TYPE", Width: 15, Getter: getResourceType},
				{Name: "SIZE", Width: 12, Getter: getSize},
				{Name: "PROGRESS", Width: 10, Getter: getProgress},
				{Name: "CREATED", Width: 20, Getter: getCreated},
			},
		},
	}
}

func getJobId(r dao.Resource) string {
	if j, ok := r.(*RestoreJobResource); ok {
		return j.JobId()
	}
	return ""
}

func getStatus(r dao.Resource) string {
	if j, ok := r.(*RestoreJobResource); ok {
		return j.Status()
	}
	return ""
}

func getResourceType(r dao.Resource) string {
	if j, ok := r.(*RestoreJobResource); ok {
		return j.ResourceType()
	}
	return ""
}

func getSize(r dao.Resource) string {
	if j, ok := r.(*RestoreJobResource); ok {
		return j.BackupSizeFormatted()
	}
	return "-"
}

func getProgress(r dao.Resource) string {
	if j, ok := r.(*RestoreJobResource); ok {
		if pct := j.PercentDone(); pct != "" {
			return pct + "%"
		}
	}
	return "-"
}

func getCreated(r dao.Resource) string {
	if j, ok := r.(*RestoreJobResource); ok {
		return j.CreationDate()
	}
	return "-"
}

// RenderDetail renders detailed restore job information
func (r *RestoreJobRenderer) RenderDetail(resource dao.Resource) string {
	job, ok := resource.(*RestoreJobResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Restore Job", job.JobId())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Job ID", job.JobId())
	d.Field("Status", job.Status())
	if msg := job.StatusMessage(); msg != "" {
		d.Field("Status Message", msg)
	}

	// Resource
	d.Section("Resource")
	d.Field("Type", job.ResourceType())
	if arn := job.CreatedResourceArn(); arn != "" {
		d.Field("Created Resource ARN", arn)
	}

	// Recovery Point
	d.Section("Recovery Point")
	if rpArn := job.RecoveryPointArn(); rpArn != "" {
		d.Field("ARN", rpArn)
	}
	if rpDate := job.RecoveryPointCreationDate(); rpDate != "" {
		d.Field("Creation Date", rpDate)
	}

	// Restore Details
	d.Section("Restore Details")
	if size := job.BackupSizeFormatted(); size != "-" {
		d.Field("Size", size)
	}
	if pct := job.PercentDone(); pct != "" {
		d.Field("Progress", pct+"%")
	}
	if expectedMin := job.ExpectedCompletionTimeMinutes(); expectedMin > 0 {
		d.Field("Expected Completion", fmt.Sprintf("%d minutes", expectedMin))
	}

	// Validation
	if validationStatus := job.ValidationStatus(); validationStatus != "" {
		d.Section("Validation")
		d.Field("Status", validationStatus)
		if msg := job.ValidationStatusMessage(); msg != "" {
			d.Field("Message", msg)
		}
	}

	// Deletion
	if deletionStatus := job.DeletionStatus(); deletionStatus != "" {
		d.Section("Deletion")
		d.Field("Status", deletionStatus)
		if msg := job.DeletionStatusMessage(); msg != "" {
			d.Field("Message", msg)
		}
	}

	// IAM
	if roleArn := job.IamRoleArn(); roleArn != "" {
		d.Section("IAM")
		d.Field("Role ARN", roleArn)
	}

	// Timestamps
	d.Section("Timestamps")
	if created := job.CreationDate(); created != "" {
		d.Field("Created", created)
	}
	if completed := job.CompletionDate(); completed != "" {
		d.Field("Completed", completed)
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *RestoreJobRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	job, ok := resource.(*RestoreJobResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Job ID", Value: job.JobId()},
		{Label: "Status", Value: job.Status()},
		{Label: "Resource Type", Value: job.ResourceType()},
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
func (r *RestoreJobRenderer) Navigations(resource dao.Resource) []render.Navigation {
	// Restore jobs don't have direct navigation to vault (recovery point ARN contains vault info but parsing is complex)
	return nil
}
