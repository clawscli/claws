package graphs

import (
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// GraphRenderer renders Detective graphs.
// Ensure GraphRenderer implements render.Navigator
var _ render.Navigator = (*GraphRenderer)(nil)

type GraphRenderer struct {
	render.BaseRenderer
}

// NewGraphRenderer creates a new GraphRenderer.
func NewGraphRenderer() render.Renderer {
	return &GraphRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "detective",
			Resource: "graphs",
			Cols: []render.Column{
				{Name: "GRAPH ID", Width: 40, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "CREATED", Width: 20, Getter: getCreated},
			},
		},
	}
}

func getCreated(r dao.Resource) string {
	graph, ok := r.(*GraphResource)
	if !ok {
		return ""
	}
	if t := graph.CreatedTime(); t != nil {
		return render.FormatAge(*t)
	}
	return ""
}

// RenderDetail renders the detail view for a graph.
func (r *GraphRenderer) RenderDetail(resource dao.Resource) string {
	graph, ok := resource.(*GraphResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Detective Graph", graph.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Graph ID", graph.GetID())
	d.Field("ARN", graph.GraphArn())

	// Timestamps
	d.Section("Timestamps")
	if t := graph.CreatedTime(); t != nil {
		d.Field("Created", t.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary renders summary fields for a graph.
func (r *GraphRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	graph, ok := resource.(*GraphResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Graph ID", Value: graph.GetID()},
		{Label: "ARN", Value: graph.GraphArn()},
	}
}

// Navigations returns available navigations from a graph.
func (r *GraphRenderer) Navigations(resource dao.Resource) []render.Navigation {
	graph, ok := resource.(*GraphResource)
	if !ok {
		return nil
	}
	return []render.Navigation{
		{
			Key:         "i",
			Label:       "Investigations",
			Service:     "detective",
			Resource:    "investigations",
			FilterField: "GraphArn",
			FilterValue: graph.GraphArn(),
		},
	}
}
