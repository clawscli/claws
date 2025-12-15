package statemachines

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/clawscli/claws/internal/action"
	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	action.Global.Register("sfn", "state-machines", []action.Action{
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteStateMachine",
			Confirm:   true,
			Dangerous: true,
		},
	})

	action.RegisterExecutor("sfn", "state-machines", executeStateMachineAction)
}

func executeStateMachineAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "DeleteStateMachine":
		return executeDeleteStateMachine(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func getSFNClient(ctx context.Context) (*sfn.Client, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, err
	}
	return sfn.NewFromConfig(cfg), nil
}

func executeDeleteStateMachine(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := getSFNClient(ctx)
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
