package userpools

import (
	"fmt"
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// UserPoolRenderer renders Cognito user pools
// Ensure UserPoolRenderer implements render.Navigator
var _ render.Navigator = (*UserPoolRenderer)(nil)

type UserPoolRenderer struct {
	render.BaseRenderer
}

// NewUserPoolRenderer creates a new UserPoolRenderer
func NewUserPoolRenderer() *UserPoolRenderer {
	return &UserPoolRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "cognito",
			Resource: "user-pools",
			Cols: []render.Column{
				{Name: "ID", Width: 25, Getter: getPoolId},
				{Name: "NAME", Width: 25, Getter: getPoolName},
				{Name: "STATUS", Width: 10, Getter: getStatus},
				{Name: "AGE", Width: 12, Getter: getAge},
			},
		},
	}
}

func getPoolId(r dao.Resource) string {
	if pool, ok := r.(*UserPoolResource); ok {
		return pool.PoolId()
	}
	return ""
}

func getPoolName(r dao.Resource) string {
	if pool, ok := r.(*UserPoolResource); ok {
		return pool.PoolName()
	}
	return ""
}

func getStatus(r dao.Resource) string {
	if pool, ok := r.(*UserPoolResource); ok {
		return pool.Status()
	}
	return ""
}

func getAge(r dao.Resource) string {
	if pool, ok := r.(*UserPoolResource); ok {
		if pool.Summary != nil && pool.Summary.CreationDate != nil {
			return render.FormatAge(*pool.Summary.CreationDate)
		}
		if pool.Detail != nil && pool.Detail.CreationDate != nil {
			return render.FormatAge(*pool.Detail.CreationDate)
		}
	}
	return "-"
}

// RenderDetail renders detailed user pool information
func (r *UserPoolRenderer) RenderDetail(resource dao.Resource) string {
	pool, ok := resource.(*UserPoolResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Cognito User Pool", pool.PoolName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Pool ID", pool.PoolId())
	d.Field("Pool Name", pool.PoolName())
	if arn := pool.GetARN(); arn != "" {
		d.Field("ARN", arn)
	}
	d.Field("Status", pool.Status())

	// Users
	if users := pool.EstimatedNumberOfUsers(); users > 0 {
		d.Section("Users")
		d.Field("Estimated Users", fmt.Sprintf("%d", users))
	}

	// Domain
	if domain := pool.Domain(); domain != "" {
		d.Section("Domain")
		d.Field("Domain Prefix", domain)
	}
	if customDomain := pool.CustomDomain(); customDomain != "" {
		d.Field("Custom Domain", customDomain)
	}

	// Authentication
	d.Section("Authentication")
	if mfa := pool.MfaConfiguration(); mfa != "" {
		d.Field("MFA Configuration", mfa)
	}

	if usernameAttrs := pool.UsernameAttributes(); len(usernameAttrs) > 0 {
		d.Field("Username Attributes", strings.Join(usernameAttrs, ", "))
	}

	if autoVerify := pool.AutoVerifiedAttributes(); len(autoVerify) > 0 {
		d.Field("Auto-Verified", strings.Join(autoVerify, ", "))
	}

	// Lambda Triggers
	if lambdaConfig := pool.LambdaConfig(); len(lambdaConfig) > 0 {
		d.Section("Lambda Triggers")
		for trigger, fn := range lambdaConfig {
			d.Field(trigger, fn)
		}
	}

	// Protection
	if protection := pool.DeletionProtection(); protection != "" {
		d.Section("Protection")
		d.Field("Deletion Protection", protection)
	}

	// Timestamps
	d.Section("Timestamps")
	if created := pool.CreatedAt(); created != "" {
		d.Field("Created", created)
	}
	if modified := pool.LastModifiedDate(); modified != "" {
		d.Field("Last Modified", modified)
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *UserPoolRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	pool, ok := resource.(*UserPoolResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Pool ID", Value: pool.PoolId()},
		{Label: "Pool Name", Value: pool.PoolName()},
	}

	if arn := pool.GetARN(); arn != "" {
		fields = append(fields, render.SummaryField{Label: "ARN", Value: arn})
	}

	fields = append(fields, render.SummaryField{Label: "Status", Value: pool.Status()})

	if users := pool.EstimatedNumberOfUsers(); users > 0 {
		fields = append(fields, render.SummaryField{Label: "Users", Value: fmt.Sprintf("%d", users)})
	}

	if mfa := pool.MfaConfiguration(); mfa != "" {
		fields = append(fields, render.SummaryField{Label: "MFA", Value: mfa})
	}

	if created := pool.CreatedAt(); created != "" {
		fields = append(fields, render.SummaryField{Label: "Created", Value: created})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *UserPoolRenderer) Navigations(resource dao.Resource) []render.Navigation {
	pool, ok := resource.(*UserPoolResource)
	if !ok {
		return nil
	}

	return []render.Navigation{
		{
			Key: "u", Label: "Users", Service: "cognito", Resource: "users",
			FilterField: "UserPoolId", FilterValue: pool.PoolId(),
		},
	}
}
