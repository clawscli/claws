package instances

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
)

func TestNewInstanceResource(t *testing.T) {
	instance := types.DBInstance{
		DBInstanceIdentifier: aws.String("my-database"),
		DBInstanceClass:      aws.String("db.t3.micro"),
		Engine:               aws.String("postgres"),
		EngineVersion:        aws.String("15.4"),
		DBInstanceStatus:     aws.String("available"),
		AvailabilityZone:     aws.String("us-east-1a"),
		MultiAZ:              aws.Bool(true),
		StorageType:          aws.String("gp3"),
		AllocatedStorage:     aws.Int32(100),
		Endpoint: &types.Endpoint{
			Address: aws.String("my-database.123456789012.us-east-1.rds.amazonaws.com"),
			Port:    aws.Int32(5432),
		},
		TagList: []types.Tag{
			{Key: aws.String("Environment"), Value: aws.String("prod")},
		},
	}

	resource := NewInstanceResource(instance)

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"GetID", resource.GetID(), "my-database"},
		{"GetName", resource.GetName(), "my-database"},
		{"State", resource.State(), "available"},
		{"Engine", resource.Engine(), "postgres"},
		{"EngineVersion", resource.EngineVersion(), "15.4"},
		{"InstanceClass", resource.InstanceClass(), "db.t3.micro"},
		{"AZ", resource.AZ(), "us-east-1a"},
		{"MultiAZ", resource.MultiAZ(), true},
		{"StorageType", resource.StorageType(), "gp3"},
		{"AllocatedStorage", resource.AllocatedStorage(), int32(100)},
		{"Endpoint", resource.Endpoint(), "my-database.123456789012.us-east-1.rds.amazonaws.com"},
		{"Port", resource.Port(), int32(5432)},
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

func TestInstanceResource_MinimalInstance(t *testing.T) {
	instance := types.DBInstance{
		DBInstanceIdentifier: aws.String("minimal-db"),
	}

	resource := NewInstanceResource(instance)

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"GetID", resource.GetID(), "minimal-db"},
		{"State", resource.State(), "unknown"},
		{"Engine", resource.Engine(), ""},
		{"EngineVersion", resource.EngineVersion(), ""},
		{"InstanceClass", resource.InstanceClass(), ""},
		{"AZ", resource.AZ(), ""},
		{"MultiAZ", resource.MultiAZ(), false},
		{"StorageType", resource.StorageType(), ""},
		{"AllocatedStorage", resource.AllocatedStorage(), int32(0)},
		{"Endpoint", resource.Endpoint(), ""},
		{"Port", resource.Port(), int32(0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestInstanceResource_StateVariations(t *testing.T) {
	states := []struct {
		status   *string
		expected string
	}{
		{aws.String("available"), "available"},
		{aws.String("creating"), "creating"},
		{aws.String("deleting"), "deleting"},
		{aws.String("modifying"), "modifying"},
		{aws.String("starting"), "starting"},
		{aws.String("stopping"), "stopping"},
		{aws.String("stopped"), "stopped"},
		{nil, "unknown"},
	}

	for _, tc := range states {
		name := "nil"
		if tc.status != nil {
			name = *tc.status
		}
		t.Run(name, func(t *testing.T) {
			instance := types.DBInstance{
				DBInstanceIdentifier: aws.String("test"),
				DBInstanceStatus:     tc.status,
			}
			resource := NewInstanceResource(instance)
			if got := resource.State(); got != tc.expected {
				t.Errorf("State() = %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestInstanceResource_EngineVariations(t *testing.T) {
	engines := []struct {
		engine  string
		version string
	}{
		{"postgres", "15.4"},
		{"mysql", "8.0.35"},
		{"aurora-postgresql", "15.4"},
		{"aurora-mysql", "3.04.0"},
		{"mariadb", "10.11.4"},
		{"oracle-ee", "19.0.0.0.ru-2024-01.rur-2024-01.r1"},
		{"sqlserver-se", "15.00.4345.5.v1"},
	}

	for _, tc := range engines {
		t.Run(tc.engine, func(t *testing.T) {
			instance := types.DBInstance{
				DBInstanceIdentifier: aws.String("test"),
				Engine:               aws.String(tc.engine),
				EngineVersion:        aws.String(tc.version),
			}
			resource := NewInstanceResource(instance)
			if got := resource.Engine(); got != tc.engine {
				t.Errorf("Engine() = %q, want %q", got, tc.engine)
			}
			if got := resource.EngineVersion(); got != tc.version {
				t.Errorf("EngineVersion() = %q, want %q", got, tc.version)
			}
		})
	}
}

func TestRdsTagsToMap(t *testing.T) {
	tests := []struct {
		name     string
		tags     []types.Tag
		expected map[string]string
	}{
		{
			name:     "empty tags",
			tags:     []types.Tag{},
			expected: map[string]string{},
		},
		{
			name: "single tag",
			tags: []types.Tag{
				{Key: aws.String("Name"), Value: aws.String("test")},
			},
			expected: map[string]string{"Name": "test"},
		},
		{
			name: "multiple tags",
			tags: []types.Tag{
				{Key: aws.String("Name"), Value: aws.String("test")},
				{Key: aws.String("Environment"), Value: aws.String("prod")},
			},
			expected: map[string]string{"Name": "test", "Environment": "prod"},
		},
		{
			name: "nil key",
			tags: []types.Tag{
				{Key: nil, Value: aws.String("test")},
			},
			expected: map[string]string{},
		},
		{
			name: "nil value",
			tags: []types.Tag{
				{Key: aws.String("Name"), Value: nil},
			},
			expected: map[string]string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := rdsTagsToMap(tc.tags)
			if len(result) != len(tc.expected) {
				t.Errorf("len(result) = %d, want %d", len(result), len(tc.expected))
			}
			for k, v := range tc.expected {
				if result[k] != v {
					t.Errorf("result[%q] = %q, want %q", k, result[k], v)
				}
			}
		})
	}
}
