package guardrails

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// GuardrailRenderer renders Bedrock Guardrail resources
// Ensure GuardrailRenderer implements render.Navigator
var _ render.Navigator = (*GuardrailRenderer)(nil)

type GuardrailRenderer struct {
	render.BaseRenderer
}

// NewGuardrailRenderer creates a new GuardrailRenderer
func NewGuardrailRenderer() render.Renderer {
	return &GuardrailRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "bedrock",
			Resource: "guardrails",
			Cols: []render.Column{
				{Name: "NAME", Width: 30, Getter: func(r dao.Resource) string { return r.GetName() }},
				{Name: "STATUS", Width: 12, Getter: getGRStatus},
				{Name: "VERSION", Width: 10, Getter: getGRVersion},
				{Name: "DESCRIPTION", Width: 35, Getter: getGRDescription},
				{Name: "UPDATED", Width: 12, Getter: getGRAge},
			},
		},
	}
}

func getGRStatus(r dao.Resource) string {
	if gr, ok := r.(*GuardrailResource); ok {
		return gr.Status()
	}
	return ""
}

func getGRVersion(r dao.Resource) string {
	if gr, ok := r.(*GuardrailResource); ok {
		return gr.Version()
	}
	return ""
}

func getGRDescription(r dao.Resource) string {
	if gr, ok := r.(*GuardrailResource); ok {
		desc := gr.Description()
		if len(desc) > 35 {
			return desc[:32] + "..."
		}
		return desc
	}
	return ""
}

func getGRAge(r dao.Resource) string {
	if gr, ok := r.(*GuardrailResource); ok {
		if updated := gr.UpdatedAt(); updated != nil {
			return render.FormatAge(*updated)
		}
	}
	return "-"
}

// RenderDetail renders detailed guardrail information
func (r *GuardrailRenderer) RenderDetail(resource dao.Resource) string {
	gr, ok := resource.(*GuardrailResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Bedrock Guardrail", gr.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", gr.GetName())
	d.Field("ID", gr.GetID())
	d.Field("Status", gr.Status())
	d.Field("Version", gr.Version())

	if arn := gr.GetARN(); arn != "" {
		d.Field("ARN", arn)
	}

	if desc := gr.Description(); desc != "" {
		d.Field("Description", desc)
	}

	// Messaging
	d.Section("Messaging")
	if msg := gr.BlockedInputMessaging(); msg != "" {
		d.Field("Blocked Input", msg)
	}
	if msg := gr.BlockedOutputsMessaging(); msg != "" {
		d.Field("Blocked Output", msg)
	}

	// Timestamps
	d.Section("Timestamps")
	if created := gr.CreatedAt(); created != nil {
		d.Field("Created", created.Format("2006-01-02 15:04:05"))
	}
	if updated := gr.UpdatedAt(); updated != nil {
		d.Field("Updated", updated.Format("2006-01-02 15:04:05"))
	}

	// Failure Recommendations
	if recommendations := gr.FailureRecommendations(); len(recommendations) > 0 {
		d.Section("Failure Recommendations")
		for _, rec := range recommendations {
			d.Field("", rec)
		}
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *GuardrailRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	gr, ok := resource.(*GuardrailResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: gr.GetName()},
		{Label: "ID", Value: gr.GetID()},
		{Label: "Status", Value: gr.Status()},
		{Label: "Version", Value: gr.Version()},
	}

	if arn := gr.GetARN(); arn != "" {
		fields = append(fields, render.SummaryField{Label: "ARN", Value: arn})
	}

	if desc := gr.Description(); desc != "" {
		fields = append(fields, render.SummaryField{Label: "Description", Value: desc})
	}

	if created := gr.CreatedAt(); created != nil {
		fields = append(fields, render.SummaryField{Label: "Created", Value: fmt.Sprintf("%s ago", render.FormatAge(*created))})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *GuardrailRenderer) Navigations(resource dao.Resource) []render.Navigation {
	return nil
}
