package firewalls

import (
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// FirewallRenderer renders Network Firewalls.
type FirewallRenderer struct {
	render.BaseRenderer
}

// NewFirewallRenderer creates a new FirewallRenderer.
func NewFirewallRenderer() render.Renderer {
	return &FirewallRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "network-firewall",
			Resource: "firewalls",
			Cols: []render.Column{
				{Name: "FIREWALL NAME", Width: 35, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "STATUS", Width: 15, Getter: getStatus},
				{Name: "VPC", Width: 22, Getter: getVpc},
				{Name: "DELETE PROTECTION", Width: 18, Getter: getDeleteProtection},
			},
		},
	}
}

func getStatus(r dao.Resource) string {
	fw, ok := r.(*FirewallResource)
	if !ok {
		return ""
	}
	return fw.StatusValue()
}

func getVpc(r dao.Resource) string {
	fw, ok := r.(*FirewallResource)
	if !ok {
		return ""
	}
	return fw.VpcId()
}

func getDeleteProtection(r dao.Resource) string {
	fw, ok := r.(*FirewallResource)
	if !ok {
		return ""
	}
	if fw.DeleteProtection() {
		return "Enabled"
	}
	return "Disabled"
}

// RenderDetail renders the detail view for a Network Firewall.
func (r *FirewallRenderer) RenderDetail(resource dao.Resource) string {
	fw, ok := resource.(*FirewallResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Network Firewall", fw.FirewallName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Firewall Name", fw.FirewallName())
	d.Field("ARN", fw.GetARN())
	if status := fw.StatusValue(); status != "" {
		d.Field("Status", status)
	}
	if syncState := fw.ConfigurationSyncStateSummary(); syncState != "" {
		d.Field("Config Sync State", syncState)
	}
	if desc := fw.Description(); desc != "" {
		d.Field("Description", desc)
	}

	// VPC Configuration
	d.Section("VPC Configuration")
	if vpcId := fw.VpcId(); vpcId != "" {
		d.Field("VPC ID", vpcId)
	}
	if subnets := fw.SubnetMappings(); len(subnets) > 0 {
		d.Field("Subnets", strings.Join(subnets, ", "))
	}

	// Firewall Endpoints
	if states := fw.SyncStates(); len(states) > 0 {
		d.Section("Firewall Endpoints")
		for az, endpoint := range states {
			d.Field(az, endpoint)
		}
	}

	// Policy
	if policy := fw.FirewallPolicyArn(); policy != "" {
		d.Section("Firewall Policy")
		d.Field("Policy ARN", policy)
	}

	// Protection Settings
	d.Section("Protection Settings")
	if fw.DeleteProtection() {
		d.Field("Delete Protection", "Enabled")
	} else {
		d.Field("Delete Protection", "Disabled")
	}
	if fw.SubnetChangeProtection() {
		d.Field("Subnet Change Protection", "Enabled")
	} else {
		d.Field("Subnet Change Protection", "Disabled")
	}
	if fw.PolicyChangeProtection() {
		d.Field("Policy Change Protection", "Enabled")
	} else {
		d.Field("Policy Change Protection", "Disabled")
	}

	// Encryption
	if keyId := fw.EncryptionKeyId(); keyId != "" {
		d.Section("Encryption")
		d.Field("KMS Key ID", keyId)
	}

	// Tags
	if tags := fw.Tags(); len(tags) > 0 {
		d.Section("Tags")
		for k, v := range tags {
			d.Field(k, v)
		}
	}

	return d.String()
}

// RenderSummary renders summary fields for a Network Firewall.
func (r *FirewallRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	fw, ok := resource.(*FirewallResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Firewall Name", Value: fw.FirewallName()},
		{Label: "ARN", Value: fw.GetARN()},
	}

	if status := fw.StatusValue(); status != "" {
		fields = append(fields, render.SummaryField{Label: "Status", Value: status})
	}

	if vpcId := fw.VpcId(); vpcId != "" {
		fields = append(fields, render.SummaryField{Label: "VPC", Value: vpcId})
	}

	return fields
}
