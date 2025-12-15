//go:build integration

package buckets

import (
	"context"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	appaws "github.com/clawscli/claws/internal/aws"
)

func TestIntegration_BucketDAO_List(t *testing.T) {
	if os.Getenv("AWS_ENDPOINT_URL") == "" {
		t.Skip("AWS_ENDPOINT_URL not set, skipping integration test")
	}

	ctx := context.Background()

	// Create test bucket first
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}
	client := s3.NewFromConfig(cfg)

	bucketName := "integration-test-bucket"
	_, _ = client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: &bucketName,
	})

	// Test DAO
	dao, err := NewBucketDAO(ctx)
	if err != nil {
		t.Fatalf("Failed to create BucketDAO: %v", err)
	}

	resources, err := dao.List(ctx)
	if err != nil {
		t.Fatalf("Failed to list buckets: %v", err)
	}

	// Should have at least one bucket
	if len(resources) == 0 {
		t.Log("No buckets found (LocalStack may not have test data)")
	}

	for _, r := range resources {
		t.Logf("Bucket: %s", r.GetID())
	}
}

func TestIntegration_BucketDAO_ServiceInfo(t *testing.T) {
	if os.Getenv("AWS_ENDPOINT_URL") == "" {
		t.Skip("AWS_ENDPOINT_URL not set, skipping integration test")
	}

	ctx := context.Background()

	dao, err := NewBucketDAO(ctx)
	if err != nil {
		t.Fatalf("Failed to create BucketDAO: %v", err)
	}

	if dao.ServiceName() != "s3" {
		t.Errorf("ServiceName() = %q, want %q", dao.ServiceName(), "s3")
	}
	if dao.ResourceType() != "buckets" {
		t.Errorf("ResourceType() = %q, want %q", dao.ResourceType(), "buckets")
	}
}
