package stacks

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"

	cfnClient "github.com/clawscli/claws/custom/cfn"
	"github.com/clawscli/claws/internal/action"
	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	action.Global.Register("cloudformation", "stacks", []action.Action{
		{
			Name:         "Delete",
			Shortcut:     "D",
			Type:         action.ActionTypeAPI,
			Operation:    "DeleteStack",
			Confirm:      action.ConfirmDangerous,
			ConfirmToken: action.ConfirmTokenName,
		},
		{
			Name:      "Detect Drift",
			Shortcut:  "d",
			Type:      action.ActionTypeAPI,
			Operation: "DetectStackDrift",
		},
		{
			Name:      "Cancel Update",
			Shortcut:  "C",
			Type:      action.ActionTypeAPI,
			Operation: "CancelUpdateStack",
			Confirm:   action.ConfirmSimple,
		},
	})

	// Register executor for this resource
	action.RegisterExecutor("cloudformation", "stacks", executeStackAction)
}

// executeStackAction executes an action on a Stack resource
func executeStackAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "DeleteStack":
		return executeDeleteStack(ctx, resource)
	case "DetectStackDrift":
		return executeDetectStackDrift(ctx, resource)
	case "CancelUpdateStack":
		return executeCancelUpdateStack(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeDeleteStack(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := cfnClient.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	// Use stack name for deletion (more reliable than stack ID for active stacks)
	stackName := resource.GetName()

	input := &cloudformation.DeleteStackInput{
		StackName: &stackName,
	}

	_, err = client.DeleteStack(ctx, input)
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete stack: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Delete initiated for stack %s", stackName),
	}
}

func executeDetectStackDrift(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := cfnClient.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	stackName := resource.GetName()

	input := &cloudformation.DetectStackDriftInput{
		StackName: &stackName,
	}

	output, err := client.DetectStackDrift(ctx, input)
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("detect stack drift: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Drift detection started for %s (ID: %s)", stackName, appaws.Str(output.StackDriftDetectionId)),
	}
}

func executeCancelUpdateStack(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := cfnClient.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	stackName := resource.GetName()

	input := &cloudformation.CancelUpdateStackInput{
		StackName: &stackName,
	}

	_, err = client.CancelUpdateStack(ctx, input)
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("cancel update stack: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Update cancelled for stack %s", stackName),
	}
}
