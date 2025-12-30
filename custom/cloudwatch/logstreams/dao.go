package logstreams

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// LogStreamDAO provides data access for CloudWatch Log Streams
type LogStreamDAO struct {
	dao.BaseDAO
	client *cloudwatchlogs.Client
}

// NewLogStreamDAO creates a new LogStreamDAO
func NewLogStreamDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new cloudwatch/logstreams dao")
	}
	return &LogStreamDAO{
		BaseDAO: dao.NewBaseDAO("cloudwatch", "log-streams"),
		client:  cloudwatchlogs.NewFromConfig(cfg),
	}, nil
}

// List returns log streams (first page only for backwards compatibility).
// For paginated access, use ListPage instead.
func (d *LogStreamDAO) List(ctx context.Context) ([]dao.Resource, error) {
	resources, _, err := d.ListPage(ctx, 50, "")
	return resources, err
}

// ListPage returns a page of log streams.
// Implements dao.PaginatedDAO interface.
func (d *LogStreamDAO) ListPage(ctx context.Context, pageSize int, pageToken string) ([]dao.Resource, string, error) {
	logGroupName := dao.GetFilterFromContext(ctx, "LogGroupName")
	if logGroupName == "" {
		return nil, "", fmt.Errorf("LogGroupName required: navigate from log-groups using 's' key")
	}

	limit := int32(pageSize)
	if limit > 50 {
		limit = 50 // AWS API max
	}

	input := &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName: &logGroupName,
		Descending:   appaws.BoolPtr(true),
		OrderBy:      types.OrderByLastEventTime,
		Limit:        &limit,
	}
	if pageToken != "" {
		input.NextToken = &pageToken
	}

	output, err := d.client.DescribeLogStreams(ctx, input)
	if err != nil {
		return nil, "", apperrors.Wrap(err, "describe log streams")
	}

	resources := make([]dao.Resource, 0, len(output.LogStreams))
	for _, ls := range output.LogStreams {
		resources = append(resources, NewLogStreamResource(ls, logGroupName))
	}

	nextToken := ""
	if output.NextToken != nil {
		nextToken = *output.NextToken
	}

	return resources, nextToken, nil
}

func (d *LogStreamDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	logGroupName := dao.GetFilterFromContext(ctx, "LogGroupName")
	if logGroupName == "" {
		return nil, fmt.Errorf("LogGroupName required: navigate from log-groups using 's' key")
	}

	input := &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName:        &logGroupName,
		LogStreamNamePrefix: &id,
	}

	output, err := d.client.DescribeLogStreams(ctx, input)
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe log stream %s", id)
	}

	// Find exact match
	for _, ls := range output.LogStreams {
		if ls.LogStreamName != nil && *ls.LogStreamName == id {
			return NewLogStreamResource(ls, logGroupName), nil
		}
	}

	return nil, fmt.Errorf("log stream not found: %s", id)
}

func (d *LogStreamDAO) Delete(ctx context.Context, id string) error {
	logGroupName := dao.GetFilterFromContext(ctx, "LogGroupName")
	if logGroupName == "" {
		return fmt.Errorf("LogGroupName required: navigate from log-groups using 's' key")
	}

	input := &cloudwatchlogs.DeleteLogStreamInput{
		LogGroupName:  &logGroupName,
		LogStreamName: &id,
	}

	_, err := d.client.DeleteLogStream(ctx, input)
	if err != nil {
		return apperrors.Wrapf(err, "delete log stream %s", id)
	}

	return nil
}

// LogStreamResource wraps a CloudWatch Log Stream
type LogStreamResource struct {
	dao.BaseResource
	Item         types.LogStream
	logGroupName string
}

// NewLogStreamResource creates a new LogStreamResource
func NewLogStreamResource(ls types.LogStream, logGroupName string) *LogStreamResource {
	name := appaws.Str(ls.LogStreamName)

	return &LogStreamResource{
		BaseResource: dao.BaseResource{
			ID:   name,
			Name: name,
			ARN:  appaws.Str(ls.Arn),
			Tags: nil,
			Data: ls,
		},
		Item:         ls,
		logGroupName: logGroupName,
	}
}

// LogStreamName returns the log stream name
func (r *LogStreamResource) LogStreamName() string {
	return appaws.Str(r.Item.LogStreamName)
}

// LogGroupName returns the parent log group name
func (r *LogStreamResource) LogGroupName() string {
	return r.logGroupName
}

// FirstEventTimestamp returns the first event timestamp
func (r *LogStreamResource) FirstEventTimestamp() int64 {
	if r.Item.FirstEventTimestamp != nil {
		return *r.Item.FirstEventTimestamp
	}
	return 0
}

// LastEventTimestamp returns the last event timestamp
func (r *LogStreamResource) LastEventTimestamp() int64 {
	if r.Item.LastEventTimestamp != nil {
		return *r.Item.LastEventTimestamp
	}
	return 0
}

// LastIngestionTime returns the last ingestion time
func (r *LogStreamResource) LastIngestionTime() int64 {
	if r.Item.LastIngestionTime != nil {
		return *r.Item.LastIngestionTime
	}
	return 0
}

// CreationTime returns the creation time
func (r *LogStreamResource) CreationTime() int64 {
	if r.Item.CreationTime != nil {
		return *r.Item.CreationTime
	}
	return 0
}

// UploadSequenceToken returns the upload sequence token
func (r *LogStreamResource) UploadSequenceToken() string {
	return appaws.Str(r.Item.UploadSequenceToken)
}
