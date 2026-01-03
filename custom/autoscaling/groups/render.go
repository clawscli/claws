package groups

import (
	"fmt"
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
	"github.com/clawscli/claws/internal/ui"
)

// AutoScalingGroupRenderer renders Auto Scaling Groups
// Ensure AutoScalingGroupRenderer implements render.Navigator
var _ render.Navigator = (*AutoScalingGroupRenderer)(nil)

type AutoScalingGroupRenderer struct {
	render.BaseRenderer
}

// NewAutoScalingGroupRenderer creates a new AutoScalingGroupRenderer
func NewAutoScalingGroupRenderer() render.Renderer {
	return &AutoScalingGroupRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "autoscaling",
			Resource: "groups",
			Cols: []render.Column{
				{
					Name:  "NAME",
					Width: 40,
					Getter: func(r dao.Resource) string {
						return r.GetName()
					},
					Priority: 0,
				},
				{
					Name:  "INSTANCES",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*AutoScalingGroupResource); ok {
							return fmt.Sprintf("%d/%d", rr.InstanceCount(), rr.DesiredCapacity())
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "MIN/MAX",
					Width: 10,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*AutoScalingGroupResource); ok {
							return fmt.Sprintf("%d/%d", rr.MinSize(), rr.MaxSize())
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "HEALTH",
					Width: 8,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*AutoScalingGroupResource); ok {
							return rr.HealthCheckType()
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "LAUNCH CONFIG/TEMPLATE",
					Width: 30,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*AutoScalingGroupResource); ok {
							if lt := rr.LaunchTemplateName(); lt != "" {
								return lt
							}
							return rr.LaunchConfigurationName()
						}
						return ""
					},
					Priority: 4,
				},
				{
					Name:  "AZS",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*AutoScalingGroupResource); ok {
							return fmt.Sprintf("%d zones", len(rr.AvailabilityZones()))
						}
						return ""
					},
					Priority: 5,
				},
				{
					Name:  "CREATED",
					Width: 20,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*AutoScalingGroupResource); ok {
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

// RenderDetail renders detailed Auto Scaling Group information
func (r *AutoScalingGroupRenderer) RenderDetail(resource dao.Resource) string {
	rr, ok := resource.(*AutoScalingGroupResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Auto Scaling Group", rr.AutoScalingGroupName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", rr.AutoScalingGroupName())
	d.Field("ARN", rr.AutoScalingGroupARN())
	if rr.Status() != "" {
		d.Field("Status", rr.Status())
	}
	d.Field("Created", rr.CreatedTime().Format("2006-01-02 15:04:05 MST"))

	// Capacity
	d.Section("Capacity")
	d.Field("Desired Capacity", fmt.Sprintf("%d", rr.DesiredCapacity()))
	d.Field("Min Size", fmt.Sprintf("%d", rr.MinSize()))
	d.Field("Max Size", fmt.Sprintf("%d", rr.MaxSize()))
	d.Field("Running Instances", fmt.Sprintf("%d", rr.InstanceCount()))
	d.Field("Healthy Instances", fmt.Sprintf("%d", rr.HealthyInstanceCount()))

	// Launch Configuration/Template
	d.Section("Launch Configuration")
	if lt := rr.LaunchTemplateName(); lt != "" {
		d.Field("Launch Template", lt)
		d.Field("Template ID", rr.LaunchTemplateId())
		d.Field("Template Version", rr.LaunchTemplateVersion())
	} else if lc := rr.LaunchConfigurationName(); lc != "" {
		d.Field("Launch Configuration", lc)
	}

	// Health Check
	d.Section("Health Check")
	d.Field("Type", rr.HealthCheckType())
	d.Field("Grace Period", fmt.Sprintf("%d seconds", rr.HealthCheckGracePeriod()))

	// Network
	d.Section("Network")
	if len(rr.AvailabilityZones()) > 0 {
		d.Field("Availability Zones", strings.Join(rr.AvailabilityZones(), ", "))
	}
	if rr.VPCZoneIdentifier() != "" {
		d.Field("Subnets", rr.VPCZoneIdentifier())
	}

	// Load Balancers
	if len(rr.TargetGroupARNs()) > 0 {
		d.Section("Target Groups")
		for i, arn := range rr.TargetGroupARNs() {
			d.Field(fmt.Sprintf("TG %d", i+1), arn)
		}
	}
	if len(rr.LoadBalancerNames()) > 0 {
		d.Section("Classic Load Balancers")
		for _, name := range rr.LoadBalancerNames() {
			d.Field("CLB", name)
		}
	}

	// Scaling Settings
	d.Section("Scaling Settings")
	d.Field("Default Cooldown", fmt.Sprintf("%d seconds", rr.DefaultCooldown()))
	d.Field("New Instances Protected", fmt.Sprintf("%v", rr.NewInstancesProtectedFromScaleIn()))
	if len(rr.TerminationPolicies()) > 0 {
		d.Field("Termination Policies", strings.Join(rr.TerminationPolicies(), ", "))
	}

	// Suspended Processes
	if len(rr.Item.SuspendedProcesses) > 0 {
		d.Section("Suspended Processes")
		for _, proc := range rr.Item.SuspendedProcesses {
			reason := ""
			if proc.SuspensionReason != nil {
				reason = *proc.SuspensionReason
			}
			if proc.ProcessName != nil {
				if reason != "" {
					d.FieldStyled(*proc.ProcessName, reason, ui.WarningStyle())
				} else {
					d.FieldStyled(*proc.ProcessName, "Suspended", ui.WarningStyle())
				}
			}
		}
	}

	// Warm Pool
	if rr.Item.WarmPoolConfiguration != nil {
		wp := rr.Item.WarmPoolConfiguration
		d.Section("Warm Pool")
		if wp.PoolState != "" {
			d.Field("Pool State", string(wp.PoolState))
		}
		if wp.MinSize != nil {
			d.Field("Min Size", fmt.Sprintf("%d", *wp.MinSize))
		}
		if wp.MaxGroupPreparedCapacity != nil {
			d.Field("Max Prepared Capacity", fmt.Sprintf("%d", *wp.MaxGroupPreparedCapacity))
		}
		if wp.InstanceReusePolicy != nil && wp.InstanceReusePolicy.ReuseOnScaleIn != nil {
			d.Field("Reuse On Scale In", fmt.Sprintf("%v", *wp.InstanceReusePolicy.ReuseOnScaleIn))
		}
	}

	// Tags
	if len(rr.GetTags()) > 0 {
		d.Section("Tags")
		for k, v := range rr.GetTags() {
			d.Field(k, v)
		}
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *AutoScalingGroupRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	rr, ok := resource.(*AutoScalingGroupResource)
	if !ok {
		return nil
	}

	launchConfig := rr.LaunchTemplateName()
	if launchConfig == "" {
		launchConfig = rr.LaunchConfigurationName()
	}

	return []render.SummaryField{
		{Label: "Name", Value: rr.AutoScalingGroupName()},
		{Label: "Launch", Value: launchConfig},
		{Label: "Instances", Value: fmt.Sprintf("%d/%d", rr.InstanceCount(), rr.DesiredCapacity())},
		{Label: "Min/Max", Value: fmt.Sprintf("%d/%d", rr.MinSize(), rr.MaxSize())},
		{Label: "Health", Value: rr.HealthCheckType()},
	}
}

// Navigations returns navigation shortcuts
func (r *AutoScalingGroupRenderer) Navigations(resource dao.Resource) []render.Navigation {
	rr, ok := resource.(*AutoScalingGroupResource)
	if !ok {
		return nil
	}

	return []render.Navigation{
		{
			Key: "a", Label: "Activities", Service: "autoscaling", Resource: "activities",
			FilterField: "AutoScalingGroupName", FilterValue: rr.AutoScalingGroupName(),
		},
		{
			Key: "e", Label: "Instances", Service: "ec2", Resource: "instances",
			FilterField: "AutoScalingGroupName", FilterValue: rr.AutoScalingGroupName(),
		},
	}
}
