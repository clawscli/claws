package rulegroups

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// RuleGroupRenderer renders Network Firewall rule groups.
type RuleGroupRenderer struct {
	render.BaseRenderer
}

// NewRuleGroupRenderer creates a new RuleGroupRenderer.
func NewRuleGroupRenderer() render.Renderer {
	return &RuleGroupRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "network-firewall",
			Resource: "rule-groups",
			Cols: []render.Column{
				{Name: "RULE GROUP NAME", Width: 35, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "TYPE", Width: 12, Getter: getType},
				{Name: "STATUS", Width: 12, Getter: getStatus},
				{Name: "CAPACITY", Width: 10, Getter: getCapacity},
				{Name: "ASSOCIATIONS", Width: 14, Getter: getAssociations},
			},
		},
	}
}

func getType(r dao.Resource) string {
	rg, ok := r.(*RuleGroupResource)
	if !ok {
		return ""
	}
	return rg.Type()
}

func getStatus(r dao.Resource) string {
	rg, ok := r.(*RuleGroupResource)
	if !ok {
		return ""
	}
	return rg.Status()
}

func getCapacity(r dao.Resource) string {
	rg, ok := r.(*RuleGroupResource)
	if !ok {
		return ""
	}
	if c := rg.Capacity(); c > 0 {
		return fmt.Sprintf("%d", c)
	}
	return ""
}

func getAssociations(r dao.Resource) string {
	rg, ok := r.(*RuleGroupResource)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%d", rg.NumberOfAssociations())
}

// RenderDetail renders the detail view for a Network Firewall rule group.
func (r *RuleGroupRenderer) RenderDetail(resource dao.Resource) string {
	rg, ok := resource.(*RuleGroupResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Network Firewall Rule Group", rg.RuleGroupName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Rule Group Name", rg.RuleGroupName())
	d.Field("ARN", rg.GetARN())
	d.Field("Type", rg.Type())
	d.Field("Status", rg.Status())
	if desc := rg.Description(); desc != "" {
		d.Field("Description", desc)
	}

	// Capacity
	d.Section("Capacity")
	if c := rg.Capacity(); c > 0 {
		d.Field("Total Capacity", fmt.Sprintf("%d", c))
	}
	if c := rg.ConsumedCapacity(); c > 0 {
		d.Field("Consumed Capacity", fmt.Sprintf("%d", c))
	}

	// Associations
	d.Section("Usage")
	d.Field("Policy Associations", fmt.Sprintf("%d", rg.NumberOfAssociations()))

	return d.String()
}

// RenderSummary renders summary fields for a Network Firewall rule group.
func (r *RuleGroupRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	rg, ok := resource.(*RuleGroupResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Rule Group Name", Value: rg.RuleGroupName()},
		{Label: "Type", Value: rg.Type()},
		{Label: "Status", Value: rg.Status()},
	}

	if c := rg.Capacity(); c > 0 {
		fields = append(fields, render.SummaryField{Label: "Capacity", Value: fmt.Sprintf("%d", c)})
	}

	return fields
}
