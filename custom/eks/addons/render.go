package addons

import (
	"encoding/json"
	"fmt"
	"strings"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// AddonRenderer renders EKS add-on resources
type AddonRenderer struct {
	render.BaseRenderer
}

// NewAddonRenderer creates a new AddonRenderer
func NewAddonRenderer() render.Renderer {
	return &AddonRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "eks",
			Resource: "addons",
			Cols: []render.Column{
				{
					Name:     "NAME",
					Width:    30,
					Priority: 0,
					Getter: func(r dao.Resource) string {
						return r.GetName()
					},
				},
				{
					Name:     "VERSION",
					Width:    20,
					Priority: 1,
					Getter: func(r dao.Resource) string {
						if ar, ok := r.(*AddonResource); ok {
							return ar.Version()
						}
						return ""
					},
				},
				{
					Name:     "STATUS",
					Width:    15,
					Priority: 2,
					Getter: func(r dao.Resource) string {
						if ar, ok := r.(*AddonResource); ok {
							return ar.Status()
						}
						return ""
					},
				},
				{
					Name:     "AGE",
					Width:    10,
					Priority: 3,
					Getter: func(r dao.Resource) string {
						if ar, ok := r.(*AddonResource); ok {
							return ar.CreatedAge()
						}
						return ""
					},
				},
			},
		},
	}
}

func (rnd *AddonRenderer) RenderDetail(resource dao.Resource) string {
	ar, ok := resource.(*AddonResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()
	d.Title("EKS Add-on", ar.GetName())

	// Basic Information
	d.Section("Basic Information")
	d.Field("Name", ar.GetName())
	d.Field("ARN", ar.GetARN())
	d.Field("Cluster", appaws.Str(ar.Addon.ClusterName))
	d.Field("Status", ar.Status())
	d.Field("Version", ar.Version())
	d.Field("Created", ar.CreatedAge())
	if modified := appaws.Time(ar.Addon.ModifiedAt); !modified.IsZero() {
		d.Field("Modified", render.FormatAge(modified))
	}

	// Configuration
	d.Section("Configuration")
	d.Field("Service Account Role", appaws.Str(ar.Addon.ServiceAccountRoleArn))
	if publisher := appaws.Str(ar.Addon.Publisher); publisher != "" {
		d.Field("Publisher", publisher)
	}
	if owner := appaws.Str(ar.Addon.Owner); owner != "" {
		d.Field("Owner", owner)
	}
	if mi := ar.Addon.MarketplaceInformation; mi != nil {
		d.Field("Product ID", appaws.Str(mi.ProductId))
		d.Field("Product URL", appaws.Str(mi.ProductUrl))
	}

	// Configuration Values
	if config := appaws.Str(ar.Addon.ConfigurationValues); config != "" {
		d.Section("Configuration Values")
		d.Line(config)
	}

	// Pod Identity Associations
	if len(ar.Addon.PodIdentityAssociations) > 0 {
		d.Section("Pod Identity Associations")
		for i, pia := range ar.Addon.PodIdentityAssociations {
			d.Field(fmt.Sprintf("Association #%d", i+1), pia)
		}
	}

	// Health Issues
	if health := ar.Addon.Health; health != nil && len(health.Issues) > 0 {
		d.Section("Health Issues")
		for i, issue := range health.Issues {
			d.Field(fmt.Sprintf("Issue #%d Code", i+1), string(issue.Code))
			if msg := appaws.Str(issue.Message); msg != "" {
				d.Field(fmt.Sprintf("Issue #%d Message", i+1), msg)
			}
			if len(issue.ResourceIds) > 0 {
				d.Field(fmt.Sprintf("Issue #%d Resources", i+1), strings.Join(issue.ResourceIds, ", "))
			}
		}
	}

	// Tags
	if len(ar.Addon.Tags) > 0 {
		d.Section("Tags")
		d.Tags(ar.Addon.Tags)
	}

	// Full Details
	d.Section("Full Details")
	if jsonBytes, err := json.MarshalIndent(ar.Addon, "", "  "); err == nil {
		d.Line(string(jsonBytes))
	}

	return d.String()
}

func (rnd *AddonRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	ar, ok := resource.(*AddonResource)
	if !ok {
		return nil
	}

	return []render.SummaryField{
		{Label: "Name", Value: ar.GetName()},
		{Label: "Version", Value: ar.Version()},
		{Label: "Status", Value: ar.Status()},
	}
}
