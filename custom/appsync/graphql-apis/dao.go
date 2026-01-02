package graphqlapis

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/appsync"
	"github.com/aws/aws-sdk-go-v2/service/appsync/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// GraphQLApiDAO provides data access for AppSync GraphQL APIs.
type GraphQLApiDAO struct {
	dao.BaseDAO
	client *appsync.Client
}

// NewGraphQLApiDAO creates a new GraphQLApiDAO.
func NewGraphQLApiDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new appsync/graphql-apis dao")
	}
	return &GraphQLApiDAO{
		BaseDAO: dao.NewBaseDAO("appsync", "graphql-apis"),
		client:  appsync.NewFromConfig(cfg),
	}, nil
}

// List returns all GraphQL APIs.
func (d *GraphQLApiDAO) List(ctx context.Context) ([]dao.Resource, error) {
	apis, err := appaws.Paginate(ctx, func(token *string) ([]types.GraphqlApi, *string, error) {
		output, err := d.client.ListGraphqlApis(ctx, &appsync.ListGraphqlApisInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list appsync graphql apis")
		}
		return output.GraphqlApis, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(apis))
	for i, api := range apis {
		resources[i] = NewGraphQLApiResource(api)
	}
	return resources, nil
}

// Get returns a specific GraphQL API.
func (d *GraphQLApiDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.GetGraphqlApi(ctx, &appsync.GetGraphqlApiInput{
		ApiId: &id,
	})
	if err != nil {
		return nil, apperrors.Wrap(err, "get appsync graphql api")
	}
	return NewGraphQLApiResource(*output.GraphqlApi), nil
}

// Delete deletes a GraphQL API.
func (d *GraphQLApiDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteGraphqlApi(ctx, &appsync.DeleteGraphqlApiInput{
		ApiId: &id,
	})
	if err != nil {
		return apperrors.Wrap(err, "delete appsync graphql api")
	}
	return nil
}

// GraphQLApiResource wraps an AppSync GraphQL API.
type GraphQLApiResource struct {
	dao.BaseResource
	Api *types.GraphqlApi
}

// NewGraphQLApiResource creates a new GraphQLApiResource.
func NewGraphQLApiResource(api types.GraphqlApi) *GraphQLApiResource {
	return &GraphQLApiResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(api.ApiId),
			ARN: appaws.Str(api.Arn),
		},
		Api: &api,
	}
}

// Name returns the API name.
func (r *GraphQLApiResource) Name() string {
	if r.Api != nil && r.Api.Name != nil {
		return *r.Api.Name
	}
	return ""
}

// AuthenticationType returns the authentication type.
func (r *GraphQLApiResource) AuthenticationType() string {
	if r.Api != nil {
		return string(r.Api.AuthenticationType)
	}
	return ""
}

// ApiType returns the API type.
func (r *GraphQLApiResource) ApiType() string {
	if r.Api != nil {
		return string(r.Api.ApiType)
	}
	return ""
}

// Endpoint returns the GraphQL endpoint.
func (r *GraphQLApiResource) Endpoint() string {
	if r.Api != nil && r.Api.Uris != nil {
		if endpoint, ok := r.Api.Uris["GRAPHQL"]; ok {
			return endpoint
		}
	}
	return ""
}

// XrayEnabled returns whether X-Ray is enabled.
func (r *GraphQLApiResource) XrayEnabled() bool {
	if r.Api != nil {
		return r.Api.XrayEnabled
	}
	return false
}
