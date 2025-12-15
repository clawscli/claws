//go:build integration

package vpcs

import (
	"context"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	appaws "github.com/clawscli/claws/internal/aws"
)

func TestIntegration_VPCDAO_List(t *testing.T) {
	if os.Getenv("AWS_ENDPOINT_URL") == "" {
		t.Skip("AWS_ENDPOINT_URL not set, skipping integration test")
	}

	ctx := context.Background()

	// Create test VPC first
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}
	client := ec2.NewFromConfig(cfg)

	cidr := "10.99.0.0/16"
	_, _ = client.CreateVpc(ctx, &ec2.CreateVpcInput{
		CidrBlock: &cidr,
	})

	// Test DAO
	dao, err := NewVPCDAO(ctx)
	if err != nil {
		t.Fatalf("Failed to create VPCDAO: %v", err)
	}

	resources, err := dao.List(ctx)
	if err != nil {
		t.Fatalf("Failed to list VPCs: %v", err)
	}

	// Should have at least one VPC (default VPC or created one)
	if len(resources) == 0 {
		t.Log("No VPCs found (LocalStack may not have test data)")
	}

	for _, r := range resources {
		t.Logf("VPC: %s (Name: %s)", r.GetID(), r.GetName())
	}
}

func TestIntegration_VPCDAO_Get(t *testing.T) {
	if os.Getenv("AWS_ENDPOINT_URL") == "" {
		t.Skip("AWS_ENDPOINT_URL not set, skipping integration test")
	}

	ctx := context.Background()

	dao, err := NewVPCDAO(ctx)
	if err != nil {
		t.Fatalf("Failed to create VPCDAO: %v", err)
	}

	// First list to get a VPC ID
	resources, err := dao.List(ctx)
	if err != nil {
		t.Fatalf("Failed to list VPCs: %v", err)
	}

	if len(resources) == 0 {
		t.Skip("No VPCs available to test Get")
	}

	// Test Get
	vpcID := resources[0].GetID()
	resource, err := dao.Get(ctx, vpcID)
	if err != nil {
		t.Fatalf("Failed to get VPC %s: %v", vpcID, err)
	}

	if resource.GetID() != vpcID {
		t.Errorf("GetID() = %q, want %q", resource.GetID(), vpcID)
	}
}

func TestIntegration_VPCDAO_ServiceInfo(t *testing.T) {
	if os.Getenv("AWS_ENDPOINT_URL") == "" {
		t.Skip("AWS_ENDPOINT_URL not set, skipping integration test")
	}

	ctx := context.Background()

	dao, err := NewVPCDAO(ctx)
	if err != nil {
		t.Fatalf("Failed to create VPCDAO: %v", err)
	}

	if dao.ServiceName() != "ec2" {
		t.Errorf("ServiceName() = %q, want %q", dao.ServiceName(), "ec2")
	}
	if dao.ResourceType() != "vpcs" {
		t.Errorf("ResourceType() = %q, want %q", dao.ResourceType(), "vpcs")
	}
}
