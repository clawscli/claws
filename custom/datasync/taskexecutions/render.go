package taskexecutions

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// TaskExecutionRenderer renders DataSync task executions.
type TaskExecutionRenderer struct {
	render.BaseRenderer
}

// NewTaskExecutionRenderer creates a new TaskExecutionRenderer.
func NewTaskExecutionRenderer() render.Renderer {
	return &TaskExecutionRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "datasync",
			Resource: "task-executions",
			Cols: []render.Column{
				{Name: "EXECUTION ID", Width: 35, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "STATUS", Width: 15, Getter: getStatus},
			},
		},
	}
}

func getStatus(r dao.Resource) string {
	exec, ok := r.(*TaskExecutionResource)
	if !ok {
		return ""
	}
	return exec.Status()
}

// RenderDetail renders the detail view for a task execution.
func (r *TaskExecutionRenderer) RenderDetail(resource dao.Resource) string {
	exec, ok := resource.(*TaskExecutionResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("DataSync Task Execution", exec.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Execution ID", exec.GetID())
	d.Field("ARN", exec.GetARN())
	d.Field("Status", exec.Status())

	// Timing
	if t := exec.GetStartTime(); t != nil {
		d.Section("Timing")
		d.Field("Started", t.Format("2006-01-02 15:04:05"))
	}

	// File Statistics
	d.Section("File Statistics")
	if exec.FilesTransferred > 0 {
		d.Field("Files Transferred", fmt.Sprintf("%d", exec.FilesTransferred))
	}
	if exec.EstimatedFiles > 0 {
		d.Field("Estimated Files", fmt.Sprintf("%d", exec.EstimatedFiles))
	}
	if exec.GetFilesVerified() > 0 {
		d.Field("Files Verified", fmt.Sprintf("%d", exec.GetFilesVerified()))
	}
	if exec.GetFilesDeleted() > 0 {
		d.Field("Files Deleted", fmt.Sprintf("%d", exec.GetFilesDeleted()))
	}
	if exec.GetFilesSkipped() > 0 {
		d.Field("Files Skipped", fmt.Sprintf("%d", exec.GetFilesSkipped()))
	}

	// Byte Statistics
	d.Section("Byte Statistics")
	if exec.BytesTransferred > 0 {
		d.Field("Bytes Transferred", render.FormatSize(exec.BytesTransferred))
	}
	if exec.BytesWritten > 0 {
		d.Field("Bytes Written", render.FormatSize(exec.BytesWritten))
	}
	if exec.GetBytesCompressed() > 0 {
		d.Field("Bytes Compressed", render.FormatSize(exec.GetBytesCompressed()))
	}
	if exec.EstimatedBytes > 0 {
		d.Field("Estimated Bytes", render.FormatSize(exec.EstimatedBytes))
	}

	// Result
	if result := exec.GetResult(); result != nil {
		d.Section("Result")
		if result.PrepareDuration != nil {
			d.Field("Prepare Duration", fmt.Sprintf("%dms", *result.PrepareDuration))
		}
		if result.TransferDuration != nil {
			d.Field("Transfer Duration", fmt.Sprintf("%dms", *result.TransferDuration))
		}
		if result.VerifyDuration != nil {
			d.Field("Verify Duration", fmt.Sprintf("%dms", *result.VerifyDuration))
		}
		if result.TotalDuration != nil {
			d.Field("Total Duration", fmt.Sprintf("%dms", *result.TotalDuration))
		}
		if result.ErrorCode != nil {
			d.Field("Error Code", *result.ErrorCode)
		}
		if result.ErrorDetail != nil {
			d.Field("Error Detail", *result.ErrorDetail)
		}
	}

	return d.String()
}

// RenderSummary renders summary fields for a task execution.
func (r *TaskExecutionRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	exec, ok := resource.(*TaskExecutionResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Execution ID", Value: exec.GetID()},
		{Label: "Status", Value: exec.Status()},
	}
}
