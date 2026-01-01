package inferenceprofiles

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// InferenceProfileRenderer renders Bedrock Inference Profile resources
// Ensure InferenceProfileRenderer implements render.Navigator
var _ render.Navigator = (*InferenceProfileRenderer)(nil)

type InferenceProfileRenderer struct {
	render.BaseRenderer
}

// NewInferenceProfileRenderer creates a new InferenceProfileRenderer
func NewInferenceProfileRenderer() render.Renderer {
	return &InferenceProfileRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "bedrock",
			Resource: "inference-profiles",
			Cols: []render.Column{
				{Name: "NAME", Width: 40, Getter: func(r dao.Resource) string { return r.GetName() }},
				{Name: "STATUS", Width: 10, Getter: getIPStatus},
				{Name: "TYPE", Width: 18, Getter: getIPType},
				{Name: "MODELS", Width: 8, Getter: getIPModelCount},
				{Name: "UPDATED", Width: 12, Getter: getIPAge},
			},
		},
	}
}

func getIPStatus(r dao.Resource) string {
	if ip, ok := r.(*InferenceProfileResource); ok {
		return ip.Status()
	}
	return ""
}

func getIPType(r dao.Resource) string {
	if ip, ok := r.(*InferenceProfileResource); ok {
		return ip.ProfileType()
	}
	return ""
}

func getIPModelCount(r dao.Resource) string {
	if ip, ok := r.(*InferenceProfileResource); ok {
		return fmt.Sprintf("%d", ip.ModelCount())
	}
	return ""
}

func getIPAge(r dao.Resource) string {
	if ip, ok := r.(*InferenceProfileResource); ok {
		if updated := ip.UpdatedAt(); updated != nil {
			return render.FormatAge(*updated)
		}
	}
	return "-"
}

// RenderDetail renders detailed inference profile information
func (r *InferenceProfileRenderer) RenderDetail(resource dao.Resource) string {
	ip, ok := resource.(*InferenceProfileResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Bedrock Inference Profile", ip.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", ip.GetName())
	d.Field("ID", ip.GetID())
	d.Field("Status", ip.Status())
	d.Field("Type", ip.ProfileType())

	if arn := ip.GetARN(); arn != "" {
		d.Field("ARN", arn)
	}

	if desc := ip.Description(); desc != "" {
		d.Field("Description", desc)
	}

	// Models
	d.Section("Models")
	d.Field("Count", fmt.Sprintf("%d", ip.ModelCount()))
	if models := ip.Models(); models != "" {
		d.Field("Model ARNs", models)
	}

	// Timestamps
	d.Section("Timestamps")
	if created := ip.CreatedAt(); created != nil {
		d.Field("Created", created.Format("2006-01-02 15:04:05"))
	}
	if updated := ip.UpdatedAt(); updated != nil {
		d.Field("Updated", updated.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *InferenceProfileRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	ip, ok := resource.(*InferenceProfileResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: ip.GetName()},
		{Label: "ID", Value: ip.GetID()},
		{Label: "Status", Value: ip.Status()},
		{Label: "Type", Value: ip.ProfileType()},
	}

	if arn := ip.GetARN(); arn != "" {
		fields = append(fields, render.SummaryField{Label: "ARN", Value: arn})
	}

	fields = append(fields, render.SummaryField{Label: "Models", Value: fmt.Sprintf("%d", ip.ModelCount())})

	if created := ip.CreatedAt(); created != nil {
		fields = append(fields, render.SummaryField{Label: "Created", Value: fmt.Sprintf("%s ago", render.FormatAge(*created))})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *InferenceProfileRenderer) Navigations(resource dao.Resource) []render.Navigation {
	return nil
}
