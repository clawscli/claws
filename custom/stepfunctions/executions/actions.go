package executions

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/sfn"

	sfnClient "github.com/clawscli/claws/custom/stepfunctions"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	// Register actions for Step Functions executions
	action.Global.Register("stepfunctions", "executions", []action.Action{
		{
			Name:      "Stop",
			Shortcut:  "S",
			Type:      action.ActionTypeAPI,
			Operation: "StopExecution",
			Confirm:   action.ConfirmSimple,
		},
	})

	// Register executor
	action.RegisterExecutor("stepfunctions", "executions", executeExecutionAction)
}

// executeExecutionAction executes an action on a Step Functions execution
func executeExecutionAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "StopExecution":
		return executeStopExecution(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeStopExecution(ctx context.Context, resource dao.Resource) action.ActionResult {
	exec, ok := resource.(*ExecutionResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	client, err := sfnClient.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	arn := exec.ARN()
	input := &sfn.StopExecutionInput{
		ExecutionArn: &arn,
	}

	_, err = client.StopExecution(ctx, input)
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("stop execution: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Stopped execution %s", exec.GetName()),
	}
}
