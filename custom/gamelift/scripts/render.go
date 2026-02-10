package scripts

import (
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// ScriptRenderer renders GameLift scripts.
type ScriptRenderer struct {
	render.BaseRenderer
}

// NewScriptRenderer creates a new ScriptRenderer.
func NewScriptRenderer() render.Renderer {
	return &ScriptRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "gamelift",
			Resource: "scripts",
			Cols: []render.Column{
				{Name: "NAME", Width: 30, Getter: func(r dao.Resource) string { return r.GetName() }},
				{Name: "SCRIPT ID", Width: 24, Getter: func(r dao.Resource) string { return r.GetID() }, Priority: 3},
				{Name: "VERSION", Width: 16, Getter: getScriptVersion},
				{Name: "NODE.JS", Width: 10, Getter: getScriptNodeJsVersion, Priority: 2},
				{Name: "SIZE", Width: 12, Getter: getScriptSize, Priority: 2},
				{Name: "CREATED", Width: 20, Getter: getScriptCreated},
			},
		},
	}
}

func getScriptVersion(r dao.Resource) string {
	script, ok := r.(*ScriptResource)
	if !ok {
		return ""
	}
	return script.Version()
}

func getScriptNodeJsVersion(r dao.Resource) string {
	script, ok := r.(*ScriptResource)
	if !ok {
		return ""
	}
	return script.NodeJsVersion()
}

func getScriptSize(r dao.Resource) string {
	script, ok := r.(*ScriptResource)
	if !ok {
		return ""
	}
	size := script.SizeOnDisk()
	if size == 0 {
		return "-"
	}
	return render.FormatSize(size)
}

func getScriptCreated(r dao.Resource) string {
	script, ok := r.(*ScriptResource)
	if !ok {
		return ""
	}
	if t := script.CreationTime(); t != nil {
		return t.Format("2006-01-02 15:04")
	}
	return ""
}

// RenderDetail renders the detail view for a GameLift script.
func (rr *ScriptRenderer) RenderDetail(resource dao.Resource) string {
	script, ok := resource.(*ScriptResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("GameLift Script", script.GetName())

	d.Section("Basic Information")
	d.Field("Name", script.GetName())
	d.Field("Script ID", script.GetID())
	d.Field("ARN", script.GetARN())

	d.Section("Configuration")
	if v := script.Version(); v != "" {
		d.Field("Version", v)
	}
	if v := script.NodeJsVersion(); v != "" {
		d.Field("Node.js Version", v)
	}

	d.Section("Storage")
	size := script.SizeOnDisk()
	if size > 0 {
		d.Field("Size on Disk", render.FormatSize(size))
	}
	if bucket := script.StorageLocationBucket(); bucket != "" {
		d.Field("S3 Bucket", bucket)
		if key := script.StorageLocationKey(); key != "" {
			d.Field("S3 Key", key)
		}
	}

	d.Section("Timestamps")
	if t := script.CreationTime(); t != nil {
		d.Field("Created", t.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary renders summary fields for a GameLift script.
func (rr *ScriptRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	script, ok := resource.(*ScriptResource)
	if !ok {
		return rr.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Name", Value: script.GetName()},
		{Label: "Script ID", Value: script.GetID()},
		{Label: "Version", Value: script.Version()},
	}
}
