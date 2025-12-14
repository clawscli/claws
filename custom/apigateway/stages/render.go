package stages

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// StageRenderer renders API Gateway stages
type StageRenderer struct {
	render.BaseRenderer
}

// NewStageRenderer creates a new StageRenderer
func NewStageRenderer() render.Renderer {
	return &StageRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "apigateway",
			Resource: "stages",
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
						if rr, ok := r.(*StageResource); ok {
							return rr.DeploymentId()
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "CACHE",
					Width: 10,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*StageResource); ok {
							if rr.CacheClusterEnabled() {
								return rr.CacheClusterSize()
							}
							return "Disabled"
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "TRACING",
					Width: 8,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*StageResource); ok {
							if rr.TracingEnabled() {
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
						if rr, ok := r.(*StageResource); ok {
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
						if rr, ok := r.(*StageResource); ok {
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
func (r *StageRenderer) RenderDetail(resource dao.Resource) string {
	rr, ok := resource.(*StageResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("API Gateway Stage", rr.StageName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Stage Name", rr.StageName())
	d.Field("REST API ID", rr.RestApiId)
	d.Field("Deployment ID", rr.DeploymentId())
	if rr.Description() != "" {
		d.Field("Description", rr.Description())
	}
	d.Field("Created", rr.CreatedDate().Format("2006-01-02 15:04:05 MST"))
	d.Field("Last Updated", rr.LastUpdatedDate().Format("2006-01-02 15:04:05 MST"))

	// Cache
	d.Section("Cache Settings")
	d.Field("Cache Enabled", fmt.Sprintf("%v", rr.CacheClusterEnabled()))
	if rr.CacheClusterEnabled() {
		d.Field("Cache Size", rr.CacheClusterSize())
		d.Field("Cache Status", rr.CacheClusterStatus())
	}

	// Tracing & Logging
	d.Section("Observability")
	d.Field("X-Ray Tracing", fmt.Sprintf("%v", rr.TracingEnabled()))
	d.Field("Access Logs", fmt.Sprintf("%v", rr.HasAccessLogs()))
	if rr.HasAccessLogs() {
		d.Field("Log Destination", rr.AccessLogDestination())
	}

	// WAF
	if rr.WebAclArn() != "" {
		d.Section("Security")
		d.Field("WAF Web ACL", rr.WebAclArn())
	}

	// Stage Variables
	if len(rr.Variables()) > 0 {
		d.Section("Stage Variables")
		for k, v := range rr.Variables() {
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
func (r *StageRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	rr, ok := resource.(*StageResource)
	if !ok {
		return nil
	}

	cacheStatus := "Disabled"
	if rr.CacheClusterEnabled() {
		cacheStatus = rr.CacheClusterSize()
	}

	tracingStatus := "Disabled"
	if rr.TracingEnabled() {
		tracingStatus = "Enabled"
	}

	return []render.SummaryField{
		{Label: "Stage", Value: rr.StageName()},
		{Label: "Deployment", Value: rr.DeploymentId()},
		{Label: "Cache", Value: cacheStatus},
		{Label: "X-Ray", Value: tracingStatus},
	}
}
