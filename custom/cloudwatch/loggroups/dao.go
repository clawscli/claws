package loggroups

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// LogGroupDAO provides data access for CloudWatch Log Groups
type LogGroupDAO struct {
	dao.BaseDAO
	client *cloudwatchlogs.Client
}

// NewLogGroupDAO creates a new LogGroupDAO
func NewLogGroupDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new cloudwatch/loggroups dao")
	}
	return &LogGroupDAO{
		BaseDAO: dao.NewBaseDAO("cloudwatch", "log-groups"),
		client:  cloudwatchlogs.NewFromConfig(cfg),
	}, nil
}

func (d *LogGroupDAO) List(ctx context.Context) ([]dao.Resource, error) {
	// Check for prefix filter (for navigation from Lambda/ECS)
	prefix := dao.GetFilterFromContext(ctx, "LogGroupPrefix")

	logGroups, err := appaws.Paginate(ctx, func(token *string) ([]types.LogGroup, *string, error) {
		input := &cloudwatchlogs.DescribeLogGroupsInput{
			NextToken: token,
		}
		if prefix != "" {
			input.LogGroupNamePrefix = &prefix
		}
		output, err := d.client.DescribeLogGroups(ctx, input)
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "describe log groups")
		}
		return output.LogGroups, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(logGroups))
	for i, lg := range logGroups {
		resources[i] = NewLogGroupResource(lg)
	}
	return resources, nil
}

func (d *LogGroupDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	input := &cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix: &id,
	}

	output, err := d.client.DescribeLogGroups(ctx, input)
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe log group %s", id)
	}

	// Find exact match
	for _, lg := range output.LogGroups {
		if lg.LogGroupName != nil && *lg.LogGroupName == id {
			return NewLogGroupResource(lg), nil
		}
	}

	return nil, fmt.Errorf("log group not found: %s", id)
}

func (d *LogGroupDAO) Delete(ctx context.Context, id string) error {
	input := &cloudwatchlogs.DeleteLogGroupInput{
		LogGroupName: &id,
	}

	_, err := d.client.DeleteLogGroup(ctx, input)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil // Already deleted
		}
		return apperrors.Wrapf(err, "delete log group %s", id)
	}

	return nil
}

// LogGroupResource wraps a CloudWatch Log Group
type LogGroupResource struct {
	dao.BaseResource
	Item types.LogGroup
}

// NewLogGroupResource creates a new LogGroupResource
func NewLogGroupResource(lg types.LogGroup) *LogGroupResource {
	name := appaws.Str(lg.LogGroupName)

	// Extract short name for display (last part of path)
	shortName := name
	if idx := strings.LastIndex(name, "/"); idx != -1 {
		shortName = name[idx+1:]
	}

	return &LogGroupResource{
		BaseResource: dao.BaseResource{
			ID:   name,
			Name: shortName,
			ARN:  appaws.Str(lg.Arn),
			Tags: nil, // Tags require separate API call
			Data: lg,
		},
		Item: lg,
	}
}

// LogGroupName returns the full log group name
func (r *LogGroupResource) LogGroupName() string {
	return appaws.Str(r.Item.LogGroupName)
}

// StoredBytes returns the stored bytes
func (r *LogGroupResource) StoredBytes() int64 {
	if r.Item.StoredBytes != nil {
		return *r.Item.StoredBytes
	}
	return 0
}

// RetentionDays returns the retention in days
func (r *LogGroupResource) RetentionDays() int32 {
	if r.Item.RetentionInDays != nil {
		return *r.Item.RetentionInDays
	}
	return 0 // Never expire
}

// CreationTime returns the creation time in milliseconds
func (r *LogGroupResource) CreationTime() int64 {
	if r.Item.CreationTime != nil {
		return *r.Item.CreationTime
	}
	return 0
}

// LogGroupClass returns the log group class
func (r *LogGroupResource) LogGroupClass() string {
	return string(r.Item.LogGroupClass)
}

// KmsKeyId returns the KMS key ID if encrypted
func (r *LogGroupResource) KmsKeyId() string {
	return appaws.Str(r.Item.KmsKeyId)
}

// MetricFilterCount returns the number of metric filters
func (r *LogGroupResource) MetricFilterCount() int32 {
	if r.Item.MetricFilterCount != nil {
		return *r.Item.MetricFilterCount
	}
	return 0
}

// DataProtectionStatus returns the data protection status
func (r *LogGroupResource) DataProtectionStatus() string {
	return string(r.Item.DataProtectionStatus)
}
