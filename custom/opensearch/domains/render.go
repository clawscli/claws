package domains

import (
	"fmt"
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// DomainRenderer renders OpenSearch domains
// Ensure DomainRenderer implements render.Navigator
var _ render.Navigator = (*DomainRenderer)(nil)

type DomainRenderer struct {
	render.BaseRenderer
}

// NewDomainRenderer creates a new DomainRenderer
func NewDomainRenderer() *DomainRenderer {
	return &DomainRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "opensearch",
			Resource: "domains",
			Cols: []render.Column{
				{Name: "DOMAIN", Width: 25, Getter: getDomainName},
				{Name: "VERSION", Width: 18, Getter: getVersion},
				{Name: "TYPE", Width: 18, Getter: getInstanceType},
				{Name: "INSTANCES", Width: 10, Getter: getInstanceCount},
				{Name: "STORAGE", Width: 12, Getter: getStorage},
				{Name: "STATUS", Width: 12, Getter: getStatus},
			},
		},
	}
}

func getDomainName(r dao.Resource) string {
	if domain, ok := r.(*DomainResource); ok {
		return domain.DomainName()
	}
	return ""
}

func getVersion(r dao.Resource) string {
	if domain, ok := r.(*DomainResource); ok {
		return domain.EngineVersion()
	}
	return ""
}

func getInstanceType(r dao.Resource) string {
	if domain, ok := r.(*DomainResource); ok {
		return domain.InstanceType()
	}
	return ""
}

func getInstanceCount(r dao.Resource) string {
	if domain, ok := r.(*DomainResource); ok {
		return fmt.Sprintf("%d", domain.InstanceCount())
	}
	return ""
}

func getStorage(r dao.Resource) string {
	if domain, ok := r.(*DomainResource); ok {
		if domain.EBSEnabled() {
			return fmt.Sprintf("%dGB %s", domain.VolumeSize(), domain.VolumeType())
		}
		return "Instance"
	}
	return ""
}

func getStatus(r dao.Resource) string {
	if domain, ok := r.(*DomainResource); ok {
		return domain.Status()
	}
	return ""
}

// RenderDetail renders detailed domain information
func (r *DomainRenderer) RenderDetail(resource dao.Resource) string {
	domain, ok := resource.(*DomainResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("OpenSearch Domain", domain.DomainName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Domain Name", domain.DomainName())
	d.Field("Domain ID", domain.DomainId())
	d.Field("ARN", domain.GetARN())
	d.Field("Status", domain.Status())
	d.Field("Engine Version", domain.EngineVersion())

	// Endpoint
	d.Section("Endpoint")
	if endpoint := domain.Endpoint(); endpoint != "" {
		d.Field("Endpoint", endpoint)
	}
	if endpoints := domain.Endpoints(); len(endpoints) > 0 {
		for k, v := range endpoints {
			d.Field(k, v)
		}
	}

	// Cluster Configuration
	d.Section("Cluster Configuration")
	d.Field("Instance Type", domain.InstanceType())
	d.Field("Instance Count", fmt.Sprintf("%d", domain.InstanceCount()))
	d.Field("Zone Awareness", formatBool(domain.ZoneAwarenessEnabled()))

	// Dedicated Master
	if domain.DedicatedMasterEnabled() {
		d.Section("Dedicated Master")
		d.Field("Enabled", "Yes")
		d.Field("Type", domain.DedicatedMasterType())
		d.Field("Count", fmt.Sprintf("%d", domain.DedicatedMasterCount()))
	}

	// Warm Storage
	if domain.WarmEnabled() {
		d.Section("Warm Storage")
		d.Field("Enabled", "Yes")
		d.Field("Type", domain.WarmType())
		d.Field("Count", fmt.Sprintf("%d", domain.WarmCount()))
	}

	// Storage
	d.Section("Storage")
	if domain.EBSEnabled() {
		d.Field("EBS Enabled", "Yes")
		d.Field("Volume Type", domain.VolumeType())
		d.Field("Volume Size", fmt.Sprintf("%d GB", domain.VolumeSize()))
	} else {
		d.Field("Storage Type", "Instance Storage")
	}

	// Security
	d.Section("Security")
	d.Field("Encryption at Rest", formatBool(domain.EncryptionAtRestEnabled()))
	d.Field("Node-to-Node Encryption", formatBool(domain.NodeToNodeEncryptionEnabled()))
	d.Field("Enforce HTTPS", formatBool(domain.EnforceHTTPS()))
	if tlsPolicy := domain.TLSSecurityPolicy(); tlsPolicy != "" {
		d.Field("TLS Policy", tlsPolicy)
	}
	d.Field("Fine-Grained Access Control", formatBool(domain.AdvancedSecurityEnabled()))

	// VPC
	if vpcId := domain.VPCId(); vpcId != "" {
		d.Section("VPC Configuration")
		d.Field("VPC ID", vpcId)
		if subnets := domain.SubnetIds(); len(subnets) > 0 {
			d.Field("Subnets", strings.Join(subnets, ", "))
		}
		if sgs := domain.SecurityGroupIds(); len(sgs) > 0 {
			d.Field("Security Groups", strings.Join(sgs, ", "))
		}
	}

	// Auto-Tune
	if autoTune := domain.AutoTuneState(); autoTune != "" {
		d.Section("Auto-Tune")
		d.Field("State", autoTune)
	}

	return d.String()
}

func formatBool(b bool) string {
	if b {
		return "Enabled"
	}
	return "Disabled"
}

// RenderSummary returns summary fields for the header panel
func (r *DomainRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	domain, ok := resource.(*DomainResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Domain Name", Value: domain.DomainName()},
		{Label: "ARN", Value: domain.GetARN()},
		{Label: "Status", Value: domain.Status()},
		{Label: "Version", Value: domain.EngineVersion()},
		{Label: "Instance Type", Value: domain.InstanceType()},
		{Label: "Instances", Value: fmt.Sprintf("%d", domain.InstanceCount())},
	}

	if domain.EBSEnabled() {
		fields = append(fields, render.SummaryField{
			Label: "Storage",
			Value: fmt.Sprintf("%d GB %s", domain.VolumeSize(), domain.VolumeType()),
		})
	}

	if endpoint := domain.Endpoint(); endpoint != "" {
		fields = append(fields, render.SummaryField{Label: "Endpoint", Value: endpoint})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *DomainRenderer) Navigations(resource dao.Resource) []render.Navigation {
	// No navigations for now
	return nil
}
