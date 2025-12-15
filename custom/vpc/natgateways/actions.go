package natgateways

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	appec2 "github.com/clawscli/claws/custom/ec2"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	action.Global.Register("vpc", "nat-gateways", []action.Action{
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteNatGateway",
			Confirm:   true,
			Dangerous: true,
		},
	})

	action.RegisterExecutor("vpc", "nat-gateways", executeNatGatewayAction)
}

func executeNatGatewayAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "DeleteNatGateway":
		return executeDeleteNatGateway(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeDeleteNatGateway(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := appec2.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	natGatewayID := resource.GetID()
	_, err = client.DeleteNatGateway(ctx, &ec2.DeleteNatGatewayInput{
		NatGatewayId: &natGatewayID,
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete nat gateway: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleted NAT gateway %s", natGatewayID),
	}
}
