package builds

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// BuildRenderer renders GameLift builds.
type BuildRenderer struct {
	render.BaseRenderer
}

// NewBuildRenderer creates a new BuildRenderer.
func NewBuildRenderer() render.Renderer {
	return &BuildRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "gamelift",
			Resource: "builds",
			Cols: []render.Column{
				{Name: "NAME", Width: 30, Getter: func(r dao.Resource) string { return r.GetName() }},
				{Name: "BUILD ID", Width: 24, Getter: func(r dao.Resource) string { return r.GetID() }, Priority: 3},
				{Name: "STATUS", Width: 14, Getter: getBuildStatus},
				{Name: "VERSION", Width: 16, Getter: getBuildVersion},
				{Name: "OS", Width: 16, Getter: getBuildOS},
				{Name: "SIZE", Width: 12, Getter: getBuildSize, Priority: 2},
				{Name: "SDK VERSION", Width: 14, Getter: getBuildSdkVersion, Priority: 3},
				{Name: "CREATED", Width: 20, Getter: getBuildCreated, Priority: 2},
			},
		},
	}
}

func getBuildStatus(r dao.Resource) string {
	build, ok := r.(*BuildResource)
	if !ok {
		return ""
	}
	return build.Status()
}

func getBuildVersion(r dao.Resource) string {
	build, ok := r.(*BuildResource)
	if !ok {
		return ""
	}
	return build.Version()
}

func getBuildOS(r dao.Resource) string {
	build, ok := r.(*BuildResource)
	if !ok {
		return ""
	}
	return build.OperatingSystem()
}

func getBuildSize(r dao.Resource) string {
	build, ok := r.(*BuildResource)
	if !ok {
		return ""
	}
	size := build.SizeOnDisk()
	if size == 0 {
		return "-"
	}
	return render.FormatSize(size)
}

func getBuildSdkVersion(r dao.Resource) string {
	build, ok := r.(*BuildResource)
	if !ok {
		return ""
	}
	return build.ServerSdkVersion()
}

func getBuildCreated(r dao.Resource) string {
	build, ok := r.(*BuildResource)
	if !ok {
		return ""
	}
	if t := build.CreationTime(); t != nil {
		return t.Format("2006-01-02 15:04")
	}
	return ""
}

// RenderDetail renders the detail view for a GameLift build.
func (rr *BuildRenderer) RenderDetail(resource dao.Resource) string {
	build, ok := resource.(*BuildResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("GameLift Build", build.GetName())

	d.Section("Basic Information")
	d.Field("Name", build.GetName())
	d.Field("Build ID", build.GetID())
	d.Field("ARN", build.GetARN())
	d.Field("Status", build.Status())

	d.Section("Configuration")
	if v := build.Version(); v != "" {
		d.Field("Version", v)
	}
	d.Field("Operating System", build.OperatingSystem())
	if v := build.ServerSdkVersion(); v != "" {
		d.Field("Server SDK Version", v)
	}

	d.Section("Storage")
	size := build.SizeOnDisk()
	if size > 0 {
		d.Field("Size on Disk", render.FormatSize(size))
	} else {
		d.Field("Size on Disk", fmt.Sprintf("%d bytes", size))
	}

	d.Section("Timestamps")
	if t := build.CreationTime(); t != nil {
		d.Field("Created", t.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary renders summary fields for a GameLift build.
func (rr *BuildRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	build, ok := resource.(*BuildResource)
	if !ok {
		return rr.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Name", Value: build.GetName()},
		{Label: "Build ID", Value: build.GetID()},
		{Label: "Status", Value: build.Status()},
		{Label: "Version", Value: build.Version()},
	}
}
