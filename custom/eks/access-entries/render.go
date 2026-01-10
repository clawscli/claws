package accessentries

import (
	"encoding/json"
	"strings"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// AccessEntryRenderer renders EKS access entry resources
type AccessEntryRenderer struct {
	render.BaseRenderer
}

var _ render.Navigator = (*AccessEntryRenderer)(nil)

// NewAccessEntryRenderer creates a new AccessEntryRenderer
func NewAccessEntryRenderer() render.Renderer {
	return &AccessEntryRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "eks",
			Resource: "access-entries",
			Cols: []render.Column{
				{
					Name:     "PRINCIPAL ARN",
					Width:    60,
					Priority: 0,
					Getter: func(r dao.Resource) string {
						return r.GetName()
					},
				},
				{
					Name:     "TYPE",
					Width:    20,
					Priority: 1,
					Getter: func(r dao.Resource) string {
						if aer, ok := r.(*AccessEntryResource); ok {
							return aer.Type()
						}
						return ""
					},
				},
				{
					Name:     "USERNAME",
					Width:    30,
					Priority: 2,
					Getter: func(r dao.Resource) string {
						if aer, ok := r.(*AccessEntryResource); ok {
							return aer.Username()
						}
						return ""
					},
				},
				{
					Name:     "AGE",
					Width:    10,
					Priority: 3,
					Getter: func(r dao.Resource) string {
						if aer, ok := r.(*AccessEntryResource); ok {
							return aer.CreatedAge()
						}
						return ""
					},
				},
			},
		},
	}
}

func (rnd *AccessEntryRenderer) RenderDetail(resource dao.Resource) string {
	aer, ok := resource.(*AccessEntryResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()
	d.Title("EKS Access Entry", aer.GetName())

	// Basic Information
	d.Section("Basic Information")
	d.Field("Principal ARN", aer.GetName())
	d.Field("Access Entry ARN", aer.GetARN())
	d.Field("Cluster", appaws.Str(aer.AccessEntry.ClusterName))
	d.Field("Type", aer.Type())
	d.Field("Created", aer.CreatedAge())
	if modified := appaws.Time(aer.AccessEntry.ModifiedAt); !modified.IsZero() {
		d.Field("Modified", render.FormatAge(modified))
	}

	// Kubernetes Identity
	d.Section("Kubernetes Identity")
	d.Field("Username", aer.Username())
	if len(aer.AccessEntry.KubernetesGroups) > 0 {
		d.Field("Groups", strings.Join(aer.AccessEntry.KubernetesGroups, ", "))
	} else {
		d.Field("Groups", render.Empty)
	}

	// Tags
	if len(aer.AccessEntry.Tags) > 0 {
		d.Section("Tags")
		d.Tags(aer.AccessEntry.Tags)
	}

	// Full Details
	d.Section("Full Details")
	if jsonBytes, err := json.MarshalIndent(aer.AccessEntry, "", "  "); err == nil {
		d.Line(string(jsonBytes))
	}

	return d.String()
}

func (rnd *AccessEntryRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	aer, ok := resource.(*AccessEntryResource)
	if !ok {
		return nil
	}

	return []render.SummaryField{
		{Label: "Principal", Value: aer.GetName()},
		{Label: "Type", Value: aer.Type()},
		{Label: "Username", Value: aer.Username()},
	}
}

func (rnd *AccessEntryRenderer) Navigations(resource dao.Resource) []render.Navigation {
	aer, ok := resource.(*AccessEntryResource)
	if !ok {
		return nil
	}

	var navs []render.Navigation

	// Parent cluster (always present)
	if clusterName := appaws.Str(aer.AccessEntry.ClusterName); clusterName != "" {
		navs = append(navs, render.Navigation{
			Key:         "p",
			Label:       "Cluster",
			Service:     "eks",
			Resource:    "clusters",
			FilterField: "ClusterName",
			FilterValue: clusterName,
		})
	}

	// IAM Principal (User or Role)
	if principalArn := appaws.Str(aer.AccessEntry.PrincipalArn); principalArn != "" {
		principalName := appaws.ExtractResourceName(principalArn)

		// Determine if it's a role or user from ARN
		var service, resource string
		if strings.Contains(principalArn, ":user/") {
			service = "iam"
			resource = "users"
		} else if strings.Contains(principalArn, ":role/") {
			service = "iam"
			resource = "roles"
		}

		if service != "" {
			// Determine filter field name based on resource type
			var filterField string
			if resource == "users" {
				filterField = "UserName"
			} else {
				filterField = "RoleName"
			}

			navs = append(navs, render.Navigation{
				Key:         "i",
				Label:       "IAM Principal",
				Service:     service,
				Resource:    resource,
				FilterField: filterField,
				FilterValue: principalName,
			})
		}
	}

	return navs
}
