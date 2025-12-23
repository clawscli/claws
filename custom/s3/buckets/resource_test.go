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

func TestBucketResource_BucketRegion(t *testing.T) {
	// Test that BucketRegion from ListBuckets response is handled correctly
	bucket := types.Bucket{
		Name:         aws.String("regional-bucket"),
		BucketRegion: aws.String("ap-northeast-1"),
	}

	resource := NewBucketResource(bucket)

	// BucketRegion is set separately in DAO.List(), so initially empty
	if resource.Region != "" {
		t.Errorf("Region should be empty initially, got %q", resource.Region)
	}

	// Simulate what DAO.List() does
	if bucket.BucketRegion != nil {
		resource.Region = *bucket.BucketRegion
	}

	if resource.Region != "ap-northeast-1" {
		t.Errorf("Region = %q, want %q", resource.Region, "ap-northeast-1")
	}
}

func TestBucketResource_BucketRegionNil(t *testing.T) {
	// Test that nil BucketRegion doesn't cause issues
	bucket := types.Bucket{
		Name:         aws.String("no-region-bucket"),
		BucketRegion: nil,
	}

	resource := NewBucketResource(bucket)

	// Region should remain empty when BucketRegion is nil
	if resource.Region != "" {
		t.Errorf("Region should be empty for nil BucketRegion, got %q", resource.Region)
	}
}

func TestBucketResource_MergeFrom(t *testing.T) {
	creationTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	// Original from List() has CreationDate
	original := &BucketResource{
		BucketName:   "test-bucket",
		CreationDate: creationTime,
	}

	// Refreshed from Get() has extended info but no CreationDate
	refreshed := &BucketResource{
		BucketName: "test-bucket",
		Region:     "us-west-2",
		Versioning: "Enabled",
		// CreationDate is zero
	}

	refreshed.MergeFrom(original)

	// CreationDate should be preserved from original
	if !refreshed.CreationDate.Equal(creationTime) {
		t.Errorf("CreationDate = %v, want %v", refreshed.CreationDate, creationTime)
	}

	// Refreshed fields should be retained
	if refreshed.Versioning != "Enabled" {
		t.Errorf("Versioning = %q, want %q", refreshed.Versioning, "Enabled")
	}
	if refreshed.Region != "us-west-2" {
		t.Errorf("Region = %q, want %q", refreshed.Region, "us-west-2")
	}
}

func TestBucketResource_MergeFrom_NoOverwrite(t *testing.T) {
	originalTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	refreshedTime := time.Date(2024, 6, 20, 15, 45, 0, 0, time.UTC)

	original := &BucketResource{
		CreationDate: originalTime,
	}

	// If refreshed already has CreationDate, don't overwrite
	refreshed := &BucketResource{
		CreationDate: refreshedTime,
	}

	refreshed.MergeFrom(original)

	// Should keep refreshed value, not original
	if !refreshed.CreationDate.Equal(refreshedTime) {
		t.Errorf("CreationDate = %v, want %v (should not overwrite)", refreshed.CreationDate, refreshedTime)
	}
}

func TestBucketResource_MergeFrom_WrongType(t *testing.T) {
	creationTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	refreshed := &BucketResource{
		BucketName:   "test-bucket",
		CreationDate: creationTime,
	}

	// Should not panic when passed nil
	refreshed.MergeFrom(nil)

	// Should not panic and should not modify when passed wrong type
	other := &otherResource{}
	refreshed.MergeFrom(other)

	// CreationDate should remain unchanged
	if !refreshed.CreationDate.Equal(creationTime) {
		t.Errorf("CreationDate should not change when merging wrong type")
	}
}

// otherResource is a mock for testing wrong type handling
type otherResource struct{}

func (r *otherResource) GetID() string              { return "" }
func (r *otherResource) GetName() string            { return "" }
func (r *otherResource) GetARN() string             { return "" }
func (r *otherResource) GetTags() map[string]string { return nil }
func (r *otherResource) Raw() any                   { return nil }
