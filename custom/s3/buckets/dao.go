package buckets

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/log"
)

// BucketDAO provides data access for S3 buckets
type BucketDAO struct {
	dao.BaseDAO
	client *s3.Client
}

// NewBucketDAO creates a new BucketDAO
func NewBucketDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &BucketDAO{
		BaseDAO: dao.NewBaseDAO("s3", "buckets"),
		client:  s3.NewFromConfig(cfg),
	}, nil
}

func (d *BucketDAO) List(ctx context.Context) ([]dao.Resource, error) {
	output, err := d.client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, fmt.Errorf("list buckets: %w", err)
	}

	resources := make([]dao.Resource, 0, len(output.Buckets))
	for _, bucket := range output.Buckets {
		r := NewBucketResource(bucket)
		// Fetch bucket region
		if bucket.Name != nil {
			locOutput, err := d.client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
				Bucket: bucket.Name,
			})
			if err != nil {
				// Log non-access-denied errors (access denied is common for cross-account buckets)
				if !appaws.IsAccessDenied(err) {
					log.Warn("failed to get bucket location", "bucket", *bucket.Name, "error", err)
				}
			} else if locOutput != nil {
				if locOutput.LocationConstraint != "" {
					r.Region = string(locOutput.LocationConstraint)
				} else {
					r.Region = "us-east-1" // default for buckets without explicit location
				}
			}
		}
		resources = append(resources, r)
	}

	return resources, nil
}

func (d *BucketDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	// Get bucket location first (this works from any region)
	region := "us-east-1" // default for buckets without explicit location
	locOutput, err := d.client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: &id,
	})
	if err != nil {
		return nil, fmt.Errorf("get bucket location for %s: %w", id, err)
	}
	if locOutput != nil && locOutput.LocationConstraint != "" {
		region = string(locOutput.LocationConstraint)
	}

	resource := &BucketResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: id,
		},
		BucketName: id,
		Region:     region,
	}

	// Create a region-specific client for bucket operations
	regionClient, err := d.getRegionClient(ctx, region)
	if err != nil {
		// Fall back to default client if we can't create region-specific one
		regionClient = d.client
	}

	// Fetch extended details (errors are ignored for each API call)
	d.fetchVersioning(ctx, regionClient, id, resource)
	d.fetchEncryption(ctx, regionClient, id, resource)
	d.fetchPublicAccessBlock(ctx, regionClient, id, resource)
	d.fetchLifecycle(ctx, regionClient, id, resource)
	d.fetchObjectLock(ctx, regionClient, id, resource)
	d.fetchTags(ctx, regionClient, id, resource)

	return resource, nil
}

// getRegionClient creates an S3 client for the specified region
func (d *BucketDAO) getRegionClient(ctx context.Context, region string) (*s3.Client, error) {
	cfg, err := appaws.NewConfigWithRegion(ctx, region)
	if err != nil {
		return nil, err
	}
	return s3.NewFromConfig(cfg), nil
}

// fetchVersioning fetches bucket versioning configuration
func (d *BucketDAO) fetchVersioning(ctx context.Context, client *s3.Client, bucket string, r *BucketResource) {
	output, err := client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
		Bucket: &bucket,
	})
	if err != nil {
		return
	}
	if output.Status != "" {
		r.Versioning = string(output.Status)
	} else {
		r.Versioning = "Disabled"
	}
	if output.MFADelete != "" {
		r.MFADelete = string(output.MFADelete)
	}
}

// fetchEncryption fetches bucket encryption configuration
func (d *BucketDAO) fetchEncryption(ctx context.Context, client *s3.Client, bucket string, r *BucketResource) {
	output, err := client.GetBucketEncryption(ctx, &s3.GetBucketEncryptionInput{
		Bucket: &bucket,
	})
	if err != nil {
		return
	}
	if output.ServerSideEncryptionConfiguration != nil && len(output.ServerSideEncryptionConfiguration.Rules) > 0 {
		r.EncryptionEnabled = true
		rule := output.ServerSideEncryptionConfiguration.Rules[0]
		if rule.ApplyServerSideEncryptionByDefault != nil {
			r.EncryptionAlgorithm = string(rule.ApplyServerSideEncryptionByDefault.SSEAlgorithm)
			if rule.ApplyServerSideEncryptionByDefault.KMSMasterKeyID != nil {
				r.EncryptionKMSKeyID = *rule.ApplyServerSideEncryptionByDefault.KMSMasterKeyID
			}
		}
		if rule.BucketKeyEnabled != nil {
			r.BucketKeyEnabled = *rule.BucketKeyEnabled
		}
	}
}

// fetchPublicAccessBlock fetches public access block configuration
func (d *BucketDAO) fetchPublicAccessBlock(ctx context.Context, client *s3.Client, bucket string, r *BucketResource) {
	output, err := client.GetPublicAccessBlock(ctx, &s3.GetPublicAccessBlockInput{
		Bucket: &bucket,
	})
	if err != nil {
		return
	}
	if output.PublicAccessBlockConfiguration != nil {
		cfg := output.PublicAccessBlockConfiguration
		r.PublicAccessBlock = &PublicAccessBlockInfo{
			BlockPublicAcls:       cfg.BlockPublicAcls != nil && *cfg.BlockPublicAcls,
			IgnorePublicAcls:      cfg.IgnorePublicAcls != nil && *cfg.IgnorePublicAcls,
			BlockPublicPolicy:     cfg.BlockPublicPolicy != nil && *cfg.BlockPublicPolicy,
			RestrictPublicBuckets: cfg.RestrictPublicBuckets != nil && *cfg.RestrictPublicBuckets,
		}
	}
}

// fetchLifecycle fetches bucket lifecycle configuration
func (d *BucketDAO) fetchLifecycle(ctx context.Context, client *s3.Client, bucket string, r *BucketResource) {
	output, err := client.GetBucketLifecycleConfiguration(ctx, &s3.GetBucketLifecycleConfigurationInput{
		Bucket: &bucket,
	})
	if err != nil {
		return
	}
	r.LifecycleRulesCount = len(output.Rules)
}

// fetchObjectLock fetches object lock configuration
func (d *BucketDAO) fetchObjectLock(ctx context.Context, client *s3.Client, bucket string, r *BucketResource) {
	output, err := client.GetObjectLockConfiguration(ctx, &s3.GetObjectLockConfigurationInput{
		Bucket: &bucket,
	})
	if err != nil {
		return
	}
	if output.ObjectLockConfiguration != nil {
		r.ObjectLockEnabled = output.ObjectLockConfiguration.ObjectLockEnabled == types.ObjectLockEnabledEnabled
		if output.ObjectLockConfiguration.Rule != nil && output.ObjectLockConfiguration.Rule.DefaultRetention != nil {
			retention := output.ObjectLockConfiguration.Rule.DefaultRetention
			r.ObjectLockMode = string(retention.Mode)
			if retention.Days != nil {
				r.ObjectLockRetention = fmt.Sprintf("%d days", *retention.Days)
			} else if retention.Years != nil {
				r.ObjectLockRetention = fmt.Sprintf("%d years", *retention.Years)
			}
		}
	}
}

// fetchTags fetches bucket tags
func (d *BucketDAO) fetchTags(ctx context.Context, client *s3.Client, bucket string, r *BucketResource) {
	output, err := client.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{
		Bucket: &bucket,
	})
	if err != nil {
		return
	}
	tags := make(map[string]string)
	for _, tag := range output.TagSet {
		if tag.Key != nil && tag.Value != nil {
			tags[*tag.Key] = *tag.Value
		}
	}
	r.BaseResource.Tags = tags
}

func (d *BucketDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: &id,
	})
	if err != nil {
		if appaws.IsNotFound(err) {
			return nil // Already deleted
		}
		if appaws.IsResourceInUse(err) {
			return fmt.Errorf("bucket %s is not empty (must delete all objects first)", id)
		}
		return fmt.Errorf("delete bucket %s: %w", id, err)
	}
	return nil
}

// BucketResource wraps an S3 bucket
type BucketResource struct {
	dao.BaseResource
	BucketName   string
	Region       string
	CreationDate time.Time

	// Extended info (fetched in Get() only)
	Versioning          string
	MFADelete           string
	EncryptionEnabled   bool
	EncryptionAlgorithm string
	EncryptionKMSKeyID  string
	BucketKeyEnabled    bool
	PublicAccessBlock   *PublicAccessBlockInfo
	LifecycleRulesCount int
	ObjectLockEnabled   bool
	ObjectLockMode      string
	ObjectLockRetention string
}

// PublicAccessBlockInfo holds public access block settings
type PublicAccessBlockInfo struct {
	BlockPublicAcls       bool
	IgnorePublicAcls      bool
	BlockPublicPolicy     bool
	RestrictPublicBuckets bool
}

// NewBucketResource creates a new BucketResource
func NewBucketResource(bucket types.Bucket) *BucketResource {
	name := appaws.Str(bucket.Name)

	return &BucketResource{
		BaseResource: dao.BaseResource{
			ID:   name,
			Name: name,
		},
		BucketName:   name,
		CreationDate: appaws.Time(bucket.CreationDate),
	}
}

func (r *BucketResource) Age() time.Duration {
	if r.CreationDate.IsZero() {
		return 0
	}
	return time.Since(r.CreationDate)
}
