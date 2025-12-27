package profile

import (
	"strings"

	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// ProfileRenderer renders local AWS profiles
type ProfileRenderer struct {
	render.BaseRenderer
}

// NewProfileRenderer creates a new ProfileRenderer
func NewProfileRenderer() render.Renderer {
	return &ProfileRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "local",
			Resource: "profile",
			Cols: []render.Column{
				{
					Name:  "NAME",
					Width: 25,
					Getter: func(r dao.Resource) string {
						if pr, ok := r.(*ProfileResource); ok {
							if pr.Data.IsCurrent {
								return pr.Data.Name + " *"
							}
							return pr.Data.Name
						}
						return r.GetName()
					},
					Priority: 0,
				},
				{
					Name:  "TYPE",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if pr, ok := r.(*ProfileResource); ok {
							return getProfileType(pr.Data)
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "REGION",
					Width: 15,
					Getter: func(r dao.Resource) string {
						if pr, ok := r.(*ProfileResource); ok {
							return pr.Data.Region
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "SOURCE",
					Width: 20,
					Getter: func(r dao.Resource) string {
						if pr, ok := r.(*ProfileResource); ok {
							if pr.Data.SourceProfile != "" {
								return pr.Data.SourceProfile
							}
							if pr.Data.SSOSession != "" {
								return pr.Data.SSOSession
							}
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "ACCESS KEY",
					Width: 20,
					Getter: func(r dao.Resource) string {
						if pr, ok := r.(*ProfileResource); ok {
							return maskAccessKey(pr.Data.AccessKeyID)
						}
						return ""
					},
					Priority: 4,
				},
			},
		},
	}
}

func getProfileType(data *ProfileData) string {
	if data.ID == config.EnvOnly().ID() {
		return "Env/IMDS"
	}
	if data.SSOStartURL != "" || data.SSOSession != "" {
		return "SSO"
	}
	if data.RoleArn != "" {
		return "AssumeRole"
	}
	if data.HasCredentials {
		return "Static"
	}
	return "Default"
}

func maskAccessKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

// RenderDetail renders detailed profile information
func (r *ProfileRenderer) RenderDetail(resource dao.Resource) string {
	pr, ok := resource.(*ProfileResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()
	data := pr.Data

	// Special handling for Env/IMDS Only option
	if data.ID == config.EnvOnly().ID() {
		title := config.EnvOnly().DisplayName()
		if data.IsCurrent {
			title += " (current)"
		}
		d.Title("AWS Profile", title)
		d.Section("Description")
		d.Field("Purpose", "Ignore ~/.aws config files and use environment credentials")
		d.Field("Behavior", "Uses environment variables, instance profile, ECS task role, Lambda execution role, etc.")
		d.Field("Note", "Useful when ~/.aws/config has a [default] section that interferes with IMDS")
		return d.String()
	}

	title := data.Name
	if data.IsCurrent {
		title += " (current)"
	}
	d.Title("AWS Profile", title)

	// Basic Info
	d.Section("Configuration")
	d.Field("Profile Name", data.Name)
	d.Field("Type", getProfileType(data))
	if data.Region != "" {
		d.Field("Region", data.Region)
	}
	if data.Output != "" {
		d.Field("Output Format", data.Output)
	}

	// Source files
	sources := []string{}
	if data.InConfig {
		sources = append(sources, "~/.aws/config")
	}
	if data.InCredentials {
		sources = append(sources, "~/.aws/credentials")
	}
	if len(sources) > 0 {
		d.Field("Defined In", strings.Join(sources, ", "))
	}

	// Credentials
	if data.HasCredentials {
		d.Section("Credentials")
		d.Field("Access Key ID", maskAccessKey(data.AccessKeyID))
		d.Field("Secret Key", "********")
	}

	// Role assumption
	if data.RoleArn != "" {
		d.Section("Role Assumption")
		d.Field("Role ARN", data.RoleArn)
		if data.SourceProfile != "" {
			d.Field("Source Profile", data.SourceProfile)
		}
		if data.ExternalID != "" {
			d.Field("External ID", data.ExternalID)
		}
		if data.MFASerial != "" {
			d.Field("MFA Serial", data.MFASerial)
		}
		if data.RoleSessionName != "" {
			d.Field("Session Name", data.RoleSessionName)
		}
		if data.DurationSeconds != "" {
			d.Field("Duration", data.DurationSeconds+"s")
		}
	}

	// SSO settings
	if data.SSOStartURL != "" || data.SSOSession != "" {
		d.Section("SSO Configuration")
		if data.SSOSession != "" {
			d.Field("SSO Session", data.SSOSession)
		}
		if data.SSOStartURL != "" {
			d.Field("Start URL", data.SSOStartURL)
		}
		if data.SSORegion != "" {
			d.Field("SSO Region", data.SSORegion)
		}
		if data.SSOAccountID != "" {
			d.Field("Account ID", data.SSOAccountID)
		}
		if data.SSORoleName != "" {
			d.Field("Role Name", data.SSORoleName)
		}
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *ProfileRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	pr, ok := resource.(*ProfileResource)
	if !ok {
		return nil
	}

	data := pr.Data
	fields := []render.SummaryField{
		{Label: "Profile", Value: data.Name},
		{Label: "Type", Value: getProfileType(data)},
	}

	if data.Region != "" {
		fields = append(fields, render.SummaryField{Label: "Region", Value: data.Region})
	}

	if data.IsCurrent {
		fields = append(fields, render.SummaryField{Label: "Status", Value: "Current"})
	}

	return fields
}
