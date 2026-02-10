package matchmakingconfigs

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
	action.Global.Register("gamelift", "matchmaking-configs", []action.Action{
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteMatchmakingConfiguration",
			Confirm:   action.ConfirmDangerous,
		},
	})

	action.RegisterExecutor("gamelift", "matchmaking-configs", executeMatchmakingConfigAction)
}

func executeMatchmakingConfigAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "DeleteMatchmakingConfiguration":
		return executeDeleteMatchmakingConfig(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeDeleteMatchmakingConfig(ctx context.Context, resource dao.Resource) action.ActionResult {
	config, ok := resource.(*MatchmakingConfigResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: apperrors.Wrap(err, "create gamelift client")}
	}
	client := gamelift.NewFromConfig(cfg)

	name := config.GetName()
	_, err = client.DeleteMatchmakingConfiguration(ctx, &gamelift.DeleteMatchmakingConfigurationInput{
		Name: &name,
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete matchmaking configuration: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleted matchmaking configuration %s", name),
	}
}
