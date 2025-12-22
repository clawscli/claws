package buckets

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3vectors/types"
)

func TestNewVectorBucketResource(t *testing.T) {
	creationTime := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	bucket := &types.VectorBucket{
		VectorBucketName: aws.String("my-vector-bucket"),
		VectorBucketArn:  aws.String("arn:aws:s3vectors:us-east-1:123456789012:bucket/my-vector-bucket"),
		CreationTime:     &creationTime,
		EncryptionConfiguration: &types.EncryptionConfiguration{
			SseType: types.SseTypeAes256,
		},
	}

	resource := NewVectorBucketResource(bucket)

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"GetID", resource.GetID(), "my-vector-bucket"},
		{"GetName", resource.GetName(), "my-vector-bucket"},
		{"GetARN", resource.GetARN(), "arn:aws:s3vectors:us-east-1:123456789012:bucket/my-vector-bucket"},
		{"BucketName", resource.BucketName(), "my-vector-bucket"},
		{"EncryptionType", resource.EncryptionType(), "AES256"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.expected)
			}
		})
	}

	// Test CreationDate
	if resource.CreationDate() != "2024-06-15 10:30:00" {
		t.Errorf("CreationDate() = %q, want %q", resource.CreationDate(), "2024-06-15 10:30:00")
	}

	// Test Item is set
	if resource.Item == nil {
		t.Error("Item should not be nil")
	}
	if resource.Summary != nil {
		t.Error("Summary should be nil for resource created from detail")
	}
}

func TestNewVectorBucketResourceFromSummary(t *testing.T) {
	creationTime := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	summary := types.VectorBucketSummary{
		VectorBucketName: aws.String("my-vector-bucket"),
		VectorBucketArn:  aws.String("arn:aws:s3vectors:us-east-1:123456789012:bucket/my-vector-bucket"),
		CreationTime:     &creationTime,
	}

	resource := NewVectorBucketResourceFromSummary(summary)

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"GetID", resource.GetID(), "my-vector-bucket"},
		{"GetName", resource.GetName(), "my-vector-bucket"},
		{"GetARN", resource.GetARN(), "arn:aws:s3vectors:us-east-1:123456789012:bucket/my-vector-bucket"},
		{"BucketName", resource.BucketName(), "my-vector-bucket"},
		{"CreationDate", resource.CreationDate(), "2024-06-15 10:30:00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.expected)
			}
		})
	}

	// EncryptionType should return "-" when only Summary is available
	if resource.EncryptionType() != "-" {
		t.Errorf("EncryptionType() = %q, want %q for Summary-only resource", resource.EncryptionType(), "-")
	}

	// Summary is set, Item is nil
	if resource.Summary == nil {
		t.Error("Summary should not be nil")
	}
	if resource.Item != nil {
		t.Error("Item should be nil for resource created from summary")
	}
}

func TestVectorBucketResource_KmsEncryption(t *testing.T) {
	bucket := &types.VectorBucket{
		VectorBucketName: aws.String("kms-bucket"),
		VectorBucketArn:  aws.String("arn:aws:s3vectors:us-east-1:123456789012:bucket/kms-bucket"),
		EncryptionConfiguration: &types.EncryptionConfiguration{
			SseType:   types.SseTypeAwsKms,
			KmsKeyArn: aws.String("arn:aws:kms:us-east-1:123456789012:key/abc123"),
		},
	}

	resource := NewVectorBucketResource(bucket)

	if resource.EncryptionType() != "aws:kms" {
		t.Errorf("EncryptionType() = %q, want %q", resource.EncryptionType(), "aws:kms")
	}

	if resource.KmsKeyArn() != "arn:aws:kms:us-east-1:123456789012:key/abc123" {
		t.Errorf("KmsKeyArn() = %q, want KMS key ARN", resource.KmsKeyArn())
	}
}

func TestVectorBucketResource_NilFields(t *testing.T) {
	// Test with completely empty resource
	resource := &VectorBucketResource{}

	if resource.BucketName() != "" {
		t.Errorf("BucketName() = %q, want empty", resource.BucketName())
	}
	if resource.CreationDate() != "" {
		t.Errorf("CreationDate() = %q, want empty", resource.CreationDate())
	}
	if resource.EncryptionType() != "-" {
		t.Errorf("EncryptionType() = %q, want %q", resource.EncryptionType(), "-")
	}
	if resource.KmsKeyArn() != "" {
		t.Errorf("KmsKeyArn() = %q, want empty", resource.KmsKeyArn())
	}
}
