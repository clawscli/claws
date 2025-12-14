package instances

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestNewInstanceResourceWithRole(t *testing.T) {
	instance := types.Instance{
		InstanceId:       aws.String("i-1234567890abcdef0"),
		InstanceType:     types.InstanceTypeT2Micro,
		PrivateIpAddress: aws.String("10.0.0.1"),
		PublicIpAddress:  aws.String("54.1.2.3"),
		State: &types.InstanceState{
			Name: types.InstanceStateNameRunning,
		},
		Placement: &types.Placement{
			AvailabilityZone: aws.String("us-east-1a"),
		},
		Tags: []types.Tag{
			{Key: aws.String("Name"), Value: aws.String("test-instance")},
			{Key: aws.String("Environment"), Value: aws.String("prod")},
		},
	}

	resource := NewInstanceResourceWithRole(instance, "test-role")

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"GetID", resource.GetID(), "i-1234567890abcdef0"},
		{"GetName", resource.GetName(), "test-instance"},
		{"GetRoleName", resource.GetRoleName(), "test-role"},
		{"State", resource.State(), "running"},
		{"InstanceType", resource.InstanceType(), "t2.micro"},
		{"PrivateIP", resource.PrivateIP(), "10.0.0.1"},
		{"PublicIP", resource.PublicIP(), "54.1.2.3"},
		{"AZ", resource.AZ(), "us-east-1a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.expected)
			}
		})
	}

	// Test tags
	tags := resource.GetTags()
	if tags["Name"] != "test-instance" {
		t.Errorf("GetTags()[Name] = %q, want %q", tags["Name"], "test-instance")
	}
	if tags["Environment"] != "prod" {
		t.Errorf("GetTags()[Environment] = %q, want %q", tags["Environment"], "prod")
	}
}

func TestInstanceResource_NilFields(t *testing.T) {
	// Test with minimal instance (many nil fields)
	instance := types.Instance{
		InstanceId:   aws.String("i-minimal"),
		InstanceType: types.InstanceTypeT2Micro,
	}

	resource := NewInstanceResourceWithRole(instance, "")

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"GetID", resource.GetID(), "i-minimal"},
		{"GetName", resource.GetName(), ""}, // Empty when no Name tag
		{"GetRoleName", resource.GetRoleName(), ""},
		{"State", resource.State(), "unknown"},
		{"PrivateIP", resource.PrivateIP(), ""},
		{"PublicIP", resource.PublicIP(), ""},
		{"AZ", resource.AZ(), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestInstanceResource_StateVariations(t *testing.T) {
	states := []struct {
		state    types.InstanceStateName
		expected string
	}{
		{types.InstanceStateNamePending, "pending"},
		{types.InstanceStateNameRunning, "running"},
		{types.InstanceStateNameStopping, "stopping"},
		{types.InstanceStateNameStopped, "stopped"},
		{types.InstanceStateNameTerminated, "terminated"},
	}

	for _, tc := range states {
		t.Run(string(tc.state), func(t *testing.T) {
			instance := types.Instance{
				InstanceId: aws.String("i-test"),
				State: &types.InstanceState{
					Name: tc.state,
				},
			}
			resource := NewInstanceResourceWithRole(instance, "")
			if got := resource.State(); got != tc.expected {
				t.Errorf("State() = %q, want %q", got, tc.expected)
			}
		})
	}
}
