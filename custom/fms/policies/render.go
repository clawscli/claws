package policies

import (
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// PolicyRenderer renders FMS policies.
type PolicyRenderer struct {
	render.BaseRenderer
}

// NewPolicyRenderer creates a new PolicyRenderer.
func NewPolicyRenderer() render.Renderer {
	return &PolicyRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "fms",
			Resource: "policies",
			Cols: []render.Column{
				{Name: "POLICY NAME", Width: 35, Getter: getPolicyName},
				{Name: "POLICY ID", Width: 38, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "SERVICE TYPE", Width: 20, Getter: getServiceType},
				{Name: "RESOURCE TYPE", Width: 20, Getter: getResourceType},
				{Name: "REMEDIATION", Width: 12, Getter: getRemediation},
			},
		},
	}
}

func getPolicyName(r dao.Resource) string {
	policy, ok := r.(*PolicyResource)
	if !ok {
		return ""
	}
	return policy.PolicyName()
}

func getServiceType(r dao.Resource) string {
	policy, ok := r.(*PolicyResource)
	if !ok {
		return ""
	}
	return policy.SecurityServiceType()
}

func getResourceType(r dao.Resource) string {
	policy, ok := r.(*PolicyResource)
	if !ok {
		return ""
	}
	return policy.ResourceType()
}

func getRemediation(r dao.Resource) string {
	policy, ok := r.(*PolicyResource)
	if !ok {
		return ""
	}
	if policy.RemediationEnabled() {
		return "Enabled"
	}
	return "Disabled"
}

// RenderDetail renders the detail view for an FMS policy.
func (r *PolicyRenderer) RenderDetail(resource dao.Resource) string {
	policy, ok := resource.(*PolicyResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Firewall Manager Policy", policy.PolicyName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Policy Name", policy.PolicyName())
	d.Field("Policy ID", policy.PolicyId())
	d.Field("ARN", policy.GetARN())

	// Service Configuration
	d.Section("Service Configuration")
	d.Field("Security Service Type", policy.SecurityServiceType())
	d.Field("Resource Type", policy.ResourceType())
	if types := policy.ResourceTypeList(); len(types) > 0 {
		d.Field("Resource Types", strings.Join(types, ", "))
	}

	// Settings
	d.Section("Settings")
	if policy.RemediationEnabled() {
		d.Field("Remediation", "Enabled")
	} else {
		d.Field("Remediation", "Disabled")
	}
	if policy.DeleteUnusedFMManagedResources() {
		d.Field("Delete Unused Resources", "Yes")
	} else {
		d.Field("Delete Unused Resources", "No")
	}
	if policy.ExcludeResourceTags() {
		d.Field("Exclude Resource Tags", "Yes")
	} else {
		d.Field("Exclude Resource Tags", "No")
	}

	// Scope
	if includes := policy.IncludeMap(); len(includes) > 0 {
		d.Section("Include Scope")
		for key, values := range includes {
			d.Field(key, strings.Join(values, ", "))
		}
	}
	if excludes := policy.ExcludeMap(); len(excludes) > 0 {
		d.Section("Exclude Scope")
		for key, values := range excludes {
			d.Field(key, strings.Join(values, ", "))
		}
	}

	// Resource Tags
	if tags := policy.ResourceTags(); len(tags) > 0 {
		d.Section("Resource Tags")
		for k, v := range tags {
			d.Field(k, v)
		}
	}

	return d.String()
}

// RenderSummary renders summary fields for an FMS policy.
func (r *PolicyRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	policy, ok := resource.(*PolicyResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Policy Name", Value: policy.PolicyName()},
		{Label: "Policy ID", Value: policy.PolicyId()},
		{Label: "Service Type", Value: policy.SecurityServiceType()},
		{Label: "Resource Type", Value: policy.ResourceType()},
	}

	if policy.RemediationEnabled() {
		fields = append(fields, render.SummaryField{Label: "Remediation", Value: "Enabled"})
	} else {
		fields = append(fields, render.SummaryField{Label: "Remediation", Value: "Disabled"})
	}

	return fields
}
