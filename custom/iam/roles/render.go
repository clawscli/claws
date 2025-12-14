package roles

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// RoleRenderer renders IAM Roles
type RoleRenderer struct {
	render.BaseRenderer
}

// NewRoleRenderer creates a new RoleRenderer
func NewRoleRenderer() render.Renderer {
	return &RoleRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "iam",
			Resource: "roles",
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
						if rr, ok := r.(*RoleResource); ok {
							return rr.Path()
						}
						return ""
					},
					Priority: 1,
				},
				{
					Name:  "CREATED",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*RoleResource); ok {
							if rr.Item.CreateDate != nil {
								return render.FormatAge(*rr.Item.CreateDate)
							}
						}
						return ""
					},
					Priority: 2,
				},
				{
					Name:  "MAX SESSION",
					Width: 12,
					Getter: func(r dao.Resource) string {
						if rr, ok := r.(*RoleResource); ok {
							duration := rr.MaxSessionDuration()
							if duration > 0 {
								return fmt.Sprintf("%dh", duration/3600)
							}
						}
						return ""
					},
					Priority: 3,
				},
			},
		},
	}
}

// RenderDetail renders detailed role information
func (r *RoleRenderer) RenderDetail(resource dao.Resource) string {
	rr, ok := resource.(*RoleResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("IAM Role", rr.GetName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Role Name", rr.GetName())
	d.Field("Path", rr.Path())
	d.Field("ARN", rr.Arn())

	if rr.Item.RoleId != nil {
		d.Field("Role ID", *rr.Item.RoleId)
	}

	if rr.Item.Description != nil && *rr.Item.Description != "" {
		d.Field("Description", *rr.Item.Description)
	}

	// Session & Creation
	d.Section("Session Settings")
	d.Field("Max Session Duration", fmt.Sprintf("%d seconds (%d hours)", rr.MaxSessionDuration(), rr.MaxSessionDuration()/3600))

	if rr.Item.CreateDate != nil {
		d.Field("Created", rr.Item.CreateDate.Format(time.RFC3339))
		d.Field("Age", render.FormatAge(*rr.Item.CreateDate))
	}

	// Last Used
	if rr.Item.RoleLastUsed != nil {
		d.Section("Last Used")
		if rr.Item.RoleLastUsed.LastUsedDate != nil {
			d.Field("Last Used", rr.Item.RoleLastUsed.LastUsedDate.Format(time.RFC3339))
			d.Field("Last Used Age", render.FormatAge(*rr.Item.RoleLastUsed.LastUsedDate)+" ago")
		} else {
			d.Field("Last Used", "Never")
		}
		if rr.Item.RoleLastUsed.Region != nil {
			d.Field("Region", *rr.Item.RoleLastUsed.Region)
		}
	}

	// Attached Policies
	d.Section("Attached Policies")
	if len(rr.AttachedPolicies) == 0 && len(rr.InlinePolicies) == 0 {
		d.Field("Policies", "None")
	} else {
		if len(rr.AttachedPolicies) > 0 {
			d.Field("Managed Policies", fmt.Sprintf("%d", len(rr.AttachedPolicies)))
			for _, policy := range rr.AttachedPolicies {
				d.Field("  Policy", appaws.Str(policy.PolicyName))
			}
		}
		if len(rr.InlinePolicies) > 0 {
			d.Field("Inline Policies", fmt.Sprintf("%d", len(rr.InlinePolicies)))
			for _, name := range rr.InlinePolicies {
				d.Field("  Policy", name)
			}
		}
	}

	// Trust Policy (AssumeRolePolicyDocument)
	if rr.Item.AssumeRolePolicyDocument != nil && *rr.Item.AssumeRolePolicyDocument != "" {
		d.Section("Trust Relationship (AssumeRolePolicyDocument)")
		d.Line(formatPolicyDocument(*rr.Item.AssumeRolePolicyDocument))
	}

	// Permissions Boundary
	if rr.Item.PermissionsBoundary != nil {
		d.Section("Permissions Boundary")
		if rr.Item.PermissionsBoundary.PermissionsBoundaryArn != nil {
			d.Field("ARN", *rr.Item.PermissionsBoundary.PermissionsBoundaryArn)
		}
	}

	// Tags
	d.Tags(appaws.TagsToMap(rr.Item.Tags))

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *RoleRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	rr, ok := resource.(*RoleResource)
	if !ok {
		return nil
	}

	fields := []render.SummaryField{
		{Label: "Role Name", Value: rr.GetName()},
		{Label: "Path", Value: rr.Path()},
		{Label: "ARN", Value: rr.Arn()},
		{Label: "Max Session", Value: fmt.Sprintf("%dh", rr.MaxSessionDuration()/3600)},
	}

	if rr.Item.CreateDate != nil {
		fields = append(fields, render.SummaryField{
			Label: "Created",
			Value: render.FormatAge(*rr.Item.CreateDate),
		})
	}

	// Add trust policy summary
	if rr.Item.AssumeRolePolicyDocument != nil {
		if summary := trustPolicySummary(*rr.Item.AssumeRolePolicyDocument); summary != "" {
			fields = append(fields, render.SummaryField{
				Label: "Trusted By",
				Value: summary,
			})
		}
	}

	return fields
}

// formatPolicyDocument decodes URL-encoded policy and formats it as indented JSON
func formatPolicyDocument(encoded string) string {
	// URL decode
	decoded, err := url.QueryUnescape(encoded)
	if err != nil {
		return encoded // Return as-is if decode fails
	}

	// Pretty print JSON
	var obj any
	if err := json.Unmarshal([]byte(decoded), &obj); err != nil {
		return decoded // Return decoded but unformatted if not valid JSON
	}

	pretty, err := json.MarshalIndent(obj, "  ", "  ")
	if err != nil {
		return decoded
	}

	return "  " + string(pretty)
}

// trustPolicySummary extracts a human-readable summary from AssumeRolePolicyDocument
func trustPolicySummary(encoded string) string {
	decoded, err := url.QueryUnescape(encoded)
	if err != nil {
		return ""
	}

	var policy struct {
		Statement []struct {
			Effect    string `json:"Effect"`
			Principal struct {
				Service   any    `json:"Service"`
				AWS       any    `json:"AWS"`
				Federated string `json:"Federated"`
			} `json:"Principal"`
			Action    any `json:"Action"`
			Condition any `json:"Condition"`
		} `json:"Statement"`
	}

	if err := json.Unmarshal([]byte(decoded), &policy); err != nil {
		return ""
	}

	var principals []string
	for _, stmt := range policy.Statement {
		if stmt.Effect != "Allow" {
			continue
		}
		// Service principals
		switch v := stmt.Principal.Service.(type) {
		case string:
			principals = append(principals, "Service: "+v)
		case []any:
			for _, s := range v {
				if svc, ok := s.(string); ok {
					principals = append(principals, "Service: "+svc)
				}
			}
		}
		// AWS principals
		switch v := stmt.Principal.AWS.(type) {
		case string:
			principals = append(principals, "AWS: "+v)
		case []any:
			for _, a := range v {
				if acct, ok := a.(string); ok {
					principals = append(principals, "AWS: "+acct)
				}
			}
		}
		// Federated principals
		if stmt.Principal.Federated != "" {
			principals = append(principals, "Federated: "+stmt.Principal.Federated)
		}
	}

	if len(principals) == 0 {
		return ""
	}

	// Limit to first 3 principals for summary
	if len(principals) > 3 {
		return fmt.Sprintf("%s (+%d more)", principals[0], len(principals)-1)
	}

	result := ""
	for i, p := range principals {
		if i > 0 {
			result += ", "
		}
		result += p
	}
	return result
}
