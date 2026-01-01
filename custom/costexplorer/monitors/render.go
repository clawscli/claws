package monitors

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// MonitorRenderer renders Cost Anomaly Monitor data.
type MonitorRenderer struct {
	render.BaseRenderer
}

// NewMonitorRenderer creates a new MonitorRenderer.
func NewMonitorRenderer() render.Renderer {
	return &MonitorRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "ce",
			Resource: "monitors",
			Cols: []render.Column{
				{Name: "NAME", Width: 35, Getter: getName},
				{Name: "TYPE", Width: 12, Getter: getType},
				{Name: "DIMENSION", Width: 15, Getter: getDimension},
				{Name: "VALUES", Width: 8, Getter: getValueCount},
				{Name: "LAST EVALUATED", Width: 14, Getter: getLastEvaluated},
				{Name: "CREATED", Width: 12, Getter: getCreated},
			},
		},
	}
}

func getName(r dao.Resource) string {
	m, ok := r.(*MonitorResource)
	if !ok {
		return ""
	}
	return m.MonitorName()
}

func getType(r dao.Resource) string {
	m, ok := r.(*MonitorResource)
	if !ok {
		return ""
	}
	return m.MonitorType()
}

func getDimension(r dao.Resource) string {
	m, ok := r.(*MonitorResource)
	if !ok {
		return ""
	}
	return m.MonitorDimension()
}

func getValueCount(r dao.Resource) string {
	m, ok := r.(*MonitorResource)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%d", m.DimensionalValueCount())
}

func getLastEvaluated(r dao.Resource) string {
	m, ok := r.(*MonitorResource)
	if !ok {
		return ""
	}
	return m.LastEvaluatedDate()
}

func getCreated(r dao.Resource) string {
	m, ok := r.(*MonitorResource)
	if !ok {
		return ""
	}
	return m.CreationDate()
}

// RenderDetail renders the detail view for a monitor.
func (r *MonitorRenderer) RenderDetail(resource dao.Resource) string {
	m, ok := resource.(*MonitorResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Cost Anomaly Monitor", m.MonitorName())

	// Basic Info
	d.Section("Monitor Information")
	d.Field("Name", m.MonitorName())
	d.Field("ARN", m.GetARN())
	d.Field("Type", m.MonitorType())
	d.Field("Dimension", m.MonitorDimension())

	// Statistics
	d.Section("Statistics")
	d.Field("Dimensional Value Count", fmt.Sprintf("%d", m.DimensionalValueCount()))

	// Dates
	d.Section("Dates")
	d.Field("Created", m.CreationDate())
	d.Field("Last Evaluated", m.LastEvaluatedDate())
	d.Field("Last Updated", m.LastUpdatedDate())

	return d.String()
}

// RenderSummary renders summary fields for a monitor.
func (r *MonitorRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	m, ok := resource.(*MonitorResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Name", Value: m.MonitorName()},
		{Label: "Type", Value: m.MonitorType()},
		{Label: "Dimension", Value: m.MonitorDimension()},
		{Label: "Last Evaluated", Value: m.LastEvaluatedDate()},
	}
}
