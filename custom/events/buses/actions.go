package buses

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/eventbridge"

	ebClient "github.com/clawscli/claws/custom/events"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	action.Global.Register("events", "buses", []action.Action{
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteEventBus",
			Confirm:   action.ConfirmDangerous,
		},
	})

	action.RegisterExecutor("events", "buses", executeBusAction)
}

func executeBusAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "DeleteEventBus":
		return executeDeleteEventBus(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeDeleteEventBus(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := ebClient.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	busName := resource.GetName()
	_, err = client.DeleteEventBus(ctx, &eventbridge.DeleteEventBusInput{
		Name: &busName,
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete event bus: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleted event bus %s", busName),
	}
}
