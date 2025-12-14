package rules

import (
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// RuleRenderer renders AWS Config rules.
type RuleRenderer struct {
	render.BaseRenderer
}

// NewRuleRenderer creates a new RuleRenderer.
func NewRuleRenderer() render.Renderer {
	return &RuleRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "config",
			Resource: "rules",
			Cols: []render.Column{
				{Name: "RULE NAME", Width: 45, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "STATE", Width: 10, Getter: getState},
				{Name: "OWNER", Width: 15, Getter: getOwner},
				{Name: "SOURCE", Width: 40, Getter: getSourceIdentifier},
			},
		},
	}
}

func getState(r dao.Resource) string {
	rule, ok := r.(*RuleResource)
	if !ok {
		return ""
	}
	return rule.State()
}

func getOwner(r dao.Resource) string {
	rule, ok := r.(*RuleResource)
	if !ok {
		return ""
	}
	return rule.SourceOwner()
}

func getSourceIdentifier(r dao.Resource) string {
	rule, ok := r.(*RuleResource)
	if !ok {
		return ""
	}
	return rule.SourceIdentifier()
}

// RenderDetail renders the detail view for a Config rule.
func (r *RuleRenderer) RenderDetail(resource dao.Resource) string {
	rule, ok := resource.(*RuleResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("AWS Config Rule", rule.Name())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Rule Name", rule.Name())
	d.Field("ARN", rule.GetARN())
	d.Field("State", rule.State())
	if desc := rule.Description(); desc != "" {
		d.Field("Description", desc)
	}

	// Source Configuration
	d.Section("Source Configuration")
	d.Field("Owner", rule.SourceOwner())
	d.Field("Source Identifier", rule.SourceIdentifier())

	// Execution
	if freq := rule.MaximumExecutionFrequency(); freq != "" {
		d.Section("Execution")
		d.Field("Max Frequency", freq)
	}

	// Parameters
	if params := rule.InputParameters(); params != "" {
		d.Section("Input Parameters")
		d.Field("Parameters", params)
	}

	// Metadata
	if createdBy := rule.CreatedBy(); createdBy != "" {
		d.Section("Metadata")
		d.Field("Created By", createdBy)
	}

	return d.String()
}

// RenderSummary renders summary fields for a Config rule.
func (r *RuleRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	rule, ok := resource.(*RuleResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Rule Name", Value: rule.Name()},
		{Label: "ARN", Value: rule.GetARN()},
		{Label: "State", Value: rule.State()},
		{Label: "Owner", Value: rule.SourceOwner()},
		{Label: "Source", Value: rule.SourceIdentifier()},
	}

	if desc := rule.Description(); desc != "" {
		fields = append(fields, render.SummaryField{Label: "Description", Value: desc})
	}

	return fields
}
