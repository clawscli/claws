package view

import (
	"fmt"
	"strings"

	"github.com/clawscli/claws/internal/ai"
	"github.com/clawscli/claws/internal/config"
)

// formatProfileName converts internal profile ID to display name
func formatProfileName(profileID string) string {
	sel := config.ProfileSelectionFromID(profileID)
	if sel.Mode == config.ModeNamedProfile {
		return sel.ProfileName
	}
	return sel.Mode.String()
}

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
All resource tools require a region parameter. Use profile parameter when querying cross-profile resources.

Available tools:
- list_resources(service): Lists resource types for a service
- query_resources(service, resource_type, region, profile?, limit?, offset?): Lists resources (default: 100, max: 2000, supports pagination)
- get_resource_detail(service, resource_type, region, id, cluster?, profile?): Gets resource details
- tail_logs(service, resource_type, region, id, cluster?, profile?): Fetches CloudWatch logs for a resource
  - Supported: lambda/functions, ecs/services, ecs/tasks, ecs/task-definitions, codebuild/projects, codebuild/builds, cloudtrail/trails, apigateway/stages, apigateway/stages-v2, stepfunctions/state-machines
  - cluster parameter required for ecs/services and ecs/tasks
- search_aws_docs(query): Search AWS documentation
</tool_usage>

<response_format>
Be concise. Use markdown for formatting.
</response_format>`, serviceList)

	if c.aiCtx != nil {
		if len(c.aiCtx.UserRegions) > 0 {
			prompt += "\n\n<user_selected_regions>"
			prompt += strings.Join(c.aiCtx.UserRegions, ", ")
			prompt += "\nThese are ALL regions the user is currently browsing."
			prompt += "\nIn list mode, query resources across ALL these regions (call query_resources for each)."
			prompt += "\nFor specific resources (detail/diff mode), use the region from current_context instead."
			prompt += "\n</user_selected_regions>"
		}

		if len(c.aiCtx.UserProfiles) > 0 {
			prompt += "\n\n<user_selected_profiles>"
			prompt += strings.Join(c.aiCtx.UserProfiles, ", ")
			prompt += "\nThese are ALL profile IDs the user is currently browsing."
			prompt += "\nIn list mode, query resources across ALL these profiles (call query_resources for each)."
			prompt += "\nFor specific resources (detail/diff mode), use the profile from current_context instead."
			prompt += "\n</user_selected_profiles>"
		}

		switch c.aiCtx.Mode {
		case ai.ContextModeList:
			prompt += c.buildListContextPrompt()
		case ai.ContextModeDiff:
			prompt += c.buildDiffContextPrompt()
		default:
			prompt += c.buildSingleContextPrompt()
		}
	}

	return prompt
}

func (c *ChatOverlay) buildListContextPrompt() string {
	ctx := c.aiCtx
	if ctx.Service == "" {
		return ""
	}

	prompt := fmt.Sprintf("\n<current_context mode=\"list\">\nservice=%s, resource_type=%s", ctx.Service, ctx.ResourceType)
	prompt += fmt.Sprintf(", count=%d", ctx.ResourceCount)
	if ctx.FilterText != "" {
		prompt += fmt.Sprintf(", filter=\"%s\"", ctx.FilterText)
	}
	if ctx.Service == "securityhub" && ctx.ResourceType == "findings" {
		if ctx.Toggles["ShowResolved"] {
			prompt += ", show_resolved=true"
		} else {
			prompt += ", show_resolved=false (use include_resolved=true in query_resources for all)"
		}
	}
	prompt += "\n</current_context>"
	prompt += "\nIMPORTANT: When the user asks to list or analyze resources, call query_resources for EACH combination of user_selected_regions and user_selected_profiles to get the complete view across all selected contexts."
	return prompt
}

func (c *ChatOverlay) buildDiffContextPrompt() string {
	ctx := c.aiCtx
	if ctx.DiffLeft == nil || ctx.DiffRight == nil {
		return ""
	}

	prompt := fmt.Sprintf("\n<current_context mode=\"diff\">\nservice=%s, resource_type=%s", ctx.Service, ctx.ResourceType)
	prompt += fmt.Sprintf("\nleft: id=%s, name=%s", ctx.DiffLeft.ID, ctx.DiffLeft.Name)
	if ctx.DiffLeft.Region != "" {
		prompt += fmt.Sprintf(", region=%s", ctx.DiffLeft.Region)
	}
	if ctx.DiffLeft.Profile != "" {
		prompt += fmt.Sprintf(", profile=%s", ctx.DiffLeft.Profile)
	}
	if ctx.DiffLeft.Cluster != "" {
		prompt += fmt.Sprintf(", cluster=%s", ctx.DiffLeft.Cluster)
	}
	prompt += fmt.Sprintf("\nright: id=%s, name=%s", ctx.DiffRight.ID, ctx.DiffRight.Name)
	if ctx.DiffRight.Region != "" {
		prompt += fmt.Sprintf(", region=%s", ctx.DiffRight.Region)
	}
	if ctx.DiffRight.Profile != "" {
		prompt += fmt.Sprintf(", profile=%s", ctx.DiffRight.Profile)
	}
	if ctx.DiffRight.Cluster != "" {
		prompt += fmt.Sprintf(", cluster=%s", ctx.DiffRight.Cluster)
	}
	prompt += "\n</current_context>"
	prompt += "\nIMPORTANT: Call get_resource_detail twice (once for left, once for right) using each resource's specific region and profile."
	return prompt
}

func (c *ChatOverlay) buildSingleContextPrompt() string {
	ctx := c.aiCtx
	if ctx.Service == "" {
		return ""
	}

	prompt := fmt.Sprintf("\n<current_context>service=%s", ctx.Service)
	if ctx.ResourceType != "" {
		prompt += ", resource_type=" + ctx.ResourceType
	}
	if ctx.ResourceRegion != "" {
		prompt += ", region=" + ctx.ResourceRegion
	}
	if ctx.ResourceID != "" {
		prompt += ", id=" + ctx.ResourceID
	}
	if ctx.ResourceProfile != "" {
		prompt += ", profile=" + ctx.ResourceProfile
	}
	if ctx.Cluster != "" {
		prompt += ", cluster=" + ctx.Cluster
	}
	prompt += "</current_context>"
	prompt += "\nIMPORTANT: Use the region and profile from current_context when querying this resource."
	return prompt
}

func (c *ChatOverlay) renderContextParams() string {
	ctx := c.aiCtx
	if ctx == nil {
		return ""
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("  mode: %s", ctx.Mode))

	if len(ctx.UserRegions) > 0 {
		lines = append(lines, fmt.Sprintf("  regions: %s", strings.Join(ctx.UserRegions, ", ")))
	}
	if len(ctx.UserProfiles) > 0 {
		var profileNames []string
		for _, pid := range ctx.UserProfiles {
			profileNames = append(profileNames, formatProfileName(pid))
		}
		lines = append(lines, fmt.Sprintf("  profiles: %s", strings.Join(profileNames, ", ")))
	}
	if ctx.ResourceCount > 0 {
		lines = append(lines, fmt.Sprintf("  count: %d", ctx.ResourceCount))
	}
	if ctx.FilterText != "" {
		lines = append(lines, fmt.Sprintf("  filter: %s", ctx.FilterText))
	}
	if ctx.ResourceID != "" {
		lines = append(lines, fmt.Sprintf("  id: %s", ctx.ResourceID))
	}
	if ctx.ResourceRegion != "" {
		lines = append(lines, fmt.Sprintf("  region: %s", ctx.ResourceRegion))
	}
	if ctx.ResourceProfile != "" {
		profileName := formatProfileName(ctx.ResourceProfile)
		lines = append(lines, fmt.Sprintf("  profile: %s", profileName))
	}
	if ctx.Cluster != "" {
		lines = append(lines, fmt.Sprintf("  cluster: %s", ctx.Cluster))
	}
	if ctx.DiffLeft != nil && ctx.DiffRight != nil {
		left := fmt.Sprintf("%s/%s", ctx.DiffLeft.ID, ctx.DiffLeft.Name)
		if ctx.DiffLeft.Profile != "" {
			profileName := formatProfileName(ctx.DiffLeft.Profile)
			left += fmt.Sprintf(" [%s]", profileName)
		}
		if ctx.DiffLeft.Region != "" {
			left += fmt.Sprintf(" (%s)", ctx.DiffLeft.Region)
		}
		right := fmt.Sprintf("%s/%s", ctx.DiffRight.ID, ctx.DiffRight.Name)
		if ctx.DiffRight.Profile != "" {
			profileName := formatProfileName(ctx.DiffRight.Profile)
			right += fmt.Sprintf(" [%s]", profileName)
		}
		if ctx.DiffRight.Region != "" {
			right += fmt.Sprintf(" (%s)", ctx.DiffRight.Region)
		}
		lines = append(lines, fmt.Sprintf("  left: %s", left))
		lines = append(lines, fmt.Sprintf("  right: %s", right))
	}
	if ctx.Service == "securityhub" && ctx.ResourceType == "findings" {
		showResolved := "false"
		if ctx.Toggles["ShowResolved"] {
			showResolved = "true"
		}
		lines = append(lines, fmt.Sprintf("  show_resolved: %s", showResolved))
	}

	return strings.Join(lines, "\n") + "\n"
}
