package roles

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

func TestNewRoleResource(t *testing.T) {
	createDate := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	role := types.Role{
		RoleName:           aws.String("my-role"),
		RoleId:             aws.String("AROAEXAMPLE12345"),
		Arn:                aws.String("arn:aws:iam::123456789012:role/my-role"),
		Path:               aws.String("/service-role/"),
		Description:        aws.String("Test role"),
		MaxSessionDuration: aws.Int32(7200),
		CreateDate:         &createDate,
		Tags: []types.Tag{
			{Key: aws.String("Environment"), Value: aws.String("prod")},
		},
	}

	resource := NewRoleResource(role)

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"GetID", resource.GetID(), "my-role"},
		{"GetName", resource.GetName(), "my-role"},
		{"Path", resource.Path(), "/service-role/"},
		{"Arn", resource.Arn(), "arn:aws:iam::123456789012:role/my-role"},
		{"Description", resource.Description(), "Test role"},
		{"MaxSessionDuration", resource.MaxSessionDuration(), int32(7200)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}

	// Test tags
	tags := resource.GetTags()
	if tags["Environment"] != "prod" {
		t.Errorf("GetTags()[Environment] = %q, want %q", tags["Environment"], "prod")
	}
}

func TestRoleResource_MinimalRole(t *testing.T) {
	role := types.Role{
		RoleName: aws.String("minimal-role"),
	}

	resource := NewRoleResource(role)

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"GetID", resource.GetID(), "minimal-role"},
		{"Path", resource.Path(), ""},
		{"Arn", resource.Arn(), ""},
		{"Description", resource.Description(), ""},
		{"MaxSessionDuration", resource.MaxSessionDuration(), int32(0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestRoleResource_PathVariations(t *testing.T) {
	paths := []struct {
		path     *string
		expected string
	}{
		{aws.String("/"), "/"},
		{aws.String("/service-role/"), "/service-role/"},
		{aws.String("/application/admin/"), "/application/admin/"},
		{nil, ""},
	}

	for _, tc := range paths {
		name := "nil"
		if tc.path != nil {
			name = *tc.path
		}
		t.Run(name, func(t *testing.T) {
			role := types.Role{
				RoleName: aws.String("test"),
				Path:     tc.path,
			}
			resource := NewRoleResource(role)
			if got := resource.Path(); got != tc.expected {
				t.Errorf("Path() = %q, want %q", got, tc.expected)
			}
		})
	}
}
