package securitygroups

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	appec2 "github.com/clawscli/claws/custom/ec2"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	action.Global.Register("ec2", "security-groups", []action.Action{
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteSecurityGroup",
			Confirm:   true,
			Dangerous: true,
		},
	})

	action.RegisterExecutor("ec2", "security-groups", executeSecurityGroupAction)
}

func executeSecurityGroupAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "DeleteSecurityGroup":
		return executeDeleteSecurityGroup(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeDeleteSecurityGroup(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := appec2.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	groupID := resource.GetID()
	_, err = client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
		GroupId: &groupID,
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete security group: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleted security group %s", groupID),
	}
}
