package gamesessionqueues

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/gamelift"

	"github.com/clawscli/claws/internal/action"
	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

func init() {
	action.Global.Register("gamelift", "game-session-queues", []action.Action{
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteGameSessionQueue",
			Confirm:   action.ConfirmDangerous,
		},
	})

	action.RegisterExecutor("gamelift", "game-session-queues", executeQueueAction)
}

func executeQueueAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "DeleteGameSessionQueue":
		return executeDeleteQueue(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeDeleteQueue(ctx context.Context, resource dao.Resource) action.ActionResult {
	queue, ok := resource.(*QueueResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: apperrors.Wrap(err, "create gamelift client")}
	}
	client := gamelift.NewFromConfig(cfg)

	name := queue.GetName()
	_, err = client.DeleteGameSessionQueue(ctx, &gamelift.DeleteGameSessionQueueInput{
		Name: &name,
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete game session queue: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleted game session queue %s", name),
	}
}
