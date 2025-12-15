package users

import (
	"fmt"
	"time"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// UserRenderer renders IAM Users
type UserRenderer struct {
	render.BaseRenderer
}

// NewUserRenderer creates a new UserRenderer
func NewUserRenderer() render.Renderer {
	return &UserRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "iam",
			Resource: "users",
			Cols: []render.Column{
				{
					Name:  "NAME",
					Width: 30,
					Getter: func(r dao.Resource) string {
						return r.GetName()
					},
					Priority: 0,
				},
				{
					Name:  "PATH",
					Width: 20,
					Getter: func(r dao.Resource) string {
						if ur, ok := r.(*UserResource); ok {
							return ur.Path()
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "CREATED",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if ur, ok := r.(*UserResource); ok {
							if ur.Item.CreateDate != nil {
								return render.FormatAge(*ur.Item.CreateDate)
							}
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "LAST USED",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if ur, ok := r.(*UserResource); ok {
							if ur.Item.PasswordLastUsed != nil {
								return render.FormatAge(*ur.Item.PasswordLastUsed)
							}
						}
						return "Never"
					},
					Priority: 3,
				},
				{
					Name:  "USER ID",
					Width: 24,
					Getter: func(r dao.Resource) string {
						if ur, ok := r.(*UserResource); ok {
							return ur.UserId()
						}
						return ""
					},
					Priority: 4,
				},
			},
		},
	}
}

// RenderDetail renders detailed user information
func (r *UserRenderer) RenderDetail(resource dao.Resource) string {
	ur, ok := resource.(*UserResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("IAM User", ur.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("User Name", ur.GetName())
	d.Field("User ID", ur.UserId())
	d.Field("Path", ur.Path())
	d.Field("ARN", ur.Arn())

	// Dates
	d.Section("Activity")
	if ur.Item.CreateDate != nil {
		d.Field("Created", ur.Item.CreateDate.Format(time.RFC3339))
		d.Field("Age", render.FormatAge(*ur.Item.CreateDate))
	}
	if ur.Item.PasswordLastUsed != nil {
		d.Field("Password Last Used", ur.Item.PasswordLastUsed.Format(time.RFC3339))
		d.Field("Last Active", render.FormatAge(*ur.Item.PasswordLastUsed)+" ago")
	} else {
		d.Field("Password Last Used", "Never (no console access or never used)")
	}

	// Access Keys
	d.Section("Access Keys")
	if len(ur.AccessKeys) == 0 {
		d.Field("Access Keys", "None")
	} else {
		d.Field("Access Key Count", fmt.Sprintf("%d", len(ur.AccessKeys)))
		for i, key := range ur.AccessKeys {
			keyInfo := fmt.Sprintf("%s (%s)", appaws.Str(key.AccessKeyId), key.Status)
			if key.CreateDate != nil {
				keyInfo += fmt.Sprintf(", created %s ago", render.FormatAge(*key.CreateDate))
			}
			d.Field(fmt.Sprintf("  Key %d", i+1), keyInfo)
		}
	}

	// MFA Devices
	d.Section("MFA")
	if len(ur.MFADevices) == 0 {
		d.Field("MFA Status", "Not enabled")
	} else {
		d.Field("MFA Status", "Enabled")
		d.Field("MFA Device Count", fmt.Sprintf("%d", len(ur.MFADevices)))
		for i, mfa := range ur.MFADevices {
			serial := appaws.Str(mfa.SerialNumber)
			if mfa.EnableDate != nil {
				d.Field(fmt.Sprintf("  Device %d", i+1), fmt.Sprintf("%s (enabled %s ago)", serial, render.FormatAge(*mfa.EnableDate)))
			} else {
				d.Field(fmt.Sprintf("  Device %d", i+1), serial)
			}
		}
	}

	// Groups
	d.Section("Groups")
	if len(ur.Groups) == 0 {
		d.Field("Groups", "None")
	} else {
		d.Field("Group Count", fmt.Sprintf("%d", len(ur.Groups)))
		for _, group := range ur.Groups {
			d.Field("  Group", appaws.Str(group.GroupName))
		}
	}

	// Attached Policies
	d.Section("Attached Policies")
	if len(ur.AttachedPolicies) == 0 && len(ur.InlinePolicies) == 0 {
		d.Field("Policies", "None")
	} else {
		if len(ur.AttachedPolicies) > 0 {
			d.Field("Managed Policies", fmt.Sprintf("%d", len(ur.AttachedPolicies)))
			for _, policy := range ur.AttachedPolicies {
				d.Field("  Policy", appaws.Str(policy.PolicyName))
			}
		}
		if len(ur.InlinePolicies) > 0 {
			d.Field("Inline Policies", fmt.Sprintf("%d", len(ur.InlinePolicies)))
			for _, name := range ur.InlinePolicies {
				d.Field("  Policy", name)
			}
		}
	}

	// Permissions Boundary
	if ur.Item.PermissionsBoundary != nil {
		d.Section("Permissions Boundary")
		if ur.Item.PermissionsBoundary.PermissionsBoundaryArn != nil {
			d.Field("ARN", *ur.Item.PermissionsBoundary.PermissionsBoundaryArn)
		}
	}

	// Tags
	d.Tags(appaws.TagsToMap(ur.Item.Tags))

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *UserRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	ur, ok := resource.(*UserResource)
	if !ok {
		return nil
	}

	fields := []render.SummaryField{
		{Label: "User Name", Value: ur.GetName()},
		{Label: "User ID", Value: ur.UserId()},
		{Label: "Path", Value: ur.Path()},
		{Label: "ARN", Value: ur.Arn()},
	}

	if ur.Item.CreateDate != nil {
		fields = append(fields, render.SummaryField{
			Label: "Created",
			Value: render.FormatAge(*ur.Item.CreateDate),
		})
	}

	if ur.Item.PasswordLastUsed != nil {
		fields = append(fields, render.SummaryField{
			Label: "Last Active",
			Value: render.FormatAge(*ur.Item.PasswordLastUsed) + " ago",
		})
	}

	return fields
}
