package knowledgebases

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// KnowledgeBaseRenderer renders Bedrock Knowledge Base resources
// Ensure KnowledgeBaseRenderer implements render.Navigator
var _ render.Navigator = (*KnowledgeBaseRenderer)(nil)

type KnowledgeBaseRenderer struct {
	render.BaseRenderer
}

// NewKnowledgeBaseRenderer creates a new KnowledgeBaseRenderer
func NewKnowledgeBaseRenderer() render.Renderer {
	return &KnowledgeBaseRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "bedrock-agent",
			Resource: "knowledge-bases",
			Cols: []render.Column{
				{Name: "NAME", Width: 30, Getter: func(r dao.Resource) string { return r.GetName() }},
				{Name: "STATUS", Width: 12, Getter: getKBStatus},
				{Name: "DESCRIPTION", Width: 40, Getter: getKBDescription},
				{Name: "UPDATED", Width: 12, Getter: getKBAge},
			},
		},
	}
}

func getKBStatus(r dao.Resource) string {
	if kb, ok := r.(*KnowledgeBaseResource); ok {
		return kb.Status()
	}
	return ""
}

func getKBDescription(r dao.Resource) string {
	if kb, ok := r.(*KnowledgeBaseResource); ok {
		desc := kb.Description()
		if len(desc) > 40 {
			return desc[:37] + "..."
		}
		return desc
	}
	return ""
}

func getKBAge(r dao.Resource) string {
	if kb, ok := r.(*KnowledgeBaseResource); ok {
		if updated := kb.UpdatedAt(); updated != nil {
			return render.FormatAge(*updated)
		}
	}
	return "-"
}

// RenderDetail renders detailed knowledge base information
func (r *KnowledgeBaseRenderer) RenderDetail(resource dao.Resource) string {
	kb, ok := resource.(*KnowledgeBaseResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Bedrock Knowledge Base", kb.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", kb.GetName())
	d.Field("ID", kb.GetID())
	d.Field("Status", kb.Status())

	if arn := kb.GetARN(); arn != "" {
		d.Field("ARN", arn)
	}

	if desc := kb.Description(); desc != "" {
		d.Field("Description", desc)
	}

	// Configuration
	d.Section("Configuration")
	if roleArn := kb.RoleArn(); roleArn != "" {
		d.Field("Role ARN", roleArn)
	}
	if embedModel := kb.EmbeddingModelArn(); embedModel != "" {
		d.Field("Embedding Model", embedModel)
	}
	if storageType := kb.StorageType(); storageType != "" {
		d.Field("Storage Type", storageType)
	}

	// Timestamps
	d.Section("Timestamps")
	if created := kb.CreatedAt(); created != nil {
		d.Field("Created", created.Format("2006-01-02 15:04:05"))
	}
	if updated := kb.UpdatedAt(); updated != nil {
		d.Field("Updated", updated.Format("2006-01-02 15:04:05"))
	}

	// Failure Reasons
	if failures := kb.FailureReasons(); len(failures) > 0 {
		d.Section("Failure Reasons")
		for _, reason := range failures {
			d.Field("", reason)
		}
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *KnowledgeBaseRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	kb, ok := resource.(*KnowledgeBaseResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: kb.GetName()},
		{Label: "ID", Value: kb.GetID()},
		{Label: "Status", Value: kb.Status()},
	}

	if arn := kb.GetARN(); arn != "" {
		fields = append(fields, render.SummaryField{Label: "ARN", Value: arn})
	}

	if desc := kb.Description(); desc != "" {
		fields = append(fields, render.SummaryField{Label: "Description", Value: desc})
	}

	if created := kb.CreatedAt(); created != nil {
		fields = append(fields, render.SummaryField{Label: "Created", Value: fmt.Sprintf("%s ago", render.FormatAge(*created))})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *KnowledgeBaseRenderer) Navigations(resource dao.Resource) []render.Navigation {
	kb, ok := resource.(*KnowledgeBaseResource)
	if !ok {
		return nil
	}
	return []render.Navigation{
		{
			Key:         "s",
			Label:       "Data Sources",
			Service:     "bedrock-agent",
			Resource:    "data-sources",
			FilterField: "KnowledgeBaseId",
			FilterValue: kb.GetID(),
		},
	}
}
