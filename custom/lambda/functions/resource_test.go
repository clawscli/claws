package functions

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

func TestNewFunctionResource(t *testing.T) {
	fn := types.FunctionConfiguration{
		FunctionName: aws.String("my-function"),
		FunctionArn:  aws.String("arn:aws:lambda:us-east-1:123456789012:function:my-function"),
		Runtime:      types.RuntimePython312,
		Handler:      aws.String("index.handler"),
		CodeSize:     1024,
		MemorySize:   aws.Int32(128),
		Timeout:      aws.Int32(30),
		Description:  aws.String("Test function"),
		State:        types.StateActive,
		LastModified: aws.String("2024-01-15T10:30:00.000+0000"),
		Role:         aws.String("arn:aws:iam::123456789012:role/lambda-role"),
		PackageType:  types.PackageTypeZip,
		Version:      aws.String("$LATEST"),
		CodeSha256:   aws.String("abc123sha256"),
	}

	resource := NewFunctionResource(fn)

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"GetID", resource.GetID(), "my-function"},
		{"GetName", resource.GetName(), "my-function"},
		{"GetARN", resource.GetARN(), "arn:aws:lambda:us-east-1:123456789012:function:my-function"},
		{"Runtime", resource.Runtime(), "python3.12"},
		{"Handler", resource.Handler(), "index.handler"},
		{"CodeSize", resource.CodeSize(), int64(1024)},
		{"MemorySize", resource.MemorySize(), int32(128)},
		{"Timeout", resource.Timeout(), int32(30)},
		{"Description", resource.Description(), "Test function"},
		{"State", resource.State(), "Active"},
		{"Role", resource.Role(), "arn:aws:iam::123456789012:role/lambda-role"},
		{"PackageType", resource.PackageType(), "Zip"},
		{"Version", resource.Version(), "$LATEST"},
		{"CodeSha256", resource.CodeSha256(), "abc123sha256"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestFunctionResource_MinimalFunction(t *testing.T) {
	fn := types.FunctionConfiguration{
		FunctionName: aws.String("minimal-function"),
	}

	resource := NewFunctionResource(fn)

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"GetID", resource.GetID(), "minimal-function"},
		{"Runtime", resource.Runtime(), ""},
		{"Handler", resource.Handler(), ""},
		{"CodeSize", resource.CodeSize(), int64(0)},
		{"MemorySize", resource.MemorySize(), int32(0)},
		{"Timeout", resource.Timeout(), int32(0)},
		{"Description", resource.Description(), ""},
		{"State", resource.State(), ""},
		{"Role", resource.Role(), ""},
		{"EphemeralStorageSize", resource.EphemeralStorageSize(), int32(512)}, // default
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestFunctionResource_RuntimeVariations(t *testing.T) {
	runtimes := []struct {
		runtime  types.Runtime
		expected string
	}{
		{types.RuntimePython312, "python3.12"},
		{types.RuntimeNodejs20x, "nodejs20.x"},
		{types.RuntimeGo1x, "go1.x"},
		{types.RuntimeJava21, "java21"},
	}

	for _, tc := range runtimes {
		t.Run(string(tc.runtime), func(t *testing.T) {
			fn := types.FunctionConfiguration{
				FunctionName: aws.String("test"),
				Runtime:      tc.runtime,
			}
			resource := NewFunctionResource(fn)
			if got := resource.Runtime(); got != tc.expected {
				t.Errorf("Runtime() = %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestFunctionResource_TracingConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   *types.TracingConfigResponse
		expected string
	}{
		{"nil config", nil, ""},
		{"Active", &types.TracingConfigResponse{Mode: types.TracingModeActive}, "Active"},
		{"PassThrough", &types.TracingConfigResponse{Mode: types.TracingModePassThrough}, "PassThrough"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fn := types.FunctionConfiguration{
				FunctionName:  aws.String("test"),
				TracingConfig: tc.config,
			}
			resource := NewFunctionResource(fn)
			if got := resource.TracingConfig(); got != tc.expected {
				t.Errorf("TracingConfig() = %q, want %q", got, tc.expected)
			}
		})
	}
}
