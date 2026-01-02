package datasources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/appsync"
	"github.com/aws/aws-sdk-go-v2/service/appsync/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// DataSourceDAO provides data access for AppSync data sources.
type DataSourceDAO struct {
	dao.BaseDAO
	client *appsync.Client
}

// NewDataSourceDAO creates a new DataSourceDAO.
func NewDataSourceDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new appsync/data-sources dao")
	}
	return &DataSourceDAO{
		BaseDAO: dao.NewBaseDAO("appsync", "data-sources"),
		client:  appsync.NewFromConfig(cfg),
	}, nil
}

// List returns data sources for the specified API.
func (d *DataSourceDAO) List(ctx context.Context) ([]dao.Resource, error) {
	apiId := dao.GetFilterFromContext(ctx, "ApiId")
	if apiId == "" {
		return nil, fmt.Errorf("API ID filter required")
	}

	dataSources, err := appaws.Paginate(ctx, func(token *string) ([]types.DataSource, *string, error) {
		output, err := d.client.ListDataSources(ctx, &appsync.ListDataSourcesInput{
			ApiId:     &apiId,
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list appsync data sources")
		}
		return output.DataSources, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(dataSources))
	for i, ds := range dataSources {
		resources[i] = NewDataSourceResource(ds, apiId)
	}
	return resources, nil
}

// Get returns a specific data source.
func (d *DataSourceDAO) Get(ctx context.Context, name string) (dao.Resource, error) {
	apiId := dao.GetFilterFromContext(ctx, "ApiId")
	if apiId == "" {
		return nil, fmt.Errorf("API ID filter required")
	}

	output, err := d.client.GetDataSource(ctx, &appsync.GetDataSourceInput{
		ApiId: &apiId,
		Name:  &name,
	})
	if err != nil {
		return nil, apperrors.Wrap(err, "get appsync data source")
	}
	return NewDataSourceResource(*output.DataSource, apiId), nil
}

// Delete deletes a data source.
func (d *DataSourceDAO) Delete(ctx context.Context, name string) error {
	apiId := dao.GetFilterFromContext(ctx, "ApiId")
	if apiId == "" {
		return fmt.Errorf("API ID filter required")
	}

	_, err := d.client.DeleteDataSource(ctx, &appsync.DeleteDataSourceInput{
		ApiId: &apiId,
		Name:  &name,
	})
	if err != nil {
		return apperrors.Wrap(err, "delete appsync data source")
	}
	return nil
}

// DataSourceResource wraps an AppSync data source.
type DataSourceResource struct {
	dao.BaseResource
	DataSource *types.DataSource
	apiId      string
}

// NewDataSourceResource creates a new DataSourceResource.
func NewDataSourceResource(ds types.DataSource, apiId string) *DataSourceResource {
	return &DataSourceResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(ds.Name),
			ARN: appaws.Str(ds.DataSourceArn),
		},
		DataSource: &ds,
		apiId:      apiId,
	}
}

// Name returns the data source name.
func (r *DataSourceResource) Name() string {
	if r.DataSource != nil && r.DataSource.Name != nil {
		return *r.DataSource.Name
	}
	return ""
}

// Type returns the data source type.
func (r *DataSourceResource) Type() string {
	if r.DataSource != nil {
		return string(r.DataSource.Type)
	}
	return ""
}

// Description returns the data source description.
func (r *DataSourceResource) Description() string {
	if r.DataSource != nil && r.DataSource.Description != nil {
		return *r.DataSource.Description
	}
	return ""
}

// ServiceRoleArn returns the service role ARN.
func (r *DataSourceResource) ServiceRoleArn() string {
	if r.DataSource != nil && r.DataSource.ServiceRoleArn != nil {
		return *r.DataSource.ServiceRoleArn
	}
	return ""
}
