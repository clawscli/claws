package queues

import (
	"testing"
)

func TestNewQueueResource(t *testing.T) {
	queueUrl := "https://sqs.us-east-1.amazonaws.com/123456789012/my-queue"
	attrs := map[string]string{
		"QueueArn":                              "arn:aws:sqs:us-east-1:123456789012:my-queue",
		"ApproximateNumberOfMessages":           "10",
		"ApproximateNumberOfMessagesNotVisible": "5",
		"ApproximateNumberOfMessagesDelayed":    "2",
		"VisibilityTimeout":                     "30",
		"MessageRetentionPeriod":                "345600",
		"DelaySeconds":                          "0",
		"ReceiveMessageWaitTimeSeconds":         "20",
		"CreatedTimestamp":                      "1234567890",
		"LastModifiedTimestamp":                 "1234567899",
		"RedrivePolicy":                         `{"deadLetterTargetArn":"arn:aws:sqs:us-east-1:123456789012:my-dlq","maxReceiveCount":3}`,
	}

	resource := NewQueueResource(queueUrl, attrs)

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"GetID", resource.GetID(), "my-queue"},
		{"GetName", resource.GetName(), "my-queue"},
		{"GetARN", resource.GetARN(), "arn:aws:sqs:us-east-1:123456789012:my-queue"},
		{"URL", resource.URL, queueUrl},
		{"ApproximateNumberOfMessages", resource.ApproximateNumberOfMessages(), "10"},
		{"ApproximateNumberOfMessagesNotVisible", resource.ApproximateNumberOfMessagesNotVisible(), "5"},
		{"ApproximateNumberOfMessagesDelayed", resource.ApproximateNumberOfMessagesDelayed(), "2"},
		{"VisibilityTimeout", resource.VisibilityTimeout(), "30"},
		{"MessageRetentionPeriod", resource.MessageRetentionPeriod(), "345600"},
		{"DelaySeconds", resource.DelaySeconds(), "0"},
		{"ReceiveMessageWaitTimeSeconds", resource.ReceiveMessageWaitTimeSeconds(), "20"},
		{"CreatedTimestamp", resource.CreatedTimestamp(), "1234567890"},
		{"LastModifiedTimestamp", resource.LastModifiedTimestamp(), "1234567899"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestQueueResource_IsFIFO(t *testing.T) {
	tests := []struct {
		queueUrl string
		expected bool
	}{
		{"https://sqs.us-east-1.amazonaws.com/123456789012/standard-queue", false},
		{"https://sqs.us-east-1.amazonaws.com/123456789012/fifo-queue.fifo", true},
	}

	for _, tc := range tests {
		t.Run(tc.queueUrl, func(t *testing.T) {
			resource := NewQueueResource(tc.queueUrl, map[string]string{})
			if got := resource.IsFIFO(); got != tc.expected {
				t.Errorf("IsFIFO() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestQueueResource_MissingAttributes(t *testing.T) {
	queueUrl := "https://sqs.us-east-1.amazonaws.com/123456789012/empty-queue"
	resource := NewQueueResource(queueUrl, map[string]string{})

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"ApproximateNumberOfMessages", resource.ApproximateNumberOfMessages(), "0"},
		{"ApproximateNumberOfMessagesNotVisible", resource.ApproximateNumberOfMessagesNotVisible(), "0"},
		{"ApproximateNumberOfMessagesDelayed", resource.ApproximateNumberOfMessagesDelayed(), "0"},
		{"VisibilityTimeout", resource.VisibilityTimeout(), ""},
		{"MessageRetentionPeriod", resource.MessageRetentionPeriod(), ""},
		{"RedrivePolicy", resource.RedrivePolicy(), ""},
		{"DeadLetterTargetArn", resource.DeadLetterTargetArn(), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.expected)
			}
		})
	}
}
