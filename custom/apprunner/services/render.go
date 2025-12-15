package services

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// ServiceRenderer renders App Runner services.
// Ensure ServiceRenderer implements render.Navigator
var _ render.Navigator = (*ServiceRenderer)(nil)

type ServiceRenderer struct {
	render.BaseRenderer
}

// NewServiceRenderer creates a new ServiceRenderer.
func NewServiceRenderer() render.Renderer {
	return &ServiceRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "apprunner",
			Resource: "services",
			Cols: []render.Column{
				{Name: "SERVICE NAME", Width: 30, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "STATUS", Width: 18, Getter: getStatus},
				{Name: "SERVICE URL", Width: 50, Getter: getServiceUrl},
				{Name: "UPDATED", Width: 20, Getter: getUpdated},
			},
		},
	}
}

func getStatus(r dao.Resource) string {
	svc, ok := r.(*ServiceResource)
	if !ok {
		return ""
	}
	return svc.Status()
}

func getServiceUrl(r dao.Resource) string {
	svc, ok := r.(*ServiceResource)
	if !ok {
		return ""
	}
	url := svc.ServiceUrl()
	if len(url) > 47 {
		return url[:47] + "..."
	}
	return url
}

func getUpdated(r dao.Resource) string {
	svc, ok := r.(*ServiceResource)
	if !ok {
		return ""
	}
	if t := svc.UpdatedAt(); t != nil {
		return t.Format("2006-01-02 15:04")
	}
	return ""
}

// RenderDetail renders the detail view for an App Runner service.
func (r *ServiceRenderer) RenderDetail(resource dao.Resource) string {
	svc, ok := resource.(*ServiceResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("App Runner Service", svc.ServiceName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Service Name", svc.ServiceName())
	d.Field("ARN", svc.GetARN())
	if id := svc.ServiceId(); id != "" {
		d.Field("Service ID", id)
	}
	d.Field("Status", svc.Status())

	// URL
	if url := svc.ServiceUrl(); url != "" {
		d.Section("Access")
		d.Field("Service URL", "https://"+url)
		if svc.IngressIsPublic() {
			d.Field("Public Access", "Enabled")
		} else {
			d.Field("Public Access", "Disabled")
		}
	}

	// Instance Configuration
	d.Section("Instance Configuration")
	if cpu := svc.Cpu(); cpu != "" {
		d.Field("CPU", cpu)
	}
	if mem := svc.Memory(); mem != "" {
		d.Field("Memory", mem)
	}
	if role := svc.InstanceRoleArn(); role != "" {
		d.Field("Instance Role ARN", role)
	}

	// Source Configuration
	if sourceType := svc.SourceType(); sourceType != "" {
		d.Section("Source Configuration")
		d.Field("Source Type", sourceType)
		if sourceType == "IMAGE_REPOSITORY" {
			if img := svc.ImageIdentifier(); img != "" {
				d.Field("Image", img)
			}
		} else if sourceType == "CODE_REPOSITORY" {
			if repo := svc.RepositoryUrl(); repo != "" {
				d.Field("Repository URL", repo)
			}
		}
		if svc.AutoDeploymentsEnabled() {
			d.Field("Auto Deployments", "Enabled")
		} else {
			d.Field("Auto Deployments", "Disabled")
		}
	}

	// Health Check Configuration
	if proto := svc.HealthCheckProtocol(); proto != "" {
		d.Section("Health Check")
		d.Field("Protocol", proto)
		if path := svc.HealthCheckPath(); path != "" {
			d.Field("Path", path)
		}
		if interval := svc.HealthCheckInterval(); interval > 0 {
			d.Field("Interval", fmt.Sprintf("%d seconds", interval))
		}
		if timeout := svc.HealthCheckTimeout(); timeout > 0 {
			d.Field("Timeout", fmt.Sprintf("%d seconds", timeout))
		}
	}

	// Network Configuration
	if egress := svc.NetworkEgressType(); egress != "" {
		d.Section("Network Configuration")
		d.Field("Egress Type", egress)
		if vpc := svc.VpcConnectorArn(); vpc != "" {
			d.Field("VPC Connector ARN", vpc)
		}
	}

	// Observability
	d.Section("Observability")
	if svc.ObservabilityEnabled() {
		d.Field("Observability", "Enabled")
	} else {
		d.Field("Observability", "Disabled")
	}

	// Encryption
	if kms := svc.EncryptionKeyArn(); kms != "" {
		d.Section("Encryption")
		d.Field("KMS Key ARN", kms)
	}

	// Timestamps
	d.Section("Timestamps")
	if t := svc.CreatedAt(); t != nil {
		d.Field("Created", t.Format("2006-01-02 15:04:05"))
	}
	if t := svc.UpdatedAt(); t != nil {
		d.Field("Updated", t.Format("2006-01-02 15:04:05"))
	}
	if t := svc.DeletedAt(); t != nil {
		d.Field("Deleted", t.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary renders summary fields for an App Runner service.
func (r *ServiceRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	svc, ok := resource.(*ServiceResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Service Name", Value: svc.ServiceName()},
		{Label: "ARN", Value: svc.GetARN()},
		{Label: "Status", Value: svc.Status()},
	}

	if url := svc.ServiceUrl(); url != "" {
		fields = append(fields, render.SummaryField{Label: "URL", Value: "https://" + url})
	}

	return fields
}

// Navigations returns available navigations from an App Runner service.
func (r *ServiceRenderer) Navigations(resource dao.Resource) []render.Navigation {
	svc, ok := resource.(*ServiceResource)
	if !ok {
		return nil
	}
	return []render.Navigation{
		{
			Key:         "o",
			Label:       "Operations",
			Service:     "apprunner",
			Resource:    "operations",
			FilterField: "ServiceArn",
			FilterValue: svc.GetARN(),
		},
	}
}
