package clusters

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

func TestNewClusterResource(t *testing.T) {
	cluster := types.Cluster{
		ClusterName:                       aws.String("my-cluster"),
		ClusterArn:                        aws.String("arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster"),
		Status:                            aws.String("ACTIVE"),
		RunningTasksCount:                 10,
		PendingTasksCount:                 2,
		ActiveServicesCount:               5,
		RegisteredContainerInstancesCount: 3,
		CapacityProviders:                 []string{"FARGATE", "FARGATE_SPOT"},
		Settings: []types.ClusterSetting{
			{Name: types.ClusterSettingNameContainerInsights, Value: aws.String("enabled")},
		},
	}

	resource := NewClusterResource(cluster)

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"GetID", resource.GetID(), "my-cluster"},
		{"GetName", resource.GetName(), "my-cluster"},
		{"GetARN", resource.GetARN(), "arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster"},
		{"Status", resource.Status(), "ACTIVE"},
		{"RunningTasksCount", resource.RunningTasksCount(), int32(10)},
		{"PendingTasksCount", resource.PendingTasksCount(), int32(2)},
		{"ActiveServicesCount", resource.ActiveServicesCount(), int32(5)},
		{"RegisteredContainerInstancesCount", resource.RegisteredContainerInstancesCount(), int32(3)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}

	// Test CapacityProviders
	providers := resource.CapacityProviders()
	if len(providers) != 2 {
		t.Errorf("CapacityProviders length = %d, want 2", len(providers))
	}
	if providers[0] != "FARGATE" {
		t.Errorf("CapacityProviders[0] = %q, want %q", providers[0], "FARGATE")
	}

	// Test Settings
	settings := resource.Settings()
	if len(settings) != 1 {
		t.Errorf("Settings length = %d, want 1", len(settings))
	}
}

func TestClusterResource_MinimalCluster(t *testing.T) {
	cluster := types.Cluster{
		ClusterName: aws.String("minimal-cluster"),
	}

	resource := NewClusterResource(cluster)

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"GetID", resource.GetID(), "minimal-cluster"},
		{"Status", resource.Status(), ""},
		{"RunningTasksCount", resource.RunningTasksCount(), int32(0)},
		{"PendingTasksCount", resource.PendingTasksCount(), int32(0)},
		{"ActiveServicesCount", resource.ActiveServicesCount(), int32(0)},
		{"RegisteredContainerInstancesCount", resource.RegisteredContainerInstancesCount(), int32(0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}

	// Test empty slices
	if len(resource.CapacityProviders()) != 0 {
		t.Errorf("CapacityProviders should be empty")
	}
	if len(resource.Settings()) != 0 {
		t.Errorf("Settings should be empty")
	}
}
