package tgwattachments

import (
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// TGWAttachmentRenderer renders Transit Gateway attachments.
type TGWAttachmentRenderer struct {
	render.BaseRenderer
}

// NewTGWAttachmentRenderer creates a new TGWAttachmentRenderer.
func NewTGWAttachmentRenderer() render.Renderer {
	return &TGWAttachmentRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "vpc",
			Resource: "tgw-attachments",
			Cols: []render.Column{
				{Name: "ATTACHMENT ID", Width: 26, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "NAME", Width: 25, Getter: getName},
				{Name: "TYPE", Width: 12, Getter: getType},
				{Name: "RESOURCE ID", Width: 24, Getter: getResourceId},
				{Name: "STATE", Width: 12, Getter: getState},
				{Name: "CREATED", Width: 18, Getter: getCreated},
			},
		},
	}
}

func getName(r dao.Resource) string {
	att, ok := r.(*TGWAttachmentResource)
	if !ok {
		return ""
	}
	return att.Name()
}

func getType(r dao.Resource) string {
	att, ok := r.(*TGWAttachmentResource)
	if !ok {
		return ""
	}
	return att.ResourceType()
}

func getResourceId(r dao.Resource) string {
	att, ok := r.(*TGWAttachmentResource)
	if !ok {
		return ""
	}
	return att.ResourceId()
}

func getState(r dao.Resource) string {
	att, ok := r.(*TGWAttachmentResource)
	if !ok {
		return ""
	}
	return att.State()
}

func getCreated(r dao.Resource) string {
	att, ok := r.(*TGWAttachmentResource)
	if !ok {
		return ""
	}
	if t := att.CreationTime(); t != nil {
		return t.Format("2006-01-02 15:04")
	}
	return ""
}

// RenderDetail renders the detail view for a Transit Gateway attachment.
func (r *TGWAttachmentRenderer) RenderDetail(resource dao.Resource) string {
	att, ok := resource.(*TGWAttachmentResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	title := att.GetID()
	if name := att.Name(); name != "" {
		title = name
	}
	d.Title("Transit Gateway Attachment", title)

	// Basic Info
	d.Section("Basic Information")
	d.Field("Attachment ID", att.GetID())
	if name := att.Name(); name != "" {
		d.Field("Name", name)
	}
	d.Field("Transit Gateway ID", att.TransitGatewayId())
	d.Field("Transit Gateway Owner", att.TransitGatewayOwnerId())
	d.Field("State", att.State())

	// Resource
	d.Section("Attached Resource")
	d.Field("Resource Type", att.ResourceType())
	d.Field("Resource ID", att.ResourceId())
	d.Field("Resource Owner", att.ResourceOwnerId())

	// Association
	if assoc := att.Association(); assoc != "" {
		d.Section("Association")
		d.Field("Route Table ID", assoc)
		if state := att.AssociationState(); state != "" {
			d.Field("Association State", state)
		}
	}

	// Tags
	if tags := att.Tags(); len(tags) > 0 {
		d.Section("Tags")
		for k, v := range tags {
			if k != "Name" {
				d.Field(k, v)
			}
		}
	}

	// Timestamps
	if t := att.CreationTime(); t != nil {
		d.Section("Timestamps")
		d.Field("Created", t.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary renders summary fields for a Transit Gateway attachment.
func (r *TGWAttachmentRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	att, ok := resource.(*TGWAttachmentResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Attachment ID", Value: att.GetID()},
		{Label: "Type", Value: att.ResourceType()},
		{Label: "Resource ID", Value: att.ResourceId()},
		{Label: "State", Value: att.State()},
	}

	if name := att.Name(); name != "" {
		fields = append([]render.SummaryField{{Label: "Name", Value: name}}, fields...)
	}

	return fields
}
