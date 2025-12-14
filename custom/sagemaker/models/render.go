package models

import (
	"fmt"
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// ModelRenderer renders SageMaker models.
type ModelRenderer struct {
	render.BaseRenderer
}

// NewModelRenderer creates a new ModelRenderer.
func NewModelRenderer() render.Renderer {
	return &ModelRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "sagemaker",
			Resource: "models",
			Cols: []render.Column{
				{Name: "NAME", Width: 50, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "AGE", Width: 12, Getter: getAge},
			},
		},
	}
}

func getAge(r dao.Resource) string {
	model, ok := r.(*ModelResource)
	if !ok {
		return ""
	}
	if t := model.CreatedAt(); t != nil {
		return render.FormatAge(*t)
	}
	return ""
}

// RenderDetail renders the detail view for a model.
func (r *ModelRenderer) RenderDetail(resource dao.Resource) string {
	model, ok := resource.(*ModelResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("SageMaker Model", model.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", model.GetID())
	d.Field("ARN", model.GetARN())
	if model.GetEnableNetworkIsolation() {
		d.Field("Network Isolation", "Enabled")
	}

	// Primary Container
	if model.GetPrimaryContainerImage() != "" || model.GetPrimaryContainerModel() != "" {
		d.Section("Primary Container")
		if model.GetPrimaryContainerImage() != "" {
			d.Field("Image", model.GetPrimaryContainerImage())
		}
		if model.GetPrimaryContainerModel() != "" {
			d.Field("Model Data", model.GetPrimaryContainerModel())
		}
		if model.GetPrimaryContainerMode() != "" {
			d.Field("Mode", model.GetPrimaryContainerMode())
		}
	}

	// Multi-container info
	if model.GetContainerCount() > 1 {
		d.Section("Containers")
		d.Field("Container Count", fmt.Sprintf("%d", model.GetContainerCount()))
	}

	// VPC Config
	if vpc := model.GetVpcConfig(); vpc != nil {
		d.Section("VPC Configuration")
		if len(vpc.Subnets) > 0 {
			d.Field("Subnets", strings.Join(vpc.Subnets, ", "))
		}
		if len(vpc.SecurityGroupIds) > 0 {
			d.Field("Security Groups", strings.Join(vpc.SecurityGroupIds, ", "))
		}
	}

	// IAM
	if model.GetExecutionRoleArn() != "" {
		d.Section("IAM")
		d.Field("Execution Role", model.GetExecutionRoleArn())
	}

	// Timestamps
	d.Section("Timestamps")
	if t := model.CreatedAt(); t != nil {
		d.Field("Created", t.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary renders summary fields for a model.
func (r *ModelRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	model, ok := resource.(*ModelResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Name", Value: model.GetID()},
	}
}
