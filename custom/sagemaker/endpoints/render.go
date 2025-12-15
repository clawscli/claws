package endpoints

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// EndpointRenderer renders SageMaker endpoints.
type EndpointRenderer struct {
	render.BaseRenderer
}

// NewEndpointRenderer creates a new EndpointRenderer.
func NewEndpointRenderer() render.Renderer {
	return &EndpointRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "sagemaker",
			Resource: "endpoints",
			Cols: []render.Column{
				{Name: "NAME", Width: 40, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "STATUS", Width: 15, Getter: getStatus},
				{Name: "AGE", Width: 12, Getter: getAge},
			},
		},
	}
}

func getStatus(r dao.Resource) string {
	endpoint, ok := r.(*EndpointResource)
	if !ok {
		return ""
	}
	return endpoint.Status()
}

func getAge(r dao.Resource) string {
	endpoint, ok := r.(*EndpointResource)
	if !ok {
		return ""
	}
	if t := endpoint.CreatedAt(); t != nil {
		return render.FormatAge(*t)
	}
	return ""
}

// RenderDetail renders the detail view for an endpoint.
func (r *EndpointRenderer) RenderDetail(resource dao.Resource) string {
	endpoint, ok := resource.(*EndpointResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("SageMaker Endpoint", endpoint.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", endpoint.GetID())
	d.Field("ARN", endpoint.GetARN())
	d.Field("Status", endpoint.Status())
	if endpoint.GetEndpointConfigName() != "" {
		d.Field("Endpoint Config", endpoint.GetEndpointConfigName())
	}
	if endpoint.GetFailureReason() != "" {
		d.Field("Failure Reason", endpoint.GetFailureReason())
	}

	// Production Variants
	if variants := endpoint.GetProductionVariants(); len(variants) > 0 {
		d.Section("Production Variants")
		for _, v := range variants {
			variantName := ""
			if v.VariantName != nil {
				variantName = *v.VariantName
			}
			d.Field("Variant", variantName)
			if v.CurrentInstanceCount != nil {
				d.Field("  Instance Count", fmt.Sprintf("%d", *v.CurrentInstanceCount))
			}
			if v.DesiredInstanceCount != nil && v.CurrentInstanceCount != nil && *v.DesiredInstanceCount != *v.CurrentInstanceCount {
				d.Field("  Desired Count", fmt.Sprintf("%d", *v.DesiredInstanceCount))
			}
			if v.CurrentWeight != nil {
				d.Field("  Weight", fmt.Sprintf("%.2f", *v.CurrentWeight))
			}
			if len(v.DeployedImages) > 0 && v.DeployedImages[0].SpecifiedImage != nil {
				d.Field("  Image", *v.DeployedImages[0].SpecifiedImage)
			}
		}
	}

	// Data Capture Config
	if dc := endpoint.GetDataCaptureConfig(); dc != nil {
		d.Section("Data Capture")
		d.Field("Enabled", fmt.Sprintf("%v", dc.EnableCapture))
		if dc.CurrentSamplingPercentage != nil {
			d.Field("Sampling %", fmt.Sprintf("%d%%", *dc.CurrentSamplingPercentage))
		}
		if dc.DestinationS3Uri != nil {
			d.Field("Destination", *dc.DestinationS3Uri)
		}
		d.Field("Capture Status", string(dc.CaptureStatus))
	}

	// Timestamps
	d.Section("Timestamps")
	if t := endpoint.CreatedAt(); t != nil {
		d.Field("Created", t.Format("2006-01-02 15:04:05"))
	}
	if t := endpoint.LastModifiedAt(); t != nil {
		d.Field("Last Modified", t.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary renders summary fields for an endpoint.
func (r *EndpointRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	endpoint, ok := resource.(*EndpointResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Name", Value: endpoint.GetID()},
		{Label: "Status", Value: endpoint.Status()},
	}
}
