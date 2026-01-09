package fargateprofiles

import (
	"encoding/json"
	"fmt"
	"strings"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// FargateProfileRenderer renders EKS Fargate profile resources
type FargateProfileRenderer struct {
	render.BaseRenderer
}

// NewFargateProfileRenderer creates a new FargateProfileRenderer
func NewFargateProfileRenderer() render.Renderer {
	return &FargateProfileRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "eks",
			Resource: "fargate-profiles",
			Cols: []render.Column{
				{
					Name:     "NAME",
					Width:    30,
					Priority: 0,
					Getter: func(r dao.Resource) string {
						return r.GetName()
					},
				},
				{
					Name:     "STATUS",
					Width:    12,
					Priority: 1,
					Getter: func(r dao.Resource) string {
						if fpr, ok := r.(*FargateProfileResource); ok {
							return fpr.Status()
						}
						return ""
					},
				},
				{
					Name:     "POD EXECUTION ROLE",
					Width:    50,
					Priority: 2,
					Getter: func(r dao.Resource) string {
						if fpr, ok := r.(*FargateProfileResource); ok {
							return appaws.Str(fpr.FargateProfile.PodExecutionRoleArn)
						}
						return ""
					},
				},
				{
					Name:     "AGE",
					Width:    10,
					Priority: 3,
					Getter: func(r dao.Resource) string {
						if fpr, ok := r.(*FargateProfileResource); ok {
							return fpr.CreatedAge()
						}
						return ""
					},
				},
			},
		},
	}
}

func (rnd *FargateProfileRenderer) RenderDetail(resource dao.Resource) string {
	fpr, ok := resource.(*FargateProfileResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()
	d.Title("EKS Fargate Profile", fpr.GetName())

	// Basic Information
	d.Section("Basic Information")
	d.Field("Name", fpr.GetName())
	d.Field("ARN", fpr.GetARN())
	d.Field("Cluster", appaws.Str(fpr.FargateProfile.ClusterName))
	d.Field("Status", fpr.Status())
	d.Field("Created", fpr.CreatedAge())
	d.Field("Pod Execution Role", appaws.Str(fpr.FargateProfile.PodExecutionRoleArn))

	// Selectors
	if len(fpr.FargateProfile.Selectors) > 0 {
		d.Section("Selectors")
		for i, sel := range fpr.FargateProfile.Selectors {
			d.Field(fmt.Sprintf("Selector #%d Namespace", i+1), appaws.Str(sel.Namespace))
			if len(sel.Labels) > 0 {
				d.Field(fmt.Sprintf("Selector #%d Labels", i+1), "")
				d.Tags(sel.Labels)
			}
		}
	}

	// Subnets
	if len(fpr.FargateProfile.Subnets) > 0 {
		d.Section("Subnets")
		d.Field("Subnet IDs", strings.Join(fpr.FargateProfile.Subnets, ", "))
	}

	// Tags
	if len(fpr.FargateProfile.Tags) > 0 {
		d.Section("Tags")
		d.Tags(fpr.FargateProfile.Tags)
	}

	// Full Details
	d.Section("Full Details")
	if jsonBytes, err := json.MarshalIndent(fpr.FargateProfile, "", "  "); err == nil {
		d.Line(string(jsonBytes))
	}

	return d.String()
}

func (rnd *FargateProfileRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	fpr, ok := resource.(*FargateProfileResource)
	if !ok {
		return nil
	}

	return []render.SummaryField{
		{Label: "Name", Value: fpr.GetName()},
		{Label: "Status", Value: fpr.Status()},
		{Label: "Created", Value: fpr.CreatedAge()},
	}
}
