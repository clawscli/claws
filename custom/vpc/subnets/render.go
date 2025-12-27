package subnets

import (
	"fmt"

	"charm.land/lipgloss/v2"
	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// Ensure SubnetRenderer implements render.Navigator
var _ render.Navigator = (*SubnetRenderer)(nil)

// SubnetRenderer renders Subnets
type SubnetRenderer struct {
	render.BaseRenderer
}

// NewSubnetRenderer creates a new SubnetRenderer
func NewSubnetRenderer() render.Renderer {
	return &SubnetRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "vpc",
			Resource: "subnets",
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
					Name:  "SUBNET ID",
					Width: 26,
					Getter: func(r dao.Resource) string {
						return r.GetID()
					},
					Priority: 1,
				},
				{
					Name:  "STATE",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if sr, ok := r.(*SubnetResource); ok {
							return sr.State()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "CIDR",
					Width: 18,
					Getter: func(r dao.Resource) string {
						if sr, ok := r.(*SubnetResource); ok {
							return sr.CidrBlock()
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "AZ",
					Width: 14,
					Getter: func(r dao.Resource) string {
						if sr, ok := r.(*SubnetResource); ok {
							return sr.AvailabilityZone()
						}
						return ""
					},
					Priority: 4,
				},
				{
					Name:  "TYPE",
					Width: 7,
					Getter: func(r dao.Resource) string {
						if sr, ok := r.(*SubnetResource); ok {
							if sr.IsPublic() {
								return "Public"
							}
							return "Private"
						}
						return ""
					},
					Priority: 5,
				},
				{
					Name:  "IPs",
					Width: 6,
					Getter: func(r dao.Resource) string {
						if sr, ok := r.(*SubnetResource); ok {
							return fmt.Sprintf("%d", sr.AvailableIpAddressCount())
						}
						return ""
					},
					Priority: 6,
				},
				render.TagsColumn(30, 7),
			},
		},
	}
}

// RenderDetail renders detailed subnet information
func (r *SubnetRenderer) RenderDetail(resource dao.Resource) string {
	sr, ok := resource.(*SubnetResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Subnet", sr.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Subnet ID", sr.GetID())
	d.FieldStyled("State", sr.State(), render.StateColorer()(sr.State()))
	d.Field("VPC ID", sr.VpcId())
	d.Field("CIDR Block", sr.CidrBlock())
	d.Field("Availability Zone", sr.AvailabilityZone())

	// Public/Private subnet indicator
	if sr.IsPublic() {
		d.FieldStyled("Subnet Type", "Public", lipgloss.NewStyle().Foreground(lipgloss.Color("42")))
	} else {
		d.FieldStyled("Subnet Type", "Private", lipgloss.NewStyle().Foreground(lipgloss.Color("33")))
	}

	if sr.Item.AvailabilityZoneId != nil {
		d.Field("AZ ID", *sr.Item.AvailabilityZoneId)
	}

	// IP Address Info
	d.Section("IP Address Settings")
	d.Field("Available IPs", fmt.Sprintf("%d", sr.AvailableIpAddressCount()))
	d.Field("Auto-assign Public IP", fmt.Sprintf("%v", sr.MapPublicIpOnLaunch()))

	if sr.Item.AssignIpv6AddressOnCreation != nil {
		d.Field("Auto-assign IPv6", fmt.Sprintf("%v", *sr.Item.AssignIpv6AddressOnCreation))
	}

	// IPv6 CIDR Blocks
	if len(sr.Item.Ipv6CidrBlockAssociationSet) > 0 {
		d.Section("IPv6 CIDR Blocks")
		for _, assoc := range sr.Item.Ipv6CidrBlockAssociationSet {
			if assoc.Ipv6CidrBlock != nil {
				state := ""
				if assoc.Ipv6CidrBlockState != nil {
					state = string(assoc.Ipv6CidrBlockState.State)
				}
				d.Field(*assoc.Ipv6CidrBlock, state)
			}
		}
	}

	// Owner
	if sr.Item.OwnerId != nil {
		d.Section("Owner")
		d.Field("Owner ID", *sr.Item.OwnerId)
	}

	// Tags
	d.Tags(appaws.TagsToMap(sr.Item.Tags))

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *SubnetRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	sr, ok := resource.(*SubnetResource)
	if !ok {
		return nil
	}

	stateStyle := render.StateColorer()(sr.State())

	fields := []render.SummaryField{
		{Label: "Subnet ID", Value: sr.GetID()},
		{Label: "Name", Value: sr.GetName()},
		{Label: "State", Value: sr.State(), Style: stateStyle},
		{Label: "VPC ID", Value: sr.VpcId()},
		{Label: "CIDR", Value: sr.CidrBlock()},
		{Label: "AZ", Value: sr.AvailabilityZone()},
		{Label: "Available IPs", Value: fmt.Sprintf("%d", sr.AvailableIpAddressCount())},
	}

	return fields
}

// Navigations returns navigation shortcuts for Subnet resources
func (r *SubnetRenderer) Navigations(resource dao.Resource) []render.Navigation {
	sr, ok := resource.(*SubnetResource)
	if !ok {
		return nil
	}

	subnetId := sr.GetID()
	vpcId := sr.VpcId()

	return []render.Navigation{
		{Key: "v", Label: "VPC", Service: "vpc", Resource: "vpcs", FilterField: "VpcId", FilterValue: vpcId},
		{Key: "e", Label: "Instances", Service: "ec2", Resource: "instances", FilterField: "SubnetId", FilterValue: subnetId},
	}
}
