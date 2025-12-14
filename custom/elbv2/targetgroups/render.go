package targetgroups

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// TargetGroupRenderer renders ELBv2 Target Groups
// Ensure TargetGroupRenderer implements render.Navigator
var _ render.Navigator = (*TargetGroupRenderer)(nil)

type TargetGroupRenderer struct {
	render.BaseRenderer
}

// NewTargetGroupRenderer creates a new TargetGroupRenderer
func NewTargetGroupRenderer() render.Renderer {
	return &TargetGroupRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "elbv2",
			Resource: "target-groups",
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
					Name:  "PROTOCOL:PORT",
					Width: 14,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*TargetGroupResource); ok {
							return rr.ProtocolPort()
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "TARGET TYPE",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*TargetGroupResource); ok {
							return rr.TargetType()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "VPC",
					Width: 24,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*TargetGroupResource); ok {
							return rr.VpcId()
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "HEALTH CHECK",
					Width: 20,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*TargetGroupResource); ok {
							if rr.HealthCheckEnabled() {
								return fmt.Sprintf("%s:%s", rr.HealthCheckProtocol(), rr.HealthCheckPort())
							}
							return "Disabled"
						}
						return ""
					},
					Priority: 4,
				},
				{
					Name:  "LBS",
					Width: 6,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*TargetGroupResource); ok {
							return fmt.Sprintf("%d", len(rr.LoadBalancerArns()))
						}
						return ""
					},
					Priority: 5,
				},
			},
		},
	}
}

// RenderDetail renders detailed target group information
func (r *TargetGroupRenderer) RenderDetail(resource dao.Resource) string {
	rr, ok := resource.(*TargetGroupResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Target Group", rr.TargetGroupName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", rr.TargetGroupName())
	d.Field("ARN", rr.TargetGroupArn())
	d.Field("Protocol", rr.Protocol())
	if rr.Port() > 0 {
		d.Field("Port", fmt.Sprintf("%d", rr.Port()))
	}
	if rr.ProtocolVersion() != "" {
		d.Field("Protocol Version", rr.ProtocolVersion())
	}
	d.Field("Target Type", rr.TargetType())
	d.Field("IP Address Type", rr.IpAddressType())
	d.Field("VPC ID", rr.VpcId())

	// Health Check
	d.Section("Health Check")
	d.Field("Enabled", fmt.Sprintf("%v", rr.HealthCheckEnabled()))
	if rr.HealthCheckEnabled() {
		d.Field("Protocol", rr.HealthCheckProtocol())
		d.Field("Port", rr.HealthCheckPort())
		if rr.HealthCheckPath() != "" {
			d.Field("Path", rr.HealthCheckPath())
		}
		d.Field("Interval", fmt.Sprintf("%d seconds", rr.HealthCheckIntervalSeconds()))
		d.Field("Timeout", fmt.Sprintf("%d seconds", rr.HealthCheckTimeoutSeconds()))
		d.Field("Healthy Threshold", fmt.Sprintf("%d", rr.HealthyThresholdCount()))
		d.Field("Unhealthy Threshold", fmt.Sprintf("%d", rr.UnhealthyThresholdCount()))
		if rr.Matcher() != "" {
			d.Field("Success Codes", rr.Matcher())
		}
	}

	// Load Balancers
	lbArns := rr.LoadBalancerArns()
	if len(lbArns) > 0 {
		d.Section("Associated Load Balancers")
		for i, arn := range lbArns {
			d.Field(fmt.Sprintf("LB %d", i+1), arn)
		}
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *TargetGroupRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	rr, ok := resource.(*TargetGroupResource)
	if !ok {
		return nil
	}

	healthCheck := "Disabled"
	if rr.HealthCheckEnabled() {
		healthCheck = fmt.Sprintf("%s:%s", rr.HealthCheckProtocol(), rr.HealthCheckPort())
	}

	return []render.SummaryField{
		{Label: "Name", Value: rr.TargetGroupName()},
		{Label: "Protocol:Port", Value: rr.ProtocolPort()},
		{Label: "Target Type", Value: rr.TargetType()},
		{Label: "Health Check", Value: healthCheck},
	}
}

// Navigations returns available navigation options
func (r *TargetGroupRenderer) Navigations(resource dao.Resource) []render.Navigation {
	rr, ok := resource.(*TargetGroupResource)
	if !ok {
		return nil
	}

	navs := []render.Navigation{
		{
			Key:         "t",
			Label:       "Targets",
			Service:     "elbv2",
			Resource:    "targets",
			FilterField: "TargetGroupArn",
			FilterValue: rr.TargetGroupArn(),
		},
	}

	// Load Balancer navigation (if associated)
	lbArns := rr.LoadBalancerArns()
	if len(lbArns) > 0 {
		// Navigate to the first associated LB
		navs = append(navs, render.Navigation{
			Key:         "l",
			Label:       "Load Balancer",
			Service:     "elbv2",
			Resource:    "load-balancers",
			FilterField: "LoadBalancerArn",
			FilterValue: lbArns[0],
		})
	}

	return navs
}
