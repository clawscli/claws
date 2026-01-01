package instanceprofiles

import (
	"time"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// Ensure InstanceProfileRenderer implements render.Navigator
var _ render.Navigator = (*InstanceProfileRenderer)(nil)

// InstanceProfileRenderer renders IAM Instance Profiles
type InstanceProfileRenderer struct {
	render.BaseRenderer
}

// NewInstanceProfileRenderer creates a new InstanceProfileRenderer
func NewInstanceProfileRenderer() render.Renderer {
	return &InstanceProfileRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "iam",
			Resource: "instance-profiles",
			Cols: []render.Column{
				{
					Name:  "NAME",
					Width: 35,
					Getter: func(r dao.Resource) string {
						return r.GetName()
					},
					Priority: 0,
				},
				{
					Name:  "ID",
					Width: 22,
					Getter: func(r dao.Resource) string {
						return r.GetID()
					},
					Priority: 1,
				},
				{
					Name:  "ROLES",
					Width: 35,
					Getter: func(r dao.Resource) string {
						if ip, ok := r.(*InstanceProfileResource); ok {
							return ip.RoleNamesString()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "PATH",
					Width: 20,
					Getter: func(r dao.Resource) string {
						if ip, ok := r.(*InstanceProfileResource); ok {
							return ip.Path()
						}
						return ""
					},
					Priority: 3,
				},
			},
		},
	}
}

// RenderDetail renders detailed instance profile information
func (r *InstanceProfileRenderer) RenderDetail(resource dao.Resource) string {
	ip, ok := resource.(*InstanceProfileResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Instance Profile", ip.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Name", ip.GetName())
	d.Field("ID", ip.GetID())
	d.Field("ARN", ip.GetARN())
	d.Field("Path", ip.Path())
	if ip.Item.CreateDate != nil {
		d.Field("Created", ip.Item.CreateDate.Format(time.RFC3339))
	}

	// Associated Roles
	d.Section("Associated Roles")
	if len(ip.Item.Roles) > 0 {
		for _, role := range ip.Item.Roles {
			if role.RoleName != nil {
				roleInfo := *role.RoleName
				if role.Arn != nil {
					roleInfo += " (" + *role.Arn + ")"
				}
				d.Line("  " + roleInfo)
			}
		}
	} else {
		d.DimIndent("(none)")
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *InstanceProfileRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	ip, ok := resource.(*InstanceProfileResource)
	if !ok {
		return nil
	}

	fields := []render.SummaryField{
		{Label: "Name", Value: ip.GetName()},
		{Label: "ID", Value: ip.GetID()},
		{Label: "Path", Value: ip.Path()},
		{Label: "Roles", Value: ip.RoleNamesString()},
	}

	if ip.Item.CreateDate != nil {
		fields = append(fields, render.SummaryField{
			Label: "Created",
			Value: ip.Item.CreateDate.Format("2006-01-02"),
		})
	}

	return fields
}

// Navigations returns navigation shortcuts for Instance Profile resources
func (r *InstanceProfileRenderer) Navigations(resource dao.Resource) []render.Navigation {
	ip, ok := resource.(*InstanceProfileResource)
	if !ok {
		return nil
	}

	var navs []render.Navigation

	// Navigate to associated role (if exactly one role)
	if len(ip.Item.Roles) == 1 && ip.Item.Roles[0].RoleName != nil {
		navs = append(navs, render.Navigation{
			Key: "r", Label: "Role", Service: "iam", Resource: "roles",
			FilterField: "RoleName", FilterValue: *ip.Item.Roles[0].RoleName,
		})
	}

	return navs
}
