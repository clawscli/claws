package workgroups

import (
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// WorkgroupRenderer renders Athena workgroups.
// Ensure WorkgroupRenderer implements render.Navigator
var _ render.Navigator = (*WorkgroupRenderer)(nil)

type WorkgroupRenderer struct {
	render.BaseRenderer
}

// NewWorkgroupRenderer creates a new WorkgroupRenderer.
func NewWorkgroupRenderer() render.Renderer {
	return &WorkgroupRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "athena",
			Resource: "workgroups",
			Cols: []render.Column{
				{Name: "WORKGROUP NAME", Width: 30, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "STATE", Width: 12, Getter: getState},
				{Name: "ENGINE", Width: 15, Getter: getEngine},
				{Name: "DESCRIPTION", Width: 40, Getter: getDescription},
				{Name: "CREATED", Width: 20, Getter: getCreated},
			},
		},
	}
}

func getState(r dao.Resource) string {
	wg, ok := r.(*WorkgroupResource)
	if !ok {
		return ""
	}
	return wg.State()
}

func getEngine(r dao.Resource) string {
	wg, ok := r.(*WorkgroupResource)
	if !ok {
		return ""
	}
	return wg.EngineVersion()
}

func getDescription(r dao.Resource) string {
	wg, ok := r.(*WorkgroupResource)
	if !ok {
		return ""
	}
	desc := wg.Description()
	if len(desc) > 37 {
		return desc[:37] + "..."
	}
	return desc
}

func getCreated(r dao.Resource) string {
	wg, ok := r.(*WorkgroupResource)
	if !ok {
		return ""
	}
	if t := wg.CreationTime(); t != nil {
		return t.Format("2006-01-02 15:04")
	}
	return ""
}

// RenderDetail renders the detail view for an Athena workgroup.
func (r *WorkgroupRenderer) RenderDetail(resource dao.Resource) string {
	wg, ok := resource.(*WorkgroupResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Athena Workgroup", wg.Name())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Workgroup Name", wg.Name())
	d.Field("State", wg.State())
	if desc := wg.Description(); desc != "" {
		d.Field("Description", desc)
	}

	// Engine
	if engine := wg.EngineVersion(); engine != "" {
		d.Section("Engine")
		d.Field("Engine Version", engine)
	}

	// Output Location
	if output := wg.OutputLocation(); output != "" {
		d.Section("Results Configuration")
		d.Field("Output Location", output)
	}

	// Timestamps
	if t := wg.CreationTime(); t != nil {
		d.Section("Timestamps")
		d.Field("Created", t.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary renders summary fields for an Athena workgroup.
func (r *WorkgroupRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	wg, ok := resource.(*WorkgroupResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Workgroup Name", Value: wg.Name()},
		{Label: "State", Value: wg.State()},
	}

	if engine := wg.EngineVersion(); engine != "" {
		fields = append(fields, render.SummaryField{Label: "Engine", Value: engine})
	}

	if desc := wg.Description(); desc != "" {
		fields = append(fields, render.SummaryField{Label: "Description", Value: desc})
	}

	return fields
}

// Navigations returns available navigations from an Athena workgroup.
func (r *WorkgroupRenderer) Navigations(resource dao.Resource) []render.Navigation {
	wg, ok := resource.(*WorkgroupResource)
	if !ok {
		return nil
	}
	return []render.Navigation{
		{
			Key:         "q",
			Label:       "Queries",
			Service:     "athena",
			Resource:    "query-executions",
			FilterField: "WorkGroup",
			FilterValue: wg.Name(),
		},
	}
}
