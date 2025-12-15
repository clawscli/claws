package investigations

import (
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// InvestigationRenderer renders Detective investigations.
type InvestigationRenderer struct {
	render.BaseRenderer
}

// NewInvestigationRenderer creates a new InvestigationRenderer.
func NewInvestigationRenderer() render.Renderer {
	return &InvestigationRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "detective",
			Resource: "investigations",
			Cols: []render.Column{
				{Name: "INVESTIGATION ID", Width: 30, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "ENTITY TYPE", Width: 15, Getter: getEntityType},
				{Name: "SEVERITY", Width: 12, Getter: getSeverity},
				{Name: "STATUS", Width: 12, Getter: getStatus},
				{Name: "STATE", Width: 12, Getter: getState},
				{Name: "CREATED", Width: 20, Getter: getCreated},
			},
		},
	}
}

func getEntityType(r dao.Resource) string {
	inv, ok := r.(*InvestigationResource)
	if !ok {
		return ""
	}
	return inv.EntityType()
}

func getSeverity(r dao.Resource) string {
	inv, ok := r.(*InvestigationResource)
	if !ok {
		return ""
	}
	return inv.Severity()
}

func getStatus(r dao.Resource) string {
	inv, ok := r.(*InvestigationResource)
	if !ok {
		return ""
	}
	return inv.Status()
}

func getState(r dao.Resource) string {
	inv, ok := r.(*InvestigationResource)
	if !ok {
		return ""
	}
	return inv.State()
}

func getCreated(r dao.Resource) string {
	inv, ok := r.(*InvestigationResource)
	if !ok {
		return ""
	}
	if t := inv.CreatedTime(); t != nil {
		return render.FormatAge(*t)
	}
	return ""
}

// RenderDetail renders the detail view for an investigation.
func (r *InvestigationRenderer) RenderDetail(resource dao.Resource) string {
	inv, ok := resource.(*InvestigationResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Detective Investigation", inv.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Investigation ID", inv.GetID())
	d.Field("Entity ARN", inv.EntityArn())
	d.Field("Entity Type", inv.EntityType())

	// Status
	d.Section("Status")
	d.Field("Status", inv.Status())
	d.Field("State", inv.State())
	d.Field("Severity", inv.Severity())

	// Timestamps
	d.Section("Timestamps")
	if t := inv.CreatedTime(); t != nil {
		d.Field("Created", t.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary renders summary fields for an investigation.
func (r *InvestigationRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	inv, ok := resource.(*InvestigationResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Investigation ID", Value: inv.GetID()},
		{Label: "Entity Type", Value: inv.EntityType()},
		{Label: "Severity", Value: inv.Severity()},
		{Label: "Status", Value: inv.Status()},
	}
}
