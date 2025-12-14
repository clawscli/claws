package buses

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/clawscli/claws/internal/action"
	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	action.Global.Register("eventbridge", "buses", []action.Action{
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteEventBus",
			Confirm:   true,
			Dangerous: true,
		},
	})

	action.RegisterExecutor("eventbridge", "buses", executeBusAction)
}

func executeBusAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "DeleteEventBus":
		return executeDeleteEventBus(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func getEventBridgeClient(ctx context.Context) (*eventbridge.Client, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, err
	}
	return eventbridge.NewFromConfig(cfg), nil
}

func executeDeleteEventBus(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := getEventBridgeClient(ctx)
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
