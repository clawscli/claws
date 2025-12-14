package natgateways

import (
	"time"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// Ensure NatGatewayRenderer implements render.Navigator
var _ render.Navigator = (*NatGatewayRenderer)(nil)

// NatGatewayRenderer renders NAT Gateways
type NatGatewayRenderer struct {
	render.BaseRenderer
}

// NewNatGatewayRenderer creates a new NatGatewayRenderer
func NewNatGatewayRenderer() render.Renderer {
	return &NatGatewayRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "vpc",
			Resource: "nat-gateways",
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
					Name:  "NAT GW ID",
					Width: 24,
					Getter: func(r dao.Resource) string {
						return r.GetID()
					},
					Priority: 1,
				},
				{
					Name:  "STATE",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if ngwr, ok := r.(*NatGatewayResource); ok {
							return ngwr.State()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "TYPE",
					Width: 10,
					Getter: func(r dao.Resource) string {
						if ngwr, ok := r.(*NatGatewayResource); ok {
							return ngwr.ConnectivityType()
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "SUBNET",
					Width: 26,
					Getter: func(r dao.Resource) string {
						if ngwr, ok := r.(*NatGatewayResource); ok {
							return ngwr.SubnetId()
						}
						return ""
					},
					Priority: 4,
				},
				{
					Name:  "PUBLIC IP",
					Width: 16,
					Getter: func(r dao.Resource) string {
						if ngwr, ok := r.(*NatGatewayResource); ok {
							return ngwr.PublicIp()
						}
						return ""
					},
					Priority: 5,
				},
				{
					Name:  "PRIVATE IP",
					Width: 16,
					Getter: func(r dao.Resource) string {
						if ngwr, ok := r.(*NatGatewayResource); ok {
							return ngwr.PrivateIp()
						}
						return ""
					},
					Priority: 6,
				},
			},
		},
	}
}

// RenderDetail renders detailed NAT gateway information
func (r *NatGatewayRenderer) RenderDetail(resource dao.Resource) string {
	ngwr, ok := resource.(*NatGatewayResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()
	styles := d.Styles()

	d.Title("NAT Gateway", ngwr.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("NAT Gateway ID", ngwr.GetID())
	d.FieldStyled("State", ngwr.State(), render.StateColorer()(ngwr.State()))
	d.Field("Connectivity Type", ngwr.ConnectivityType())
	d.Field("VPC ID", ngwr.VpcId())
	d.Field("Subnet ID", ngwr.SubnetId())

	if ngwr.Item.CreateTime != nil {
		d.Field("Created", ngwr.Item.CreateTime.Format(time.RFC3339))
	}

	// Network Addresses
	if len(ngwr.Item.NatGatewayAddresses) > 0 {
		d.Section("Network Addresses")
		for _, addr := range ngwr.Item.NatGatewayAddresses {
			if addr.PublicIp != nil {
				d.Field("Public IP", *addr.PublicIp)
			}
			if addr.PrivateIp != nil {
				d.Field("Private IP", *addr.PrivateIp)
			}
			if addr.AllocationId != nil {
				d.Field("Allocation ID", *addr.AllocationId)
			}
			if addr.NetworkInterfaceId != nil {
				d.Field("Network Interface", *addr.NetworkInterfaceId)
			}
		}
	}

	// Failure info
	if ngwr.Item.FailureCode != nil && *ngwr.Item.FailureCode != "" {
		d.Section("Failure Information")
		d.Line("  " + styles.Value.Render("Code: ") + *ngwr.Item.FailureCode)
		if ngwr.Item.FailureMessage != nil {
			d.Line("  " + styles.Value.Render("Message: ") + *ngwr.Item.FailureMessage)
		}
	}

	// Tags
	d.Tags(appaws.TagsToMap(ngwr.Item.Tags))

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *NatGatewayRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	ngwr, ok := resource.(*NatGatewayResource)
	if !ok {
		return nil
	}

	stateStyle := render.StateColorer()(ngwr.State())

	fields := []render.SummaryField{
		{Label: "NAT GW ID", Value: ngwr.GetID()},
		{Label: "Name", Value: ngwr.GetName()},
		{Label: "State", Value: ngwr.State(), Style: stateStyle},
		{Label: "Type", Value: ngwr.ConnectivityType()},
		{Label: "VPC ID", Value: ngwr.VpcId()},
		{Label: "Subnet ID", Value: ngwr.SubnetId()},
		{Label: "Public IP", Value: ngwr.PublicIp()},
		{Label: "Private IP", Value: ngwr.PrivateIp()},
	}

	return fields
}

// Navigations returns navigation shortcuts for NAT Gateway resources
func (r *NatGatewayRenderer) Navigations(resource dao.Resource) []render.Navigation {
	ngwr, ok := resource.(*NatGatewayResource)
	if !ok {
		return nil
	}

	vpcId := ngwr.VpcId()
	subnetId := ngwr.SubnetId()

	var navs []render.Navigation

	if vpcId != "" {
		navs = append(navs, render.Navigation{Key: "v", Label: "VPC", Service: "vpc", Resource: "vpcs", FilterField: "VpcId", FilterValue: vpcId})
	}
	if subnetId != "" {
		navs = append(navs, render.Navigation{Key: "u", Label: "Subnet", Service: "vpc", Resource: "subnets", FilterField: "SubnetId", FilterValue: subnetId})
	}

	return navs
}
