package policies

import (
	"bytes"
	"encoding/json"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// PolicyRenderer renders Organizations policies.
type PolicyRenderer struct {
	render.BaseRenderer
}

// NewPolicyRenderer creates a new PolicyRenderer.
func NewPolicyRenderer() render.Renderer {
	return &PolicyRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "organizations",
			Resource: "policies",
			Cols: []render.Column{
				{Name: "POLICY ID", Width: 20, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "NAME", Width: 35, Getter: getName},
				{Name: "TYPE", Width: 25, Getter: getType},
				{Name: "AWS MANAGED", Width: 12, Getter: getAwsManaged},
			},
		},
	}
}

func getName(r dao.Resource) string {
	policy, ok := r.(*PolicyResource)
	if !ok {
		return ""
	}
	return policy.Name()
}

func getType(r dao.Resource) string {
	policy, ok := r.(*PolicyResource)
	if !ok {
		return ""
	}
	return policy.Type()
}

func getAwsManaged(r dao.Resource) string {
	policy, ok := r.(*PolicyResource)
	if !ok {
		return ""
	}
	if policy.AwsManaged() {
		return "Yes"
	}
	return "No"
}

// RenderDetail renders the detail view for a policy.
func (r *PolicyRenderer) RenderDetail(resource dao.Resource) string {
	policy, ok := resource.(*PolicyResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Organizations Policy", policy.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Policy ID", policy.GetID())
	d.Field("Name", policy.Name())
	d.Field("Type", policy.Type())
	d.Field("ARN", policy.GetARN())
	if policy.AwsManaged() {
		d.Field("AWS Managed", "Yes")
	}

	if policy.Description() != "" {
		d.Field("Description", policy.Description())
	}

	// Policy Content (at bottom for readability)
	if policy.Content != "" {
		d.Section("Policy Document")
		d.Line(prettyJSON(policy.Content))
	}

	return d.String()
}

// prettyJSON formats JSON string with indentation
func prettyJSON(s string) string {
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(s), "", "  "); err != nil {
		return s
	}
	return buf.String()
}

// RenderSummary renders summary fields for a policy.
func (r *PolicyRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	policy, ok := resource.(*PolicyResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Policy ID", Value: policy.GetID()},
		{Label: "Name", Value: policy.Name()},
		{Label: "Type", Value: policy.Type()},
	}
}
