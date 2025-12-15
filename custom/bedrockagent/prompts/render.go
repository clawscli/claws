package prompts

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// PromptRenderer renders Bedrock Prompt resources
// Ensure PromptRenderer implements render.Navigator
var _ render.Navigator = (*PromptRenderer)(nil)

type PromptRenderer struct {
	render.BaseRenderer
}

// NewPromptRenderer creates a new PromptRenderer
func NewPromptRenderer() render.Renderer {
	return &PromptRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "bedrock-agent",
			Resource: "prompts",
			Cols: []render.Column{
				{Name: "NAME", Width: 30, Getter: func(r dao.Resource) string { return r.GetName() }},
				{Name: "VERSION", Width: 10, Getter: getPromptVersion},
				{Name: "DESCRIPTION", Width: 35, Getter: getPromptDescription},
				{Name: "UPDATED", Width: 12, Getter: getPromptAge},
			},
		},
	}
}

func getPromptVersion(r dao.Resource) string {
	if prompt, ok := r.(*PromptResource); ok {
		return prompt.Version()
	}
	return ""
}

func getPromptDescription(r dao.Resource) string {
	if prompt, ok := r.(*PromptResource); ok {
		desc := prompt.Description()
		if len(desc) > 35 {
			return desc[:32] + "..."
		}
		return desc
	}
	return ""
}

func getPromptAge(r dao.Resource) string {
	if prompt, ok := r.(*PromptResource); ok {
		if updated := prompt.UpdatedAt(); updated != nil {
			return render.FormatAge(*updated)
		}
	}
	return "-"
}

// RenderDetail renders detailed prompt information
func (r *PromptRenderer) RenderDetail(resource dao.Resource) string {
	prompt, ok := resource.(*PromptResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Bedrock Prompt", prompt.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", prompt.GetName())
	d.Field("ID", prompt.GetID())
	d.Field("Version", prompt.Version())

	if arn := prompt.GetARN(); arn != "" {
		d.Field("ARN", arn)
	}

	if desc := prompt.Description(); desc != "" {
		d.Field("Description", desc)
	}

	// Configuration
	d.Section("Configuration")
	if defaultVariant := prompt.DefaultVariant(); defaultVariant != "" {
		d.Field("Default Variant", defaultVariant)
	}
	if variantCount := prompt.VariantCount(); variantCount > 0 {
		d.Field("Variant Count", fmt.Sprintf("%d", variantCount))
	}

	// Variants
	if variants := prompt.Variants(); len(variants) > 0 {
		d.Section("Variants")
		for _, variant := range variants {
			name := ""
			if variant.Name != nil {
				name = *variant.Name
			}
			modelId := ""
			if variant.ModelId != nil {
				modelId = *variant.ModelId
			}
			d.Field(name, fmt.Sprintf("Model: %s", modelId))
		}
	}

	// Timestamps
	d.Section("Timestamps")
	if created := prompt.CreatedAt(); created != nil {
		d.Field("Created", created.Format("2006-01-02 15:04:05"))
	}
	if updated := prompt.UpdatedAt(); updated != nil {
		d.Field("Updated", updated.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *PromptRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	prompt, ok := resource.(*PromptResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: prompt.GetName()},
		{Label: "ID", Value: prompt.GetID()},
		{Label: "Version", Value: prompt.Version()},
	}

	if arn := prompt.GetARN(); arn != "" {
		fields = append(fields, render.SummaryField{Label: "ARN", Value: arn})
	}

	if desc := prompt.Description(); desc != "" {
		fields = append(fields, render.SummaryField{Label: "Description", Value: desc})
	}

	if created := prompt.CreatedAt(); created != nil {
		fields = append(fields, render.SummaryField{Label: "Created", Value: fmt.Sprintf("%s ago", render.FormatAge(*created))})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *PromptRenderer) Navigations(resource dao.Resource) []render.Navigation {
	return nil
}
