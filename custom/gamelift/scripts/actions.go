package scripts

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
	action.Global.Register("gamelift", "scripts", []action.Action{
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteScript",
			Confirm:   action.ConfirmDangerous,
		},
	})

	action.RegisterExecutor("gamelift", "scripts", executeScriptAction)
}

func executeScriptAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "DeleteScript":
		return executeDeleteScript(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeDeleteScript(ctx context.Context, resource dao.Resource) action.ActionResult {
	script, ok := resource.(*ScriptResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: apperrors.Wrap(err, "create gamelift client")}
	}
	client := gamelift.NewFromConfig(cfg)

	scriptId := script.GetID()
	_, err = client.DeleteScript(ctx, &gamelift.DeleteScriptInput{
		ScriptId: &scriptId,
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete script: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleted script %s", script.GetName()),
	}
}
