package stagesv2

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// StageV2Renderer renders API Gateway HTTP/WebSocket API stages (v2)
type StageV2Renderer struct {
	render.BaseRenderer
}

// NewStageV2Renderer creates a new StageV2Renderer
func NewStageV2Renderer() render.Renderer {
	return &StageV2Renderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "apigateway",
			Resource: "stages-v2",
			Cols: []render.Column{
				{
					Name:  "STAGE",
					Width: 20,
					Getter: func(r dao.Resource) string {
						return r.GetName()
					},
					Priority: 0,
				},
				{
					Name:  "DEPLOYMENT",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*StageV2Resource); ok {
							return rr.DeploymentId()
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "AUTO DEPLOY",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*StageV2Resource); ok {
							if rr.AutoDeploy() {
								return "Yes"
							}
							return "No"
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "MANAGED",
					Width: 8,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*StageV2Resource); ok {
							if rr.ApiGatewayManaged() {
								return "Yes"
							}
							return "No"
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "LOGS",
					Width: 6,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*StageV2Resource); ok {
							if rr.HasAccessLogs() {
								return "Yes"
							}
							return "No"
						}
						return ""
					},
					Priority: 4,
				},
				{
					Name:  "UPDATED",
					Width: 20,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*StageV2Resource); ok {
							t := rr.LastUpdatedDate()
							if !t.IsZero() {
								return t.Format("2006-01-02 15:04")
							}
						}
						return ""
					},
					Priority: 5,
				},
			},
		},
	}
}

// RenderDetail renders detailed stage information
func (r *StageV2Renderer) RenderDetail(resource dao.Resource) string {
	rr, ok := resource.(*StageV2Resource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("API Gateway Stage (HTTP/WebSocket)", rr.StageName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Stage Name", rr.StageName())
	d.Field("API ID", rr.ApiId)
	d.Field("Deployment ID", rr.DeploymentId())
	if rr.Description() != "" {
		d.Field("Description", rr.Description())
	}
	d.Field("Auto Deploy", fmt.Sprintf("%v", rr.AutoDeploy()))
	d.Field("API Gateway Managed", fmt.Sprintf("%v", rr.ApiGatewayManaged()))
	d.Field("Created", rr.CreatedDate().Format("2006-01-02 15:04:05 MST"))
	d.Field("Last Updated", rr.LastUpdatedDate().Format("2006-01-02 15:04:05 MST"))

	// Throttling
	d.Section("Default Route Settings")
	d.Field("Throttling Burst Limit", fmt.Sprintf("%d", rr.ThrottlingBurstLimit()))
	d.Field("Throttling Rate Limit", fmt.Sprintf("%.2f", rr.ThrottlingRateLimit()))

	// Logging
	d.Section("Observability")
	d.Field("Access Logs", fmt.Sprintf("%v", rr.HasAccessLogs()))
	if rr.HasAccessLogs() {
		d.Field("Log Destination", rr.AccessLogDestination())
	}

	// Stage Variables
	if len(rr.StageVariables()) > 0 {
		d.Section("Stage Variables")
		for k, v := range rr.StageVariables() {
			d.Field(k, v)
		}
	}

	// Tags
	if len(rr.GetTags()) > 0 {
		d.Section("Tags")
		for k, v := range rr.GetTags() {
			d.Field(k, v)
		}
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *StageV2Renderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	rr, ok := resource.(*StageV2Resource)
	if !ok {
		return nil
	}

	autoDeployStatus := "Disabled"
	if rr.AutoDeploy() {
		autoDeployStatus = "Enabled"
	}

	logsStatus := "Disabled"
	if rr.HasAccessLogs() {
		logsStatus = "Enabled"
	}

	return []render.SummaryField{
		{Label: "Stage", Value: rr.StageName()},
		{Label: "Deployment", Value: rr.DeploymentId()},
		{Label: "Auto Deploy", Value: autoDeployStatus},
		{Label: "Logs", Value: logsStatus},
	}
}
