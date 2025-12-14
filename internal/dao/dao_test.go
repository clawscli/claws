package dao

import (
	"context"
	"testing"
)

func TestBaseResource(t *testing.T) {
	r := &BaseResource{
		ID:   "test-id",
		Name: "test-name",
		ARN:  "arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0",
		Tags: map[string]string{"Environment": "production", "Team": "platform"},
		Data: map[string]string{"key": "value"},
	}

	if r.GetID() != "test-id" {
		t.Errorf("GetID() = %q, want %q", r.GetID(), "test-id")
	}
	if r.GetName() != "test-name" {
		t.Errorf("GetName() = %q, want %q", r.GetName(), "test-name")
	}
	if r.GetARN() != "arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0" {
		t.Errorf("GetARN() = %q, want ARN", r.GetARN())
	}
	if r.GetTags() == nil {
		t.Error("GetTags() should not be nil")
	}
	if r.GetTags()["Environment"] != "production" {
		t.Errorf("GetTags()[Environment] = %q, want %q", r.GetTags()["Environment"], "production")
	}
	if r.Raw() == nil {
		t.Error("Raw() should not be nil")
	}
}

func TestBaseDAO(t *testing.T) {
	dao := NewBaseDAO("ec2", "instances")

	if dao.ServiceName() != "ec2" {
		t.Errorf("ServiceName() = %q, want %q", dao.ServiceName(), "ec2")
	}
	if dao.ResourceType() != "instances" {
		t.Errorf("ResourceType() = %q, want %q", dao.ResourceType(), "instances")
	}
}

func TestBaseDAO_Supports(t *testing.T) {
	dao := NewBaseDAO("ec2", "instances")

	tests := []struct {
		op   Operation
		want bool
	}{
		{OpList, true},
		{OpGet, true},
		{OpDelete, true},
		{OpCreate, false},
		{OpUpdate, false},
		{Operation("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.op), func(t *testing.T) {
			if got := dao.Supports(tt.op); got != tt.want {
				t.Errorf("Supports(%q) = %v, want %v", tt.op, got, tt.want)
			}
		})
	}
}

func TestWithFilter(t *testing.T) {
	ctx := context.Background()

	// Add filter
	ctx = WithFilter(ctx, "VpcId", "vpc-123")

	// Retrieve filter
	got := GetFilterFromContext(ctx, "VpcId")
	if got != "vpc-123" {
		t.Errorf("GetFilterFromContext() = %q, want %q", got, "vpc-123")
	}
}

func TestGetFilterFromContext_NotFound(t *testing.T) {
	ctx := context.Background()

	// Try to get non-existent filter
	got := GetFilterFromContext(ctx, "NonExistent")
	if got != "" {
		t.Errorf("GetFilterFromContext() = %q, want empty string", got)
	}
}

func TestWithFilter_MultipleFilters(t *testing.T) {
	ctx := context.Background()

	// Add multiple filters
	ctx = WithFilter(ctx, "VpcId", "vpc-123")
	ctx = WithFilter(ctx, "SubnetId", "subnet-456")

	// Both should be retrievable
	if got := GetFilterFromContext(ctx, "VpcId"); got != "vpc-123" {
		t.Errorf("GetFilterFromContext(VpcId) = %q, want %q", got, "vpc-123")
	}
	if got := GetFilterFromContext(ctx, "SubnetId"); got != "subnet-456" {
		t.Errorf("GetFilterFromContext(SubnetId) = %q, want %q", got, "subnet-456")
	}
}

func TestOperationConstants(t *testing.T) {
	// Verify operation constants have expected values
	tests := []struct {
		op   Operation
		want string
	}{
		{OpList, "list"},
		{OpGet, "get"},
		{OpCreate, "create"},
		{OpDelete, "delete"},
		{OpUpdate, "update"},
	}

	for _, tt := range tests {
		if string(tt.op) != tt.want {
			t.Errorf("Operation %v = %q, want %q", tt.op, string(tt.op), tt.want)
		}
	}
}
