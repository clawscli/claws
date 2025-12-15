package builds

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// BuildRenderer renders CodeBuild builds
// Ensure BuildRenderer implements render.Navigator
var _ render.Navigator = (*BuildRenderer)(nil)

type BuildRenderer struct {
	render.BaseRenderer
}

// NewBuildRenderer creates a new BuildRenderer
func NewBuildRenderer() *BuildRenderer {
	return &BuildRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "codebuild",
			Resource: "builds",
			Cols: []render.Column{
				{Name: "BUILD #", Width: 10, Getter: getBuildNumber},
				{Name: "STATUS", Width: 12, Getter: getStatus},
				{Name: "PHASE", Width: 15, Getter: getPhase},
				{Name: "STARTED", Width: 20, Getter: getStarted},
				{Name: "DURATION", Width: 12, Getter: getDuration},
			},
		},
	}
}

func getBuildNumber(r dao.Resource) string {
	if b, ok := r.(*BuildResource); ok {
		if num := b.BuildNumber(); num > 0 {
			return fmt.Sprintf("#%d", num)
		}
	}
	return "-"
}

func getStatus(r dao.Resource) string {
	if b, ok := r.(*BuildResource); ok {
		return b.Status()
	}
	return ""
}

func getPhase(r dao.Resource) string {
	if b, ok := r.(*BuildResource); ok {
		return b.CurrentPhase()
	}
	return ""
}

func getStarted(r dao.Resource) string {
	if b, ok := r.(*BuildResource); ok {
		return b.StartTime()
	}
	return "-"
}

func getDuration(r dao.Resource) string {
	if b, ok := r.(*BuildResource); ok {
		if dur := b.Duration(); dur != "" {
			return dur
		}
	}
	return "-"
}

// RenderDetail renders detailed build information
func (r *BuildRenderer) RenderDetail(resource dao.Resource) string {
	build, ok := resource.(*BuildResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("CodeBuild Build", fmt.Sprintf("#%d", build.BuildNumber()))

	// Basic Info
	d.Section("Basic Information")
	d.Field("Build ID", build.BuildId())
	if arn := build.GetARN(); arn != "" {
		d.Field("ARN", arn)
	}
	d.Field("Project", build.ProjectName)
	d.Field("Build Number", fmt.Sprintf("%d", build.BuildNumber()))
	d.Field("Status", build.Status())
	d.Field("Current Phase", build.CurrentPhase())

	// Source
	d.Section("Source")
	if sourceVer := build.SourceVersion(); sourceVer != "" {
		d.Field("Source Version", sourceVer)
	}
	if resolvedVer := build.ResolvedSourceVersion(); resolvedVer != "" {
		d.Field("Resolved Version", resolvedVer)
	}
	d.Field("Initiator", build.Initiator())

	// Environment
	d.Section("Environment")
	if img := build.EnvironmentImage(); img != "" {
		d.Field("Image", img)
	}
	if compute := build.ComputeType(); compute != "" {
		d.Field("Compute Type", compute)
	}

	// Phases
	if phases := build.Phases(); len(phases) > 0 {
		d.Section("Build Phases")
		for _, phase := range phases {
			status := string(phase.PhaseStatus)
			if status == "" {
				status = "IN_PROGRESS"
			}
			duration := ""
			if phase.DurationInSeconds != nil {
				duration = fmt.Sprintf(" (%ds)", *phase.DurationInSeconds)
			}
			d.Field(string(phase.PhaseType), status+duration)
		}
	}

	// Logs
	if logGroup := build.LogsGroupName(); logGroup != "" {
		d.Section("CloudWatch Logs")
		d.Field("Log Group", logGroup)
		if stream := build.LogsStreamName(); stream != "" {
			d.Field("Log Stream", stream)
		}
	}

	// Timestamps
	d.Section("Timestamps")
	if started := build.StartTime(); started != "" {
		d.Field("Started", started)
	}
	if ended := build.EndTime(); ended != "" {
		d.Field("Ended", ended)
	}
	if dur := build.Duration(); dur != "" {
		d.Field("Duration", dur)
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *BuildRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	build, ok := resource.(*BuildResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Build #", Value: fmt.Sprintf("%d", build.BuildNumber())},
		{Label: "Project", Value: build.ProjectName},
		{Label: "Status", Value: build.Status()},
		{Label: "Phase", Value: build.CurrentPhase()},
	}

	if started := build.StartTime(); started != "" {
		fields = append(fields, render.SummaryField{Label: "Started", Value: started})
	}

	if dur := build.Duration(); dur != "" {
		fields = append(fields, render.SummaryField{Label: "Duration", Value: dur})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *BuildRenderer) Navigations(resource dao.Resource) []render.Navigation {
	return nil
}
