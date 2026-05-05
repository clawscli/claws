package ai

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
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

func TestToolExecuteRejectsOutOfScopeProfile(t *testing.T) {
	executor := &ToolExecutor{
		registry: nil,
		aiCtx: &Context{
			Mode:         ContextModeList,
			Service:      "ec2",
			ResourceType: "instances",
			UserRegions:  []string{"us-east-1"},
			UserProfiles: []string{"dev"},
		},
	}

	result := executor.Execute(context.TODO(), &ToolUseContent{
		ID:   "test-123",
		Name: "query_resources",
		Input: map[string]any{
			"service":       "ec2",
			"resource_type": "instances",
			"region":        "us-east-1",
			"profile":       "prod",
		},
	})

	if !result.IsError {
		t.Fatal("expected out-of-scope profile to be rejected")
	}
	if !strings.Contains(result.Content, "profile prod is outside the current AI context") {
		t.Fatalf("unexpected error content: %q", result.Content)
	}
}

func TestToolExecuteRejectsOutOfScopeResource(t *testing.T) {
	executor := &ToolExecutor{
		registry: nil,
		aiCtx: &Context{
			Mode:            ContextModeSingle,
			Service:         "ec2",
			ResourceType:    "instances",
			ResourceID:      "i-allowed",
			ResourceRegion:  "us-east-1",
			ResourceProfile: "dev",
		},
	}

	result := executor.Execute(context.TODO(), &ToolUseContent{
		ID:   "test-123",
		Name: "get_resource_detail",
		Input: map[string]any{
			"service":       "ec2",
			"resource_type": "instances",
			"region":        "us-east-1",
			"profile":       "dev",
			"id":            "i-other",
		},
	})

	if !result.IsError {
		t.Fatal("expected out-of-scope resource to be rejected")
	}
	if !strings.Contains(result.Content, "resource i-other is outside the current AI context") {
		t.Fatalf("unexpected error content: %q", result.Content)
	}
}

func TestToolExecutorDefaultsSingleResourceProfileScope(t *testing.T) {
	executor := &ToolExecutor{
		aiCtx: &Context{
			Mode:            ContextModeSingle,
			Service:         "ec2",
			ResourceType:    "instances",
			ResourceID:      "i-allowed",
			ResourceRegion:  "us-east-1",
			ResourceProfile: "dev",
		},
	}

	profile, cluster, err := executor.validateScope("ec2", "instances", "us-east-1", "", "i-allowed", "")
	if err != nil {
		t.Fatalf("validateScope() returned error: %v", err)
	}
	if profile != "dev" {
		t.Fatalf("profile = %q, want context resource profile", profile)
	}
	if cluster != "" {
		t.Fatalf("cluster = %q, want empty", cluster)
	}
}

func TestToolExecuteRejectsOutOfScopeCluster(t *testing.T) {
	executor := &ToolExecutor{
		registry: nil,
		aiCtx: &Context{
			Mode:            ContextModeSingle,
			Service:         "ecs",
			ResourceType:    "services",
			ResourceID:      "svc-allowed",
			ResourceRegion:  "us-east-1",
			ResourceProfile: "dev",
			Cluster:         "cluster-a",
		},
	}

	result := executor.Execute(context.TODO(), &ToolUseContent{
		ID:   "test-123",
		Name: "get_resource_detail",
		Input: map[string]any{
			"service":       "ecs",
			"resource_type": "services",
			"region":        "us-east-1",
			"profile":       "dev",
			"id":            "svc-allowed",
			"cluster":       "cluster-b",
		},
	})

	if !result.IsError {
		t.Fatal("expected out-of-scope cluster to be rejected")
	}
	if !strings.Contains(result.Content, "cluster cluster-b is outside the current AI context") {
		t.Fatalf("unexpected error content: %q", result.Content)
	}
}

func TestToolExecutorDefaultsSingleResourceClusterScope(t *testing.T) {
	executor := &ToolExecutor{
		aiCtx: &Context{
			Mode:            ContextModeSingle,
			Service:         "ecs",
			ResourceType:    "services",
			ResourceID:      "svc-allowed",
			ResourceRegion:  "us-east-1",
			ResourceProfile: "dev",
			Cluster:         "cluster-a",
		},
	}

	profile, cluster, err := executor.validateScope("ecs", "services", "us-east-1", "", "svc-allowed", "")
	if err != nil {
		t.Fatalf("validateScope() returned error: %v", err)
	}
	if profile != "dev" {
		t.Fatalf("profile = %q, want context resource profile", profile)
	}
	if cluster != "cluster-a" {
		t.Fatalf("cluster = %q, want context cluster", cluster)
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

func TestPrepareDocsSearchQueryAllowsGeneralQueryBeforeAWSData(t *testing.T) {
	executor := &ToolExecutor{
		aiCtx: &Context{
			Mode:            ContextModeSingle,
			Service:         "ec2",
			ResourceType:    "instances",
			ResourceID:      "i-private123",
			ResourceProfile: "production-profile",
		},
	}

	query, err := executor.prepareDocsSearchQuery("EC2 instance metadata options")
	if err != nil {
		t.Fatalf("prepareDocsSearchQuery returned error: %v", err)
	}
	if query != "EC2 instance metadata options" {
		t.Fatalf("query = %q, want unchanged general query", query)
	}
}

func TestToolExecuteRejectsSensitiveDocsSearchQuery(t *testing.T) {
	executor := &ToolExecutor{
		aiCtx: &Context{
			Mode:            ContextModeSingle,
			Service:         "ec2",
			ResourceType:    "instances",
			ResourceID:      "i-private123",
			ResourceProfile: "production-profile",
		},
	}

	result := executor.Execute(context.TODO(), &ToolUseContent{
		ID:   "test-123",
		Name: "search_aws_docs",
		Input: map[string]any{
			"query": "why is i-private123 in production-profile failing with token=plain-secret",
		},
	})

	if !result.IsError {
		t.Fatal("expected sensitive documentation query to be rejected")
	}
	if !strings.Contains(result.Content, "private or sensitive context") {
		t.Fatalf("unexpected rejection message: %q", result.Content)
	}
	for _, leaked := range []string{"i-private123", "production-profile", "plain-secret"} {
		if strings.Contains(result.Content, leaked) {
			t.Fatalf("rejection message leaked %q: %q", leaked, result.Content)
		}
	}
}

func TestPrepareDocsSearchQueryRejectsAWSIdentifiers(t *testing.T) {
	executor := &ToolExecutor{}

	for _, query := range []string{
		"explain arn:aws:lambda:us-east-1:123456789012:function:prod-handler timeout",
		"why does account 123456789012 see access denied",
	} {
		t.Run(query, func(t *testing.T) {
			if _, err := executor.prepareDocsSearchQuery(query); err == nil {
				t.Fatal("expected AWS identifier query to be rejected")
			}
		})
	}
}

func TestToolExecuteAllowsDocsSearchAfterAWSDataTool(t *testing.T) {
	reg := registry.New()
	reg.RegisterCustom("ec2", "instances", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return &mockDAO{
				BaseDAO: dao.NewBaseDAO("ec2", "instances"),
				resources: []dao.Resource{
					&mockResource{id: "i-123", name: "app-server"},
				},
			}, nil
		},
	})
	executor := &ToolExecutor{
		registry: reg,
		docsSearcher: func(ctx context.Context, query string) string {
			return "docs: " + query
		},
	}

	queryResult := executor.Execute(context.TODO(), &ToolUseContent{
		ID:   "query-123",
		Name: "query_resources",
		Input: map[string]any{
			"service":       "ec2",
			"resource_type": "instances",
			"region":        "us-east-1",
		},
	})
	if queryResult.IsError {
		t.Fatalf("expected query_resources to succeed, got %q", queryResult.Content)
	}

	result := executor.Execute(context.TODO(), &ToolUseContent{
		ID:   "test-123",
		Name: "search_aws_docs",
		Input: map[string]any{
			"query": "how to rotate access keys",
		},
	})

	if result.IsError {
		t.Fatalf("expected documentation search to remain allowed after AWS data tools, got %q", result.Content)
	}
	if result.Content != "docs: how to rotate access keys" {
		t.Fatalf("unexpected documentation search result: %q", result.Content)
	}
}

func TestToolExecuteAllowsDocsSearchAfterResourceDetailTool(t *testing.T) {
	reg := registry.New()
	reg.RegisterCustom("ec2", "instances", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return &mockDAO{
				BaseDAO: dao.NewBaseDAO("ec2", "instances"),
				resources: []dao.Resource{
					&mockResource{id: "i-123", name: "app-server"},
				},
			}, nil
		},
	})
	executor := &ToolExecutor{
		registry: reg,
		docsSearcher: func(ctx context.Context, query string) string {
			return "docs: " + query
		},
	}

	detailResult := executor.Execute(context.TODO(), &ToolUseContent{
		ID:   "detail-123",
		Name: "get_resource_detail",
		Input: map[string]any{
			"service":       "ec2",
			"resource_type": "instances",
			"region":        "us-east-1",
			"id":            "i-123",
		},
	})
	if detailResult.IsError {
		t.Fatalf("expected get_resource_detail to succeed, got %q", detailResult.Content)
	}

	result := executor.Execute(context.TODO(), &ToolUseContent{
		ID:   "test-123",
		Name: "search_aws_docs",
		Input: map[string]any{
			"query": "EC2 instance metadata options",
		},
	})
	if result.IsError {
		t.Fatalf("expected documentation search to remain allowed after get_resource_detail, got %q", result.Content)
	}
	if result.Content != "docs: EC2 instance metadata options" {
		t.Fatalf("unexpected documentation search result: %q", result.Content)
	}
}

func TestToolExecuteAllowsDocsSearchAfterFailedAWSDataTool(t *testing.T) {
	reg := registry.New()
	reg.RegisterCustom("ec2", "instances", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return &mockDAO{
				BaseDAO: dao.NewBaseDAO("ec2", "instances"),
				listErr: errors.New("operation failed for arn:aws:ec2:us-east-1:123456789012:instance/i-private token=plain-secret"),
			}, nil
		},
	})
	executor := &ToolExecutor{
		registry: reg,
		docsSearcher: func(ctx context.Context, query string) string {
			return "docs: " + query
		},
	}

	queryResult := executor.Execute(context.TODO(), &ToolUseContent{
		ID:   "query-123",
		Name: "query_resources",
		Input: map[string]any{
			"service":       "ec2",
			"resource_type": "instances",
			"region":        "us-east-1",
		},
	})
	if !queryResult.IsError {
		t.Fatal("expected query_resources to fail")
	}
	for _, leaked := range []string{"arn:aws:ec2", "123456789012", "plain-secret"} {
		if strings.Contains(queryResult.Content, leaked) {
			t.Fatalf("expected failed data tool output to redact %q, got %q", leaked, queryResult.Content)
		}
	}

	result := executor.Execute(context.TODO(), &ToolUseContent{
		ID:   "test-123",
		Name: "search_aws_docs",
		Input: map[string]any{
			"query": "EC2 instance metadata options",
		},
	})
	if result.IsError {
		t.Fatalf("expected documentation search to remain allowed after failed AWS data tool, got %q", result.Content)
	}
	if result.Content != "docs: EC2 instance metadata options" {
		t.Fatalf("unexpected documentation search result: %q", result.Content)
	}
}

func TestToolExecuteRedactsFailedResourceDetailOutput(t *testing.T) {
	reg := registry.New()
	reg.RegisterCustom("ec2", "instances", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return &mockDAO{
				BaseDAO: dao.NewBaseDAO("ec2", "instances"),
				getErr:  errors.New("lookup failed for arn:aws:ec2:us-east-1:123456789012:instance/i-private password=plain-secret"),
			}, nil
		},
	})
	executor := &ToolExecutor{registry: reg}

	result := executor.Execute(context.TODO(), &ToolUseContent{
		ID:   "detail-123",
		Name: "get_resource_detail",
		Input: map[string]any{
			"service":       "ec2",
			"resource_type": "instances",
			"region":        "us-east-1",
			"id":            "i-private",
		},
	})
	if !result.IsError {
		t.Fatal("expected get_resource_detail to fail")
	}
	for _, leaked := range []string{"arn:aws:ec2", "123456789012", "plain-secret"} {
		if strings.Contains(result.Content, leaked) {
			t.Fatalf("expected failed detail output to redact %q, got %q", leaked, result.Content)
		}
	}
}

func TestToolExecuteAllowsDocsSearchAfterMissingDataToolParam(t *testing.T) {
	executor := &ToolExecutor{
		docsSearcher: func(ctx context.Context, query string) string {
			return "docs: " + query
		},
	}

	missingParamResult := executor.Execute(context.TODO(), &ToolUseContent{
		ID:    "query-123",
		Name:  "query_resources",
		Input: map[string]any{"service": "ec2", "resource_type": "instances"},
	})
	if !missingParamResult.IsError {
		t.Fatal("expected missing parameter to fail")
	}

	result := executor.Execute(context.TODO(), &ToolUseContent{
		ID:   "test-123",
		Name: "search_aws_docs",
		Input: map[string]any{
			"query": "EC2 instance metadata options",
		},
	})
	if result.IsError {
		t.Fatalf("expected documentation search to remain allowed after failed AWS data tool attempt, got %q", result.Content)
	}
	if result.Content != "docs: EC2 instance metadata options" {
		t.Fatalf("unexpected documentation search result: %q", result.Content)
	}
}

func TestIsPrivateDataToolIncludesDataTools(t *testing.T) {
	for _, toolName := range []string{"query_resources", "get_resource_detail", "tail_logs"} {
		t.Run(toolName, func(t *testing.T) {
			if !isPrivateDataTool(toolName) {
				t.Fatalf("expected %s to be treated as private data tool", toolName)
			}
		})
	}
	for _, toolName := range []string{"list_resources", "search_aws_docs", "unknown"} {
		t.Run(toolName, func(t *testing.T) {
			if isPrivateDataTool(toolName) {
				t.Fatalf("expected %s not to be treated as private data tool", toolName)
			}
		})
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

func TestFormatResourceDetailRedactsSensitiveRawData(t *testing.T) {
	resource := &mockResource{
		id:   "func-1",
		name: "my-function",
		raw: map[string]any{
			"FunctionName": "my-function",
			"Environment": map[string]any{
				"Variables": map[string]any{
					"API_KEY": "super-secret-value",
				},
			},
			"VpcConfig": map[string]any{
				"VpcId": "vpc-123",
			},
		},
	}

	result := formatResourceDetail(resource)

	if strings.Contains(result, "super-secret-value") {
		t.Fatalf("expected sensitive environment values to be redacted, got %q", result)
	}
	if !strings.Contains(result, "API_KEY") {
		t.Fatalf("expected sensitive key name to be preserved for context, got %q", result)
	}
	if !strings.Contains(result, "[REDACTED]") {
		t.Fatalf("expected redaction marker, got %q", result)
	}
	if !strings.Contains(result, "vpc-123") {
		t.Fatalf("expected non-sensitive raw fields to remain, got %q", result)
	}
}

func TestFormatResourceDetailRedactsSensitiveTags(t *testing.T) {
	resource := &mockResource{
		id:   "resource-1",
		name: "resource",
		tags: map[string]string{
			"Environment": "prod",
			"ApiToken":    "plain-secret-token",
		},
	}

	result := formatResourceDetail(resource)

	if strings.Contains(result, "plain-secret-token") {
		t.Fatalf("expected sensitive tag value to be redacted, got %q", result)
	}
	if !strings.Contains(result, "ApiToken") || !strings.Contains(result, "[REDACTED]") {
		t.Fatalf("expected sensitive tag key with redaction marker, got %q", result)
	}
	if !strings.Contains(result, "Environment: prod") {
		t.Fatalf("expected non-sensitive tag to remain, got %q", result)
	}
}

func TestFormatResourceDetailRedactsSensitiveTagValuePatterns(t *testing.T) {
	resource := &mockResource{
		id:   "resource-1",
		name: "resource",
		tags: map[string]string{
			"Environment": "prod",
			"Endpoint":    "postgres://app:super-secret-password@db.example.com:5432/app",
		},
	}

	result := formatResourceDetail(resource)

	if strings.Contains(result, "super-secret-password") {
		t.Fatalf("expected sensitive tag value pattern to be redacted, got %q", result)
	}
	if !strings.Contains(result, "Environment: prod") {
		t.Fatalf("expected non-sensitive tag to remain, got %q", result)
	}
}

func TestFormatResourceDetailRedactsSensitiveLabelValueRecords(t *testing.T) {
	resource := &mockResource{
		id:   "stack-1",
		name: "my-stack",
		raw: map[string]any{
			"Outputs": []map[string]any{
				{
					"OutputKey":   "DB_PASSWORD",
					"OutputValue": "plain-secret-output",
				},
				{
					"OutputKey":   "Endpoint",
					"OutputValue": "https://example.com",
				},
			},
			"Parameters": []map[string]any{
				{
					"ParameterKey":   "ApiToken",
					"ParameterValue": "plain-secret-parameter",
					"ResolvedValue":  "plain-secret-resolved",
				},
			},
		},
	}

	result := formatResourceDetail(resource)

	for _, secret := range []string{"DB_PASSWORD", "plain-secret-output", "ApiToken", "plain-secret-parameter", "plain-secret-resolved"} {
		if strings.Contains(result, secret) {
			t.Fatalf("expected sensitive label/value record data to be redacted, found %q in %q", secret, result)
		}
	}
	if !strings.Contains(result, "https://example.com") {
		t.Fatalf("expected non-sensitive output value to remain, got %q", result)
	}
}

func TestIsSensitiveRawKeyAvoidsSubstringFalsePositives(t *testing.T) {
	for _, key := range []string{"environment", "variables", "tokenization", "tokencount", "accessTokens_issued", "secretsmanager_arn", "credentialsexpiry"} {
		if isSensitiveRawKey(key) {
			t.Fatalf("isSensitiveRawKey(%q) = true, want false", key)
		}
	}

	for _, key := range []string{"DB_PASSWORD", "DBPassword", "DBMasterPassword", "MasterUserPassword", "ApiToken", "clientSecret", "SecretAccessKey", "EnvironmentVariables"} {
		if !isSensitiveRawKey(key) {
			t.Fatalf("isSensitiveRawKey(%q) = false, want true", key)
		}
	}
}

func TestFormatResourceDetailKeepsNonSensitiveEnvironmentFields(t *testing.T) {
	resource := &mockResource{
		id:   "instance-1",
		name: "my-instance",
		raw: map[string]any{
			"environment": "production",
			"variables":   "some-list",
			"DBPassword":  "secret-db-password",
		},
	}

	result := formatResourceDetail(resource)

	if !strings.Contains(result, "production") || !strings.Contains(result, "some-list") {
		t.Fatalf("expected non-sensitive environment fields to remain, got %q", result)
	}
	if strings.Contains(result, "secret-db-password") {
		t.Fatalf("expected DBPassword value to be redacted, got %q", result)
	}
	if !strings.Contains(result, "DBPassword") {
		t.Fatalf("expected sensitive key name to be preserved, got %q", result)
	}
}

func TestFormatResourceDetailPreservesMultipleSensitiveKeyNames(t *testing.T) {
	resource := &mockResource{
		id:   "resource-1",
		name: "resource",
		raw: map[string]any{
			"DBPassword":   "db-secret",
			"ApiToken":     "api-secret",
			"clientSecret": "client-secret",
		},
	}

	result := formatResourceDetail(resource)

	for _, key := range []string{"DBPassword", "ApiToken", "clientSecret"} {
		if !strings.Contains(result, key) {
			t.Fatalf("expected sensitive key %q to be preserved, got %q", key, result)
		}
	}
	for _, secret := range []string{"db-secret", "api-secret", "client-secret"} {
		if strings.Contains(result, secret) {
			t.Fatalf("expected sensitive value %q to be redacted, got %q", secret, result)
		}
	}
}

func TestFormatResourceDetailRedactsSensitiveValuePatterns(t *testing.T) {
	resource := &mockResource{
		id:   "resource-1",
		name: "resource",
		raw: map[string]any{
			"DatabaseURL": "postgres://app:super-secret-password@db.example.com:5432/app",
			"Header":      "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.payload.signature",
			"Certificate": "-----BEGIN PRIVATE KEY-----\nplain-private-key\n-----END PRIVATE KEY-----",
			"PublicURL":   "https://example.com/health",
		},
	}

	result := formatResourceDetail(resource)

	for _, secret := range []string{"super-secret-password", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.payload.signature", "plain-private-key"} {
		if strings.Contains(result, secret) {
			t.Fatalf("expected value-only secret %q to be redacted, got %q", secret, result)
		}
	}
	if !strings.Contains(result, "https://example.com/health") {
		t.Fatalf("expected non-sensitive URL to remain, got %q", result)
	}
	if !strings.Contains(result, "[REDACTED]") {
		t.Fatalf("expected redaction marker, got %q", result)
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

type mockDAO struct {
	dao.BaseDAO
	resources []dao.Resource
	listErr   error
	getErr    error
}

func (d *mockDAO) List(ctx context.Context) ([]dao.Resource, error) {
	if d.listErr != nil {
		return nil, d.listErr
	}
	return d.resources, nil
}

func (d *mockDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	if d.getErr != nil {
		return nil, d.getErr
	}
	for _, resource := range d.resources {
		if resource.GetID() == id {
			return resource, nil
		}
	}
	return nil, nil
}

func (d *mockDAO) Delete(ctx context.Context, id string) error {
	return nil
}
