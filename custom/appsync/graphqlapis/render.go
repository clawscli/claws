package graphqlapis

import (
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// GraphQLApiRenderer renders AppSync GraphQL APIs.
// Ensure GraphQLApiRenderer implements render.Navigator
var _ render.Navigator = (*GraphQLApiRenderer)(nil)

type GraphQLApiRenderer struct {
	render.BaseRenderer
}

// NewGraphQLApiRenderer creates a new GraphQLApiRenderer.
func NewGraphQLApiRenderer() render.Renderer {
	return &GraphQLApiRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "appsync",
			Resource: "graphql-apis",
			Cols: []render.Column{
				{Name: "API ID", Width: 30, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "NAME", Width: 30, Getter: getName},
				{Name: "AUTH TYPE", Width: 20, Getter: getAuthType},
				{Name: "API TYPE", Width: 12, Getter: getApiType},
			},
		},
	}
}

func getName(r dao.Resource) string {
	api, ok := r.(*GraphQLApiResource)
	if !ok {
		return ""
	}
	return api.Name()
}

func getAuthType(r dao.Resource) string {
	api, ok := r.(*GraphQLApiResource)
	if !ok {
		return ""
	}
	return api.AuthenticationType()
}

func getApiType(r dao.Resource) string {
	api, ok := r.(*GraphQLApiResource)
	if !ok {
		return ""
	}
	return api.ApiType()
}

// RenderDetail renders the detail view for a GraphQL API.
func (r *GraphQLApiRenderer) RenderDetail(resource dao.Resource) string {
	api, ok := resource.(*GraphQLApiResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()
	a := api.Api

	d.Title("AppSync GraphQL API", api.Name())

	// Basic Info
	d.Section("Basic Information")
	d.Field("API ID", api.GetID())
	d.Field("Name", api.Name())
	d.Field("ARN", api.GetARN())
	d.Field("API Type", api.ApiType())
	if a.Visibility != "" {
		d.Field("Visibility", string(a.Visibility))
	}
	if a.Owner != nil {
		d.Field("Owner", *a.Owner)
	}
	d.Field("Introspection", string(a.IntrospectionConfig))
	if a.QueryDepthLimit > 0 {
		d.Field("Query Depth Limit", fmt.Sprintf("%d", a.QueryDepthLimit))
	}
	if a.ResolverCountLimit > 0 {
		d.Field("Resolver Count Limit", fmt.Sprintf("%d", a.ResolverCountLimit))
	}

	// Authentication
	d.Section("Authentication")
	d.Field("Primary Auth Type", api.AuthenticationType())
	if a.UserPoolConfig != nil {
		d.Field("User Pool ID", *a.UserPoolConfig.UserPoolId)
		if a.UserPoolConfig.AppIdClientRegex != nil {
			d.Field("App ID Regex", *a.UserPoolConfig.AppIdClientRegex)
		}
	}
	if a.OpenIDConnectConfig != nil && a.OpenIDConnectConfig.Issuer != nil {
		d.Field("OIDC Issuer", *a.OpenIDConnectConfig.Issuer)
	}
	if a.LambdaAuthorizerConfig != nil && a.LambdaAuthorizerConfig.AuthorizerUri != nil {
		d.Field("Lambda Authorizer", *a.LambdaAuthorizerConfig.AuthorizerUri)
	}
	if len(a.AdditionalAuthenticationProviders) > 0 {
		d.Field("Additional Auth Providers", fmt.Sprintf("%d configured", len(a.AdditionalAuthenticationProviders)))
		for i, p := range a.AdditionalAuthenticationProviders {
			d.Field(fmt.Sprintf("  Provider %d", i+1), string(p.AuthenticationType))
		}
	}

	// Endpoints
	d.Section("Endpoints")
	if api.Endpoint() != "" {
		d.Field("GraphQL Endpoint", api.Endpoint())
	}
	if a.Uris != nil {
		if realtime, ok := a.Uris["REALTIME"]; ok {
			d.Field("Realtime Endpoint", realtime)
		}
	}
	if len(a.Dns) > 0 {
		for k, v := range a.Dns {
			d.Field("DNS: "+k, v)
		}
	}

	// Logging
	if a.LogConfig != nil {
		d.Section("Logging")
		d.Field("CloudWatch Logs Role", *a.LogConfig.CloudWatchLogsRoleArn)
		d.Field("Field Log Level", string(a.LogConfig.FieldLogLevel))
		d.Field("Exclude Verbose Content", fmt.Sprintf("%v", a.LogConfig.ExcludeVerboseContent))
	}

	// WAF
	if a.WafWebAclArn != nil {
		d.Section("Security")
		d.Field("WAF Web ACL", *a.WafWebAclArn)
	}

	// Monitoring
	d.Section("Monitoring")
	if api.XrayEnabled() {
		d.Field("X-Ray", "Enabled")
	} else {
		d.Field("X-Ray", "Disabled")
	}

	return d.String()
}

// RenderSummary renders summary fields for a GraphQL API.
func (r *GraphQLApiRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	api, ok := resource.(*GraphQLApiResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	return []render.SummaryField{
		{Label: "API ID", Value: api.GetID()},
		{Label: "Name", Value: api.Name()},
		{Label: "Auth Type", Value: api.AuthenticationType()},
	}
}

// Navigations returns available navigations from a GraphQL API.
func (r *GraphQLApiRenderer) Navigations(resource dao.Resource) []render.Navigation {
	api, ok := resource.(*GraphQLApiResource)
	if !ok {
		return nil
	}
	return []render.Navigation{
		{
			Key:         "d",
			Label:       "Data Sources",
			Service:     "appsync",
			Resource:    "data-sources",
			FilterField: "ApiId",
			FilterValue: api.GetID(),
		},
	}
}
