package clusters

import (
	"fmt"
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// ClusterRenderer renders ECS clusters
// Ensure ClusterRenderer implements render.Navigator
var _ render.Navigator = (*ClusterRenderer)(nil)

type ClusterRenderer struct {
	render.BaseRenderer
}

// NewClusterRenderer creates a new ClusterRenderer
func NewClusterRenderer() render.Renderer {
	return &ClusterRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "ecs",
			Resource: "clusters",
			Cols: []render.Column{
				{Name: "NAME", Width: 35, Getter: func(r dao.Resource) string { return r.GetName() }},
				{Name: "STATUS", Width: 10, Getter: getStatus},
				{Name: "SERVICES", Width: 10, Getter: getServicesCount},
				{Name: "RUNNING", Width: 10, Getter: getRunningTasks},
				{Name: "PENDING", Width: 10, Getter: getPendingTasks},
				{Name: "INSTANCES", Width: 10, Getter: getInstancesCount},
			},
		},
	}
}

func getStatus(r dao.Resource) string {
	if cluster, ok := r.(*ClusterResource); ok {
		status := cluster.Status()
		switch status {
		case "ACTIVE":
			return "active"
		case "PROVISIONING":
			return "pending"
		case "DEPROVISIONING":
			return "deleting"
		case "FAILED":
			return "failed"
		case "INACTIVE":
			return "stopped"
		default:
			return strings.ToLower(status)
		}
	}
	return ""
}

func getServicesCount(r dao.Resource) string {
	if cluster, ok := r.(*ClusterResource); ok {
		return fmt.Sprintf("%d", cluster.ActiveServicesCount())
	}
	return ""
}

func getRunningTasks(r dao.Resource) string {
	if cluster, ok := r.(*ClusterResource); ok {
		return fmt.Sprintf("%d", cluster.RunningTasksCount())
	}
	return ""
}

func getPendingTasks(r dao.Resource) string {
	if cluster, ok := r.(*ClusterResource); ok {
		count := cluster.PendingTasksCount()
		if count == 0 {
			return "-"
		}
		return fmt.Sprintf("%d", count)
	}
	return ""
}

func getInstancesCount(r dao.Resource) string {
	if cluster, ok := r.(*ClusterResource); ok {
		count := cluster.RegisteredContainerInstancesCount()
		if count == 0 {
			return "Fargate"
		}
		return fmt.Sprintf("%d", count)
	}
	return ""
}

// RenderDetail renders detailed cluster information
func (r *ClusterRenderer) RenderDetail(resource dao.Resource) string {
	cluster, ok := resource.(*ClusterResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("ECS Cluster", cluster.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", cluster.GetName())
	d.Field("ARN", cluster.GetARN())
	d.FieldStyled("Status", cluster.Status(), render.StateColorer()(strings.ToLower(cluster.Status())))

	// Statistics
	d.Section("Statistics")
	d.Field("Active Services", fmt.Sprintf("%d", cluster.ActiveServicesCount()))
	d.Field("Running Tasks", fmt.Sprintf("%d", cluster.RunningTasksCount()))
	d.Field("Pending Tasks", fmt.Sprintf("%d", cluster.PendingTasksCount()))
	d.Field("Container Instances", fmt.Sprintf("%d", cluster.RegisteredContainerInstancesCount()))

	// Capacity Providers
	if providers := cluster.CapacityProviders(); len(providers) > 0 {
		d.Section("Capacity Providers")
		for _, provider := range providers {
			d.Line("  • " + provider)
		}
	}

	// Settings
	if settings := cluster.Settings(); len(settings) > 0 {
		d.Section("Settings")
		for _, setting := range settings {
			if setting.Value != nil {
				d.Field(string(setting.Name), *setting.Value)
			}
		}
	}

	// Tags
	d.Tags(cluster.GetTags())

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *ClusterRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	cluster, ok := resource.(*ClusterResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: cluster.GetName()},
		{Label: "ARN", Value: cluster.GetARN()},
		{Label: "Status", Value: cluster.Status()},
		{Label: "Active Services", Value: fmt.Sprintf("%d", cluster.ActiveServicesCount())},
		{Label: "Running Tasks", Value: fmt.Sprintf("%d", cluster.RunningTasksCount())},
		{Label: "Pending Tasks", Value: fmt.Sprintf("%d", cluster.PendingTasksCount())},
		{Label: "Container Instances", Value: fmt.Sprintf("%d", cluster.RegisteredContainerInstancesCount())},
	}

	if providers := cluster.CapacityProviders(); len(providers) > 0 {
		fields = append(fields, render.SummaryField{
			Label: "Capacity Providers",
			Value: strings.Join(providers, ", "),
		})
	}

	// Container Insights
	for _, setting := range cluster.Settings() {
		if setting.Name == "containerInsights" && setting.Value != nil {
			fields = append(fields, render.SummaryField{
				Label: "Container Insights",
				Value: *setting.Value,
			})
		}
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *ClusterRenderer) Navigations(resource dao.Resource) []render.Navigation {
	cluster, ok := resource.(*ClusterResource)
	if !ok {
		return nil
	}

	// Use cluster name for filtering (DAO expects "ClusterName" in context)
	clusterName := cluster.GetName()

	return []render.Navigation{
		{
			Key:         "s",
			Label:       "Services",
			Service:     "ecs",
			Resource:    "services",
			FilterField: "ClusterName",
			FilterValue: clusterName,
		},
		{
			Key:         "t",
			Label:       "Tasks",
			Service:     "ecs",
			Resource:    "tasks",
			FilterField: "ClusterName",
			FilterValue: clusterName,
		},
	}
}
