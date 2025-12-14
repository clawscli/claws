package operations

import (
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// OperationRenderer renders App Runner operations.
type OperationRenderer struct {
	render.BaseRenderer
}

// NewOperationRenderer creates a new OperationRenderer.
func NewOperationRenderer() render.Renderer {
	return &OperationRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "apprunner",
			Resource: "operations",
			Cols: []render.Column{
				{Name: "OPERATION ID", Width: 38, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "TYPE", Width: 20, Getter: getType},
				{Name: "STATUS", Width: 14, Getter: getStatus},
				{Name: "STARTED", Width: 18, Getter: getStarted},
				{Name: "ENDED", Width: 18, Getter: getEnded},
			},
		},
	}
}

func getType(r dao.Resource) string {
	op, ok := r.(*OperationResource)
	if !ok {
		return ""
	}
	return op.OperationType()
}

func getStatus(r dao.Resource) string {
	op, ok := r.(*OperationResource)
	if !ok {
		return ""
	}
	return op.Status()
}

func getStarted(r dao.Resource) string {
	op, ok := r.(*OperationResource)
	if !ok {
		return ""
	}
	if t := op.StartedAt(); t != nil {
		return t.Format("2006-01-02 15:04")
	}
	return ""
}

func getEnded(r dao.Resource) string {
	op, ok := r.(*OperationResource)
	if !ok {
		return ""
	}
	if t := op.EndedAt(); t != nil {
		return t.Format("2006-01-02 15:04")
	}
	return ""
}

// RenderDetail renders the detail view for an App Runner operation.
func (r *OperationRenderer) RenderDetail(resource dao.Resource) string {
	op, ok := resource.(*OperationResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("App Runner Operation", op.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Operation ID", op.GetID())
	d.Field("Type", op.OperationType())
	d.Field("Status", op.Status())

	// Target
	d.Section("Target")
	d.Field("Target ARN", op.TargetArn())

	// Timing
	d.Section("Timing")
	if t := op.StartedAt(); t != nil {
		d.Field("Started", t.Format("2006-01-02 15:04:05"))
	}
	if t := op.EndedAt(); t != nil {
		d.Field("Ended", t.Format("2006-01-02 15:04:05"))
	}
	if t := op.UpdatedAt(); t != nil {
		d.Field("Updated", t.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary renders summary fields for an App Runner operation.
func (r *OperationRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	op, ok := resource.(*OperationResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Operation ID", Value: op.GetID()},
		{Label: "Type", Value: op.OperationType()},
		{Label: "Status", Value: op.Status()},
	}
}
