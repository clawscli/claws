package view

import (
	"fmt"
	"strings"
)

func (c *ChatOverlay) buildSystemPrompt() string {
	services := c.registry.ListServices()
	serviceList := strings.Join(services, ", ")

	prompt := fmt.Sprintf(`You are an AWS resource assistant in claws TUI.

<available_services>
%s
</available_services>

<tool_usage>
When a user asks about AWS resources, you MUST call the appropriate tool. Do not just describe what you would do - actually call the tool.
Use ONLY the service names listed in available_services above. Do not guess or use similar names.
All resource tools require a region parameter.

Available tools:
- list_resources(service): Lists resource types for a service
- query_resources(service, resource_type, region): Lists actual resources
- get_resource_detail(service, resource_type, region, id): Gets resource details
- tail_logs(service, resource_type, region, id, cluster?): Fetches CloudWatch logs for a resource
  - Supported: lambda/functions, ecs/services, ecs/tasks, ecs/task-definitions, codebuild/projects, codebuild/builds, cloudtrail/trails, apigateway/stages, apigateway/stages-v2, stepfunctions/state-machines
  - cluster parameter required for ecs/services and ecs/tasks
</tool_usage>

<examples>
query_resources(service="ec2", resource_type="instances", region="us-east-1")
query_resources(service="lambda", resource_type="functions", region="us-west-2")
get_resource_detail(service="lambda", resource_type="functions", region="us-west-2", id="my-function")
tail_logs(service="lambda", resource_type="functions", region="us-east-1", id="my-func")
tail_logs(service="ecs", resource_type="tasks", region="us-east-1", id="my-task", cluster="my-cluster")
</examples>

<response_format>
Be concise. Use markdown for formatting.
</response_format>`, serviceList)

	if c.aiCtx != nil {
		if len(c.aiCtx.Regions) > 0 {
			prompt += fmt.Sprintf("\n\n<current_regions>%s</current_regions>", strings.Join(c.aiCtx.Regions, ", "))
		}
		if c.aiCtx.Service != "" {
			prompt += fmt.Sprintf("\n<current_context>service=%s", c.aiCtx.Service)
			if c.aiCtx.ResourceType != "" {
				prompt += ", resource_type=" + c.aiCtx.ResourceType
			}
			if c.aiCtx.ResourceRegion != "" {
				prompt += ", region=" + c.aiCtx.ResourceRegion
			}
			if c.aiCtx.ResourceID != "" {
				prompt += ", id=" + c.aiCtx.ResourceID
			}
			if c.aiCtx.Cluster != "" {
				prompt += ", cluster=" + c.aiCtx.Cluster
			}
			prompt += "</current_context>"
			prompt += "\nUse these values when querying this resource."
		}
	}

	return prompt
}
