package loadbalancers

import (
	"fmt"
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// LoadBalancerRenderer renders ELBv2 Load Balancers
// Ensure LoadBalancerRenderer implements render.Navigator
var _ render.Navigator = (*LoadBalancerRenderer)(nil)

type LoadBalancerRenderer struct {
	render.BaseRenderer
}

// NewLoadBalancerRenderer creates a new LoadBalancerRenderer
func NewLoadBalancerRenderer() render.Renderer {
	return &LoadBalancerRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "elbv2",
			Resource: "load-balancers",
			Cols: []render.Column{
				{
					Name:  "NAME",
					Width: 32,
					Getter: func(r dao.Resource) string {
						return r.GetName()
					},
					Priority: 0,
				},
				{
					Name:  "TYPE",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*LoadBalancerResource); ok {
							return rr.Type()
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "SCHEME",
					Width: 16,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*LoadBalancerResource); ok {
							return rr.Scheme()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "STATE",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*LoadBalancerResource); ok {
							return rr.State()
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "DNS NAME",
					Width: 50,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*LoadBalancerResource); ok {
							return rr.DNSName()
						}
						return ""
					},
					Priority: 4,
				},
				{
					Name:  "AZS",
					Width: 20,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*LoadBalancerResource); ok {
							zones := rr.AvailabilityZones()
							if len(zones) > 0 {
								return fmt.Sprintf("%d zones", len(zones))
							}
						}
						return ""
					},
					Priority: 5,
				},
				{
					Name:  "CREATED",
					Width: 20,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*LoadBalancerResource); ok {
							t := rr.CreatedTime()
							if !t.IsZero() {
								return t.Format("2006-01-02 15:04")
							}
						}
						return ""
					},
					Priority: 6,
				},
			},
		},
	}
}

// RenderDetail renders detailed load balancer information
func (r *LoadBalancerRenderer) RenderDetail(resource dao.Resource) string {
	rr, ok := resource.(*LoadBalancerResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Load Balancer", rr.LoadBalancerName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", rr.LoadBalancerName())
	d.Field("ARN", rr.LoadBalancerArn())
	d.Field("Type", rr.Type())
	d.Field("Scheme", rr.Scheme())
	d.Field("State", rr.State())
	if rr.StateReason() != "" {
		d.Field("State Reason", rr.StateReason())
	}
	d.Field("IP Address Type", rr.IpAddressType())
	d.Field("Created", rr.CreatedTime().Format("2006-01-02 15:04:05 MST"))

	// Network
	d.Section("Network")
	d.Field("DNS Name", rr.DNSName())
	d.Field("VPC ID", rr.VpcId())
	d.Field("Hosted Zone ID", rr.CanonicalHostedZoneId())

	// Availability Zones
	zones := rr.AvailabilityZones()
	if len(zones) > 0 {
		d.Field("Availability Zones", strings.Join(zones, ", "))
	}

	// Security Groups
	sgs := rr.SecurityGroups()
	if len(sgs) > 0 {
		d.Section("Security Groups")
		for _, sg := range sgs {
			d.Field("Security Group", sg)
		}
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *LoadBalancerRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	rr, ok := resource.(*LoadBalancerResource)
	if !ok {
		return nil
	}

	return []render.SummaryField{
		{Label: "Name", Value: rr.LoadBalancerName()},
		{Label: "Type", Value: rr.Type()},
		{Label: "Scheme", Value: rr.Scheme()},
		{Label: "State", Value: rr.State()},
	}
}

// Navigations returns available navigation options
func (r *LoadBalancerRenderer) Navigations(resource dao.Resource) []render.Navigation {
	rr, ok := resource.(*LoadBalancerResource)
	if !ok {
		return nil
	}

	navs := []render.Navigation{
		{
			Key:         "t",
			Label:       "Target Groups",
			Service:     "elbv2",
			Resource:    "target-groups",
			FilterField: "LoadBalancerArn",
			FilterValue: rr.LoadBalancerArn(),
		},
	}

	// VPC navigation
	if vpcId := rr.VpcId(); vpcId != "" {
		navs = append(navs, render.Navigation{
			Key:         "v",
			Label:       "VPC",
			Service:     "vpc",
			Resource:    "vpcs",
			FilterField: "VpcId",
			FilterValue: vpcId,
		})
	}

	return navs
}
