package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"

	appaws "github.com/clawscli/claws/internal/aws"
	appconfig "github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/log"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/sanitize"

	apigatewayStages "github.com/clawscli/claws/custom/apigateway/stages"
	apigatewayStagesV2 "github.com/clawscli/claws/custom/apigateway/stages-v2"
	cloudtrailtrails "github.com/clawscli/claws/custom/cloudtrail/trails"
	codebuildbuilds "github.com/clawscli/claws/custom/codebuild/builds"
	codebuildprojects "github.com/clawscli/claws/custom/codebuild/projects"
	ecsservices "github.com/clawscli/claws/custom/ecs/services"
	taskdefinitions "github.com/clawscli/claws/custom/ecs/task-definitions"
	ecstasks "github.com/clawscli/claws/custom/ecs/tasks"
	sfnStateMachines "github.com/clawscli/claws/custom/stepfunctions/state-machines"
)

type ToolExecutor struct {
	registry     *registry.Registry
	aiCtx        *Context
	docsSearcher func(context.Context, string) string
}

var (
	docsSearchARNPattern       = regexp.MustCompile(`\barn:[^\s]+`)
	docsSearchAccountIDPattern = regexp.MustCompile(`\b\d{12}\b`)
)

func NewToolExecutor(_ context.Context, reg *registry.Registry, contexts ...*Context) (*ToolExecutor, error) {
	var aiCtx *Context
	if len(contexts) > 0 {
		aiCtx = contexts[0]
	}
	return &ToolExecutor{
		registry: reg,
		aiCtx:    aiCtx,
	}, nil
}

func (e *ToolExecutor) validateScope(service, resourceType, region, profile, id, cluster string) (string, string, error) {
	ctx := e.aiCtx
	if ctx == nil {
		return profile, cluster, nil
	}
	if ctx.Service != "" && service != ctx.Service {
		return "", "", fmt.Errorf("service %s is outside the current AI context", service)
	}
	if ctx.ResourceType != "" && resourceType != ctx.ResourceType {
		return "", "", fmt.Errorf("resource type %s is outside the current AI context", resourceType)
	}
	if region != "" && !regionAllowed(ctx, region) {
		return "", "", fmt.Errorf("region %s is outside the current AI context", region)
	}
	profile = defaultProfile(ctx, profile)
	if profile != "" && !profileAllowed(ctx, profile) {
		return "", "", fmt.Errorf("profile %s is outside the current AI context", profile)
	}
	if id != "" && !resourceAllowed(ctx, id) {
		return "", "", fmt.Errorf("resource %s is outside the current AI context", id)
	}
	cluster = defaultCluster(ctx, cluster)
	if cluster != "" && !clusterAllowed(ctx, cluster) {
		return "", "", fmt.Errorf("cluster %s is outside the current AI context", cluster)
	}
	return profile, cluster, nil
}

func defaultProfile(ctx *Context, profile string) string {
	if profile != "" {
		return profile
	}
	if ctx.ResourceProfile != "" {
		return ctx.ResourceProfile
	}
	if refsHaveSameProfile(ctx.DiffLeft, ctx.DiffRight) {
		return ctx.DiffLeft.Profile
	}
	return ""
}

func defaultCluster(ctx *Context, cluster string) string {
	if cluster != "" {
		return cluster
	}
	if ctx.Cluster != "" {
		return ctx.Cluster
	}
	if refsHaveSameCluster(ctx.DiffLeft, ctx.DiffRight) {
		return ctx.DiffLeft.Cluster
	}
	return ""
}

func regionAllowed(ctx *Context, region string) bool {
	if ctx.ResourceRegion != "" {
		return region == ctx.ResourceRegion
	}
	if ctx.DiffLeft != nil || ctx.DiffRight != nil {
		return resourceRefHasRegion(ctx.DiffLeft, region) || resourceRefHasRegion(ctx.DiffRight, region)
	}
	if len(ctx.UserRegions) == 0 {
		return true
	}
	for _, allowed := range ctx.UserRegions {
		if region == allowed {
			return true
		}
	}
	return false
}

func profileAllowed(ctx *Context, profile string) bool {
	if ctx.ResourceProfile != "" {
		return profile == ctx.ResourceProfile
	}
	if ctx.DiffLeft != nil || ctx.DiffRight != nil {
		return resourceRefHasProfile(ctx.DiffLeft, profile) || resourceRefHasProfile(ctx.DiffRight, profile)
	}
	if len(ctx.UserProfiles) == 0 {
		return true
	}
	for _, allowed := range ctx.UserProfiles {
		if profile == allowed {
			return true
		}
	}
	return false
}

func clusterAllowed(ctx *Context, cluster string) bool {
	if ctx.Cluster != "" {
		return cluster == ctx.Cluster
	}
	if ctx.DiffLeft != nil || ctx.DiffRight != nil {
		return resourceRefHasCluster(ctx.DiffLeft, cluster) || resourceRefHasCluster(ctx.DiffRight, cluster)
	}
	return true
}

func resourceAllowed(ctx *Context, id string) bool {
	if ctx.ResourceID != "" {
		return id == ctx.ResourceID
	}
	if ctx.DiffLeft != nil || ctx.DiffRight != nil {
		return resourceRefHasID(ctx.DiffLeft, id) || resourceRefHasID(ctx.DiffRight, id)
	}
	return true
}

func resourceRefHasID(ref *ResourceRef, id string) bool {
	return ref != nil && ref.ID == id
}

func resourceRefHasRegion(ref *ResourceRef, region string) bool {
	return ref != nil && ref.Region == region
}

func resourceRefHasProfile(ref *ResourceRef, profile string) bool {
	return ref != nil && ref.Profile == profile
}

func resourceRefHasCluster(ref *ResourceRef, cluster string) bool {
	return ref != nil && ref.Cluster == cluster
}

func refsHaveSameProfile(left, right *ResourceRef) bool {
	return left != nil && right != nil && left.Profile != "" && left.Profile == right.Profile
}

func refsHaveSameCluster(left, right *ResourceRef) bool {
	return left != nil && right != nil && left.Cluster != "" && left.Cluster == right.Cluster
}

func (e *ToolExecutor) Tools() []Tool {
	return []Tool{
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
			Description: "List AWS resources. You MUST provide service, resource_type, and region parameters.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"service": map[string]any{
						"type":        "string",
						"description": "AWS service name. Examples: ec2, lambda, s3, rds, ecs, dynamodb",
					},
					"resource_type": map[string]any{
						"type":        "string",
						"description": "Resource type. Examples: instances (for ec2), functions (for lambda), buckets (for s3), tables (for dynamodb)",
					},
					"region": map[string]any{
						"type":        "string",
						"description": "AWS region. Examples: us-east-1, us-west-2, ap-northeast-1",
					},
					"profile": map[string]any{
						"type":        "string",
						"description": "AWS profile name (optional, uses current profile if not specified)",
					},
					"include_resolved": map[string]any{
						"type":        "boolean",
						"description": "Include resolved/archived items (securityhub/findings only, default: false)",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum resources to return (default: 100, max: 2000)",
					},
					"offset": map[string]any{
						"type":        "integer",
						"description": "Skip first N resources for pagination (default: 0)",
					},
				},
				"required": []string{"service", "resource_type", "region"},
			},
		},
		{
			Name:        "get_resource_detail",
			Description: "Get detailed information about a specific AWS resource. NOTE: For ecs/services and ecs/tasks, cluster parameter is required.",
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
					"region": map[string]any{
						"type":        "string",
						"description": "AWS region (e.g., us-east-1, us-west-2)",
					},
					"id": map[string]any{
						"type":        "string",
						"description": "Resource ID",
					},
					"cluster": map[string]any{
						"type":        "string",
						"description": "ECS cluster name (required for ecs/services and ecs/tasks)",
					},
					"profile": map[string]any{
						"type":        "string",
						"description": "AWS profile name (optional, uses current profile if not specified)",
					},
				},
				"required": []string{"service", "resource_type", "region", "id"},
			},
		},
		{
			Name:        "tail_logs",
			Description: "Fetch recent CloudWatch logs for an AWS resource. Automatically extracts log group from resource configuration. Supported: lambda/functions, ecs/services, ecs/tasks, ecs/task-definitions, codebuild/projects, codebuild/builds, cloudtrail/trails, apigateway/stages, apigateway/stages-v2, stepfunctions/state-machines. NOTE: For ecs/services and ecs/tasks, cluster parameter is required.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"service": map[string]any{
						"type":        "string",
						"description": "AWS service name (e.g., lambda, ecs, codebuild)",
					},
					"resource_type": map[string]any{
						"type":        "string",
						"description": "Resource type (e.g., functions, services, tasks, task-definitions)",
					},
					"region": map[string]any{
						"type":        "string",
						"description": "AWS region (e.g., us-east-1, ap-northeast-1)",
					},
					"id": map[string]any{
						"type":        "string",
						"description": "Resource ID",
					},
					"cluster": map[string]any{
						"type":        "string",
						"description": "ECS cluster name (required for ecs/services and ecs/tasks)",
					},
					"profile": map[string]any{
						"type":        "string",
						"description": "AWS profile name (optional, uses current profile if not specified)",
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
				"required": []string{"service", "resource_type", "region", "id"},
			},
		},
		{
			Name:        "search_aws_docs",
			Description: "Search AWS documentation for information. Queries containing private or sensitive context are rejected before external search.",
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

func (e *ToolExecutor) Execute(ctx context.Context, call *ToolUseContent) ToolResultContent {
	if call.InputError != "" {
		return ToolResultContent{
			ToolUseID: call.ID,
			Content:   fmt.Sprintf("Error: malformed tool input: %s", call.InputError),
			IsError:   true,
		}
	}

	var content string
	var isError bool

	switch call.Name {
	case "list_resources":
		service, _ := call.Input["service"].(string)
		content = e.listResources(service)
	case "query_resources":
		service, _ := call.Input["service"].(string)
		resourceType, _ := call.Input["resource_type"].(string)
		region, _ := call.Input["region"].(string)
		profile, _ := call.Input["profile"].(string)
		includeResolved, _ := call.Input["include_resolved"].(bool)
		limit, _ := call.Input["limit"].(float64)
		offset, _ := call.Input["offset"].(float64)
		content, isError = e.queryResources(ctx, service, resourceType, region, profile, includeResolved, int(limit), int(offset))
	case "get_resource_detail":
		service, _ := call.Input["service"].(string)
		resourceType, _ := call.Input["resource_type"].(string)
		region, _ := call.Input["region"].(string)
		id, _ := call.Input["id"].(string)
		cluster, _ := call.Input["cluster"].(string)
		profile, _ := call.Input["profile"].(string)
		content, isError = e.getResourceDetail(ctx, service, resourceType, region, id, cluster, profile)
	case "tail_logs":
		service, _ := call.Input["service"].(string)
		resourceType, _ := call.Input["resource_type"].(string)
		region, _ := call.Input["region"].(string)
		id, _ := call.Input["id"].(string)
		cluster, _ := call.Input["cluster"].(string)
		profile, _ := call.Input["profile"].(string)
		filter, _ := call.Input["filter"].(string)
		since, _ := call.Input["since"].(string)
		limit, _ := call.Input["limit"].(float64)
		content, isError = e.tailLogs(ctx, service, resourceType, region, id, cluster, profile, filter, since, int(limit))
	case "search_aws_docs":
		query, _ := call.Input["query"].(string)
		var err error
		query, err = e.prepareDocsSearchQuery(query)
		if err != nil {
			content = "Error: " + err.Error()
			isError = true
		} else {
			content = e.runDocsSearch(ctx, query)
		}
	default:
		content = fmt.Sprintf("Unknown tool: %s", call.Name)
		isError = true
	}

	if isPrivateDataTool(call.Name) && isError {
		content = e.redactPrivateToolOutput(content)
	}

	return ToolResultContent{
		ToolUseID: call.ID,
		Content:   content,
		IsError:   isError,
	}
}

func isPrivateDataTool(toolName string) bool {
	switch toolName {
	case "query_resources", "get_resource_detail", "tail_logs":
		return true
	default:
		return false
	}
}

func (e *ToolExecutor) runDocsSearch(ctx context.Context, query string) string {
	if e.docsSearcher != nil {
		return e.docsSearcher(ctx, query)
	}
	return e.searchDocs(ctx, query)
}

func (e *ToolExecutor) prepareDocsSearchQuery(query string) (string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "", fmt.Errorf("query parameter is required")
	}
	if sanitized := e.redactPrivateDocsSearchQuery(query); sanitized != query {
		return "", fmt.Errorf("AWS documentation search query contains private or sensitive context; ask a general AWS documentation question without resource IDs, account IDs, ARNs, profile names, logs, tags, or secrets")
	}
	return query, nil
}

func (e *ToolExecutor) redactPrivateDocsSearchQuery(query string) string {
	redacted := sanitize.SensitiveText(query)
	redacted = docsSearchARNPattern.ReplaceAllString(redacted, sanitize.Redacted)
	redacted = docsSearchAccountIDPattern.ReplaceAllString(redacted, sanitize.Redacted)
	return redactDocsSearchContextValues(redacted, e.aiCtx)
}

func (e *ToolExecutor) redactPrivateToolOutput(output string) string {
	return e.redactPrivateDocsSearchQuery(output)
}

func redactDocsSearchContextValues(query string, ctx *Context) string {
	if ctx == nil {
		return query
	}
	values := []string{
		ctx.ResourceID,
		ctx.ResourceProfile,
		ctx.Cluster,
		ctx.FilterText,
	}
	values = append(values, ctx.UserProfiles...)
	values = append(values, resourceRefPrivateValues(ctx.DiffLeft)...)
	values = append(values, resourceRefPrivateValues(ctx.DiffRight)...)
	for _, value := range values {
		if value == "" {
			continue
		}
		query = strings.ReplaceAll(query, value, sanitize.Redacted)
	}
	return query
}

func resourceRefPrivateValues(ref *ResourceRef) []string {
	if ref == nil {
		return nil
	}
	return []string{ref.ID, ref.Name, ref.Profile, ref.Cluster}
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

func (e *ToolExecutor) queryResources(ctx context.Context, service, resourceType, region, profile string, includeResolved bool, limit, offset int) (string, bool) {
	if service == "" {
		return "Error: service parameter is required", true
	}
	if resourceType == "" {
		return "Error: resource_type parameter is required", true
	}
	if region == "" {
		return "Error: region parameter is required", true
	}
	var err error
	profile, _, err = e.validateScope(service, resourceType, region, profile, "", "")
	if err != nil {
		return "Error: " + err.Error(), true
	}

	// Validate and apply limit
	if limit <= 0 {
		limit = 100 // default changed from 50
	}
	if limit > 2000 {
		limit = 2000 // max 2000
	}

	// Validate offset
	if offset < 0 {
		offset = 0
	}

	if profile != "" {
		ctx = appaws.WithSelectionOverride(ctx, appconfig.ProfileSelectionFromID(profile))
	}
	ctx = appaws.WithRegionOverride(ctx, region)
	if includeResolved {
		ctx = dao.WithFilter(ctx, "ShowResolved", "true")
	}
	d, err := e.registry.GetDAO(ctx, service, resourceType)
	if err != nil {
		return fmt.Sprintf("Error: %s/%s not found. Use list_resources(service=\"%s\") to see available types.", service, resourceType, service), true
	}

	resources, err := d.List(ctx)
	if err != nil {
		return fmt.Sprintf("Error listing %s/%s: %v", service, resourceType, err), true
	}

	if len(resources) == 0 {
		return fmt.Sprintf("No %s/%s resources found in %s", service, resourceType, region), false
	}

	filterNote := ""
	if service == "securityhub" && resourceType == "findings" {
		if includeResolved {
			filterNote = " (including resolved)"
		} else {
			filterNote = " (active only, use include_resolved=true for all)"
		}
	}

	// Apply offset
	start := offset
	if start >= len(resources) {
		return fmt.Sprintf("Offset %d exceeds total count %d", offset, len(resources)), true
	}

	end := start + limit
	if end > len(resources) {
		end = len(resources)
	}

	viewResources := resources[start:end]

	result := fmt.Sprintf("Found %d %s/%s resources in %s%s (showing %d-%d):\n\n",
		len(resources), service, resourceType, region, filterNote, start+1, end)

	for _, r := range viewResources {
		result += formatResourceSummary(r)
	}

	if end < len(resources) {
		result += fmt.Sprintf("\n... and %d more (use offset=%d to see next page)\n", len(resources)-end, end)
	}

	return result, false
}

func (e *ToolExecutor) getResourceDetail(ctx context.Context, service, resourceType, region, id, cluster, profile string) (string, bool) {
	if region == "" {
		return "Error: region parameter is required", true
	}
	var err error
	profile, cluster, err = e.validateScope(service, resourceType, region, profile, id, cluster)
	if err != nil {
		return "Error: " + err.Error(), true
	}

	if profile != "" {
		ctx = appaws.WithSelectionOverride(ctx, appconfig.ProfileSelectionFromID(profile))
	}
	ctx = appaws.WithRegionOverride(ctx, region)

	if service == "ecs" && (resourceType == "services" || resourceType == "tasks") {
		if cluster == "" {
			err := "Error: cluster parameter is required for ecs/services and ecs/tasks"
			log.Warn("getResourceDetail failed", "error", err)
			return err, true
		}
		ctx = dao.WithFilter(ctx, "ClusterName", cluster)
	}

	d, err := e.registry.GetDAO(ctx, service, resourceType)
	if err != nil {
		log.Warn("getResourceDetail GetDAO failed", "error", err)
		return fmt.Sprintf("Error getting DAO: %v", err), true
	}

	resource, err := d.Get(ctx, id)
	if err != nil {
		log.Warn("getResourceDetail Get failed", "service", service, "resourceType", resourceType, "id", id, "error", err)
		return fmt.Sprintf("Error getting resource: %v", err), true
	}

	return formatResourceDetail(resource), false
}

func (e *ToolExecutor) tailLogs(ctx context.Context, service, resourceType, region, id, cluster, profile, filter, since string, limit int) (string, bool) {
	if region == "" {
		return "Error: region parameter is required", true
	}
	var err error
	profile, cluster, err = e.validateScope(service, resourceType, region, profile, id, cluster)
	if err != nil {
		return "Error: " + err.Error(), true
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	if profile != "" {
		ctx = appaws.WithSelectionOverride(ctx, appconfig.ProfileSelectionFromID(profile))
	}
	ctx = appaws.WithRegionOverride(ctx, region)

	logGroup, err := e.extractLogGroup(ctx, service, resourceType, id, cluster)
	if err != nil {
		log.Warn("tailLogs extractLogGroup failed", "service", service, "resourceType", resourceType, "id", id, "error", err)
		return fmt.Sprintf("Error extracting log group for %s/%s/%s: %v", service, resourceType, id, err), true
	}

	cfg, err := appaws.NewConfigWithRegion(ctx, region)
	if err != nil {
		return fmt.Sprintf("Error creating config for region %s: %v", region, err), true
	}
	cwClient := cloudwatchlogs.NewFromConfig(cfg)

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

	output, err := cwClient.FilterLogEvents(ctx, input)
	if err != nil {
		return fmt.Sprintf("Error fetching logs from %s: %v", logGroup, err), true
	}

	if len(output.Events) == 0 {
		sinceStr := "15m"
		if since != "" {
			sinceStr = since
		}
		return fmt.Sprintf("No logs found in %s (since %s)", logGroup, sinceStr), false
	}

	result := fmt.Sprintf("Logs from %s (%d events):\n\n", logGroup, len(output.Events))
	for _, event := range output.Events {
		ts := time.UnixMilli(aws.ToInt64(event.Timestamp))
		result += fmt.Sprintf("[%s] %s\n", ts.Format("15:04:05"), sanitize.LogText(aws.ToString(event.Message)))
	}

	return result, false
}

func (e *ToolExecutor) extractLogGroup(ctx context.Context, service, resourceType, id, cluster string) (string, error) {
	key := service + "/" + resourceType

	switch key {
	case "lambda/functions":
		return "/aws/lambda/" + id, nil

	case "ecs/task-definitions":
		resource, err := e.getResource(ctx, service, resourceType, id)
		if err != nil {
			return "", err
		}
		td, ok := resource.(*taskdefinitions.TaskDefinitionResource)
		if !ok {
			return "", fmt.Errorf("unexpected resource type for task-definitions")
		}
		if logGroup := td.GetCloudWatchLogGroup(""); logGroup != "" {
			return logGroup, nil
		}
		return "", fmt.Errorf("no CloudWatch logs configured for task definition %s", id)

	case "ecs/services":
		if cluster == "" {
			return "", fmt.Errorf("cluster parameter is required for ecs/services")
		}
		ctxWithCluster := dao.WithFilter(ctx, "ClusterName", cluster)
		resource, err := e.getResource(ctxWithCluster, service, resourceType, id)
		if err != nil {
			return "", err
		}
		svc, ok := resource.(*ecsservices.ServiceResource)
		if !ok {
			return "", fmt.Errorf("unexpected resource type for ecs services")
		}
		taskDefArn := svc.TaskDefinition()
		if taskDefArn == "" {
			return "", fmt.Errorf("no task definition found for service %s", id)
		}
		return e.extractLogGroupFromTaskDef(ctx, taskDefArn)

	case "ecs/tasks":
		if cluster == "" {
			return "", fmt.Errorf("cluster parameter is required for ecs/tasks")
		}
		ctxWithCluster := dao.WithFilter(ctx, "ClusterName", cluster)
		resource, err := e.getResource(ctxWithCluster, service, resourceType, id)
		if err != nil {
			return "", err
		}
		task, ok := resource.(*ecstasks.TaskResource)
		if !ok {
			return "", fmt.Errorf("unexpected resource type for ecs tasks")
		}
		taskDefArn := task.TaskDefinitionArn()
		if taskDefArn == "" {
			return "", fmt.Errorf("no task definition found for task %s", id)
		}
		return e.extractLogGroupFromTaskDef(ctx, taskDefArn)

	case "codebuild/projects":
		resource, err := e.getResource(ctx, service, resourceType, id)
		if err != nil {
			return "", err
		}
		proj, ok := resource.(*codebuildprojects.ProjectResource)
		if !ok {
			return "", fmt.Errorf("unexpected resource type for codebuild projects")
		}
		if proj.Project.LogsConfig != nil &&
			proj.Project.LogsConfig.CloudWatchLogs != nil &&
			proj.Project.LogsConfig.CloudWatchLogs.GroupName != nil {
			return *proj.Project.LogsConfig.CloudWatchLogs.GroupName, nil
		}
		return "/aws/codebuild/" + id, nil

	case "codebuild/builds":
		resource, err := e.getResource(ctx, service, resourceType, id)
		if err != nil {
			return "", err
		}
		build, ok := resource.(*codebuildbuilds.BuildResource)
		if !ok {
			return "", fmt.Errorf("unexpected resource type for codebuild builds")
		}
		if build.LogsGroupName() != "" {
			return build.LogsGroupName(), nil
		}
		return "", fmt.Errorf("no CloudWatch logs configured for build %s", id)

	case "cloudtrail/trails":
		resource, err := e.getResource(ctx, service, resourceType, id)
		if err != nil {
			return "", err
		}
		trail, ok := resource.(*cloudtrailtrails.TrailResource)
		if !ok {
			return "", fmt.Errorf("unexpected resource type for cloudtrail trails")
		}
		logGroupArn := trail.CloudWatchLogsLogGroupArn()
		if logGroupArn == "" {
			return "", fmt.Errorf("no CloudWatch logs configured for trail %s", id)
		}
		return extractLogGroupNameFromArn(logGroupArn), nil

	case "apigateway/stages":
		resource, err := e.getResource(ctx, service, resourceType, id)
		if err != nil {
			return "", err
		}
		stage, ok := resource.(*apigatewayStages.StageResource)
		if !ok {
			return "", fmt.Errorf("unexpected resource type for apigateway stages")
		}
		destArn := stage.AccessLogDestination()
		if destArn == "" {
			return "", fmt.Errorf("no access logs configured for stage %s", id)
		}
		return extractLogGroupNameFromArn(destArn), nil

	case "apigateway/stages-v2":
		resource, err := e.getResource(ctx, service, resourceType, id)
		if err != nil {
			return "", err
		}
		stage, ok := resource.(*apigatewayStagesV2.StageV2Resource)
		if !ok {
			return "", fmt.Errorf("unexpected resource type for apigateway stages-v2")
		}
		destArn := stage.AccessLogDestination()
		if destArn == "" {
			return "", fmt.Errorf("no access logs configured for stage %s", id)
		}
		return extractLogGroupNameFromArn(destArn), nil

	case "stepfunctions/state-machines":
		resource, err := e.getResource(ctx, service, resourceType, id)
		if err != nil {
			return "", err
		}
		sm, ok := resource.(*sfnStateMachines.StateMachineResource)
		if !ok {
			return "", fmt.Errorf("unexpected resource type for stepfunctions state-machines")
		}
		if sm.Detail != nil && sm.Detail.LoggingConfiguration != nil {
			for _, dest := range sm.Detail.LoggingConfiguration.Destinations {
				if dest.CloudWatchLogsLogGroup != nil && dest.CloudWatchLogsLogGroup.LogGroupArn != nil {
					return extractLogGroupNameFromArn(*dest.CloudWatchLogsLogGroup.LogGroupArn), nil
				}
			}
		}
		return "", fmt.Errorf("no CloudWatch logs configured for state machine %s", id)

	default:
		return "", fmt.Errorf("log extraction not supported for %s/%s. Supported: lambda/functions, ecs/services, ecs/tasks, ecs/task-definitions, codebuild/projects, codebuild/builds, cloudtrail/trails, apigateway/stages, apigateway/stages-v2, stepfunctions/state-machines", service, resourceType)
	}
}

func (e *ToolExecutor) extractLogGroupFromTaskDef(ctx context.Context, taskDefArn string) (string, error) {
	taskDefID := appaws.ExtractResourceName(taskDefArn)
	resource, err := e.getResource(ctx, "ecs", "task-definitions", taskDefID)
	if err != nil {
		return "", fmt.Errorf("failed to get task definition %s: %w", taskDefArn, err)
	}

	td, ok := resource.(*taskdefinitions.TaskDefinitionResource)
	if !ok {
		return "", fmt.Errorf("unexpected resource type")
	}

	if logGroup := td.GetCloudWatchLogGroup(""); logGroup != "" {
		return logGroup, nil
	}

	return "", fmt.Errorf("no CloudWatch logs configured in task definition %s", taskDefArn)
}

func extractLogGroupNameFromArn(arn string) string {
	parts := strings.Split(arn, ":")
	if len(parts) >= 7 {
		logGroupPart := parts[6]
		if strings.HasPrefix(logGroupPart, "log-group:") {
			return strings.TrimPrefix(logGroupPart, "log-group:")
		}
		return logGroupPart
	}
	return arn
}

func (e *ToolExecutor) searchDocs(ctx context.Context, query string) string {
	if query == "" {
		return "Error: query parameter is required"
	}

	reqBody := map[string]any{
		"textQuery": map[string]string{
			"input": query,
		},
		"contextAttributes": []map[string]string{
			{"key": "domain", "value": "docs.aws.amazon.com"},
		},
		"acceptSuggestionBody": "RawText",
		"locales":              []string{"en_us"},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Sprintf("Error creating request: %v", err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, appconfig.File().DocsSearchTimeout())
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "POST", "https://proxy.search.docs.aws.amazon.com/search", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Sprintf("Error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Sprintf("Error searching documentation: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Sprintf("Error: received status %d from AWS documentation search", resp.StatusCode)
	}

	var result struct {
		Suggestions []struct {
			TextExcerptSuggestion struct {
				Link     string `json:"link"`
				Title    string `json:"title"`
				Metadata struct {
					SeoAbstract string `json:"seo_abstract"`
					Abstract    string `json:"abstract"`
				} `json:"metadata"`
				Summary string `json:"summary"`
			} `json:"textExcerptSuggestion"`
		} `json:"suggestions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Sprintf("Error parsing response: %v", err)
	}

	if len(result.Suggestions) == 0 {
		return fmt.Sprintf("No documentation found for: %s", query)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("AWS Documentation results for '%s':\n\n", query))
	for i, s := range result.Suggestions {
		if i >= 5 {
			break
		}
		suggestion := s.TextExcerptSuggestion
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, suggestion.Title))
		sb.WriteString(fmt.Sprintf("   URL: %s\n", suggestion.Link))
		context := suggestion.Metadata.SeoAbstract
		if context == "" {
			context = suggestion.Metadata.Abstract
		}
		if context == "" {
			context = suggestion.Summary
		}
		if context != "" {
			sb.WriteString(fmt.Sprintf("   %s\n", context))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func (e *ToolExecutor) getResource(ctx context.Context, service, resourceType, id string) (dao.Resource, error) {
	d, err := e.registry.GetDAO(ctx, service, resourceType)
	if err != nil {
		return nil, err
	}
	resource, err := d.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return dao.UnwrapResource(resource), nil
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
			if isSensitiveRawKey(k) {
				v = sanitize.Redacted
			} else {
				v = sanitize.SensitiveText(v)
			}
			result += fmt.Sprintf("  %s: %s\n", k, v)
		}
	}

	if raw := r.Raw(); raw != nil {
		data, err := json.MarshalIndent(redactSensitiveRaw(raw), "", "  ")
		if err == nil {
			result += fmt.Sprintf("\nRaw Data:\n%s\n", string(data))
		}
	}

	return result
}

func redactSensitiveRaw(raw any) any {
	switch value := raw.(type) {
	case map[string]any, []any:
		return redactSensitiveValue(value)
	}

	// Some resources may expose typed SDK structs or maps with typed values.
	// Normalize those through JSON before redacting so traversal sees map[string]any.
	data, err := json.Marshal(raw)
	if err != nil {
		return raw
	}

	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return raw
	}
	return redactSensitiveValue(decoded)
}

func redactSensitiveValue(v any) any {
	switch value := v.(type) {
	case map[string]any:
		redacted := make(map[string]any, len(value))
		sensitiveRecord := hasSensitiveLabelField(value)
		for key, nested := range value {
			normalizedKey := normalizeRawKey(key)
			if sensitiveRecord && (isSensitiveLabelField(normalizedKey) || isSensitiveValueField(normalizedKey)) {
				redacted[key] = "[REDACTED]"
				continue
			}
			if isSensitiveRawKey(key) {
				redacted[key] = "[REDACTED]"
				continue
			}
			redacted[key] = redactSensitiveValue(nested)
		}
		return redacted
	case []map[string]any:
		redacted := make([]any, len(value))
		for i, nested := range value {
			redacted[i] = redactSensitiveValue(nested)
		}
		return redacted
	case []any:
		redacted := make([]any, len(value))
		for i, nested := range value {
			redacted[i] = redactSensitiveValue(nested)
		}
		return redacted
	case string:
		return sanitize.SensitiveText(value)
	default:
		return value
	}
}

func hasSensitiveLabelField(value map[string]any) bool {
	for key, nested := range value {
		if !isSensitiveLabelField(normalizeRawKey(key)) {
			continue
		}
		label, ok := nested.(string)
		if ok && isSensitiveRawKey(label) {
			return true
		}
	}
	return false
}

func isSensitiveLabelField(normalizedKey string) bool {
	switch normalizedKey {
	case "name", "key", "parameterkey", "outputkey":
		return true
	default:
		return false
	}
}

func isSensitiveValueField(normalizedKey string) bool {
	switch normalizedKey {
	case "value", "parametervalue", "outputvalue", "resolvedvalue":
		return true
	default:
		return false
	}
}

func isSensitiveRawKey(key string) bool {
	normalized := normalizeRawKey(key)
	if exactSensitiveRawKeys[normalized] {
		return true
	}
	for _, segment := range rawKeySegments(key) {
		if sensitiveRawKeySegments[segment] {
			return true
		}
	}
	return false
}

var exactSensitiveRawKeys = map[string]bool{
	"authorization":        true,
	"clientsecret":         true,
	"credential":           true,
	"credentials":          true,
	"environmentvariables": true,
	"privatekey":           true,
	"secret":               true,
	"secrets":              true,
	"secretstring":         true,
	"secretbinary":         true,
	"password":             true,
	"token":                true,
	"apikey":               true,
	"accesskey":            true,
	"accesskeyid":          true,
	"secretaccesskey":      true,
	"sessiontoken":         true,
}

var sensitiveRawKeySegments = map[string]bool{
	"authorization": true,
	"credential":    true,
	"credentials":   true,
	"password":      true,
	"private":       true,
	"secret":        true,
	"token":         true,
}

func rawKeySegments(key string) []string {
	var segments []string
	var current []rune
	runes := []rune(key)
	for i, r := range runes {
		if r == '_' || r == '-' || r == ' ' || r == '.' || r == '/' {
			segments = appendNormalizedSegment(segments, current)
			current = nil
			continue
		}
		if shouldSplitRawKeySegment(runes, i, current) {
			segments = appendNormalizedSegment(segments, current)
			current = nil
		}
		current = append(current, unicode.ToLower(r))
	}
	return appendNormalizedSegment(segments, current)
}

func shouldSplitRawKeySegment(runes []rune, index int, current []rune) bool {
	if len(current) == 0 || index == 0 || !unicode.IsUpper(runes[index]) {
		return false
	}
	previous := runes[index-1]
	if unicode.IsLower(previous) || unicode.IsDigit(previous) {
		return true
	}
	if !unicode.IsUpper(previous) || index+1 >= len(runes) {
		return false
	}
	return unicode.IsLower(runes[index+1])
}

func appendNormalizedSegment(segments []string, segment []rune) []string {
	if len(segment) == 0 {
		return segments
	}
	return append(segments, string(segment))
}

func normalizeRawKey(key string) string {
	return strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "_", ""), "-", ""))
}
