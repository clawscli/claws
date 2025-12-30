package buckets

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/s3vectors"
	"github.com/aws/aws-sdk-go-v2/service/s3vectors/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

// VectorBucketDAO provides data access for S3 Vector Buckets
type VectorBucketDAO struct {
	dao.BaseDAO
	client *s3vectors.Client
}

// NewVectorBucketDAO creates a new VectorBucketDAO
func NewVectorBucketDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("new s3vectors/buckets dao: %w", err)
	}
	return &VectorBucketDAO{
		BaseDAO: dao.NewBaseDAO("s3vectors", "buckets"),
		client:  s3vectors.NewFromConfig(cfg),
	}, nil
}

func (d *VectorBucketDAO) List(ctx context.Context) ([]dao.Resource, error) {
	summaries, err := appaws.Paginate(ctx, func(token *string) ([]types.VectorBucketSummary, *string, error) {
		output, err := d.client.ListVectorBuckets(ctx, &s3vectors.ListVectorBucketsInput{
			NextToken:  token,
			MaxResults: appaws.Int32Ptr(100),
		})
		if err != nil {
			return nil, nil, fmt.Errorf("list vector buckets: %w", err)
		}
		return output.VectorBuckets, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	// Use Summary directly to avoid N+1 GetVectorBucket calls
	resources := make([]dao.Resource, 0, len(summaries))
	for _, bucket := range summaries {
		resources = append(resources, NewVectorBucketResourceFromSummary(bucket))
	}

	return resources, nil
}

func (d *VectorBucketDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	input := &s3vectors.GetVectorBucketInput{
		VectorBucketName: &id,
	}

	output, err := d.client.GetVectorBucket(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("get vector bucket %s: %w", id, err)
	}

	return NewVectorBucketResource(output.VectorBucket), nil
}

func (d *VectorBucketDAO) Delete(ctx context.Context, id string) error {
	input := &s3vectors.DeleteVectorBucketInput{
		VectorBucketName: &id,
	}

	_, err := d.client.DeleteVectorBucket(ctx, input)
	if err != nil {
		return fmt.Errorf("delete vector bucket %s: %w", id, err)
	}

	return nil
}

// VectorBucketResource wraps an S3 Vector Bucket
type VectorBucketResource struct {
	dao.BaseResource
	Item    *types.VectorBucket        // Full details (from Get)
	Summary *types.VectorBucketSummary // Summary (from List)
}

// NewVectorBucketResource creates a new VectorBucketResource from detail
func NewVectorBucketResource(bucket *types.VectorBucket) *VectorBucketResource {
	name := appaws.Str(bucket.VectorBucketName)
	arn := appaws.Str(bucket.VectorBucketArn)

	return &VectorBucketResource{
		BaseResource: dao.BaseResource{
			ID:   name,
			Name: name,
			ARN:  arn,
			Tags: make(map[string]string),
			Data: bucket,
		},
		Item: bucket,
	}
}

// NewVectorBucketResourceFromSummary creates a new VectorBucketResource from summary
func NewVectorBucketResourceFromSummary(summary types.VectorBucketSummary) *VectorBucketResource {
	name := appaws.Str(summary.VectorBucketName)
	arn := appaws.Str(summary.VectorBucketArn)

	return &VectorBucketResource{
		BaseResource: dao.BaseResource{
			ID:   name,
			Name: name,
			ARN:  arn,
			Tags: make(map[string]string),
			Data: summary,
		},
		Summary: &summary,
	}
}

// BucketName returns the bucket name
func (r *VectorBucketResource) BucketName() string {
	if r.Item != nil {
		return appaws.Str(r.Item.VectorBucketName)
	}
	if r.Summary != nil {
		return appaws.Str(r.Summary.VectorBucketName)
	}
	return r.GetName()
}

// CreationDate returns the creation date as string
func (r *VectorBucketResource) CreationDate() string {
	if r.Item != nil && r.Item.CreationTime != nil {
		return r.Item.CreationTime.Format("2006-01-02 15:04:05")
	}
	if r.Summary != nil && r.Summary.CreationTime != nil {
		return r.Summary.CreationTime.Format("2006-01-02 15:04:05")
	}
	return ""
}

// EncryptionType returns the encryption type (only available from Item/detail)
func (r *VectorBucketResource) EncryptionType() string {
	if r.Item != nil && r.Item.EncryptionConfiguration != nil {
		return string(r.Item.EncryptionConfiguration.SseType)
	}
	return "-"
}

// KmsKeyArn returns the KMS key ARN if using KMS encryption
func (r *VectorBucketResource) KmsKeyArn() string {
	if r.Item != nil && r.Item.EncryptionConfiguration != nil {
		return appaws.Str(r.Item.EncryptionConfiguration.KmsKeyArn)
	}
	return ""
}
