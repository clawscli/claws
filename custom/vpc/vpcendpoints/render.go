package vpcendpoints

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// VpcEndpointRenderer renders VPC endpoints.
type VpcEndpointRenderer struct {
	render.BaseRenderer
}

// NewVpcEndpointRenderer creates a new VpcEndpointRenderer.
func NewVpcEndpointRenderer() render.Renderer {
	return &VpcEndpointRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "vpc",
			Resource: "endpoints",
			Cols: []render.Column{
				{Name: "ENDPOINT ID", Width: 26, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "NAME", Width: 25, Getter: getName},
				{Name: "SERVICE", Width: 40, Getter: getService},
				{Name: "TYPE", Width: 12, Getter: getType},
				{Name: "STATE", Width: 12, Getter: getState},
				{Name: "VPC", Width: 22, Getter: getVpc},
			},
		},
	}
}

func getName(r dao.Resource) string {
	endpoint, ok := r.(*VpcEndpointResource)
	if !ok {
		return ""
	}
	return endpoint.Name()
}

func getService(r dao.Resource) string {
	endpoint, ok := r.(*VpcEndpointResource)
	if !ok {
		return ""
	}
	service := endpoint.ServiceName()
	// Truncate long service names
	if len(service) > 37 {
		return service[:37] + "..."
	}
	return service
}

func getType(r dao.Resource) string {
	endpoint, ok := r.(*VpcEndpointResource)
	if !ok {
		return ""
	}
	return endpoint.VpcEndpointType()
}

func getState(r dao.Resource) string {
	endpoint, ok := r.(*VpcEndpointResource)
	if !ok {
		return ""
	}
	return endpoint.State()
}

func getVpc(r dao.Resource) string {
	endpoint, ok := r.(*VpcEndpointResource)
	if !ok {
		return ""
	}
	return endpoint.VpcId()
}

// RenderDetail renders the detail view for a VPC endpoint.
func (r *VpcEndpointRenderer) RenderDetail(resource dao.Resource) string {
	endpoint, ok := resource.(*VpcEndpointResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	title := endpoint.GetID()
	if name := endpoint.Name(); name != "" {
		title = name
	}
	d.Title("VPC Endpoint", title)

	// Basic Info
	d.Section("Basic Information")
	d.Field("Endpoint ID", endpoint.GetID())
	if name := endpoint.Name(); name != "" {
		d.Field("Name", name)
	}
	d.Field("VPC ID", endpoint.VpcId())
	d.Field("State", endpoint.State())
	d.Field("Owner ID", endpoint.OwnerId())
	if endpoint.RequesterManaged() {
		d.Field("Requester Managed", "Yes")
	}

	// Service Configuration
	d.Section("Service Configuration")
	d.Field("Service Name", endpoint.ServiceName())
	d.Field("Endpoint Type", endpoint.VpcEndpointType())
	if ipType := endpoint.IpAddressType(); ipType != "" {
		d.Field("IP Address Type", ipType)
	}
	if endpoint.PrivateDnsEnabled() {
		d.Field("Private DNS", "Enabled")
	} else {
		d.Field("Private DNS", "Disabled")
	}

	// DNS Entries
	if dnsEntries := endpoint.DnsEntries(); len(dnsEntries) > 0 {
		d.Section("DNS Entries")
		for i, entry := range dnsEntries {
			if i >= 5 {
				d.Field("", fmt.Sprintf("... and %d more", len(dnsEntries)-5))
				break
			}
			d.Field(fmt.Sprintf("DNS %d", i+1), entry)
		}
	}

	// Networking
	d.Section("Networking")
	if subnets := endpoint.SubnetIds(); len(subnets) > 0 {
		d.Field("Subnets", strings.Join(subnets, ", "))
	}
	if sgs := endpoint.SecurityGroupIds(); len(sgs) > 0 {
		d.Field("Security Groups", strings.Join(sgs, ", "))
	}
	if rts := endpoint.RouteTableIds(); len(rts) > 0 {
		d.Field("Route Tables", strings.Join(rts, ", "))
	}
	if enis := endpoint.NetworkInterfaceIds(); len(enis) > 0 {
		d.Field("Network Interfaces", strings.Join(enis, ", "))
	}

	// Tags
	if tags := endpoint.Tags(); len(tags) > 0 {
		d.Section("Tags")
		for k, v := range tags {
			if k != "Name" {
				d.Field(k, v)
			}
		}
	}

	// Timestamps
	if t := endpoint.CreationTimestamp(); t != nil {
		d.Section("Timestamps")
		d.Field("Created", t.Format("2006-01-02 15:04:05"))
	}

	// Policy (at bottom for readability)
	if policy := endpoint.PolicyDocument(); policy != "" {
		d.Section("Policy Document")
		d.Line(prettyJSON(policy))
	}

	return d.String()
}

// prettyJSON formats JSON string with indentation
func prettyJSON(s string) string {
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(s), "", "  "); err != nil {
		return s
	}
	return buf.String()
}

// RenderSummary renders summary fields for a VPC endpoint.
func (r *VpcEndpointRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	endpoint, ok := resource.(*VpcEndpointResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Endpoint ID", Value: endpoint.GetID()},
		{Label: "Service", Value: endpoint.ServiceName()},
		{Label: "Type", Value: endpoint.VpcEndpointType()},
		{Label: "State", Value: endpoint.State()},
		{Label: "VPC", Value: endpoint.VpcId()},
	}

	if name := endpoint.Name(); name != "" {
		fields = append([]render.SummaryField{{Label: "Name", Value: name}}, fields...)
	}

	if subnets := endpoint.SubnetIds(); len(subnets) > 0 {
		fields = append(fields, render.SummaryField{Label: "Subnets", Value: fmt.Sprintf("%d", len(subnets))})
	}

	return fields
}
