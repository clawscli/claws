package jobdefinitions

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// JobDefinitionRenderer renders Batch job definitions.
type JobDefinitionRenderer struct {
	render.BaseRenderer
}

// NewJobDefinitionRenderer creates a new JobDefinitionRenderer.
func NewJobDefinitionRenderer() render.Renderer {
	return &JobDefinitionRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "batch",
			Resource: "job-definitions",
			Cols: []render.Column{
				{Name: "NAME", Width: 40, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "TYPE", Width: 12, Getter: getType},
				{Name: "STATUS", Width: 10, Getter: getStatus},
				{Name: "IMAGE", Width: 40, Getter: getImage},
			},
		},
	}
}

func getType(r dao.Resource) string {
	def, ok := r.(*JobDefinitionResource)
	if !ok {
		return ""
	}
	return def.Type()
}

func getStatus(r dao.Resource) string {
	def, ok := r.(*JobDefinitionResource)
	if !ok {
		return ""
	}
	return def.Status()
}

func getImage(r dao.Resource) string {
	def, ok := r.(*JobDefinitionResource)
	if !ok {
		return ""
	}
	return def.ContainerImage()
}

// RenderDetail renders the detail view for a job definition.
func (r *JobDefinitionRenderer) RenderDetail(resource dao.Resource) string {
	def, ok := resource.(*JobDefinitionResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Batch Job Definition", def.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", def.Name())
	d.Field("Revision", fmt.Sprintf("%d", def.Revision()))
	d.Field("ARN", def.GetARN())
	d.Field("Type", def.Type())
	d.Field("Status", def.Status())

	// Container
	if def.Def != nil && def.Def.ContainerProperties != nil {
		cp := def.Def.ContainerProperties
		d.Section("Container Properties")
		if cp.Image != nil {
			d.Field("Image", *cp.Image)
		}
		if cp.Vcpus != nil {
			d.Field("vCPUs", fmt.Sprintf("%d", *cp.Vcpus))
		}
		if cp.Memory != nil {
			d.Field("Memory (MB)", fmt.Sprintf("%d", *cp.Memory))
		}
		if len(cp.Command) > 0 {
			d.Field("Command", fmt.Sprintf("%v", cp.Command))
		}
	}

	return d.String()
}

// RenderSummary renders summary fields for a job definition.
func (r *JobDefinitionRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	def, ok := resource.(*JobDefinitionResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Name", Value: def.GetID()},
		{Label: "Type", Value: def.Type()},
		{Label: "Status", Value: def.Status()},
		{Label: "Image", Value: def.ContainerImage()},
	}
}
