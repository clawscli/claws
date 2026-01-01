package firewallpolicies

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// FirewallPolicyRenderer renders Network Firewall policies.
type FirewallPolicyRenderer struct {
	render.BaseRenderer
}

// NewFirewallPolicyRenderer creates a new FirewallPolicyRenderer.
func NewFirewallPolicyRenderer() render.Renderer {
	return &FirewallPolicyRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "network-firewall",
			Resource: "firewall-policies",
			Cols: []render.Column{
				{Name: "POLICY NAME", Width: 35, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "STATUS", Width: 12, Getter: getStatus},
				{Name: "FIREWALLS", Width: 12, Getter: getAssociations},
				{Name: "STATELESS", Width: 12, Getter: getStatelessCapacity},
				{Name: "STATEFUL", Width: 12, Getter: getStatefulCapacity},
			},
		},
	}
}

func getStatus(r dao.Resource) string {
	p, ok := r.(*FirewallPolicyResource)
	if !ok {
		return ""
	}
	return p.Status()
}

func getAssociations(r dao.Resource) string {
	p, ok := r.(*FirewallPolicyResource)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%d", p.NumberOfAssociations())
}

func getStatelessCapacity(r dao.Resource) string {
	p, ok := r.(*FirewallPolicyResource)
	if !ok {
		return ""
	}
	if c := p.ConsumedStatelessRuleCapacity(); c > 0 {
		return fmt.Sprintf("%d", c)
	}
	return "0"
}

func getStatefulCapacity(r dao.Resource) string {
	p, ok := r.(*FirewallPolicyResource)
	if !ok {
		return ""
	}
	if c := p.ConsumedStatefulRuleCapacity(); c > 0 {
		return fmt.Sprintf("%d", c)
	}
	return "0"
}

// RenderDetail renders the detail view for a Network Firewall policy.
func (r *FirewallPolicyRenderer) RenderDetail(resource dao.Resource) string {
	p, ok := resource.(*FirewallPolicyResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Network Firewall Policy", p.FirewallPolicyName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Policy Name", p.FirewallPolicyName())
	d.Field("ARN", p.GetARN())
	d.Field("Status", p.Status())
	if desc := p.Description(); desc != "" {
		d.Field("Description", desc)
	}

	// Associations
	d.Section("Usage")
	d.Field("Firewall Associations", fmt.Sprintf("%d", p.NumberOfAssociations()))

	// Rule Groups
	d.Section("Rule Groups")
	d.Field("Stateless Rule Groups", fmt.Sprintf("%d", p.StatelessRuleGroupCount()))
	d.Field("Stateful Rule Groups", fmt.Sprintf("%d", p.StatefulRuleGroupCount()))

	// Capacity
	d.Section("Capacity")
	d.Field("Consumed Stateless Capacity", fmt.Sprintf("%d", p.ConsumedStatelessRuleCapacity()))
	d.Field("Consumed Stateful Capacity", fmt.Sprintf("%d", p.ConsumedStatefulRuleCapacity()))

	return d.String()
}

// RenderSummary renders summary fields for a Network Firewall policy.
func (r *FirewallPolicyRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	p, ok := resource.(*FirewallPolicyResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Policy Name", Value: p.FirewallPolicyName()},
		{Label: "Status", Value: p.Status()},
		{Label: "Firewall Associations", Value: fmt.Sprintf("%d", p.NumberOfAssociations())},
	}
}
