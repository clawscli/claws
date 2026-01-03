package trails

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// TrailDAO provides data access for CloudTrail trails.
type TrailDAO struct {
	dao.BaseDAO
	client *cloudtrail.Client
}

// NewTrailDAO creates a new TrailDAO.
func NewTrailDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &TrailDAO{
		BaseDAO: dao.NewBaseDAO("cloudtrail", "trails"),
		client:  cloudtrail.NewFromConfig(cfg),
	}, nil
}

// List returns all CloudTrail trails.
func (d *TrailDAO) List(ctx context.Context) ([]dao.Resource, error) {
	output, err := d.client.DescribeTrails(ctx, &cloudtrail.DescribeTrailsInput{})
	if err != nil {
		return nil, apperrors.Wrap(err, "describe trails")
	}

	resources := make([]dao.Resource, len(output.TrailList))
	for i, trail := range output.TrailList {
		resources[i] = NewTrailResource(trail)
	}
	return resources, nil
}

// Get returns a specific trail by name.
func (d *TrailDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeTrails(ctx, &cloudtrail.DescribeTrailsInput{
		TrailNameList: []string{id},
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe trail %s", id)
	}
	if len(output.TrailList) == 0 {
		return nil, fmt.Errorf("trail not found: %s", id)
	}
	return NewTrailResource(output.TrailList[0]), nil
}

// Delete deletes a trail by name.
func (d *TrailDAO) Delete(ctx context.Context, id string) error {
	_, err := d.client.DeleteTrail(ctx, &cloudtrail.DeleteTrailInput{
		Name: &id,
	})
	if err != nil {
		return apperrors.Wrapf(err, "delete trail %s", id)
	}
	return nil
}

// TrailResource wraps a CloudTrail trail.
type TrailResource struct {
	dao.BaseResource
	Item types.Trail
}

// NewTrailResource creates a new TrailResource.
func NewTrailResource(trail types.Trail) *TrailResource {
	return &TrailResource{
		BaseResource: dao.BaseResource{
			ID:  appaws.Str(trail.Name),
			ARN: appaws.Str(trail.TrailARN),
		},
		Item: trail,
	}
}

// Name returns the trail name.
func (r *TrailResource) Name() string {
	return appaws.Str(r.Item.Name)
}

// S3BucketName returns the S3 bucket for log delivery.
func (r *TrailResource) S3BucketName() string {
	return appaws.Str(r.Item.S3BucketName)
}

// S3KeyPrefix returns the S3 key prefix.
func (r *TrailResource) S3KeyPrefix() string {
	return appaws.Str(r.Item.S3KeyPrefix)
}

// HomeRegion returns the home region of the trail.
func (r *TrailResource) HomeRegion() string {
	return appaws.Str(r.Item.HomeRegion)
}

// IsMultiRegionTrail returns whether the trail is multi-region.
func (r *TrailResource) IsMultiRegionTrail() bool {
	return appaws.Bool(r.Item.IsMultiRegionTrail)
}

// IsOrganizationTrail returns whether the trail is for the organization.
func (r *TrailResource) IsOrganizationTrail() bool {
	return appaws.Bool(r.Item.IsOrganizationTrail)
}

// LogFileValidationEnabled returns whether log file validation is enabled.
func (r *TrailResource) LogFileValidationEnabled() bool {
	return appaws.Bool(r.Item.LogFileValidationEnabled)
}

// IncludeGlobalServiceEvents returns whether global service events are included.
func (r *TrailResource) IncludeGlobalServiceEvents() bool {
	return appaws.Bool(r.Item.IncludeGlobalServiceEvents)
}

// HasCustomEventSelectors returns whether the trail has custom event selectors.
func (r *TrailResource) HasCustomEventSelectors() bool {
	return appaws.Bool(r.Item.HasCustomEventSelectors)
}

// HasInsightSelectors returns whether the trail has insight selectors.
func (r *TrailResource) HasInsightSelectors() bool {
	return appaws.Bool(r.Item.HasInsightSelectors)
}

// KMSKeyId returns the KMS key ID for encryption.
func (r *TrailResource) KMSKeyId() string {
	return appaws.Str(r.Item.KmsKeyId)
}

// CloudWatchLogsLogGroupArn returns the CloudWatch Logs log group ARN.
func (r *TrailResource) CloudWatchLogsLogGroupArn() string {
	return appaws.Str(r.Item.CloudWatchLogsLogGroupArn)
}

// CloudWatchLogsRoleArn returns the CloudWatch Logs role ARN.
func (r *TrailResource) CloudWatchLogsRoleArn() string {
	return appaws.Str(r.Item.CloudWatchLogsRoleArn)
}

// SnsTopicARN returns the SNS topic ARN.
func (r *TrailResource) SnsTopicARN() string {
	return appaws.Str(r.Item.SnsTopicARN)
}
