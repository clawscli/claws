package foundationmodels

import (
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// FoundationModelRenderer renders Bedrock Foundation Model resources
// Ensure FoundationModelRenderer implements render.Navigator
var _ render.Navigator = (*FoundationModelRenderer)(nil)

type FoundationModelRenderer struct {
	render.BaseRenderer
}

// NewFoundationModelRenderer creates a new FoundationModelRenderer
func NewFoundationModelRenderer() render.Renderer {
	return &FoundationModelRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "bedrock",
			Resource: "foundation-models",
			Cols: []render.Column{
				{Name: "MODEL ID", Width: 45, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "PROVIDER", Width: 15, Getter: getFMProvider},
				{Name: "INPUT", Width: 15, Getter: getFMInput},
				{Name: "OUTPUT", Width: 15, Getter: getFMOutput},
				{Name: "STREAMING", Width: 10, Getter: getFMStreaming},
			},
		},
	}
}

func getFMProvider(r dao.Resource) string {
	if model, ok := r.(*FoundationModelResource); ok {
		return model.Provider()
	}
	return ""
}

func getFMInput(r dao.Resource) string {
	if model, ok := r.(*FoundationModelResource); ok {
		return model.InputModalities()
	}
	return ""
}

func getFMOutput(r dao.Resource) string {
	if model, ok := r.(*FoundationModelResource); ok {
		return model.OutputModalities()
	}
	return ""
}

func getFMStreaming(r dao.Resource) string {
	if model, ok := r.(*FoundationModelResource); ok {
		if model.StreamingSupported() {
			return "Yes"
		}
		return "No"
	}
	return ""
}

// RenderDetail renders detailed foundation model information
func (r *FoundationModelRenderer) RenderDetail(resource dao.Resource) string {
	model, ok := resource.(*FoundationModelResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Bedrock Foundation Model", model.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", model.GetName())
	d.Field("Model ID", model.GetID())
	d.Field("Provider", model.Provider())

	if arn := model.GetARN(); arn != "" {
		d.Field("ARN", arn)
	}

	// Capabilities
	d.Section("Capabilities")
	if input := model.InputModalities(); input != "" {
		d.Field("Input Modalities", input)
	}
	if output := model.OutputModalities(); output != "" {
		d.Field("Output Modalities", output)
	}
	if inference := model.InferenceTypes(); inference != "" {
		d.Field("Inference Types", inference)
	}
	if model.StreamingSupported() {
		d.Field("Streaming", "Supported")
	} else {
		d.Field("Streaming", "Not Supported")
	}

	// Customization
	if customizations := model.CustomizationsSupported(); customizations != "" {
		d.Section("Customization")
		d.Field("Supported Types", customizations)
	}

	// Lifecycle
	if lifecycle := model.LifecycleStatus(); lifecycle != "" {
		d.Section("Lifecycle")
		d.Field("Status", lifecycle)
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *FoundationModelRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	model, ok := resource.(*FoundationModelResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: model.GetName()},
		{Label: "Model ID", Value: model.GetID()},
		{Label: "Provider", Value: model.Provider()},
	}

	if arn := model.GetARN(); arn != "" {
		fields = append(fields, render.SummaryField{Label: "ARN", Value: arn})
	}

	if input := model.InputModalities(); input != "" {
		fields = append(fields, render.SummaryField{Label: "Input", Value: input})
	}

	if output := model.OutputModalities(); output != "" {
		fields = append(fields, render.SummaryField{Label: "Output", Value: output})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *FoundationModelRenderer) Navigations(resource dao.Resource) []render.Navigation {
	return nil
}
