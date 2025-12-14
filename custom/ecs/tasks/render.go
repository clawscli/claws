package tasks

import (
	"fmt"
	"strings"
	"time"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// TaskRenderer renders ECS tasks
// Ensure TaskRenderer implements render.Navigator
var _ render.Navigator = (*TaskRenderer)(nil)

type TaskRenderer struct {
	render.BaseRenderer
}

// NewTaskRenderer creates a new TaskRenderer
func NewTaskRenderer() render.Renderer {
	return &TaskRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "ecs",
			Resource: "tasks",
			Cols: []render.Column{
				{Name: "TASK ID", Width: 38, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "STATUS", Width: 12, Getter: getStatus},
				{Name: "EXEC", Width: 5, Getter: getExecEnabled},
				{Name: "LAUNCH", Width: 10, Getter: getLaunchType},
				{Name: "CPU", Width: 6, Getter: getCPU},
				{Name: "MEM", Width: 8, Getter: getMemory},
				{Name: "AGE", Width: 8, Getter: getAge},
				{Name: "HEALTH", Width: 10, Getter: getHealth},
			},
		},
	}
}

func getExecEnabled(r dao.Resource) string {
	if task, ok := r.(*TaskResource); ok {
		if task.EnableExecuteCommand() {
			return "yes"
		}
		return "-"
	}
	return ""
}

func getStatus(r dao.Resource) string {
	if task, ok := r.(*TaskResource); ok {
		status := task.LastStatus()
		switch status {
		case "RUNNING":
			return "running"
		case "PENDING":
			return "pending"
		case "STOPPED":
			return "stopped"
		case "DEACTIVATING":
			return "stopping"
		case "PROVISIONING":
			return "pending"
		default:
			return strings.ToLower(status)
		}
	}
	return ""
}

func getLaunchType(r dao.Resource) string {
	if task, ok := r.(*TaskResource); ok {
		return task.LaunchType()
	}
	return ""
}

func getCPU(r dao.Resource) string {
	if task, ok := r.(*TaskResource); ok {
		cpu := task.CPU()
		if cpu != "" {
			return cpu
		}
	}
	return "-"
}

func getMemory(r dao.Resource) string {
	if task, ok := r.(*TaskResource); ok {
		mem := task.Memory()
		if mem != "" {
			return mem + "MB"
		}
	}
	return "-"
}

func getAge(r dao.Resource) string {
	if task, ok := r.(*TaskResource); ok {
		if task.Item.StartedAt != nil {
			return render.FormatAge(*task.Item.StartedAt)
		}
	}
	return "-"
}

func getHealth(r dao.Resource) string {
	if task, ok := r.(*TaskResource); ok {
		health := task.HealthStatus()
		switch health {
		case "HEALTHY":
			return "healthy"
		case "UNHEALTHY":
			return "unhealthy"
		case "UNKNOWN":
			return "-"
		default:
			return strings.ToLower(health)
		}
	}
	return ""
}

// RenderDetail renders detailed task information
func (r *TaskRenderer) RenderDetail(resource dao.Resource) string {
	task, ok := resource.(*TaskResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("ECS Task", task.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Task ID", task.GetID())
	d.Field("ARN", task.GetARN())
	d.FieldStyled("Status", task.LastStatus(), render.StateColorer()(strings.ToLower(task.LastStatus())))
	d.Field("Desired Status", task.DesiredStatus())

	// Extract cluster name
	d.Field("Cluster", appaws.ExtractResourceName(task.ClusterArn()))

	if group := task.Group(); group != "" {
		d.Field("Group", group)
	}

	if lt := task.LaunchType(); lt != "" {
		d.Field("Launch Type", lt)
	}

	// ECS Exec status
	execEnabled := task.EnableExecuteCommand()
	if execEnabled {
		d.FieldStyled("ECS Exec", "Enabled", render.StateColorer()("running"))
	} else {
		d.FieldStyled("ECS Exec", "Disabled", render.StateColorer()("stopped"))
	}

	// Resources
	d.Section("Resources")
	if cpu := task.CPU(); cpu != "" {
		d.Field("CPU", cpu+" units")
	}
	if mem := task.Memory(); mem != "" {
		d.Field("Memory", mem+" MB")
	}

	// Timing
	if started := task.StartedAt(); started != "" {
		d.Section("Timing")
		d.Field("Started At", started)
		if task.Item.StartedAt != nil {
			d.Field("Uptime", time.Since(*task.Item.StartedAt).Truncate(time.Second).String())
		}
	}

	// Task Definition
	if td := task.TaskDefinitionArn(); td != "" {
		d.Section("Task Definition")
		d.Field("Family:Revision", appaws.ExtractResourceName(td))
		d.Field("ARN", td)
	}

	// Containers
	containers := task.Containers()
	if len(containers) > 0 {
		d.Section("Containers")
		for _, c := range containers {
			name := appaws.Str(c.Name)
			if name == "" {
				name = "-"
			}
			status := appaws.Str(c.LastStatus)
			if status == "" {
				status = "-"
			}
			d.Field(name, status)
			if c.Reason != nil && *c.Reason != "" {
				d.Field("  Reason", *c.Reason)
			}
			if c.ExitCode != nil {
				d.Field("  Exit Code", fmt.Sprintf("%d", *c.ExitCode))
			}
		}
	}

	// Health
	if health := task.HealthStatus(); health != "" && health != "UNKNOWN" {
		d.Section("Health")
		d.FieldStyled("Status", health, render.StateColorer()(strings.ToLower(health)))
	}

	// Stop reason
	if reason := task.StoppedReason(); reason != "" {
		d.Section("Stop Information")
		d.Field("Reason", reason)
	}

	// Tags
	d.Tags(task.GetTags())

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *TaskRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	task, ok := resource.(*TaskResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Task ID", Value: task.GetID()},
		{Label: "ARN", Value: task.GetARN()},
		{Label: "Status", Value: task.LastStatus()},
		{Label: "Desired Status", Value: task.DesiredStatus()},
	}

	if lt := task.LaunchType(); lt != "" {
		fields = append(fields, render.SummaryField{Label: "Launch Type", Value: lt})
	}

	if cpu := task.CPU(); cpu != "" {
		fields = append(fields, render.SummaryField{Label: "CPU", Value: cpu + " units"})
	}

	if mem := task.Memory(); mem != "" {
		fields = append(fields, render.SummaryField{Label: "Memory", Value: mem + " MB"})
	}

	if started := task.StartedAt(); started != "" {
		fields = append(fields, render.SummaryField{Label: "Started At", Value: started})
		if task.Item.StartedAt != nil {
			fields = append(fields, render.SummaryField{
				Label: "Uptime",
				Value: time.Since(*task.Item.StartedAt).Truncate(time.Second).String(),
			})
		}
	}

	if health := task.HealthStatus(); health != "" && health != "UNKNOWN" {
		fields = append(fields, render.SummaryField{Label: "Health", Value: health})
	}

	if group := task.Group(); group != "" {
		fields = append(fields, render.SummaryField{Label: "Group", Value: group})
	}

	// Task definition
	if td := task.TaskDefinitionArn(); td != "" {
		fields = append(fields, render.SummaryField{Label: "Task Definition", Value: appaws.ExtractResourceName(td)})
	}

	// Containers
	containers := task.Containers()
	if len(containers) > 0 {
		var containerInfo []string
		for _, c := range containers {
			name := appaws.Str(c.Name)
			status := appaws.Str(c.LastStatus)
			containerInfo = append(containerInfo, fmt.Sprintf("%s(%s)", name, status))
		}
		fields = append(fields, render.SummaryField{
			Label: "Containers",
			Value: strings.Join(containerInfo, ", "),
		})
	}

	if reason := task.StoppedReason(); reason != "" {
		fields = append(fields, render.SummaryField{Label: "Stop Reason", Value: reason})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *TaskRenderer) Navigations(resource dao.Resource) []render.Navigation {
	task, ok := resource.(*TaskResource)
	if !ok {
		return nil
	}

	// Extract cluster name from ARN for filtering
	clusterName := appaws.ExtractResourceName(task.ClusterArn())

	navs := []render.Navigation{
		{
			Key:         "c",
			Label:       "Cluster",
			Service:     "ecs",
			Resource:    "clusters",
			FilterField: "ClusterName",
			FilterValue: clusterName,
		},
	}

	// If task is part of a service, add service navigation
	if group := task.Group(); strings.HasPrefix(group, "service:") {
		serviceName := strings.TrimPrefix(group, "service:")
		navs = append(navs, render.Navigation{
			Key:         "s",
			Label:       "Service",
			Service:     "ecs",
			Resource:    "services",
			FilterField: "ServiceName",
			FilterValue: serviceName,
		})
	}

	// Add logs navigation - use task definition family as log group prefix
	if taskDef := task.TaskDefinitionArn(); taskDef != "" {
		// Task definition ARN: arn:aws:ecs:region:account:task-definition/family:revision
		taskDefName := appaws.ExtractResourceName(taskDef)
		// Remove revision number (e.g., "my-task:5" -> "my-task")
		if idx := strings.LastIndex(taskDefName, ":"); idx > 0 {
			taskDefName = taskDefName[:idx]
		}
		navs = append(navs, render.Navigation{
			Key:         "l",
			Label:       "Logs",
			Service:     "cloudwatch",
			Resource:    "log-groups",
			FilterField: "LogGroupPrefix",
			FilterValue: "/ecs/" + taskDefName,
		})
	}

	// Add ECR navigation if container uses ECR image
	if len(task.Item.Containers) > 0 {
		for _, container := range task.Item.Containers {
			if container.Image != nil && strings.Contains(*container.Image, ".dkr.ecr.") {
				// Extract repository name from ECR URL
				// Format: <account>.dkr.ecr.<region>.amazonaws.com/<repository>:<tag>
				image := *container.Image
				if idx := strings.Index(image, ".amazonaws.com/"); idx > 0 {
					repoWithTag := image[idx+len(".amazonaws.com/"):]
					// Remove tag if present
					if tagIdx := strings.Index(repoWithTag, ":"); tagIdx > 0 {
						repoWithTag = repoWithTag[:tagIdx]
					}
					navs = append(navs, render.Navigation{
						Key:         "i",
						Label:       "ECR Image",
						Service:     "ecr",
						Resource:    "repositories",
						FilterField: "RepositoryName",
						FilterValue: repoWithTag,
					})
					break // Only add first ECR image
				}
			}
		}
	}

	return navs
}
