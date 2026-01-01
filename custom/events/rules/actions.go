package rules

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/eventbridge"

	ebClient "github.com/clawscli/claws/custom/events"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	// Register actions for EventBridge rules
	action.Global.Register("events", "rules", []action.Action{
		{
			Name:      "Enable",
			Shortcut:  "E",
			Type:      action.ActionTypeAPI,
			Operation: "EnableRule",
			Confirm:   action.ConfirmSimple,
		},
		{
			Name:      "Disable",
			Shortcut:  "X",
			Type:      action.ActionTypeAPI,
			Operation: "DisableRule",
			Confirm:   action.ConfirmSimple,
		},
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteRule",
			Confirm:   action.ConfirmDangerous,
		},
	})

	// Register executor
	action.RegisterExecutor("events", "rules", executeRuleAction)
}

// executeRuleAction executes an action on an EventBridge rule
func executeRuleAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "EnableRule":
		return executeEnableRule(ctx, resource)
	case "DisableRule":
		return executeDisableRule(ctx, resource)
	case "DeleteRule":
		return executeDeleteRule(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeEnableRule(ctx context.Context, resource dao.Resource) action.ActionResult {
	rule, ok := resource.(*RuleResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	client, err := ebClient.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	name := rule.GetName()
	eventBusName := rule.EventBusName()
	input := &eventbridge.EnableRuleInput{
		Name:         &name,
		EventBusName: &eventBusName,
	}

	_, err = client.EnableRule(ctx, input)
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("enable rule: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Enabled rule %s", name),
	}
}

func executeDisableRule(ctx context.Context, resource dao.Resource) action.ActionResult {
	rule, ok := resource.(*RuleResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	client, err := ebClient.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	name := rule.GetName()
	eventBusName := rule.EventBusName()
	input := &eventbridge.DisableRuleInput{
		Name:         &name,
		EventBusName: &eventBusName,
	}

	_, err = client.DisableRule(ctx, input)
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("disable rule: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Disabled rule %s", name),
	}
}

func executeDeleteRule(ctx context.Context, resource dao.Resource) action.ActionResult {
	rule, ok := resource.(*RuleResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	client, err := ebClient.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	name := rule.GetName()
	eventBusName := rule.EventBusName()

	// First, remove all targets
	targetsInput := &eventbridge.ListTargetsByRuleInput{
		Rule:         &name,
		EventBusName: &eventBusName,
	}
	targetsOutput, err := client.ListTargetsByRule(ctx, targetsInput)
	if err == nil && len(targetsOutput.Targets) > 0 {
		var targetIds []string
		for _, target := range targetsOutput.Targets {
			if target.Id != nil {
				targetIds = append(targetIds, *target.Id)
			}
		}
		if len(targetIds) > 0 {
			_, err = client.RemoveTargets(ctx, &eventbridge.RemoveTargetsInput{
				Rule:         &name,
				EventBusName: &eventBusName,
				Ids:          targetIds,
			})
			if err != nil {
				return action.ActionResult{Success: false, Error: fmt.Errorf("remove targets: %w", err)}
			}
		}
	}

	input := &eventbridge.DeleteRuleInput{
		Name:         &name,
		EventBusName: &eventBusName,
	}

	_, err = client.DeleteRule(ctx, input)
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete rule: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleted rule %s", name),
	}
}
