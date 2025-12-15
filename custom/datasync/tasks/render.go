package tasks

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// formatBytes formats bytes to human-readable format.
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// TaskRenderer renders DataSync tasks.
// Ensure TaskRenderer implements render.Navigator
var _ render.Navigator = (*TaskRenderer)(nil)

type TaskRenderer struct {
	render.BaseRenderer
}

// NewTaskRenderer creates a new TaskRenderer.
func NewTaskRenderer() render.Renderer {
	return &TaskRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "datasync",
			Resource: "tasks",
			Cols: []render.Column{
				{Name: "TASK ID", Width: 35, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "NAME", Width: 30, Getter: getName},
				{Name: "STATUS", Width: 15, Getter: getStatus},
			},
		},
	}
}

func getName(r dao.Resource) string {
	task, ok := r.(*TaskResource)
	if !ok {
		return ""
	}
	return task.Name()
}

func getStatus(r dao.Resource) string {
	task, ok := r.(*TaskResource)
	if !ok {
		return ""
	}
	return task.Status()
}

// RenderDetail renders the detail view for a task.
func (r *TaskRenderer) RenderDetail(resource dao.Resource) string {
	task, ok := resource.(*TaskResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("DataSync Task", task.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Task ID", task.GetID())
	d.Field("Name", task.Name())
	d.Field("ARN", task.TaskArn())
	d.Field("Status", task.Status())
	if task.GetTaskMode() != "" {
		d.Field("Task Mode", task.GetTaskMode())
	}

	// Schedule
	if schedule := task.GetSchedule(); schedule != "" {
		d.Section("Schedule")
		d.Field("Schedule Expression", schedule)
	}

	// Locations
	if task.SourceLocationArn != "" || task.DestLocationArn != "" {
		d.Section("Locations")
		if task.SourceLocationArn != "" {
			d.Field("Source Location", task.SourceLocationArn)
		}
		if task.DestLocationArn != "" {
			d.Field("Destination Location", task.DestLocationArn)
		}
	}

	// Current Execution
	if execArn := task.GetCurrentExecutionArn(); execArn != "" {
		d.Section("Current Execution")
		d.Field("Execution ARN", execArn)
	}

	// Options
	if opts := task.GetOptions(); opts != nil {
		d.Section("Transfer Options")
		if opts.VerifyMode != "" {
			d.Field("Verify Mode", string(opts.VerifyMode))
		}
		if opts.OverwriteMode != "" {
			d.Field("Overwrite Mode", string(opts.OverwriteMode))
		}
		if opts.Atime != "" {
			d.Field("Access Time", string(opts.Atime))
		}
		if opts.Mtime != "" {
			d.Field("Modify Time", string(opts.Mtime))
		}
		if opts.PreserveDeletedFiles != "" {
			d.Field("Preserve Deleted", string(opts.PreserveDeletedFiles))
		}
		if opts.PreserveDevices != "" {
			d.Field("Preserve Devices", string(opts.PreserveDevices))
		}
		if opts.PosixPermissions != "" {
			d.Field("POSIX Permissions", string(opts.PosixPermissions))
		}
		if opts.BytesPerSecond != nil && *opts.BytesPerSecond > 0 {
			d.Field("Bandwidth Limit", formatBytes(*opts.BytesPerSecond)+"/s")
		}
		if opts.TaskQueueing != "" {
			d.Field("Task Queueing", string(opts.TaskQueueing))
		}
		if opts.LogLevel != "" {
			d.Field("Log Level", string(opts.LogLevel))
		}
		if opts.TransferMode != "" {
			d.Field("Transfer Mode", string(opts.TransferMode))
		}
	}

	// Logging
	if logGroup := task.GetCloudWatchLogGroupArn(); logGroup != "" {
		d.Section("Logging")
		d.Field("CloudWatch Log Group", logGroup)
	}

	// Error
	if errCode := task.GetErrorCode(); errCode != "" {
		d.Section("Error")
		d.Field("Error Code", errCode)
		if errDetail := task.GetErrorDetail(); errDetail != "" {
			d.Field("Error Detail", errDetail)
		}
	}

	// Timestamps
	if t := task.GetCreationTime(); t != nil {
		d.Section("Timestamps")
		d.Field("Created", t.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary renders summary fields for a task.
func (r *TaskRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	task, ok := resource.(*TaskResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Task ID", Value: task.GetID()},
		{Label: "Name", Value: task.Name()},
		{Label: "Status", Value: task.Status()},
	}
}

// Navigations returns available navigations from a task.
func (r *TaskRenderer) Navigations(resource dao.Resource) []render.Navigation {
	task, ok := resource.(*TaskResource)
	if !ok {
		return nil
	}
	return []render.Navigation{
		{
			Key:         "e",
			Label:       "Executions",
			Service:     "datasync",
			Resource:    "task-executions",
			FilterField: "TaskArn",
			FilterValue: task.TaskArn(),
		},
	}
}
