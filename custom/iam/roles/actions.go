package roles

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	appiam "github.com/clawscli/claws/custom/iam"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	action.Global.Register("iam", "roles", []action.Action{
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteRole",
			Confirm:   true,
			Dangerous: true,
		},
	})

	action.RegisterExecutor("iam", "roles", executeRoleAction)
}

func executeRoleAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "DeleteRole":
		return executeDeleteRole(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeDeleteRole(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := appiam.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	roleName := resource.GetName()
	_, err = client.DeleteRole(ctx, &iam.DeleteRoleInput{
		RoleName: &roleName,
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete role: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleted role %s", roleName),
	}
}
