package restapis

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// RestAPIDAO provides data access for API Gateway REST APIs
type RestAPIDAO struct {
	dao.BaseDAO
	client *apigateway.Client
}

// NewRestAPIDAO creates a new RestAPIDAO
func NewRestAPIDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new apigateway/rest-apis dao")
	}
	return &RestAPIDAO{
		BaseDAO: dao.NewBaseDAO("apigateway", "rest-apis"),
		client:  apigateway.NewFromConfig(cfg),
	}, nil
}

// List returns all REST APIs
func (d *RestAPIDAO) List(ctx context.Context) ([]dao.Resource, error) {
	var resources []dao.Resource
	var position *string

	for {
		input := &apigateway.GetRestApisInput{
			Position: position,
			Limit:    intPtr(500),
		}

		output, err := d.client.GetRestApis(ctx, input)
		if err != nil {
			return nil, apperrors.Wrap(err, "list REST APIs")
		}

		for _, api := range output.Items {
			resources = append(resources, NewRestAPIResource(api))
		}

		if output.Position == nil {
			break
		}
		position = output.Position
	}

	return resources, nil
}

// Get returns a specific REST API by ID
func (d *RestAPIDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.GetRestApi(ctx, &apigateway.GetRestApiInput{
		RestApiId: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get REST API %s", id)
	}

	return NewRestAPIResourceFromGetOutput(output), nil
}

// Delete deletes a REST API
func (d *RestAPIDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteRestApi(ctx, &apigateway.DeleteRestApiInput{
		RestApiId: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete REST API %s", id)
	}
	return nil
}

func intPtr(i int32) *int32 {
	return &i
}

// RestAPIResource wraps an API Gateway REST API
type RestAPIResource struct {
	dao.BaseResource
	Item types.RestApi
}

// NewRestAPIResource creates a new RestAPIResource from list output
func NewRestAPIResource(api types.RestApi) *RestAPIResource {
	id := appaws.Str(api.Id)
	name := appaws.Str(api.Name)

	// Convert tags
	tags := make(map[string]string)
	for k, v := range api.Tags {
		tags[k] = v
	}

	return &RestAPIResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: name,
			Tags: tags,
			Data: api,
		},
		Item: api,
	}
}

// NewRestAPIResourceFromGetOutput creates a RestAPIResource from GetRestApi output
func NewRestAPIResourceFromGetOutput(output *apigateway.GetRestApiOutput) *RestAPIResource {
	api := types.RestApi{
		Id:                        output.Id,
		Name:                      output.Name,
		Description:               output.Description,
		CreatedDate:               output.CreatedDate,
		Version:                   output.Version,
		ApiKeySource:              output.ApiKeySource,
		BinaryMediaTypes:          output.BinaryMediaTypes,
		DisableExecuteApiEndpoint: output.DisableExecuteApiEndpoint,
		EndpointConfiguration:     output.EndpointConfiguration,
		MinimumCompressionSize:    output.MinimumCompressionSize,
		Policy:                    output.Policy,
		RootResourceId:            output.RootResourceId,
		Tags:                      output.Tags,
		Warnings:                  output.Warnings,
	}
	return NewRestAPIResource(api)
}

// Description returns the API description
func (r *RestAPIResource) Description() string {
	if r.Item.Description != nil {
		return *r.Item.Description
	}
	return ""
}

// Version returns the API version
func (r *RestAPIResource) Version() string {
	if r.Item.Version != nil {
		return *r.Item.Version
	}
	return ""
}

// EndpointType returns the endpoint configuration type
func (r *RestAPIResource) EndpointType() string {
	if r.Item.EndpointConfiguration != nil && len(r.Item.EndpointConfiguration.Types) > 0 {
		return string(r.Item.EndpointConfiguration.Types[0])
	}
	return ""
}

// CreatedDate returns the creation date
func (r *RestAPIResource) CreatedDate() time.Time {
	if r.Item.CreatedDate != nil {
		return *r.Item.CreatedDate
	}
	return time.Time{}
}

// ApiKeySource returns the API key source
func (r *RestAPIResource) ApiKeySource() string {
	return string(r.Item.ApiKeySource)
}

// DisableExecuteApiEndpoint returns whether the default endpoint is disabled
func (r *RestAPIResource) DisableExecuteApiEndpoint() bool {
	return r.Item.DisableExecuteApiEndpoint
}

// RootResourceId returns the root resource ID
func (r *RestAPIResource) RootResourceId() string {
	if r.Item.RootResourceId != nil {
		return *r.Item.RootResourceId
	}
	return ""
}

// MinimumCompressionSize returns the minimum compression size
func (r *RestAPIResource) MinimumCompressionSize() int32 {
	if r.Item.MinimumCompressionSize != nil {
		return *r.Item.MinimumCompressionSize
	}
	return 0
}

// BinaryMediaTypes returns the binary media types
func (r *RestAPIResource) BinaryMediaTypes() []string {
	return r.Item.BinaryMediaTypes
}
