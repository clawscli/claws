package ai

import (
	"context"
	"strings"
	"testing"
)

func TestToolExecutorTools(t *testing.T) {
	executor := &ToolExecutor{}
	tools := executor.Tools()

	expectedTools := []string{
		"list_resources",
		"query_resources",
		"get_resource_detail",
		"tail_logs",
		"search_aws_docs",
	}

	if len(tools) != len(expectedTools) {
		t.Errorf("expected %d tools, got %d", len(expectedTools), len(tools))
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	for _, name := range expectedTools {
		if !toolNames[name] {
			t.Errorf("missing tool: %s", name)
		}
	}
}

func TestToolSchemas(t *testing.T) {
	executor := &ToolExecutor{}
	tools := executor.Tools()

	for _, tool := range tools {
		t.Run(tool.Name, func(t *testing.T) {
			if tool.Name == "" {
				t.Error("tool name is empty")
			}
			if tool.Description == "" {
				t.Error("tool description is empty")
			}
			if tool.InputSchema == nil {
				t.Error("tool input schema is nil")
			}

			schemaType, ok := tool.InputSchema["type"].(string)
			if !ok || schemaType != "object" {
				t.Errorf("expected schema type 'object', got %v", tool.InputSchema["type"])
			}

			props, ok := tool.InputSchema["properties"].(map[string]any)
			if !ok {
				t.Error("schema properties is not a map")
			}

			if len(props) == 0 {
				t.Error("schema has no properties")
			}
		})
	}
}

func TestQueryResourcesRequiredParams(t *testing.T) {
	executor := &ToolExecutor{}
	tools := executor.Tools()

	var queryTool *Tool
	for i := range tools {
		if tools[i].Name == "query_resources" {
			queryTool = &tools[i]
			break
		}
	}

	if queryTool == nil {
		t.Fatal("query_resources tool not found")
	}

	required, ok := queryTool.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("required field is not []string")
	}

	expectedRequired := map[string]bool{
		"service":       true,
		"resource_type": true,
		"region":        true,
	}

	for _, r := range required {
		if !expectedRequired[r] {
			t.Errorf("unexpected required field: %s", r)
		}
		delete(expectedRequired, r)
	}

	for missing := range expectedRequired {
		t.Errorf("missing required field: %s", missing)
	}
}

func TestToolExecuteUnknownTool(t *testing.T) {
	executor := &ToolExecutor{registry: nil}

	result := executor.Execute(context.TODO(), &ToolUseContent{
		ID:    "test-123",
		Name:  "unknown_tool",
		Input: map[string]any{},
	})

	if result.ToolUseID != "test-123" {
		t.Errorf("expected tool use ID %q, got %q", "test-123", result.ToolUseID)
	}
	if !result.IsError {
		t.Error("expected IsError to be true")
	}
	if !strings.Contains(result.Content, "Unknown tool") {
		t.Errorf("expected error message about unknown tool, got %q", result.Content)
	}
}

func TestToolExecuteQueryResourcesMissingParams(t *testing.T) {
	executor := &ToolExecutor{registry: nil}

	tests := []struct {
		name          string
		input         map[string]any
		expectedError string
	}{
		{
			name:          "missing service",
			input:         map[string]any{"resource_type": "instances", "region": "us-east-1"},
			expectedError: "service parameter is required",
		},
		{
			name:          "missing resource_type",
			input:         map[string]any{"service": "ec2", "region": "us-east-1"},
			expectedError: "resource_type parameter is required",
		},
		{
			name:          "missing region",
			input:         map[string]any{"service": "ec2", "resource_type": "instances"},
			expectedError: "region parameter is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.Execute(context.TODO(), &ToolUseContent{
				ID:    "test-123",
				Name:  "query_resources",
				Input: tt.input,
			})

			if !result.IsError {
				t.Error("expected IsError to be true")
			}
			if !strings.Contains(result.Content, tt.expectedError) {
				t.Errorf("expected error %q, got %q", tt.expectedError, result.Content)
			}
		})
	}
}

func TestToolExecuteGetResourceDetailMissingRegion(t *testing.T) {
	executor := &ToolExecutor{registry: nil}

	result := executor.Execute(context.TODO(), &ToolUseContent{
		ID:   "test-123",
		Name: "get_resource_detail",
		Input: map[string]any{
			"service":       "ec2",
			"resource_type": "instances",
			"id":            "i-12345",
		},
	})

	if !result.IsError {
		t.Error("expected IsError to be true")
	}
	if !strings.Contains(result.Content, "region parameter is required") {
		t.Errorf("expected region error, got %q", result.Content)
	}
}

func TestToolExecuteTailLogsMissingRegion(t *testing.T) {
	executor := &ToolExecutor{registry: nil}

	result := executor.Execute(context.TODO(), &ToolUseContent{
		ID:   "test-123",
		Name: "tail_logs",
		Input: map[string]any{
			"service":       "lambda",
			"resource_type": "functions",
			"id":            "my-function",
		},
	})

	if !result.IsError {
		t.Error("expected IsError to be true")
	}
	if !strings.Contains(result.Content, "region parameter is required") {
		t.Errorf("expected region error, got %q", result.Content)
	}
}

func TestToolExecuteSearchDocsEmptyQuery(t *testing.T) {
	executor := &ToolExecutor{registry: nil}

	result := executor.Execute(context.TODO(), &ToolUseContent{
		ID:    "test-123",
		Name:  "search_aws_docs",
		Input: map[string]any{},
	})

	if !strings.Contains(result.Content, "query parameter is required") {
		t.Errorf("expected query error, got %q", result.Content)
	}
}

func TestExtractLogGroupNameFromArn(t *testing.T) {
	tests := []struct {
		arn      string
		expected string
	}{
		{
			arn:      "arn:aws:logs:us-east-1:123456789012:log-group:/aws/lambda/my-function",
			expected: "/aws/lambda/my-function",
		},
		{
			arn:      "arn:aws:logs:us-west-2:123456789012:log-group:/ecs/my-service",
			expected: "/ecs/my-service",
		},
		{
			arn:      "/aws/lambda/simple",
			expected: "/aws/lambda/simple",
		},
	}

	for _, tt := range tests {
		t.Run(tt.arn, func(t *testing.T) {
			result := extractLogGroupNameFromArn(tt.arn)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFormatResourceSummary(t *testing.T) {
	resource := &mockResource{
		id:   "i-12345",
		name: "my-instance",
	}

	result := formatResourceSummary(resource)

	if !strings.Contains(result, "i-12345") {
		t.Errorf("expected ID in summary, got %q", result)
	}
	if !strings.Contains(result, "my-instance") {
		t.Errorf("expected name in summary, got %q", result)
	}
}

func TestFormatResourceSummarySameIDAndName(t *testing.T) {
	resource := &mockResource{
		id:   "my-bucket",
		name: "my-bucket",
	}

	result := formatResourceSummary(resource)

	if strings.Count(result, "my-bucket") != 1 {
		t.Errorf("expected ID only once when same as name, got %q", result)
	}
}

func TestFormatResourceDetail(t *testing.T) {
	resource := &mockResource{
		id:   "i-12345",
		name: "my-instance",
		arn:  "arn:aws:ec2:us-east-1:123456789012:instance/i-12345",
		tags: map[string]string{"Environment": "prod", "Team": "platform"},
		raw:  map[string]string{"InstanceType": "t3.micro"},
	}

	result := formatResourceDetail(resource)

	if !strings.Contains(result, "i-12345") {
		t.Errorf("expected ID in detail, got %q", result)
	}
	if !strings.Contains(result, "my-instance") {
		t.Errorf("expected name in detail, got %q", result)
	}
	if !strings.Contains(result, "arn:aws:ec2") {
		t.Errorf("expected ARN in detail, got %q", result)
	}
	if !strings.Contains(result, "Environment") {
		t.Errorf("expected tags in detail, got %q", result)
	}
	if !strings.Contains(result, "InstanceType") {
		t.Errorf("expected raw data in detail, got %q", result)
	}
}

type mockResource struct {
	id   string
	name string
	arn  string
	tags map[string]string
	raw  any
}

func (m *mockResource) GetID() string              { return m.id }
func (m *mockResource) GetName() string            { return m.name }
func (m *mockResource) GetARN() string             { return m.arn }
func (m *mockResource) GetTags() map[string]string { return m.tags }
func (m *mockResource) Raw() any                   { return m.raw }
