package targets

import (
	"fmt"
	"strings"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// TargetRenderer renders ELBv2 Targets
// Ensure TargetRenderer implements render.Navigator
var _ render.Navigator = (*TargetRenderer)(nil)

type TargetRenderer struct {
	render.BaseRenderer
}

// NewTargetRenderer creates a new TargetRenderer
func NewTargetRenderer() render.Renderer {
	return &TargetRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "elbv2",
			Resource: "targets",
			Cols: []render.Column{
				{
					Name:  "TARGET",
					Width: 24,
					Getter: func(r dao.Resource) string {
						return r.GetName()
					},
					Priority: 0,
				},
				{
					Name:  "PORT",
					Width: 8,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*TargetResource); ok {
							return fmt.Sprintf("%d", rr.Port())
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "AZ",
					Width: 14,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*TargetResource); ok {
							az := rr.AvailabilityZone()
							if az == "" || az == "all" {
								return "all"
							}
							return az
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "HEALTH",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*TargetResource); ok {
							return rr.HealthState()
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "REASON",
					Width: 20,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*TargetResource); ok {
							return rr.HealthReason()
						}
						return ""
					},
					Priority: 4,
				},
				{
					Name:  "HC PORT",
					Width: 10,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*TargetResource); ok {
							return rr.HealthCheckPort()
						}
						return ""
					},
					Priority: 5,
				},
			},
		},
	}
}

// RenderDetail renders detailed target information
func (r *TargetRenderer) RenderDetail(resource dao.Resource) string {
	rr, ok := resource.(*TargetResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Target", rr.TargetId())

	// Target Info
	d.Section("Target Information")
	d.Field("Target ID", rr.TargetId())
	d.Field("Port", fmt.Sprintf("%d", rr.Port()))
	if rr.AvailabilityZone() != "" {
		d.Field("Availability Zone", rr.AvailabilityZone())
	}

	// Health Status
	d.Section("Health Status")
	d.Field("State", rr.HealthState())
	if rr.HealthReason() != "" {
		d.Field("Reason", rr.HealthReason())
	}
	if rr.HealthDescription() != "" {
		d.Field("Description", rr.HealthDescription())
	}
	d.Field("Health Check Port", rr.HealthCheckPort())

	// Parent Target Group
	d.Section("Target Group")
	d.Field("Target Group ARN", rr.TargetGroupArn)

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *TargetRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	rr, ok := resource.(*TargetResource)
	if !ok {
		return nil
	}

	return []render.SummaryField{
		{Label: "Target", Value: rr.TargetId()},
		{Label: "Port", Value: fmt.Sprintf("%d", rr.Port())},
		{Label: "Health", Value: rr.HealthState()},
		{Label: "Reason", Value: rr.HealthReason()},
	}
}

// Navigations returns available navigation options based on target type
func (r *TargetRenderer) Navigations(resource dao.Resource) []render.Navigation {
	rr, ok := resource.(*TargetResource)
	if !ok {
		return nil
	}

	targetId := rr.TargetId()
	var navs []render.Navigation

	// Target Group navigation (always available)
	navs = append(navs, render.Navigation{
		Key:         "g",
		Label:       "Target Group",
		Service:     "elbv2",
		Resource:    "target-groups",
		FilterField: "TargetGroupArn",
		FilterValue: rr.TargetGroupArn,
	})

	// EC2 instance target (i-xxxxx)
	if strings.HasPrefix(targetId, "i-") {
		navs = append(navs, render.Navigation{
			Key:         "e",
			Label:       "EC2 Instance",
			Service:     "ec2",
			Resource:    "instances",
			FilterField: "InstanceId",
			FilterValue: targetId,
		})
	}

	// Lambda function target (arn:aws:lambda:...)
	if strings.HasPrefix(targetId, "arn:aws:lambda:") {
		functionName := appaws.ExtractResourceName(targetId)
		navs = append(navs, render.Navigation{
			Key:         "l",
			Label:       "Lambda Function",
			Service:     "lambda",
			Resource:    "functions",
			FilterField: "FunctionName",
			FilterValue: functionName,
		})
	}

	return navs
}
