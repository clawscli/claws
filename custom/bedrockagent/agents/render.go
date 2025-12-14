package agents

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// AgentRenderer renders Bedrock Agent resources
// Ensure AgentRenderer implements render.Navigator
var _ render.Navigator = (*AgentRenderer)(nil)

type AgentRenderer struct {
	render.BaseRenderer
}

// NewAgentRenderer creates a new AgentRenderer
func NewAgentRenderer() render.Renderer {
	return &AgentRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "bedrock-agent",
			Resource: "agents",
			Cols: []render.Column{
				{Name: "NAME", Width: 25, Getter: func(r dao.Resource) string { return r.GetName() }},
				{Name: "STATUS", Width: 14, Getter: getAgentStatus},
				{Name: "VERSION", Width: 8, Getter: getAgentVersion},
				{Name: "DESCRIPTION", Width: 30, Getter: getAgentDescription},
				{Name: "UPDATED", Width: 12, Getter: getAgentAge},
			},
		},
	}
}

func getAgentStatus(r dao.Resource) string {
	if agent, ok := r.(*AgentResource); ok {
		return agent.Status()
	}
	return ""
}

func getAgentVersion(r dao.Resource) string {
	if agent, ok := r.(*AgentResource); ok {
		return agent.LatestVersion()
	}
	return ""
}

func getAgentDescription(r dao.Resource) string {
	if agent, ok := r.(*AgentResource); ok {
		desc := agent.Description()
		if len(desc) > 30 {
			return desc[:27] + "..."
		}
		return desc
	}
	return ""
}

func getAgentAge(r dao.Resource) string {
	if agent, ok := r.(*AgentResource); ok {
		if updated := agent.UpdatedAt(); updated != nil {
			return render.FormatAge(*updated)
		}
	}
	return "-"
}

// RenderDetail renders detailed agent information
func (r *AgentRenderer) RenderDetail(resource dao.Resource) string {
	agent, ok := resource.(*AgentResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Bedrock Agent", agent.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", agent.GetName())
	d.Field("ID", agent.GetID())
	d.Field("Status", agent.Status())
	d.Field("Version", agent.LatestVersion())

	if arn := agent.GetARN(); arn != "" {
		d.Field("ARN", arn)
	}

	if desc := agent.Description(); desc != "" {
		d.Field("Description", desc)
	}

	// Configuration
	d.Section("Configuration")
	if model := agent.FoundationModel(); model != "" {
		d.Field("Foundation Model", model)
	}
	if roleArn := agent.RoleArn(); roleArn != "" {
		d.Field("Role ARN", roleArn)
	}
	if guardrail := agent.GuardrailId(); guardrail != "" {
		d.Field("Guardrail ID", guardrail)
	}
	if ttl := agent.IdleSessionTTL(); ttl > 0 {
		d.Field("Idle Session TTL", fmt.Sprintf("%d seconds", ttl))
	}

	// Timestamps
	d.Section("Timestamps")
	if created := agent.CreatedAt(); created != nil {
		d.Field("Created", created.Format("2006-01-02 15:04:05"))
	}
	if updated := agent.UpdatedAt(); updated != nil {
		d.Field("Updated", updated.Format("2006-01-02 15:04:05"))
	}
	if prepared := agent.PreparedAt(); prepared != nil {
		d.Field("Prepared", prepared.Format("2006-01-02 15:04:05"))
	}

	// Failure Reasons
	if failures := agent.FailureReasons(); len(failures) > 0 {
		d.Section("Failure Reasons")
		for _, reason := range failures {
			d.Field("", reason)
		}
	}

	// Recommended Actions
	if actions := agent.RecommendedActions(); len(actions) > 0 {
		d.Section("Recommended Actions")
		for _, action := range actions {
			d.Field("", action)
		}
	}

	// Instructions (at bottom for readability)
	if instruction := agent.Instruction(); instruction != "" {
		d.Section("Instruction")
		d.Line(instruction)
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *AgentRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	agent, ok := resource.(*AgentResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: agent.GetName()},
		{Label: "ID", Value: agent.GetID()},
		{Label: "Status", Value: agent.Status()},
		{Label: "Version", Value: agent.LatestVersion()},
	}

	if arn := agent.GetARN(); arn != "" {
		fields = append(fields, render.SummaryField{Label: "ARN", Value: arn})
	}

	if desc := agent.Description(); desc != "" {
		fields = append(fields, render.SummaryField{Label: "Description", Value: desc})
	}

	if model := agent.FoundationModel(); model != "" {
		fields = append(fields, render.SummaryField{Label: "Model", Value: model})
	}

	if created := agent.CreatedAt(); created != nil {
		fields = append(fields, render.SummaryField{Label: "Created", Value: fmt.Sprintf("%s ago", render.FormatAge(*created))})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *AgentRenderer) Navigations(resource dao.Resource) []render.Navigation {
	return nil
}
