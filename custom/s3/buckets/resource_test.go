package buckets

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func TestNewBucketResource(t *testing.T) {
	creationTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	bucket := types.Bucket{
		Name:         aws.String("my-bucket"),
		CreationDate: &creationTime,
	}

	resource := NewBucketResource(bucket)

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"GetID", resource.GetID(), "my-bucket"},
		{"GetName", resource.GetName(), "my-bucket"},
		{"BucketName", resource.BucketName, "my-bucket"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.expected)
			}
		})
	}

	// Test CreationDate
	if !resource.CreationDate.Equal(creationTime) {
		t.Errorf("CreationDate = %v, want %v", resource.CreationDate, creationTime)
	}

	// Test Age (should be > 0 for past date)
	if resource.Age() <= 0 {
		t.Errorf("Age() = %v, want > 0", resource.Age())
	}
}

func TestBucketResource_MinimalBucket(t *testing.T) {
	bucket := types.Bucket{
		Name: aws.String("minimal-bucket"),
	}

	resource := NewBucketResource(bucket)

	if resource.GetID() != "minimal-bucket" {
		t.Errorf("GetID() = %q, want %q", resource.GetID(), "minimal-bucket")
	}

	if !resource.CreationDate.IsZero() {
		t.Errorf("CreationDate should be zero for minimal bucket, got %v", resource.CreationDate)
	}

	if resource.Age() != 0 {
		t.Errorf("Age() should be 0 for zero creation date, got %v", resource.Age())
	}
}

func TestBucketResource_ExtendedInfo(t *testing.T) {
	bucket := types.Bucket{
		Name: aws.String("test-bucket"),
	}

	resource := NewBucketResource(bucket)

	// Set extended info (normally done by DAO.Get)
	resource.Region = "us-west-2"
	resource.Versioning = "Enabled"
	resource.MFADelete = "Disabled"
	resource.EncryptionEnabled = true
	resource.EncryptionAlgorithm = "aws:kms"
	resource.EncryptionKMSKeyID = "arn:aws:kms:us-west-2:123456789012:key/abc123"
	resource.BucketKeyEnabled = true
	resource.LifecycleRulesCount = 3
	resource.ObjectLockEnabled = true
	resource.ObjectLockMode = "GOVERNANCE"
	resource.ObjectLockRetention = "30 days"
	resource.PublicAccessBlock = &PublicAccessBlockInfo{
		BlockPublicAcls:       true,
		IgnorePublicAcls:      true,
		BlockPublicPolicy:     true,
		RestrictPublicBuckets: true,
	}

	// Verify extended info
	if resource.Region != "us-west-2" {
		t.Errorf("Region = %q, want %q", resource.Region, "us-west-2")
	}
	if resource.Versioning != "Enabled" {
		t.Errorf("Versioning = %q, want %q", resource.Versioning, "Enabled")
	}
	if !resource.EncryptionEnabled {
		t.Error("EncryptionEnabled should be true")
	}
	if resource.LifecycleRulesCount != 3 {
		t.Errorf("LifecycleRulesCount = %d, want %d", resource.LifecycleRulesCount, 3)
	}
	if !resource.ObjectLockEnabled {
		t.Error("ObjectLockEnabled should be true")
	}
	if resource.PublicAccessBlock == nil {
		t.Fatal("PublicAccessBlock should not be nil")
	}
	if !resource.PublicAccessBlock.BlockPublicAcls {
		t.Error("BlockPublicAcls should be true")
	}
}

func TestBucketResource_NilName(t *testing.T) {
	bucket := types.Bucket{
		Name: nil,
	}

	resource := NewBucketResource(bucket)

	if resource.GetID() != "" {
		t.Errorf("GetID() = %q, want empty string for nil name", resource.GetID())
	}
	if resource.BucketName != "" {
		t.Errorf("BucketName = %q, want empty string for nil name", resource.BucketName)
	}
}
