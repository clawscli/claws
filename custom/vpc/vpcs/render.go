package vpcs

import (
	"fmt"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
	"github.com/clawscli/claws/internal/ui"
)

// Ensure VPCRenderer implements render.Navigator
var _ render.Navigator = (*VPCRenderer)(nil)

// VPCRenderer renders VPCs
type VPCRenderer struct {
	render.BaseRenderer
}

// NewVPCRenderer creates a new VPCRenderer
func NewVPCRenderer() render.Renderer {
	return &VPCRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "vpc",
			Resource: "vpcs",
			Cols: []render.Column{
				{
					Name:  "NAME",
					Width: 25,
					Getter: func(r dao.Resource) string {
						return r.GetName()
					},
					Priority: 0,
				},
				{
					Name:  "VPC ID",
					Width: 22,
					Getter: func(r dao.Resource) string {
						return r.GetID()
					},
					Priority: 1,
				},
				{
					Name:  "STATE",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if vr, ok := r.(*VPCResource); ok {
							return vr.State()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "CIDR",
					Width: 18,
					Getter: func(r dao.Resource) string {
						if vr, ok := r.(*VPCResource); ok {
							return vr.CidrBlock()
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "DEFAULT",
					Width: 8,
					Getter: func(r dao.Resource) string {
						if vr, ok := r.(*VPCResource); ok {
							if vr.IsDefault() {
								return "Yes"
							}
							return "No"
						}
						return ""
					},
					Priority: 4,
				},
				{
					Name:  "TENANCY",
					Width: 10,
					Getter: func(r dao.Resource) string {
						if vr, ok := r.(*VPCResource); ok {
							return vr.Tenancy()
						}
						return ""
					},
					Priority: 5,
				},
				render.TagsColumn(30, 6),
			},
		},
	}
}

// RenderDetail renders detailed VPC information
func (r *VPCRenderer) RenderDetail(resource dao.Resource) string {
	vr, ok := resource.(*VPCResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("VPC", vr.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("VPC ID", vr.GetID())
	d.FieldStyled("State", vr.State(), render.StateColorer()(vr.State()))
	d.Field("CIDR Block", vr.CidrBlock())
	d.Field("Default VPC", fmt.Sprintf("%v", vr.IsDefault()))
	d.Field("Tenancy", vr.Tenancy())
	if vr.Item.OwnerId != nil {
		d.Field("Owner ID", *vr.Item.OwnerId)
	}

	// DNS Settings
	d.Section("DNS Settings")
	if vr.EnableDnsSupport {
		d.FieldStyled("DNS Resolution", "Enabled", ui.SuccessStyle())
	} else {
		d.Field("DNS Resolution", "Disabled")
	}
	if vr.EnableDnsHostnames {
		d.FieldStyled("DNS Hostnames", "Enabled", ui.SuccessStyle())
	} else {
		d.Field("DNS Hostnames", "Disabled")
	}

	// DHCP Options
	if vr.Item.DhcpOptionsId != nil {
		d.Section("DHCP")
		d.Field("DHCP Options ID", *vr.Item.DhcpOptionsId)
	}

	// Secondary IPv4 CIDR Blocks
	if len(vr.Item.CidrBlockAssociationSet) > 1 {
		d.Section("Additional IPv4 CIDR Blocks")
		for _, assoc := range vr.Item.CidrBlockAssociationSet {
			if assoc.CidrBlock != nil && *assoc.CidrBlock != vr.CidrBlock() {
				state := ""
				if assoc.CidrBlockState != nil {
					state = string(assoc.CidrBlockState.State)
				}
				d.Field(*assoc.CidrBlock, state)
			}
		}
	}

	// IPv6 CIDR Blocks
	if len(vr.Item.Ipv6CidrBlockAssociationSet) > 0 {
		d.Section("IPv6 CIDR Blocks")
		for _, assoc := range vr.Item.Ipv6CidrBlockAssociationSet {
			if assoc.Ipv6CidrBlock != nil {
				state := ""
				if assoc.Ipv6CidrBlockState != nil {
					state = string(assoc.Ipv6CidrBlockState.State)
				}
				d.Field(*assoc.Ipv6CidrBlock, state)
			}
		}
	}

	// Tags
	d.Tags(appaws.TagsToMap(vr.Item.Tags))

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *VPCRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	vr, ok := resource.(*VPCResource)
	if !ok {
		return nil
	}

	stateStyle := render.StateColorer()(vr.State())

	fields := []render.SummaryField{
		{Label: "VPC ID", Value: vr.GetID()},
		{Label: "Name", Value: vr.GetName()},
		{Label: "State", Value: vr.State(), Style: stateStyle},
		{Label: "CIDR", Value: vr.CidrBlock()},
		{Label: "Default", Value: fmt.Sprintf("%v", vr.IsDefault())},
		{Label: "Tenancy", Value: vr.Tenancy()},
	}

	if vr.Item.OwnerId != nil {
		fields = append(fields, render.SummaryField{Label: "Owner", Value: *vr.Item.OwnerId})
	}

	return fields
}

// Navigations returns navigation shortcuts for VPC resources
func (r *VPCRenderer) Navigations(resource dao.Resource) []render.Navigation {
	vr, ok := resource.(*VPCResource)
	if !ok {
		return nil
	}

	vpcId := vr.GetID()

	return []render.Navigation{
		{Key: "s", Label: "Subnets", Service: "vpc", Resource: "subnets", FilterField: "VpcId", FilterValue: vpcId},
		{Key: "t", Label: "Route Tables", Service: "vpc", Resource: "route-tables", FilterField: "VpcId", FilterValue: vpcId},
		{Key: "i", Label: "Internet GWs", Service: "vpc", Resource: "internet-gateways", FilterField: "VpcId", FilterValue: vpcId},
		{Key: "n", Label: "NAT GWs", Service: "vpc", Resource: "nat-gateways", FilterField: "VpcId", FilterValue: vpcId},
		{Key: "g", Label: "Security Groups", Service: "ec2", Resource: "security-groups", FilterField: "VpcId", FilterValue: vpcId},
		{Key: "e", Label: "Instances", Service: "ec2", Resource: "instances", FilterField: "VpcId", FilterValue: vpcId},
	}
}
