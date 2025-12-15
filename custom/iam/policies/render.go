package policies

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// PolicyRenderer renders IAM Policies
type PolicyRenderer struct {
	render.BaseRenderer
}

// NewPolicyRenderer creates a new PolicyRenderer
func NewPolicyRenderer() render.Renderer {
	return &PolicyRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "iam",
			Resource: "policies",
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
					Name:  "PATH",
					Width: 20,
					Getter: func(r dao.Resource) string {
						if pr, ok := r.(*PolicyResource); ok {
							return pr.Path()
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "SCOPE",
					Width: 8,
					Getter: func(r dao.Resource) string {
						if pr, ok := r.(*PolicyResource); ok {
							return pr.Scope()
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "ATTACHED",
					Width: 10,
					Getter: func(r dao.Resource) string {
						if pr, ok := r.(*PolicyResource); ok {
							return fmt.Sprintf("%d", pr.AttachmentCount())
						}
						return ""
					},
					Priority: 3,
				},
				{
					Name:  "CREATED",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if pr, ok := r.(*PolicyResource); ok {
							if pr.Item.CreateDate != nil {
								return render.FormatAge(*pr.Item.CreateDate)
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

// RenderDetail renders detailed policy information
func (r *PolicyRenderer) RenderDetail(resource dao.Resource) string {
	pr, ok := resource.(*PolicyResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("IAM Policy", pr.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Policy Name", pr.GetName())
	d.Field("Policy ID", pr.PolicyId())
	d.Field("Path", pr.Path())
	d.Field("ARN", pr.Arn())
	d.Field("Scope", pr.Scope())

	if pr.Item.Description != nil && *pr.Item.Description != "" {
		d.Field("Description", *pr.Item.Description)
	}

	// Attachment Info
	d.Section("Attachment")
	d.Field("Attachable", fmt.Sprintf("%v", pr.IsAttachable()))
	d.Field("Attachment Count", fmt.Sprintf("%d", pr.AttachmentCount()))

	if pr.Item.PermissionsBoundaryUsageCount != nil {
		d.Field("Permissions Boundary Usage", fmt.Sprintf("%d", *pr.Item.PermissionsBoundaryUsageCount))
	}

	// Version Info
	d.Section("Versioning")
	if pr.Item.DefaultVersionId != nil {
		d.Field("Default Version", *pr.Item.DefaultVersionId)
	}

	// Dates
	d.Section("Timeline")
	if pr.Item.CreateDate != nil {
		d.Field("Created", pr.Item.CreateDate.Format(time.RFC3339))
		d.Field("Age", render.FormatAge(*pr.Item.CreateDate))
	}
	if pr.Item.UpdateDate != nil {
		d.Field("Last Updated", pr.Item.UpdateDate.Format(time.RFC3339))
	}

	// Attached Entities
	if len(pr.AttachedUsers) > 0 || len(pr.AttachedRoles) > 0 || len(pr.AttachedGroups) > 0 {
		d.Section("Attached To")
		if len(pr.AttachedUsers) > 0 {
			d.Field("Users", fmt.Sprintf("%d", len(pr.AttachedUsers)))
			for _, user := range pr.AttachedUsers {
				d.Field("  User", appaws.Str(user.UserName))
			}
		}
		if len(pr.AttachedRoles) > 0 {
			d.Field("Roles", fmt.Sprintf("%d", len(pr.AttachedRoles)))
			for _, role := range pr.AttachedRoles {
				d.Field("  Role", appaws.Str(role.RoleName))
			}
		}
		if len(pr.AttachedGroups) > 0 {
			d.Field("Groups", fmt.Sprintf("%d", len(pr.AttachedGroups)))
			for _, group := range pr.AttachedGroups {
				d.Field("  Group", appaws.Str(group.GroupName))
			}
		}
	}

	// Policy Document
	if pr.PolicyDocument != "" {
		d.Section("Policy Document")
		d.Line(formatPolicyDoc(pr.PolicyDocument))
	}

	// Tags
	d.Tags(appaws.TagsToMap(pr.Item.Tags))

	return d.String()
}

// formatPolicyDoc decodes URL-encoded policy and formats it as indented JSON
func formatPolicyDoc(encoded string) string {
	// URL decode
	decoded, err := url.QueryUnescape(encoded)
	if err != nil {
		return encoded
	}

	// Pretty print JSON
	var obj any
	if err := json.Unmarshal([]byte(decoded), &obj); err != nil {
		return decoded
	}

	pretty, err := json.MarshalIndent(obj, "  ", "  ")
	if err != nil {
		return decoded
	}

	return "  " + string(pretty)
}

// RenderSummary returns summary fields for the header panel
func (r *PolicyRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	pr, ok := resource.(*PolicyResource)
	if !ok {
		return nil
	}

	fields := []render.SummaryField{
		{Label: "Policy Name", Value: pr.GetName()},
		{Label: "Policy ID", Value: pr.PolicyId()},
		{Label: "Path", Value: pr.Path()},
		{Label: "Scope", Value: pr.Scope()},
		{Label: "Attachments", Value: fmt.Sprintf("%d", pr.AttachmentCount())},
		{Label: "ARN", Value: pr.Arn()},
	}

	if pr.Item.CreateDate != nil {
		fields = append(fields, render.SummaryField{
			Label: "Created",
			Value: render.FormatAge(*pr.Item.CreateDate),
		})
	}

	return fields
}
