package builds

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
	action.Global.Register("gamelift", "builds", []action.Action{
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteBuild",
			Confirm:   action.ConfirmDangerous,
		},
	})

	action.RegisterExecutor("gamelift", "builds", executeBuildAction)
}

func executeBuildAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "DeleteBuild":
		return executeDeleteBuild(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeDeleteBuild(ctx context.Context, resource dao.Resource) action.ActionResult {
	build, ok := resource.(*BuildResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: apperrors.Wrap(err, "create gamelift client")}
	}
	client := gamelift.NewFromConfig(cfg)

	buildId := build.GetID()
	_, err = client.DeleteBuild(ctx, &gamelift.DeleteBuildInput{
		BuildId: &buildId,
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete build: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleted build %s", build.GetName()),
	}
}
