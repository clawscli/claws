package routetables

import (
	"fmt"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// Ensure RouteTableRenderer implements render.Navigator
var _ render.Navigator = (*RouteTableRenderer)(nil)

// RouteTableRenderer renders Route Tables
type RouteTableRenderer struct {
	render.BaseRenderer
}

// NewRouteTableRenderer creates a new RouteTableRenderer
func NewRouteTableRenderer() render.Renderer {
	return &RouteTableRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "vpc",
			Resource: "route-tables",
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
					Name:  "ROUTE TABLE ID",
					Width: 26,
					Getter: func(r dao.Resource) string {
						return r.GetID()
					},
					Priority: 1,
				},
				{
					Name:  "VPC ID",
					Width: 22,
					Getter: func(r dao.Resource) string {
						if rtr, ok := r.(*RouteTableResource); ok {
							return rtr.VpcId()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "MAIN",
					Width: 6,
					Getter: func(r dao.Resource) string {
						if rtr, ok := r.(*RouteTableResource); ok {
							if rtr.IsMain() {
								return "Yes"
							}
							return "No"
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "ROUTES",
					Width: 8,
					Getter: func(r dao.Resource) string {
						if rtr, ok := r.(*RouteTableResource); ok {
							return fmt.Sprintf("%d", rtr.RouteCount())
						}
						return ""
					},
					Priority: 4,
				},
				{
					Name:  "SUBNETS",
					Width: 8,
					Getter: func(r dao.Resource) string {
						if rtr, ok := r.(*RouteTableResource); ok {
							return fmt.Sprintf("%d", rtr.SubnetAssociationCount())
						}
						return ""
					},
					Priority: 5,
				},
			},
		},
	}
}

// RenderDetail renders detailed route table information
func (r *RouteTableRenderer) RenderDetail(resource dao.Resource) string {
	rtr, ok := resource.(*RouteTableResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()
	styles := d.Styles()

	d.Title("Route Table", rtr.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Route Table ID", rtr.GetID())
	d.Field("VPC ID", rtr.VpcId())
	d.Field("Main Route Table", fmt.Sprintf("%v", rtr.IsMain()))

	if rtr.Item.OwnerId != nil {
		d.Field("Owner ID", *rtr.Item.OwnerId)
	}

	// Routes
	if len(rtr.Item.Routes) > 0 {
		d.Section("Routes")
		for _, route := range rtr.Item.Routes {
			dest := appaws.Str(route.DestinationCidrBlock)
			if dest == "" {
				dest = appaws.Str(route.DestinationIpv6CidrBlock)
			}
			if dest == "" {
				dest = appaws.Str(route.DestinationPrefixListId)
			}

			target := appaws.Str(route.GatewayId)
			if target == "" {
				target = appaws.Str(route.NatGatewayId)
			}
			if target == "" {
				target = appaws.Str(route.NetworkInterfaceId)
			}
			if target == "" {
				target = appaws.Str(route.VpcPeeringConnectionId)
			}
			if target == "" {
				target = appaws.Str(route.TransitGatewayId)
			}
			if target == "" {
				target = appaws.Str(route.LocalGatewayId)
			}
			if target == "" {
				target = appaws.Str(route.InstanceId)
			}

			state := string(route.State)
			d.Line("  " + styles.Value.Render(dest) + " → " + styles.Dim.Render(target) + " (" + state + ")")
		}
	}

	// Subnet Associations
	subnetAssocs := []string{}
	for _, assoc := range rtr.Item.Associations {
		if assoc.SubnetId != nil {
			subnetAssocs = append(subnetAssocs, *assoc.SubnetId)
		}
	}
	if len(subnetAssocs) > 0 {
		d.Section("Subnet Associations")
		for _, subnet := range subnetAssocs {
			d.Line("  " + styles.Value.Render(subnet))
		}
	}

	// Tags
	d.Tags(appaws.TagsToMap(rtr.Item.Tags))

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *RouteTableRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	rtr, ok := resource.(*RouteTableResource)
	if !ok {
		return nil
	}

	fields := []render.SummaryField{
		{Label: "Route Table ID", Value: rtr.GetID()},
		{Label: "Name", Value: rtr.GetName()},
		{Label: "VPC ID", Value: rtr.VpcId()},
		{Label: "Main", Value: fmt.Sprintf("%v", rtr.IsMain())},
		{Label: "Routes", Value: fmt.Sprintf("%d", rtr.RouteCount())},
		{Label: "Subnets", Value: fmt.Sprintf("%d", rtr.SubnetAssociationCount())},
	}

	return fields
}

// Navigations returns navigation shortcuts for Route Table resources
func (r *RouteTableRenderer) Navigations(resource dao.Resource) []render.Navigation {
	rtr, ok := resource.(*RouteTableResource)
	if !ok {
		return nil
	}

	vpcId := rtr.VpcId()
	if vpcId == "" {
		return nil
	}

	return []render.Navigation{
		{Key: "v", Label: "VPC", Service: "vpc", Resource: "vpcs", FilterField: "VpcId", FilterValue: vpcId},
	}
}
