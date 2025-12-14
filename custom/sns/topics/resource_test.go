package topics

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
)

func TestNewTopicResource(t *testing.T) {
	topic := types.Topic{
		TopicArn: aws.String("arn:aws:sns:us-east-1:123456789012:my-topic"),
	}
	attrs := map[string]string{
		"DisplayName":            "My Topic Display Name",
		"SubscriptionsConfirmed": "5",
		"SubscriptionsPending":   "2",
		"Owner":                  "123456789012",
		"FifoTopic":              "false",
	}

	resource := NewTopicResource(topic, attrs)

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"GetID", resource.GetID(), "arn:aws:sns:us-east-1:123456789012:my-topic"},
		{"GetName", resource.GetName(), "my-topic"},
		{"ARN", resource.ARN(), "arn:aws:sns:us-east-1:123456789012:my-topic"},
		{"DisplayName", resource.DisplayName(), "My Topic Display Name"},
		{"SubscriptionCount", resource.SubscriptionCount(), "5"},
		{"PendingSubscriptions", resource.PendingSubscriptions(), "2"},
		{"Owner", resource.Owner(), "123456789012"},
		{"IsFIFO", resource.IsFIFO(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestTopicResource_FIFOTopic(t *testing.T) {
	topic := types.Topic{
		TopicArn: aws.String("arn:aws:sns:us-east-1:123456789012:my-topic.fifo"),
	}
	attrs := map[string]string{
		"FifoTopic": "true",
	}

	resource := NewTopicResource(topic, attrs)

	if !resource.IsFIFO() {
		t.Error("IsFIFO() should be true for FIFO topic")
	}
	if resource.GetName() != "my-topic.fifo" {
		t.Errorf("GetName() = %q, want %q", resource.GetName(), "my-topic.fifo")
	}
}

func TestTopicResource_MinimalTopic(t *testing.T) {
	topic := types.Topic{
		TopicArn: aws.String("arn:aws:sns:us-east-1:123456789012:minimal"),
	}
	attrs := map[string]string{}

	resource := NewTopicResource(topic, attrs)

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"GetName", resource.GetName(), "minimal"},
		{"DisplayName", resource.DisplayName(), ""},
		{"SubscriptionCount", resource.SubscriptionCount(), "0"},
		{"PendingSubscriptions", resource.PendingSubscriptions(), "0"},
		{"Owner", resource.Owner(), ""},
		{"IsFIFO", resource.IsFIFO(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestTopicResource_NilArn(t *testing.T) {
	topic := types.Topic{
		TopicArn: nil,
	}
	attrs := map[string]string{}

	resource := NewTopicResource(topic, attrs)

	if resource.GetID() != "" {
		t.Errorf("GetID() = %q, want empty string for nil ARN", resource.GetID())
	}
	if resource.GetName() != "" {
		t.Errorf("GetName() = %q, want empty string for nil ARN", resource.GetName())
	}
	if resource.ARN() != "" {
		t.Errorf("ARN() = %q, want empty string for nil ARN", resource.ARN())
	}
}

func TestTopicResource_NameExtraction(t *testing.T) {
	tests := []struct {
		arn      string
		expected string
	}{
		{"arn:aws:sns:us-east-1:123456789012:my-topic", "my-topic"},
		{"arn:aws:sns:us-east-1:123456789012:another-topic.fifo", "another-topic.fifo"},
		{"arn:aws:sns:eu-west-1:999999999999:topic-with-dashes-123", "topic-with-dashes-123"},
		{"simple-name", "simple-name"},
	}

	for _, tc := range tests {
		t.Run(tc.arn, func(t *testing.T) {
			topic := types.Topic{
				TopicArn: aws.String(tc.arn),
			}
			resource := NewTopicResource(topic, map[string]string{})
			if got := resource.GetName(); got != tc.expected {
				t.Errorf("GetName() = %q, want %q", got, tc.expected)
			}
		})
	}
}
