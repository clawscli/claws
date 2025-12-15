package services

import (
	"fmt"
	"strings"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// ServiceRenderer renders ECS services
// Ensure ServiceRenderer implements render.Navigator
var _ render.Navigator = (*ServiceRenderer)(nil)

type ServiceRenderer struct {
	render.BaseRenderer
}

// NewServiceRenderer creates a new ServiceRenderer
func NewServiceRenderer() render.Renderer {
	return &ServiceRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "ecs",
			Resource: "services",
			Cols: []render.Column{
				{Name: "NAME", Width: 35, Getter: func(r dao.Resource) string { return r.GetName() }},
				{Name: "STATUS", Width: 10, Getter: getStatus},
				{Name: "DESIRED", Width: 8, Getter: getDesired},
				{Name: "RUNNING", Width: 8, Getter: getRunning},
				{Name: "PENDING", Width: 8, Getter: getPending},
				{Name: "LAUNCH", Width: 10, Getter: getLaunchType},
				{Name: "TASK DEF", Width: 25, Getter: getTaskDef},
			},
		},
	}
}

func getStatus(r dao.Resource) string {
	if svc, ok := r.(*ServiceResource); ok {
		status := svc.Status()
		switch status {
		case "ACTIVE":
			return "active"
		case "DRAINING":
			return "deleting"
		case "INACTIVE":
			return "stopped"
		default:
			return strings.ToLower(status)
		}
	}
	return ""
}

func getDesired(r dao.Resource) string {
	if svc, ok := r.(*ServiceResource); ok {
		return fmt.Sprintf("%d", svc.DesiredCount())
	}
	return ""
}

func getRunning(r dao.Resource) string {
	if svc, ok := r.(*ServiceResource); ok {
		return fmt.Sprintf("%d", svc.RunningCount())
	}
	return ""
}

func getPending(r dao.Resource) string {
	if svc, ok := r.(*ServiceResource); ok {
		count := svc.PendingCount()
		if count == 0 {
			return "-"
		}
		return fmt.Sprintf("%d", count)
	}
	return ""
}

func getLaunchType(r dao.Resource) string {
	if svc, ok := r.(*ServiceResource); ok {
		lt := svc.LaunchType()
		if lt == "" {
			// Check capacity provider strategy
			return "CP"
		}
		return lt
	}
	return ""
}

func getTaskDef(r dao.Resource) string {
	if svc, ok := r.(*ServiceResource); ok {
		return appaws.ExtractResourceName(svc.TaskDefinition())
	}
	return ""
}

// RenderDetail renders detailed service information
func (r *ServiceRenderer) RenderDetail(resource dao.Resource) string {
	svc, ok := resource.(*ServiceResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("ECS Service", svc.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", svc.GetName())
	d.Field("ARN", svc.GetARN())
	d.FieldStyled("Status", svc.Status(), render.StateColorer()(strings.ToLower(svc.Status())))

	// Extract cluster name
	d.Field("Cluster", appaws.ExtractResourceName(svc.ClusterArn()))

	if lt := svc.LaunchType(); lt != "" {
		d.Field("Launch Type", lt)
	}
	if strategy := svc.SchedulingStrategy(); strategy != "" {
		d.Field("Scheduling Strategy", strategy)
	}
	if pv := svc.PlatformVersion(); pv != "" {
		d.Field("Platform Version", pv)
	}
	if svc.EnableExecuteCommand() {
		d.Field("ECS Exec", "Enabled")
	}

	// Task counts
	d.Section("Task Counts")
	d.Field("Desired", fmt.Sprintf("%d", svc.DesiredCount()))
	d.Field("Running", fmt.Sprintf("%d", svc.RunningCount()))
	d.Field("Pending", fmt.Sprintf("%d", svc.PendingCount()))

	// Task Definition
	if td := svc.TaskDefinition(); td != "" {
		d.Section("Task Definition")
		d.Field("Family:Revision", appaws.ExtractResourceName(td))
		d.Field("ARN", td)
	}

	// Deployment Configuration
	if dc := svc.Item.DeploymentConfiguration; dc != nil {
		d.Section("Deployment Configuration")
		if dc.MaximumPercent != nil {
			d.Field("Maximum Percent", fmt.Sprintf("%d%%", *dc.MaximumPercent))
		}
		if dc.MinimumHealthyPercent != nil {
			d.Field("Minimum Healthy Percent", fmt.Sprintf("%d%%", *dc.MinimumHealthyPercent))
		}
		if dc.DeploymentCircuitBreaker != nil {
			cb := dc.DeploymentCircuitBreaker
			if cb.Enable {
				d.FieldStyled("Circuit Breaker", "Enabled", render.SuccessStyle())
				if cb.Rollback {
					d.Field("Rollback on Failure", "Enabled")
				}
			}
		}
	}

	// Deployments
	deployments := svc.Deployments()
	if len(deployments) > 0 {
		d.Section("Active Deployments")
		for i, dep := range deployments {
			status := appaws.Str(dep.Status)
			d.Field(fmt.Sprintf("Deployment %d", i+1), fmt.Sprintf("%s - %d/%d tasks", status, dep.RunningCount, dep.DesiredCount))
			if dep.RolloutState != "" {
				d.Field("  Rollout State", string(dep.RolloutState))
			}
		}
	}

	// Placement Constraints
	if constraints := svc.Item.PlacementConstraints; len(constraints) > 0 {
		d.Section("Placement Constraints")
		for _, c := range constraints {
			if c.Expression != nil {
				d.Field(string(c.Type), *c.Expression)
			} else {
				d.Field("Type", string(c.Type))
			}
		}
	}

	// Placement Strategy
	if strategies := svc.Item.PlacementStrategy; len(strategies) > 0 {
		d.Section("Placement Strategy")
		for _, s := range strategies {
			if s.Field != nil {
				d.Field(string(s.Type), *s.Field)
			} else {
				d.Field("Type", string(s.Type))
			}
		}
	}

	// Load Balancers
	lbs := svc.LoadBalancers()
	if len(lbs) > 0 {
		d.Section("Load Balancers")
		for _, lb := range lbs {
			if lb.TargetGroupArn != nil {
				tgParts := strings.Split(*lb.TargetGroupArn, "/")
				if len(tgParts) > 1 {
					d.Field("Target Group", tgParts[1])
				}
			}
			if lb.ContainerName != nil {
				d.Field("Container", fmt.Sprintf("%s:%d", *lb.ContainerName, appaws.Int32(lb.ContainerPort)))
			}
		}
	}

	// Capacity Provider Strategy
	if cps := svc.CapacityProviderStrategy(); len(cps) > 0 {
		d.Section("Capacity Provider Strategy")
		for _, cp := range cps {
			if cp.CapacityProvider != nil {
				d.Field(*cp.CapacityProvider, fmt.Sprintf("weight=%d, base=%d", cp.Weight, cp.Base))
			}
		}
	}

	// Network Configuration
	if nc := svc.NetworkConfiguration(); nc != nil && nc.AwsvpcConfiguration != nil {
		d.Section("Network Configuration")
		vpc := nc.AwsvpcConfiguration
		if len(vpc.Subnets) > 0 {
			d.Field("Subnets", strings.Join(vpc.Subnets, ", "))
		}
		if len(vpc.SecurityGroups) > 0 {
			d.Field("Security Groups", strings.Join(vpc.SecurityGroups, ", "))
		}
		d.Field("Assign Public IP", string(vpc.AssignPublicIp))
	}

	// Service Discovery
	if registries := svc.ServiceRegistries(); len(registries) > 0 {
		d.Section("Service Discovery")
		for _, reg := range registries {
			if reg.RegistryArn != nil {
				d.Field("Registry ARN", *reg.RegistryArn)
			}
		}
	}

	// Health Check
	if grace := svc.HealthCheckGracePeriodSeconds(); grace > 0 {
		d.Section("Health Check")
		d.Field("Grace Period", fmt.Sprintf("%d seconds", grace))
	}

	// Recent Events (show last 3)
	if events := svc.Events(); len(events) > 0 {
		d.Section("Recent Events")
		limit := 3
		if len(events) < limit {
			limit = len(events)
		}
		for i := 0; i < limit; i++ {
			event := events[i]
			if event.Message != nil {
				msg := *event.Message
				if len(msg) > 80 {
					msg = msg[:80] + "..."
				}
				timestamp := ""
				if event.CreatedAt != nil {
					timestamp = event.CreatedAt.Format("01-02 15:04")
				}
				d.Field(timestamp, msg)
			}
		}
	}

	// Timestamps
	d.Section("Timestamps")
	if created := svc.CreatedAt(); created != "" {
		d.Field("Created", created)
	}
	if createdBy := svc.CreatedBy(); createdBy != "" {
		d.Field("Created By", appaws.ExtractResourceName(createdBy))
	}

	// Tags
	d.Tags(svc.GetTags())

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *ServiceRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	svc, ok := resource.(*ServiceResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: svc.GetName()},
		{Label: "ARN", Value: svc.GetARN()},
		{Label: "Status", Value: svc.Status()},
		{Label: "Desired Count", Value: fmt.Sprintf("%d", svc.DesiredCount())},
		{Label: "Running Count", Value: fmt.Sprintf("%d", svc.RunningCount())},
		{Label: "Pending Count", Value: fmt.Sprintf("%d", svc.PendingCount())},
	}

	if lt := svc.LaunchType(); lt != "" {
		fields = append(fields, render.SummaryField{Label: "Launch Type", Value: lt})
	}

	if td := svc.TaskDefinition(); td != "" {
		fields = append(fields, render.SummaryField{Label: "Task Definition", Value: td})
	}

	// Deployments
	deployments := svc.Deployments()
	if len(deployments) > 0 {
		dep := deployments[0]
		status := appaws.Str(dep.Status)
		fields = append(fields, render.SummaryField{
			Label: "Deployment",
			Value: fmt.Sprintf("%s (%d/%d)", status, dep.RunningCount, dep.DesiredCount),
		})
	}

	// Load balancers
	lbs := svc.LoadBalancers()
	if len(lbs) > 0 {
		var lbNames []string
		for _, lb := range lbs {
			if lb.TargetGroupArn != nil {
				parts := strings.Split(*lb.TargetGroupArn, "/")
				if len(parts) > 1 {
					lbNames = append(lbNames, parts[1])
				}
			}
		}
		if len(lbNames) > 0 {
			fields = append(fields, render.SummaryField{
				Label: "Target Groups",
				Value: strings.Join(lbNames, ", "),
			})
		}
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *ServiceRenderer) Navigations(resource dao.Resource) []render.Navigation {
	svc, ok := resource.(*ServiceResource)
	if !ok {
		return nil
	}

	// Extract cluster name from ARN for filtering
	clusterName := appaws.ExtractResourceName(svc.ClusterArn())

	return []render.Navigation{
		{
			Key:         "t",
			Label:       "Tasks",
			Service:     "ecs",
			Resource:    "tasks",
			FilterField: "ServiceName",
			FilterValue: svc.GetName(),
		},
		{
			Key:         "c",
			Label:       "Cluster",
			Service:     "ecs",
			Resource:    "clusters",
			FilterField: "ClusterName",
			FilterValue: clusterName,
		},
		{
			Key:         "l",
			Label:       "Logs",
			Service:     "cloudwatch",
			Resource:    "log-groups",
			FilterField: "LogGroupPrefix",
			FilterValue: "/ecs/" + svc.GetName(),
		},
	}
}
