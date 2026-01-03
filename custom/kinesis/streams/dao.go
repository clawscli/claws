package streams

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// StreamDAO provides data access for Kinesis streams
type StreamDAO struct {
	dao.BaseDAO
	client *kinesis.Client
}

// NewStreamDAO creates a new StreamDAO
func NewStreamDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &StreamDAO{
		BaseDAO: dao.NewBaseDAO("kinesis", "streams"),
		client:  kinesis.NewFromConfig(cfg),
	}, nil
}

// List returns all Kinesis streams
func (d *StreamDAO) List(ctx context.Context) ([]dao.Resource, error) {
	streams, err := appaws.Paginate(ctx, func(token *string) ([]types.StreamSummary, *string, error) {
		output, err := d.client.ListStreams(ctx, &kinesis.ListStreamsInput{
			NextToken: token,
			Limit:     appaws.Int32Ptr(100),
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list streams")
		}
		// Kinesis uses HasMoreStreams instead of just checking NextToken
		if output.HasMoreStreams != nil && *output.HasMoreStreams {
			return output.StreamSummaries, output.NextToken, nil
		}
		return output.StreamSummaries, nil, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(streams))
	for i, summary := range streams {
		resources[i] = NewStreamResource(summary)
	}

	return resources, nil
}

// Get returns a specific Kinesis stream by name
func (d *StreamDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	input := &kinesis.DescribeStreamSummaryInput{
		StreamName: &id,
	}

	output, err := d.client.DescribeStreamSummary(ctx, input)
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe stream summary %s", id)
	}

	return NewStreamResourceFromSummary(output.StreamDescriptionSummary), nil
}

// Delete deletes a Kinesis stream
func (d *StreamDAO) Delete(ctx context.Context, id string) error {
	input := &kinesis.DeleteStreamInput{
		StreamName: &id,
	}

	_, err := d.client.DeleteStream(ctx, input)
	if err != nil {
		return apperrors.Wrapf(err, "delete stream %s", id)
	}

	return nil
}

// StreamResource represents a Kinesis stream
type StreamResource struct {
	dao.BaseResource
	Summary     *types.StreamSummary
	Description *types.StreamDescriptionSummary
}

// NewStreamResource creates a new StreamResource from StreamSummary
func NewStreamResource(summary types.StreamSummary) *StreamResource {
	streamName := appaws.Str(summary.StreamName)
	arn := appaws.Str(summary.StreamARN)

	return &StreamResource{
		BaseResource: dao.BaseResource{
			ID:   streamName,
			Name: streamName,
			ARN:  arn,
			Tags: make(map[string]string),
			Data: summary,
		},
		Summary: &summary,
	}
}

// NewStreamResourceFromSummary creates a new StreamResource from StreamDescriptionSummary
func NewStreamResourceFromSummary(desc *types.StreamDescriptionSummary) *StreamResource {
	streamName := appaws.Str(desc.StreamName)
	arn := appaws.Str(desc.StreamARN)

	return &StreamResource{
		BaseResource: dao.BaseResource{
			ID:   streamName,
			Name: streamName,
			ARN:  arn,
			Tags: make(map[string]string),
			Data: desc,
		},
		Description: desc,
	}
}

// StreamName returns the stream name
func (r *StreamResource) StreamName() string {
	if r.Summary != nil {
		return appaws.Str(r.Summary.StreamName)
	}
	if r.Description != nil {
		return appaws.Str(r.Description.StreamName)
	}
	return ""
}

// Status returns the stream status
func (r *StreamResource) Status() string {
	if r.Summary != nil {
		return string(r.Summary.StreamStatus)
	}
	if r.Description != nil {
		return string(r.Description.StreamStatus)
	}
	return ""
}

// StreamMode returns the stream mode (ON_DEMAND or PROVISIONED)
func (r *StreamResource) StreamMode() string {
	if r.Summary != nil && r.Summary.StreamModeDetails != nil {
		return string(r.Summary.StreamModeDetails.StreamMode)
	}
	if r.Description != nil && r.Description.StreamModeDetails != nil {
		return string(r.Description.StreamModeDetails.StreamMode)
	}
	return ""
}

// ShardCount returns the number of open shards
func (r *StreamResource) ShardCount() int32 {
	if r.Description != nil && r.Description.OpenShardCount != nil {
		return *r.Description.OpenShardCount
	}
	return 0
}

// RetentionPeriodHours returns the retention period in hours
func (r *StreamResource) RetentionPeriodHours() int32 {
	if r.Description != nil && r.Description.RetentionPeriodHours != nil {
		return *r.Description.RetentionPeriodHours
	}
	return 24 // Default
}

// EncryptionType returns the encryption type
func (r *StreamResource) EncryptionType() string {
	if r.Description != nil {
		return string(r.Description.EncryptionType)
	}
	return ""
}

// KeyId returns the KMS key ID for encryption
func (r *StreamResource) KeyId() string {
	if r.Description != nil {
		return appaws.Str(r.Description.KeyId)
	}
	return ""
}

// CreatedAt returns the creation time as a formatted string
func (r *StreamResource) CreatedAt() string {
	if r.Summary != nil && r.Summary.StreamCreationTimestamp != nil {
		return r.Summary.StreamCreationTimestamp.Format("2006-01-02 15:04:05")
	}
	if r.Description != nil && r.Description.StreamCreationTimestamp != nil {
		return r.Description.StreamCreationTimestamp.Format("2006-01-02 15:04:05")
	}
	return ""
}

// ConsumerCount returns the number of consumers
func (r *StreamResource) ConsumerCount() int32 {
	if r.Description != nil && r.Description.ConsumerCount != nil {
		return *r.Description.ConsumerCount
	}
	return 0
}

// EnhancedMonitoring returns the enhanced monitoring shard-level metrics
func (r *StreamResource) EnhancedMonitoring() []string {
	if r.Description != nil && len(r.Description.EnhancedMonitoring) > 0 {
		var metrics []string
		for _, em := range r.Description.EnhancedMonitoring {
			for _, m := range em.ShardLevelMetrics {
				metrics = append(metrics, string(m))
			}
		}
		return metrics
	}
	return nil
}
