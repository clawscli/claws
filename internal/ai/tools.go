package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
)

type ToolExecutor struct {
	registry *registry.Registry
	cwClient *cloudwatchlogs.Client
}

func NewToolExecutor(ctx context.Context, reg *registry.Registry) (*ToolExecutor, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	return &ToolExecutor{
		registry: reg,
		cwClient: cloudwatchlogs.NewFromConfig(cfg),
	}, nil
}

func (e *ToolExecutor) Tools() []Tool {
	return []Tool{
		{
			Name:        "list_services",
			Description: "List all available AWS services",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "list_resources",
			Description: "List resource types available for a specific AWS service",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"service": map[string]any{
						"type":        "string",
						"description": "AWS service name (e.g., ec2, lambda, s3)",
					},
				},
				"required": []string{"service"},
			},
		},
		{
			Name:        "query_resources",
			Description: "List AWS resources of a specific type",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"service": map[string]any{
						"type":        "string",
						"description": "AWS service name (e.g., ec2, lambda)",
					},
					"resource_type": map[string]any{
						"type":        "string",
						"description": "Resource type (e.g., instances, functions)",
					},
				},
				"required": []string{"service", "resource_type"},
			},
		},
		{
			Name:        "get_resource_detail",
			Description: "Get detailed information about a specific AWS resource",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"service": map[string]any{
						"type":        "string",
						"description": "AWS service name",
					},
					"resource_type": map[string]any{
						"type":        "string",
						"description": "Resource type",
					},
					"id": map[string]any{
						"type":        "string",
						"description": "Resource ID",
					},
				},
				"required": []string{"service", "resource_type", "id"},
			},
		},
		{
			Name:        "tail_logs",
			Description: "Fetch recent CloudWatch logs from a log group",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"log_group": map[string]any{
						"type":        "string",
						"description": "CloudWatch log group name (e.g., /aws/lambda/my-function)",
					},
					"filter": map[string]any{
						"type":        "string",
						"description": "Optional filter pattern for log messages",
					},
					"since": map[string]any{
						"type":        "string",
						"description": "Time range (e.g., 5m, 1h, 24h). Default: 15m",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum number of log events. Default: 100",
					},
				},
				"required": []string{"log_group"},
			},
		},
		{
			Name:        "search_aws_docs",
			Description: "Search AWS documentation for information",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Search query for AWS documentation",
					},
				},
				"required": []string{"query"},
			},
		},
	}
}

func (e *ToolExecutor) Execute(ctx context.Context, call ToolCall) ToolResult {
	var content string
	var isError bool

	switch call.Name {
	case "list_services":
		content = e.listServices()
	case "list_resources":
		service, _ := call.Input["service"].(string)
		content = e.listResources(service)
	case "query_resources":
		service, _ := call.Input["service"].(string)
		resourceType, _ := call.Input["resource_type"].(string)
		content, isError = e.queryResources(ctx, service, resourceType)
	case "get_resource_detail":
		service, _ := call.Input["service"].(string)
		resourceType, _ := call.Input["resource_type"].(string)
		id, _ := call.Input["id"].(string)
		content, isError = e.getResourceDetail(ctx, service, resourceType, id)
	case "tail_logs":
		logGroup, _ := call.Input["log_group"].(string)
		filter, _ := call.Input["filter"].(string)
		since, _ := call.Input["since"].(string)
		limit, _ := call.Input["limit"].(float64)
		content, isError = e.tailLogs(ctx, logGroup, filter, since, int(limit))
	case "search_aws_docs":
		query, _ := call.Input["query"].(string)
		content = e.searchDocs(query)
	default:
		content = fmt.Sprintf("Unknown tool: %s", call.Name)
		isError = true
	}

	return ToolResult{
		ID:      call.ID,
		Content: content,
		IsError: isError,
	}
}

func (e *ToolExecutor) listServices() string {
	categories := e.registry.ListServicesByCategory()
	var result string

	for _, cat := range categories {
		result += fmt.Sprintf("\n## %s\n", cat.Name)
		for _, svc := range cat.Services {
			displayName := e.registry.GetDisplayName(svc)
			resources := e.registry.ListResources(svc)
			result += fmt.Sprintf("- %s (%s): %d resource types\n", displayName, svc, len(resources))
		}
	}

	return result
}

func (e *ToolExecutor) listResources(service string) string {
	resources := e.registry.ListResources(service)
	if len(resources) == 0 {
		return fmt.Sprintf("No resources found for service: %s", service)
	}

	displayName := e.registry.GetDisplayName(service)
	result := fmt.Sprintf("Resource types for %s (%s):\n", displayName, service)
	for _, r := range resources {
		result += fmt.Sprintf("- %s\n", r)
	}
	return result
}

func (e *ToolExecutor) queryResources(ctx context.Context, service, resourceType string) (string, bool) {
	d, err := e.registry.GetDAO(ctx, service, resourceType)
	if err != nil {
		return fmt.Sprintf("Error getting DAO: %v", err), true
	}

	resources, err := d.List(ctx)
	if err != nil {
		return fmt.Sprintf("Error listing resources: %v", err), true
	}

	if len(resources) == 0 {
		return fmt.Sprintf("No %s/%s resources found", service, resourceType), false
	}

	result := fmt.Sprintf("Found %d %s/%s resources:\n\n", len(resources), service, resourceType)
	for i, r := range resources {
		if i >= 50 {
			result += fmt.Sprintf("\n... and %d more\n", len(resources)-50)
			break
		}
		result += formatResourceSummary(r)
	}

	return result, false
}

func (e *ToolExecutor) getResourceDetail(ctx context.Context, service, resourceType, id string) (string, bool) {
	d, err := e.registry.GetDAO(ctx, service, resourceType)
	if err != nil {
		return fmt.Sprintf("Error getting DAO: %v", err), true
	}

	resource, err := d.Get(ctx, id)
	if err != nil {
		return fmt.Sprintf("Error getting resource: %v", err), true
	}

	return formatResourceDetail(resource), false
}

func (e *ToolExecutor) tailLogs(ctx context.Context, logGroup, filter, since string, limit int) (string, bool) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	startTime := time.Now().Add(-15 * time.Minute)
	if since != "" {
		if d, err := time.ParseDuration(since); err == nil {
			startTime = time.Now().Add(-d)
		}
	}

	input := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName: aws.String(logGroup),
		StartTime:    aws.Int64(startTime.UnixMilli()),
		Limit:        aws.Int32(int32(limit)),
	}

	if filter != "" {
		input.FilterPattern = aws.String(filter)
	}

	output, err := e.cwClient.FilterLogEvents(ctx, input)
	if err != nil {
		return fmt.Sprintf("Error fetching logs: %v", err), true
	}

	if len(output.Events) == 0 {
		return fmt.Sprintf("No logs found in %s (since %s)", logGroup, since), false
	}

	result := fmt.Sprintf("Logs from %s (%d events):\n\n", logGroup, len(output.Events))
	for _, event := range output.Events {
		ts := time.UnixMilli(aws.ToInt64(event.Timestamp))
		result += fmt.Sprintf("[%s] %s\n", ts.Format("15:04:05"), aws.ToString(event.Message))
	}

	return result, false
}

func (e *ToolExecutor) searchDocs(query string) string {
	return fmt.Sprintf("Documentation search for '%s' is not yet implemented. Please check AWS documentation directly.", query)
}

func formatResourceSummary(r dao.Resource) string {
	result := fmt.Sprintf("- ID: %s", r.GetID())
	if name := r.GetName(); name != "" && name != r.GetID() {
		result += fmt.Sprintf(", Name: %s", name)
	}
	result += "\n"
	return result
}

func formatResourceDetail(r dao.Resource) string {
	result := fmt.Sprintf("ID: %s\n", r.GetID())

	if name := r.GetName(); name != "" {
		result += fmt.Sprintf("Name: %s\n", name)
	}

	if arn := r.GetARN(); arn != "" {
		result += fmt.Sprintf("ARN: %s\n", arn)
	}

	if tags := r.GetTags(); len(tags) > 0 {
		result += "\nTags:\n"
		for k, v := range tags {
			result += fmt.Sprintf("  %s: %s\n", k, v)
		}
	}

	if raw := r.Raw(); raw != nil {
		data, err := json.MarshalIndent(raw, "", "  ")
		if err == nil {
			result += fmt.Sprintf("\nRaw Data:\n%s\n", string(data))
		}
	}

	return result
}
