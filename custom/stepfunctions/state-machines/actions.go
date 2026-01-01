package statemachines

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/sfn"

	sfnClient "github.com/clawscli/claws/custom/stepfunctions"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	action.Global.Register("stepfunctions", "state-machines", []action.Action{
		{
			Name:         "Delete",
			Shortcut:     "D",
			Type:         action.ActionTypeAPI,
			Operation:    "DeleteStateMachine",
			Confirm:      action.ConfirmDangerous,
			ConfirmToken: action.ConfirmTokenName,
		},
	})

	action.RegisterExecutor("stepfunctions", "state-machines", executeStateMachineAction)
}

func executeStateMachineAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "DeleteStateMachine":
		return executeDeleteStateMachine(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeDeleteStateMachine(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := sfnClient.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	stateMachineArn := resource.GetARN()
	_, err = client.DeleteStateMachine(ctx, &sfn.DeleteStateMachineInput{
		StateMachineArn: &stateMachineArn,
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete state machine: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleted state machine %s", resource.GetName()),
	}
}
