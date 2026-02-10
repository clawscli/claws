package fleets

import (
	"fmt"
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// Ensure FleetRenderer implements render.Navigator
var _ render.Navigator = (*FleetRenderer)(nil)

// FleetRenderer renders GameLift fleets.
type FleetRenderer struct {
	render.BaseRenderer
}

// NewFleetRenderer creates a new FleetRenderer.
func NewFleetRenderer() render.Renderer {
	return &FleetRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "gamelift",
			Resource: "fleets",
			Cols: []render.Column{
				{Name: "NAME", Width: 30, Getter: func(r dao.Resource) string { return r.GetName() }},
				{Name: "FLEET ID", Width: 24, Getter: func(r dao.Resource) string { return r.GetID() }, Priority: 3},
				{Name: "STATUS", Width: 14, Getter: getFleetStatus},
				{Name: "FLEET TYPE", Width: 12, Getter: getFleetType, Priority: 2},
				{Name: "INSTANCE TYPE", Width: 16, Getter: getInstanceType},
				{Name: "BUILD/SCRIPT", Width: 24, Getter: getBuildOrScript, Priority: 2},
				{Name: "OS", Width: 16, Getter: getOperatingSystem, Priority: 3},
				{Name: "CREATED", Width: 20, Getter: getFleetCreated, Priority: 2},
			},
		},
	}
}

func getFleetStatus(r dao.Resource) string {
	fleet, ok := r.(*FleetResource)
	if !ok {
		return ""
	}
	return fleet.Status()
}

func getFleetType(r dao.Resource) string {
	fleet, ok := r.(*FleetResource)
	if !ok {
		return ""
	}
	return fleet.FleetType()
}

func getInstanceType(r dao.Resource) string {
	fleet, ok := r.(*FleetResource)
	if !ok {
		return ""
	}
	return fleet.InstanceType()
}

func getBuildOrScript(r dao.Resource) string {
	fleet, ok := r.(*FleetResource)
	if !ok {
		return ""
	}
	if id := fleet.BuildId(); id != "" {
		return "build:" + id
	}
	if id := fleet.ScriptId(); id != "" {
		return "script:" + id
	}
	return ""
}

func getOperatingSystem(r dao.Resource) string {
	fleet, ok := r.(*FleetResource)
	if !ok {
		return ""
	}
	return fleet.OperatingSystem()
}

func getFleetCreated(r dao.Resource) string {
	fleet, ok := r.(*FleetResource)
	if !ok {
		return ""
	}
	if t := fleet.CreationTime(); t != nil {
		return t.Format("2006-01-02 15:04")
	}
	return ""
}

// RenderDetail renders the detail view for a GameLift fleet.
func (rr *FleetRenderer) RenderDetail(resource dao.Resource) string {
	fleet, ok := resource.(*FleetResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("GameLift Fleet", fleet.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", fleet.GetName())
	d.Field("Fleet ID", fleet.GetID())
	d.Field("ARN", fleet.GetARN())
	d.Field("Status", fleet.Status())
	if desc := fleet.Description(); desc != "" {
		d.Field("Description", desc)
	}

	// Compute Configuration
	d.Section("Compute Configuration")
	d.Field("Fleet Type", fleet.FleetType())
	d.Field("Compute Type", fleet.ComputeType())
	if it := fleet.InstanceType(); it != "" {
		d.Field("Instance Type", it)
	}
	d.Field("Operating System", fleet.OperatingSystem())

	// Build/Script
	if buildId := fleet.BuildId(); buildId != "" {
		d.Section("Build")
		d.Field("Build ID", buildId)
		if arn := fleet.BuildArn(); arn != "" {
			d.Field("Build ARN", arn)
		}
	}
	if scriptId := fleet.ScriptId(); scriptId != "" {
		d.Section("Script")
		d.Field("Script ID", scriptId)
		if arn := fleet.ScriptArn(); arn != "" {
			d.Field("Script ARN", arn)
		}
	}

	// Security
	d.Section("Security")
	d.Field("Protection Policy", fleet.ProtectionPolicy())
	if role := fleet.InstanceRoleArn(); role != "" {
		d.Field("Instance Role ARN", role)
	}
	if cert := fleet.CertificateType(); cert != "" {
		d.Field("Certificate Type", cert)
	}

	// Metrics
	if groups := fleet.MetricGroups(); len(groups) > 0 {
		d.Section("Monitoring")
		d.Field("Metric Groups", strings.Join(groups, ", "))
	}

	// Stopped Actions
	if actions := fleet.StoppedActions(); len(actions) > 0 {
		d.Section("Stopped Actions")
		d.Field("Actions", strings.Join(actions, ", "))
	}

	// Timestamps
	d.Section("Timestamps")
	if t := fleet.CreationTime(); t != nil {
		d.Field("Created", t.Format("2006-01-02 15:04:05"))
	}
	if t := fleet.TerminationTime(); t != nil {
		d.Field("Terminated", t.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary renders summary fields for a GameLift fleet.
func (rr *FleetRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	fleet, ok := resource.(*FleetResource)
	if !ok {
		return rr.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: fleet.GetName()},
		{Label: "Fleet ID", Value: fleet.GetID()},
		{Label: "ARN", Value: fleet.GetARN()},
		{Label: "Status", Value: fleet.Status()},
		{Label: "Instance Type", Value: fleet.InstanceType()},
	}
	return fields
}

// Navigations returns available navigations from a GameLift fleet.
func (rr *FleetRenderer) Navigations(resource dao.Resource) []render.Navigation {
	fleet, ok := resource.(*FleetResource)
	if !ok {
		return nil
	}

	navs := []render.Navigation{
		{
			Key:         "s",
			Label:       "Game Sessions",
			Service:     "gamelift",
			Resource:    "game-sessions",
			FilterField: "FleetId",
			FilterValue: fleet.GetID(),
		},
	}

	if buildId := fleet.BuildId(); buildId != "" {
		navs = append(navs, render.Navigation{
			Key:         "b",
			Label:       fmt.Sprintf("Build (%s)", buildId),
			Service:     "gamelift",
			Resource:    "builds",
			FilterField: "BuildId",
			FilterValue: buildId,
		})
	}

	if scriptId := fleet.ScriptId(); scriptId != "" {
		navs = append(navs, render.Navigation{
			Key:         "c",
			Label:       fmt.Sprintf("Script (%s)", scriptId),
			Service:     "gamelift",
			Resource:    "scripts",
			FilterField: "ScriptId",
			FilterValue: scriptId,
		})
	}

	return navs
}
