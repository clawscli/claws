package datasources

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/bedrockagent"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagent/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
	"github.com/clawscli/claws/internal/log"
)

// DataSourceDAO provides data access for Bedrock Data Sources
type DataSourceDAO struct {
	dao.BaseDAO
	client *bedrockagent.Client
}

// NewDataSourceDAO creates a new DataSourceDAO
func NewDataSourceDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new bedrock-agent/data-sources dao")
	}
	return &DataSourceDAO{
		BaseDAO: dao.NewBaseDAO("bedrock-agent", "data-sources"),
		client:  bedrockagent.NewFromConfig(cfg),
	}, nil
}

func (d *DataSourceDAO) List(ctx context.Context) ([]dao.Resource, error) {
	knowledgeBaseId := dao.GetFilterFromContext(ctx, "KnowledgeBaseId")
	if knowledgeBaseId == "" {
		log.Warn("data-sources requires KnowledgeBaseId filter", "service", "bedrock-agent")
		return []dao.Resource{}, nil
	}

	dataSources, err := appaws.Paginate(ctx, func(token *string) ([]types.DataSourceSummary, *string, error) {
		output, err := d.client.ListDataSources(ctx, &bedrockagent.ListDataSourcesInput{
			KnowledgeBaseId: &knowledgeBaseId,
			NextToken:       token,
			MaxResults:      appaws.Int32Ptr(100),
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list data sources")
		}
		return output.DataSourceSummaries, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(dataSources))
	for i, ds := range dataSources {
		resources[i] = NewDataSourceResource(ds)
	}

	return resources, nil
}

func (d *DataSourceDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	knowledgeBaseId := dao.GetFilterFromContext(ctx, "KnowledgeBaseId")
	if knowledgeBaseId == "" {
		return nil, fmt.Errorf("KnowledgeBaseId filter required")
	}

	output, err := d.client.GetDataSource(ctx, &bedrockagent.GetDataSourceInput{
		DataSourceId:    &id,
		KnowledgeBaseId: &knowledgeBaseId,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "get data source %s", id)
	}

	return NewDataSourceResourceFromDetail(output.DataSource), nil
}

func (d *DataSourceDAO) Delete(ctx context.Context, id string) error {
	knowledgeBaseId := dao.GetFilterFromContext(ctx, "KnowledgeBaseId")
	if knowledgeBaseId == "" {
		return fmt.Errorf("KnowledgeBaseId filter required")
	}

	_, err := d.client.DeleteDataSource(ctx, &bedrockagent.DeleteDataSourceInput{
		DataSourceId:    &id,
		KnowledgeBaseId: &knowledgeBaseId,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete data source %s", id)
	}
	return nil
}

// DataSourceResource wraps a Bedrock Data Source
type DataSourceResource struct {
	dao.BaseResource
	Item       types.DataSourceSummary
	DetailItem *types.DataSource
	IsFromList bool
}

// NewDataSourceResource creates a new DataSourceResource from list output
func NewDataSourceResource(ds types.DataSourceSummary) *DataSourceResource {
	id := appaws.Str(ds.DataSourceId)
	name := appaws.Str(ds.Name)

	return &DataSourceResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: name,
			Data: ds,
		},
		Item:       ds,
		IsFromList: true,
	}
}

// NewDataSourceResourceFromDetail creates a DataSourceResource from detail output
func NewDataSourceResourceFromDetail(ds *types.DataSource) *DataSourceResource {
	id := appaws.Str(ds.DataSourceId)
	name := appaws.Str(ds.Name)

	return &DataSourceResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: name,
			Data: ds,
		},
		DetailItem: ds,
		IsFromList: false,
	}
}

// Status returns the data source status
func (r *DataSourceResource) Status() string {
	if r.IsFromList {
		return string(r.Item.Status)
	}
	if r.DetailItem != nil {
		return string(r.DetailItem.Status)
	}
	return ""
}

// Description returns the data source description
func (r *DataSourceResource) Description() string {
	if r.IsFromList {
		return appaws.Str(r.Item.Description)
	}
	if r.DetailItem != nil {
		return appaws.Str(r.DetailItem.Description)
	}
	return ""
}

// KnowledgeBaseId returns the knowledge base ID
func (r *DataSourceResource) KnowledgeBaseId() string {
	if r.IsFromList {
		return appaws.Str(r.Item.KnowledgeBaseId)
	}
	if r.DetailItem != nil {
		return appaws.Str(r.DetailItem.KnowledgeBaseId)
	}
	return ""
}

// UpdatedAt returns the last update time
func (r *DataSourceResource) UpdatedAt() *time.Time {
	if r.IsFromList {
		return r.Item.UpdatedAt
	}
	if r.DetailItem != nil {
		return r.DetailItem.UpdatedAt
	}
	return nil
}

// CreatedAt returns the creation time
func (r *DataSourceResource) CreatedAt() *time.Time {
	if r.DetailItem != nil {
		return r.DetailItem.CreatedAt
	}
	return nil
}

// DataSourceType returns the data source configuration type
func (r *DataSourceResource) DataSourceType() string {
	if r.DetailItem != nil && r.DetailItem.DataSourceConfiguration != nil {
		return string(r.DetailItem.DataSourceConfiguration.Type)
	}
	return ""
}

// S3BucketArn returns the S3 bucket ARN if applicable
func (r *DataSourceResource) S3BucketArn() string {
	if r.DetailItem != nil && r.DetailItem.DataSourceConfiguration != nil {
		if s3 := r.DetailItem.DataSourceConfiguration.S3Configuration; s3 != nil {
			return appaws.Str(s3.BucketArn)
		}
	}
	return ""
}

// DataDeletionPolicy returns the data deletion policy
func (r *DataSourceResource) DataDeletionPolicy() string {
	if r.DetailItem != nil {
		return string(r.DetailItem.DataDeletionPolicy)
	}
	return ""
}

// FailureReasons returns any failure reasons
func (r *DataSourceResource) FailureReasons() []string {
	if r.DetailItem != nil {
		return r.DetailItem.FailureReasons
	}
	return nil
}
