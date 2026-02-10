package fleets

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
	action.Global.Register("gamelift", "fleets", []action.Action{
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteFleet",
			Confirm:   action.ConfirmDangerous,
		},
	})

	action.RegisterExecutor("gamelift", "fleets", executeFleetAction)
}

func executeFleetAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "DeleteFleet":
		return executeDeleteFleet(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeDeleteFleet(ctx context.Context, resource dao.Resource) action.ActionResult {
	fleet, ok := resource.(*FleetResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: apperrors.Wrap(err, "create gamelift client")}
	}
	client := gamelift.NewFromConfig(cfg)

	fleetId := fleet.GetID()
	_, err = client.DeleteFleet(ctx, &gamelift.DeleteFleetInput{
		FleetId: &fleetId,
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete fleet: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleted fleet %s", fleet.GetName()),
	}
}
