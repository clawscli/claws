//go:build integration

package aws

import (
	"context"
	"os"
	"testing"
)

// Integration tests require LocalStack to be running
// Run with: task test-localstack
// or: AWS_ENDPOINT_URL=http://localhost:4566 go test -v ./... -tags=integration

func TestIntegration_EC2Client(t *testing.T) {
	if os.Getenv("AWS_ENDPOINT_URL") == "" {
		t.Skip("AWS_ENDPOINT_URL not set, skipping integration test")
	}

	ctx := context.Background()

	client, err := Global().EC2(ctx)
	if err != nil {
		t.Fatalf("Failed to get EC2 client: %v", err)
	}

	if client == nil {
		t.Fatal("EC2 client should not be nil")
	}
}

func TestIntegration_S3Client(t *testing.T) {
	if os.Getenv("AWS_ENDPOINT_URL") == "" {
		t.Skip("AWS_ENDPOINT_URL not set, skipping integration test")
	}

	ctx := context.Background()

	client, err := Global().S3(ctx)
	if err != nil {
		t.Fatalf("Failed to get S3 client: %v", err)
	}

	if client == nil {
		t.Fatal("S3 client should not be nil")
	}
}

func TestIntegration_IAMClient(t *testing.T) {
	if os.Getenv("AWS_ENDPOINT_URL") == "" {
		t.Skip("AWS_ENDPOINT_URL not set, skipping integration test")
	}

	ctx := context.Background()

	client, err := Global().IAM(ctx)
	if err != nil {
		t.Fatalf("Failed to get IAM client: %v", err)
	}

	if client == nil {
		t.Fatal("IAM client should not be nil")
	}
}

func TestIntegration_CloudFormationClient(t *testing.T) {
	if os.Getenv("AWS_ENDPOINT_URL") == "" {
		t.Skip("AWS_ENDPOINT_URL not set, skipping integration test")
	}

	ctx := context.Background()

	client, err := Global().CloudFormation(ctx)
	if err != nil {
		t.Fatalf("Failed to get CloudFormation client: %v", err)
	}

	if client == nil {
		t.Fatal("CloudFormation client should not be nil")
	}
}

func TestIntegration_NewConfig(t *testing.T) {
	if os.Getenv("AWS_ENDPOINT_URL") == "" {
		t.Skip("AWS_ENDPOINT_URL not set, skipping integration test")
	}

	ctx := context.Background()

	cfg, err := NewConfig(ctx)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Check that region is set
	if cfg.Region == "" {
		t.Error("Region should be set")
	}
}
