package tasks

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs"

	ecsClient "github.com/clawscli/claws/custom/ecs"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	// Register actions for ECS tasks
	action.Global.Register("ecs", "tasks", []action.Action{
		{
			Name:     "Exec",
			Shortcut: "x",
			Type:     action.ActionTypeExec,
			Command:  `aws ecs execute-command --cluster "${CLUSTER}" --task "${ARN}" --container "${CONTAINER}" --interactive --command "/bin/sh"`,
			Confirm:  action.ConfirmSimple,
		},
		{
			Name:      "Stop",
			Shortcut:  "S",
			Type:      action.ActionTypeAPI,
			Operation: "StopTask",
			Confirm:   action.ConfirmSimple,
		},
	})

	// Register executor
	action.RegisterExecutor("ecs", "tasks", executeTaskAction)
}

// executeTaskAction executes an action on an ECS task
func executeTaskAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "StopTask":
		return executeStopTask(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeStopTask(ctx context.Context, resource dao.Resource) action.ActionResult {
	task, ok := resource.(*TaskResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	// Check if task is already stopped
	if task.LastStatus() == "STOPPED" {
		return action.ActionResult{Success: false, Error: fmt.Errorf("task is already stopped")}
	}

	client, err := ecsClient.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	clusterArn := task.ClusterArn()
	taskArn := task.GetARN()
	reason := "Stopped via claws"

	input := &ecs.StopTaskInput{
		Cluster: &clusterArn,
		Task:    &taskArn,
		Reason:  &reason,
	}

	_, err = client.StopTask(ctx, input)
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("stop task: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Stopped task %s", task.GetID()),
	}
}
