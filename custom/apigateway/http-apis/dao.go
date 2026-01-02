package httpapis

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// HttpAPIDAO provides data access for API Gateway HTTP/WebSocket APIs (v2)
type HttpAPIDAO struct {
	dao.BaseDAO
	client *apigatewayv2.Client
}

// NewHttpAPIDAO creates a new HttpAPIDAO
func NewHttpAPIDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new apigateway/http-apis dao")
	}
	return &HttpAPIDAO{
		BaseDAO: dao.NewBaseDAO("apigateway", "http-apis"),
		client:  apigatewayv2.NewFromConfig(cfg),
	}, nil
}

// List returns all HTTP/WebSocket APIs
func (d *HttpAPIDAO) List(ctx context.Context) ([]dao.Resource, error) {
	apis, err := appaws.Paginate(ctx, func(token *string) ([]types.Api, *string, error) {
		output, err := d.client.GetApis(ctx, &apigatewayv2.GetApisInput{
			NextToken:  token,
			MaxResults: appaws.StringPtr("500"),
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list HTTP APIs")
		}
		return output.Items, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(apis))
	for i, api := range apis {
		resources[i] = NewHttpAPIResource(api)
	}

	return resources, nil
}

// Get returns a specific HTTP API by ID
func (d *HttpAPIDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.GetApi(ctx, &apigatewayv2.GetApiInput{
		ApiId: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get HTTP API %s", id)
	}

	return NewHttpAPIResourceFromGetOutput(output), nil
}

// Delete deletes an HTTP API
func (d *HttpAPIDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteApi(ctx, &apigatewayv2.DeleteApiInput{
		ApiId: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete HTTP API %s", id)
	}
	return nil
}

// HttpAPIResource wraps an API Gateway HTTP/WebSocket API (v2)
type HttpAPIResource struct {
	dao.BaseResource
	Item types.Api
}

// NewHttpAPIResource creates a new HttpAPIResource
func NewHttpAPIResource(api types.Api) *HttpAPIResource {
	id := appaws.Str(api.ApiId)
	name := appaws.Str(api.Name)

	// Convert tags
	tags := make(map[string]string)
	for k, v := range api.Tags {
		tags[k] = v
	}

	return &HttpAPIResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: name,
			Tags: tags,
			Data: api,
		},
		Item: api,
	}
}

// NewHttpAPIResourceFromGetOutput creates an HttpAPIResource from GetApi output
func NewHttpAPIResourceFromGetOutput(output *apigatewayv2.GetApiOutput) *HttpAPIResource {
	api := types.Api{
		ApiId:                     output.ApiId,
		Name:                      output.Name,
		Description:               output.Description,
		ProtocolType:              output.ProtocolType,
		ApiEndpoint:               output.ApiEndpoint,
		CreatedDate:               output.CreatedDate,
		Version:                   output.Version,
		ApiGatewayManaged:         output.ApiGatewayManaged,
		ApiKeySelectionExpression: output.ApiKeySelectionExpression,
		CorsConfiguration:         output.CorsConfiguration,
		DisableExecuteApiEndpoint: output.DisableExecuteApiEndpoint,
		DisableSchemaValidation:   output.DisableSchemaValidation,
		ImportInfo:                output.ImportInfo,
		RouteSelectionExpression:  output.RouteSelectionExpression,
		Tags:                      output.Tags,
		Warnings:                  output.Warnings,
	}
	return NewHttpAPIResource(api)
}

// Description returns the API description
func (r *HttpAPIResource) Description() string {
	if r.Item.Description != nil {
		return *r.Item.Description
	}
	return ""
}

// Version returns the API version
func (r *HttpAPIResource) Version() string {
	if r.Item.Version != nil {
		return *r.Item.Version
	}
	return ""
}

// ProtocolType returns the protocol type (HTTP or WEBSOCKET)
func (r *HttpAPIResource) ProtocolType() string {
	return string(r.Item.ProtocolType)
}

// ApiEndpoint returns the API endpoint URL
func (r *HttpAPIResource) ApiEndpoint() string {
	if r.Item.ApiEndpoint != nil {
		return *r.Item.ApiEndpoint
	}
	return ""
}

// CreatedDate returns the creation date
func (r *HttpAPIResource) CreatedDate() time.Time {
	if r.Item.CreatedDate != nil {
		return *r.Item.CreatedDate
	}
	return time.Time{}
}

// ApiGatewayManaged returns whether the API is managed by API Gateway
func (r *HttpAPIResource) ApiGatewayManaged() bool {
	if r.Item.ApiGatewayManaged != nil {
		return *r.Item.ApiGatewayManaged
	}
	return false
}

// DisableExecuteApiEndpoint returns whether the default endpoint is disabled
func (r *HttpAPIResource) DisableExecuteApiEndpoint() bool {
	if r.Item.DisableExecuteApiEndpoint != nil {
		return *r.Item.DisableExecuteApiEndpoint
	}
	return false
}

// RouteSelectionExpression returns the route selection expression
func (r *HttpAPIResource) RouteSelectionExpression() string {
	if r.Item.RouteSelectionExpression != nil {
		return *r.Item.RouteSelectionExpression
	}
	return ""
}

// CorsAllowOrigins returns the CORS allowed origins
func (r *HttpAPIResource) CorsAllowOrigins() []string {
	if r.Item.CorsConfiguration != nil {
		return r.Item.CorsConfiguration.AllowOrigins
	}
	return nil
}

// CorsAllowMethods returns the CORS allowed methods
func (r *HttpAPIResource) CorsAllowMethods() []string {
	if r.Item.CorsConfiguration != nil {
		return r.Item.CorsConfiguration.AllowMethods
	}
	return nil
}

// CorsAllowHeaders returns the CORS allowed headers
func (r *HttpAPIResource) CorsAllowHeaders() []string {
	if r.Item.CorsConfiguration != nil {
		return r.Item.CorsConfiguration.AllowHeaders
	}
	return nil
}

// HasCors returns whether CORS is configured
func (r *HttpAPIResource) HasCors() bool {
	return r.Item.CorsConfiguration != nil
}
