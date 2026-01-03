package queues

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
	"github.com/clawscli/claws/internal/log"
)

// QueueDAO provides data access for SQS queues
type QueueDAO struct {
	dao.BaseDAO
	client *sqs.Client
}

// NewQueueDAO creates a new QueueDAO
func NewQueueDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &QueueDAO{
		BaseDAO: dao.NewBaseDAO("sqs", "queues"),
		client:  sqs.NewFromConfig(cfg),
	}, nil
}

func (d *QueueDAO) List(ctx context.Context) ([]dao.Resource, error) {
	queueUrls, err := appaws.Paginate(ctx, func(token *string) ([]string, *string, error) {
		output, err := d.client.ListQueues(ctx, &sqs.ListQueuesInput{
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "list queues")
		}
		return output.QueueUrls, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, 0, len(queueUrls))
	for _, queueUrl := range queueUrls {
		// Get queue attributes
		attrsOutput, err := d.client.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
			QueueUrl: &queueUrl,
			AttributeNames: []types.QueueAttributeName{
				types.QueueAttributeNameAll,
			},
		})
		if err != nil {
			log.Warn("failed to get queue attributes", "queueUrl", queueUrl, "error", err)
			continue
		}
		resources = append(resources, NewQueueResource(queueUrl, attrsOutput.Attributes))
	}
	return resources, nil
}

func (d *QueueDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	// id could be queue URL or queue name
	queueUrl := id
	if !strings.HasPrefix(id, "https://") {
		// Get queue URL from name
		urlInput := &sqs.GetQueueUrlInput{
			QueueName: &id,
		}
		urlOutput, err := d.client.GetQueueUrl(ctx, urlInput)
		if err != nil {
			return nil, apperrors.Wrapf(err, "get queue URL for %s", id)
		}
		queueUrl = *urlOutput.QueueUrl
	}

	input := &sqs.GetQueueAttributesInput{
		QueueUrl: &queueUrl,
		AttributeNames: []types.QueueAttributeName{
			types.QueueAttributeNameAll,
		},
	}

	output, err := d.client.GetQueueAttributes(ctx, input)
	if err != nil {
		return nil, apperrors.Wrapf(err, "get queue attributes %s", id)
	}

	return NewQueueResource(queueUrl, output.Attributes), nil
}

func (d *QueueDAO) Delete(ctx context.Context, id string) error {
	queueUrl := id
	if !strings.HasPrefix(id, "https://") {
		urlInput := &sqs.GetQueueUrlInput{
			QueueName: &id,
		}
		urlOutput, err := d.client.GetQueueUrl(ctx, urlInput)
		if err != nil {
			if apperrors.IsNotFound(err) {
				return nil // Already deleted
			}
			return apperrors.Wrapf(err, "get queue URL for %s", id)
		}
		queueUrl = *urlOutput.QueueUrl
	}

	input := &sqs.DeleteQueueInput{
		QueueUrl: &queueUrl,
	}

	_, err := d.client.DeleteQueue(ctx, input)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return nil // Already deleted
		}
		if apperrors.IsResourceInUse(err) {
			return apperrors.Wrapf(err, "queue %s is in use", id)
		}
		return apperrors.Wrapf(err, "delete queue %s", id)
	}

	return nil
}

// QueueResource wraps an SQS queue
type QueueResource struct {
	dao.BaseResource
	URL        string
	Attributes map[string]string
}

// NewQueueResource creates a new QueueResource
func NewQueueResource(queueUrl string, attrs map[string]string) *QueueResource {
	// Extract queue name from URL
	queueName := appaws.ExtractResourceName(queueUrl)

	arn := ""
	if v, ok := attrs["QueueArn"]; ok {
		arn = v
	}

	return &QueueResource{
		BaseResource: dao.BaseResource{
			ID:   queueName,
			Name: queueName,
			ARN:  arn,
			Data: attrs,
		},
		URL:        queueUrl,
		Attributes: attrs,
	}
}

// IsFIFO returns true if this is a FIFO queue
func (r *QueueResource) IsFIFO() bool {
	return strings.HasSuffix(r.GetName(), ".fifo")
}

// ApproximateNumberOfMessages returns the approximate message count
func (r *QueueResource) ApproximateNumberOfMessages() string {
	if v, ok := r.Attributes["ApproximateNumberOfMessages"]; ok {
		return v
	}
	return "0"
}

// ApproximateNumberOfMessagesNotVisible returns messages in flight
func (r *QueueResource) ApproximateNumberOfMessagesNotVisible() string {
	if v, ok := r.Attributes["ApproximateNumberOfMessagesNotVisible"]; ok {
		return v
	}
	return "0"
}

// ApproximateNumberOfMessagesDelayed returns delayed messages
func (r *QueueResource) ApproximateNumberOfMessagesDelayed() string {
	if v, ok := r.Attributes["ApproximateNumberOfMessagesDelayed"]; ok {
		return v
	}
	return "0"
}

// VisibilityTimeout returns the visibility timeout in seconds
func (r *QueueResource) VisibilityTimeout() string {
	if v, ok := r.Attributes["VisibilityTimeout"]; ok {
		return v
	}
	return ""
}

// MessageRetentionPeriod returns retention period in seconds
func (r *QueueResource) MessageRetentionPeriod() string {
	if v, ok := r.Attributes["MessageRetentionPeriod"]; ok {
		return v
	}
	return ""
}

// DelaySeconds returns the delay in seconds
func (r *QueueResource) DelaySeconds() string {
	if v, ok := r.Attributes["DelaySeconds"]; ok {
		return v
	}
	return ""
}

// ReceiveMessageWaitTimeSeconds returns the long polling wait time
func (r *QueueResource) ReceiveMessageWaitTimeSeconds() string {
	if v, ok := r.Attributes["ReceiveMessageWaitTimeSeconds"]; ok {
		return v
	}
	return ""
}

// CreatedTimestamp returns when the queue was created
func (r *QueueResource) CreatedTimestamp() string {
	if v, ok := r.Attributes["CreatedTimestamp"]; ok {
		return v
	}
	return ""
}

// LastModifiedTimestamp returns when the queue was last modified
func (r *QueueResource) LastModifiedTimestamp() string {
	if v, ok := r.Attributes["LastModifiedTimestamp"]; ok {
		return v
	}
	return ""
}

// RedrivePolicy returns the redrive (DLQ) policy
func (r *QueueResource) RedrivePolicy() string {
	if v, ok := r.Attributes["RedrivePolicy"]; ok {
		return v
	}
	return ""
}

// DeadLetterTargetArn returns the DLQ ARN if configured
func (r *QueueResource) DeadLetterTargetArn() string {
	if v, ok := r.Attributes["DeadLetterTargetArn"]; ok {
		return v
	}
	return ""
}
