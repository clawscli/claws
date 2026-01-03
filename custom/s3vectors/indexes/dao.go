package indexes

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3vectors"
	"github.com/aws/aws-sdk-go-v2/service/s3vectors/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// VectorIndexDAO provides data access for S3 Vector Indexes
type VectorIndexDAO struct {
	dao.BaseDAO
	client *s3vectors.Client
}

// NewVectorIndexDAO creates a new VectorIndexDAO
func NewVectorIndexDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &VectorIndexDAO{
		BaseDAO: dao.NewBaseDAO("s3vectors", "indexes"),
		client:  s3vectors.NewFromConfig(cfg),
	}, nil
}

func (d *VectorIndexDAO) List(ctx context.Context) ([]dao.Resource, error) {
	var resources []dao.Resource

	// Check if filtering by bucket name
	bucketName := dao.GetFilterFromContext(ctx, "VectorBucketName")

	if bucketName != "" {
		// List indexes for specific bucket
		return d.listIndexesForBucket(ctx, bucketName)
	}

	// List all buckets first, then list indexes for each
	bucketIter := appaws.PaginateIter(ctx, func(token *string) ([]types.VectorBucketSummary, *string, error) {
		output, err := d.client.ListVectorBuckets(ctx, &s3vectors.ListVectorBucketsInput{
			NextToken:  token,
			MaxResults: appaws.Int32Ptr(100),
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list vector buckets")
		}
		return output.VectorBuckets, output.NextToken, nil
	})

	for bucket, err := range bucketIter {
		if err != nil {
			return nil, err
		}
		bucketIndexes, err := d.listIndexesForBucket(ctx, appaws.Str(bucket.VectorBucketName))
		if err != nil {
			// Skip buckets we can't list indexes for
			continue
		}
		resources = append(resources, bucketIndexes...)
	}

	return resources, nil
}

func (d *VectorIndexDAO) listIndexesForBucket(ctx context.Context, bucketName string) ([]dao.Resource, error) {
	var resources []dao.Resource

	indexIter := appaws.PaginateIter(ctx, func(token *string) ([]types.IndexSummary, *string, error) {
		output, err := d.client.ListIndexes(ctx, &s3vectors.ListIndexesInput{
			VectorBucketName: &bucketName,
			NextToken:        token,
			MaxResults:       appaws.Int32Ptr(100),
		})
		if err != nil {
			return nil, nil, apperrors.Wrapf(err, "list indexes for bucket %s", bucketName)
		}
		return output.Indexes, output.NextToken, nil
	})

	for indexSummary, err := range indexIter {
		if err != nil {
			return nil, err
		}
		// Get full index details for dimension, datatype, metric info
		getOutput, err := d.client.GetIndex(ctx, &s3vectors.GetIndexInput{
			IndexArn: indexSummary.IndexArn,
		})
		if err != nil {
			// Fall back to summary if we can't get details
			resources = append(resources, NewVectorIndexResourceFromSummary(indexSummary, bucketName))
			continue
		}
		resources = append(resources, NewVectorIndexResource(getOutput.Index))
	}

	return resources, nil
}

func (d *VectorIndexDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	// ID could be ARN or "bucketName/indexName"
	input := &s3vectors.GetIndexInput{
		IndexArn: &id,
	}

	output, err := d.client.GetIndex(ctx, input)
	if err != nil {
		return nil, apperrors.Wrapf(err, "get index %s", id)
	}

	return NewVectorIndexResource(output.Index), nil
}

func (d *VectorIndexDAO) Delete(ctx context.Context, id string) error {
	input := &s3vectors.DeleteIndexInput{
		IndexArn: &id,
	}

	_, err := d.client.DeleteIndex(ctx, input)
	if err != nil {
		return apperrors.Wrapf(err, "delete index %s", id)
	}

	return nil
}

// VectorIndexResource wraps an S3 Vector Index
type VectorIndexResource struct {
	dao.BaseResource
	Item       *types.Index
	Summary    *types.IndexSummary
	BucketName string
	fromDetail bool
}

// NewVectorIndexResource creates a new VectorIndexResource from detail
func NewVectorIndexResource(index *types.Index) *VectorIndexResource {
	name := appaws.Str(index.IndexName)
	arn := appaws.Str(index.IndexArn)

	return &VectorIndexResource{
		BaseResource: dao.BaseResource{
			ID:   arn,
			Name: name,
			ARN:  arn,
			Tags: make(map[string]string),
			Data: index,
		},
		Item:       index,
		fromDetail: true,
	}
}

// NewVectorIndexResourceFromSummary creates a new VectorIndexResource from summary
func NewVectorIndexResourceFromSummary(summary types.IndexSummary, bucketName string) *VectorIndexResource {
	name := appaws.Str(summary.IndexName)
	arn := appaws.Str(summary.IndexArn)

	return &VectorIndexResource{
		BaseResource: dao.BaseResource{
			ID:   arn,
			Name: name,
			ARN:  arn,
			Tags: make(map[string]string),
			Data: summary,
		},
		Summary:    &summary,
		BucketName: bucketName,
		fromDetail: false,
	}
}

// IndexName returns the index name
func (r *VectorIndexResource) IndexName() string {
	if r.fromDetail && r.Item != nil {
		return appaws.Str(r.Item.IndexName)
	}
	if r.Summary != nil {
		return appaws.Str(r.Summary.IndexName)
	}
	return r.GetName()
}

// Dimension returns the vector dimension
func (r *VectorIndexResource) Dimension() int32 {
	if r.fromDetail && r.Item != nil && r.Item.Dimension != nil {
		return *r.Item.Dimension
	}
	return 0
}

// DataType returns the data type
func (r *VectorIndexResource) DataType() string {
	if r.fromDetail && r.Item != nil {
		return string(r.Item.DataType)
	}
	return ""
}

// DistanceMetric returns the distance metric
func (r *VectorIndexResource) DistanceMetric() string {
	if r.fromDetail && r.Item != nil {
		return string(r.Item.DistanceMetric)
	}
	return ""
}

// CreationDate returns the creation date as string
func (r *VectorIndexResource) CreationDate() string {
	if r.fromDetail && r.Item != nil && r.Item.CreationTime != nil {
		return r.Item.CreationTime.Format("2006-01-02 15:04:05")
	}
	if r.Summary != nil && r.Summary.CreationTime != nil {
		return r.Summary.CreationTime.Format("2006-01-02 15:04:05")
	}
	return ""
}

// GetBucketName returns the bucket name
func (r *VectorIndexResource) GetBucketName() string {
	if r.BucketName != "" {
		return r.BucketName
	}
	if r.fromDetail && r.Item != nil {
		return appaws.Str(r.Item.VectorBucketName)
	}
	return ""
}

// EncryptionType returns the encryption type
func (r *VectorIndexResource) EncryptionType() string {
	if r.fromDetail && r.Item != nil && r.Item.EncryptionConfiguration != nil {
		return string(r.Item.EncryptionConfiguration.SseType)
	}
	return ""
}

// KmsKeyArn returns the KMS key ARN if using KMS encryption
func (r *VectorIndexResource) KmsKeyArn() string {
	if r.fromDetail && r.Item != nil && r.Item.EncryptionConfiguration != nil {
		return appaws.Str(r.Item.EncryptionConfiguration.KmsKeyArn)
	}
	return ""
}

// NonFilterableMetadataKeys returns the non-filterable metadata keys
func (r *VectorIndexResource) NonFilterableMetadataKeys() []string {
	if r.fromDetail && r.Item != nil && r.Item.MetadataConfiguration != nil {
		return r.Item.MetadataConfiguration.NonFilterableMetadataKeys
	}
	return nil
}
