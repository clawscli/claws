package users

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	appiam "github.com/clawscli/claws/custom/iam"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	action.Global.Register("iam", "users", []action.Action{
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteUser",
			Confirm:   true,
			Dangerous: true,
		},
	})

	action.RegisterExecutor("iam", "users", executeUserAction)
}

func executeUserAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "DeleteUser":
		return executeDeleteUser(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeDeleteUser(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := appiam.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	userName := resource.GetName()
	_, err = client.DeleteUser(ctx, &iam.DeleteUserInput{
		UserName: &userName,
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete user: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleted user %s", userName),
	}
}
