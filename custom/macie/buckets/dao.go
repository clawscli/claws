package buckets

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/macie2"
	"github.com/aws/aws-sdk-go-v2/service/macie2/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// BucketDAO provides data access for Macie buckets.
type BucketDAO struct {
	dao.BaseDAO
	client *macie2.Client
}

// NewBucketDAO creates a new BucketDAO.
func NewBucketDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new macie/buckets dao")
	}
	return &BucketDAO{
		BaseDAO: dao.NewBaseDAO("macie2", "buckets"),
		client:  macie2.NewFromConfig(cfg),
	}, nil
}

// List returns all buckets monitored by Macie.
func (d *BucketDAO) List(ctx context.Context) ([]dao.Resource, error) {
	buckets, err := appaws.Paginate(ctx, func(token *string) ([]types.BucketMetadata, *string, error) {
		output, err := d.client.DescribeBuckets(ctx, &macie2.DescribeBucketsInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "describe macie buckets")
		}
		return output.Buckets, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(buckets))
	for i, bucket := range buckets {
		resources[i] = NewBucketResource(bucket)
	}
	return resources, nil
}

// Get returns a specific bucket.
func (d *BucketDAO) Get(ctx context.Context, name string) (dao.Resource, error) {
	resources, err := d.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, r := range resources {
		if r.GetID() == name {
			return r, nil
		}
	}
	return nil, fmt.Errorf("bucket not found: %s", name)
}

// Delete is not supported for buckets (they're just monitored).
func (d *BucketDAO) Delete(ctx context.Context, name string) error {
	return fmt.Errorf("delete not supported for macie buckets")
}

// BucketResource wraps a Macie bucket.
type BucketResource struct {
	dao.BaseResource
	Bucket *types.BucketMetadata
}

// NewBucketResource creates a new BucketResource.
func NewBucketResource(bucket types.BucketMetadata) *BucketResource {
	return &BucketResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(bucket.BucketName),
			ARN: appaws.Str(bucket.BucketArn),
		},
		Bucket: &bucket,
	}
}

// Name returns the bucket name.
func (r *BucketResource) Name() string {
	if r.Bucket != nil && r.Bucket.BucketName != nil {
		return *r.Bucket.BucketName
	}
	return ""
}

// AccountId returns the account ID.
func (r *BucketResource) AccountId() string {
	if r.Bucket != nil && r.Bucket.AccountId != nil {
		return *r.Bucket.AccountId
	}
	return ""
}

// Region returns the bucket region.
func (r *BucketResource) Region() string {
	if r.Bucket != nil && r.Bucket.Region != nil {
		return *r.Bucket.Region
	}
	return ""
}

// ClassifiableObjectCount returns the count of classifiable objects.
func (r *BucketResource) ClassifiableObjectCount() int64 {
	if r.Bucket != nil && r.Bucket.ClassifiableObjectCount != nil {
		return *r.Bucket.ClassifiableObjectCount
	}
	return 0
}

// SizeInBytes returns the bucket size in bytes.
func (r *BucketResource) SizeInBytes() int64 {
	if r.Bucket != nil && r.Bucket.SizeInBytes != nil {
		return *r.Bucket.SizeInBytes
	}
	return 0
}
