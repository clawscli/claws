package configurations

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// ConfigurationRenderer renders License Manager configurations.
type ConfigurationRenderer struct {
	render.BaseRenderer
}

// NewConfigurationRenderer creates a new ConfigurationRenderer.
func NewConfigurationRenderer() render.Renderer {
	return &ConfigurationRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "license-manager",
			Resource: "configurations",
			Cols: []render.Column{
				{Name: "NAME", Width: 35, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "TYPE", Width: 15, Getter: getType},
				{Name: "LICENSES", Width: 12, Getter: getLicenses},
				{Name: "CONSUMED", Width: 12, Getter: getConsumed},
				{Name: "STATUS", Width: 12, Getter: getStatus},
			},
		},
	}
}

func getType(r dao.Resource) string {
	config, ok := r.(*ConfigurationResource)
	if !ok {
		return ""
	}
	return config.LicenseCountingType()
}

func getLicenses(r dao.Resource) string {
	config, ok := r.(*ConfigurationResource)
	if !ok {
		return ""
	}
	count := config.LicenseCount()
	if count == 0 {
		return "Unlimited"
	}
	return fmt.Sprintf("%d", count)
}

func getConsumed(r dao.Resource) string {
	config, ok := r.(*ConfigurationResource)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%d", config.ConsumedLicenses())
}

func getStatus(r dao.Resource) string {
	config, ok := r.(*ConfigurationResource)
	if !ok {
		return ""
	}
	return config.Status()
}

// RenderDetail renders the detail view for a configuration.
func (r *ConfigurationRenderer) RenderDetail(resource dao.Resource) string {
	config, ok := resource.(*ConfigurationResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("License Configuration", config.GetID())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", config.Name())
	d.Field("ARN", config.GetARN())
	if config.Description() != "" {
		d.Field("Description", config.Description())
	}
	d.Field("Status", config.Status())

	// License Info
	d.Section("License Information")
	d.Field("Counting Type", config.LicenseCountingType())
	if config.LicenseCount() > 0 {
		d.Field("License Count", fmt.Sprintf("%d", config.LicenseCount()))
	} else {
		d.Field("License Count", "Unlimited")
	}
	d.Field("Consumed Licenses", fmt.Sprintf("%d", config.ConsumedLicenses()))

	return d.String()
}

// RenderSummary renders summary fields for a configuration.
func (r *ConfigurationRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	config, ok := resource.(*ConfigurationResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "Name", Value: config.Name()},
		{Label: "Type", Value: config.LicenseCountingType()},
		{Label: "Status", Value: config.Status()},
	}
}
