package subnets

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	appec2 "github.com/clawscli/claws/custom/ec2"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	action.Global.Register("vpc", "subnets", []action.Action{
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteSubnet",
			Confirm:   true,
			Dangerous: true,
		},
	})

	action.RegisterExecutor("vpc", "subnets", executeSubnetAction)
}

func executeSubnetAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "DeleteSubnet":
		return executeDeleteSubnet(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeDeleteSubnet(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := appec2.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	subnetID := resource.GetID()
	_, err = client.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{
		SubnetId: &subnetID,
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete subnet: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleted subnet %s", subnetID),
	}
}
