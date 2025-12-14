package launchtemplates

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// LaunchTemplateRenderer renders EC2 Launch Templates
type LaunchTemplateRenderer struct {
	render.BaseRenderer
}

// NewLaunchTemplateRenderer creates a new LaunchTemplateRenderer
func NewLaunchTemplateRenderer() render.Renderer {
	return &LaunchTemplateRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "ec2",
			Resource: "launch-templates",
			Cols: []render.Column{
				{
					Name:  "NAME",
					Width: 40,
					Getter: func(r dao.Resource) string {
						return r.GetName()
					},
					Priority: 0,
				},
				{
					Name:  "ID",
					Width: 24,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*LaunchTemplateResource); ok {
							return rr.LaunchTemplateId()
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "DEFAULT VER",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*LaunchTemplateResource); ok {
							return fmt.Sprintf("%d", rr.DefaultVersionNumber())
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "LATEST VER",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*LaunchTemplateResource); ok {
							return fmt.Sprintf("%d", rr.LatestVersionNumber())
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "CREATED",
					Width: 20,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*LaunchTemplateResource); ok {
							t := rr.CreateTime()
							if !t.IsZero() {
								return t.Format("2006-01-02 15:04")
							}
						}
						return ""
					},
					Priority: 4,
				},
			},
		},
	}
}

// RenderDetail renders detailed Launch Template information
func (r *LaunchTemplateRenderer) RenderDetail(resource dao.Resource) string {
	rr, ok := resource.(*LaunchTemplateResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Launch Template", rr.LaunchTemplateName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", rr.LaunchTemplateName())
	d.Field("ID", rr.LaunchTemplateId())
	d.Field("Default Version", fmt.Sprintf("%d", rr.DefaultVersionNumber()))
	d.Field("Latest Version", fmt.Sprintf("%d", rr.LatestVersionNumber()))
	d.Field("Created By", rr.CreatedBy())
	d.Field("Created", rr.CreateTime().Format("2006-01-02 15:04:05 MST"))

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
func (r *LaunchTemplateRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	rr, ok := resource.(*LaunchTemplateResource)
	if !ok {
		return nil
	}

	return []render.SummaryField{
		{Label: "Name", Value: rr.LaunchTemplateName()},
		{Label: "ID", Value: rr.LaunchTemplateId()},
		{Label: "Default", Value: fmt.Sprintf("v%d", rr.DefaultVersionNumber())},
		{Label: "Latest", Value: fmt.Sprintf("v%d", rr.LatestVersionNumber())},
	}
}
