package routetables

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	appec2 "github.com/clawscli/claws/custom/ec2"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	action.Global.Register("vpc", "route-tables", []action.Action{
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteRouteTable",
			Confirm:   action.ConfirmDangerous,
		},
	})

	action.RegisterExecutor("vpc", "route-tables", executeRouteTableAction)
}

func executeRouteTableAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "DeleteRouteTable":
		return executeDeleteRouteTable(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeDeleteRouteTable(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := appec2.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	routeTableID := resource.GetID()
	_, err = client.DeleteRouteTable(ctx, &ec2.DeleteRouteTableInput{
		RouteTableId: &routeTableID,
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete route table: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleted route table %s", routeTableID),
	}
}
