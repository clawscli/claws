package stacks

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

func TestNewStackResource(t *testing.T) {
	stack := types.Stack{
		StackId:                     aws.String("arn:aws:cloudformation:us-east-1:123456789012:stack/my-stack/abc123"),
		StackName:                   aws.String("my-stack"),
		StackStatus:                 types.StackStatusCreateComplete,
		Description:                 aws.String("Test stack"),
		EnableTerminationProtection: aws.Bool(true),
		DriftInformation: &types.StackDriftInformation{
			StackDriftStatus: types.StackDriftStatusInSync,
		},
		Tags: []types.Tag{
			{Key: aws.String("Environment"), Value: aws.String("prod")},
		},
	}

	resource := NewStackResource(stack)

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"GetID", resource.GetID(), "arn:aws:cloudformation:us-east-1:123456789012:stack/my-stack/abc123"},
		{"GetName", resource.GetName(), "my-stack"},
		{"Status", resource.Status(), "CREATE_COMPLETE"},
		{"Description", resource.Description(), "Test stack"},
		{"TerminationProtection", resource.TerminationProtection(), true},
		{"DriftStatus", resource.DriftStatus(), "IN_SYNC"},
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

func TestStackResource_MinimalStack(t *testing.T) {
	stack := types.Stack{
		StackName:   aws.String("minimal-stack"),
		StackStatus: types.StackStatusCreateInProgress,
	}

	resource := NewStackResource(stack)

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"GetID", resource.GetID(), ""},
		{"GetName", resource.GetName(), "minimal-stack"},
		{"Status", resource.Status(), "CREATE_IN_PROGRESS"},
		{"Description", resource.Description(), ""},
		{"TerminationProtection", resource.TerminationProtection(), false},
		{"DriftStatus", resource.DriftStatus(), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestStackResource_StatusVariations(t *testing.T) {
	statuses := []struct {
		status   types.StackStatus
		expected string
	}{
		{types.StackStatusCreateComplete, "CREATE_COMPLETE"},
		{types.StackStatusCreateFailed, "CREATE_FAILED"},
		{types.StackStatusCreateInProgress, "CREATE_IN_PROGRESS"},
		{types.StackStatusDeleteComplete, "DELETE_COMPLETE"},
		{types.StackStatusDeleteFailed, "DELETE_FAILED"},
		{types.StackStatusDeleteInProgress, "DELETE_IN_PROGRESS"},
		{types.StackStatusUpdateComplete, "UPDATE_COMPLETE"},
		{types.StackStatusUpdateFailed, "UPDATE_FAILED"},
		{types.StackStatusUpdateInProgress, "UPDATE_IN_PROGRESS"},
		{types.StackStatusRollbackComplete, "ROLLBACK_COMPLETE"},
		{types.StackStatusRollbackFailed, "ROLLBACK_FAILED"},
	}

	for _, tc := range statuses {
		t.Run(string(tc.status), func(t *testing.T) {
			stack := types.Stack{
				StackName:   aws.String("test"),
				StackStatus: tc.status,
			}
			resource := NewStackResource(stack)
			if got := resource.Status(); got != tc.expected {
				t.Errorf("Status() = %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestStackResource_DriftStatusVariations(t *testing.T) {
	driftStatuses := []struct {
		name     string
		info     *types.StackDriftInformation
		expected string
	}{
		{"nil info", nil, ""},
		{"IN_SYNC", &types.StackDriftInformation{StackDriftStatus: types.StackDriftStatusInSync}, "IN_SYNC"},
		{"DRIFTED", &types.StackDriftInformation{StackDriftStatus: types.StackDriftStatusDrifted}, "DRIFTED"},
		{"NOT_CHECKED", &types.StackDriftInformation{StackDriftStatus: types.StackDriftStatusNotChecked}, "NOT_CHECKED"},
	}

	for _, tc := range driftStatuses {
		t.Run(tc.name, func(t *testing.T) {
			stack := types.Stack{
				StackName:        aws.String("test"),
				StackStatus:      types.StackStatusCreateComplete,
				DriftInformation: tc.info,
			}
			resource := NewStackResource(stack)
			if got := resource.DriftStatus(); got != tc.expected {
				t.Errorf("DriftStatus() = %q, want %q", got, tc.expected)
			}
		})
	}
}
