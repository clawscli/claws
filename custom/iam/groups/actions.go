package groups

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	appiam "github.com/clawscli/claws/custom/iam"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	action.Global.Register("iam", "groups", []action.Action{
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteGroup",
			Confirm:   true,
			Dangerous: true,
		},
	})

	action.RegisterExecutor("iam", "groups", executeGroupAction)
}

func executeGroupAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "DeleteGroup":
		return executeDeleteGroup(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeDeleteGroup(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := appiam.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	groupName := resource.GetName()
	_, err = client.DeleteGroup(ctx, &iam.DeleteGroupInput{
		GroupName: &groupName,
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete group: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleted group %s", groupName),
	}
}
