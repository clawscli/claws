package internetgateways

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	appec2 "github.com/clawscli/claws/custom/ec2"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	action.Global.Register("vpc", "internet-gateways", []action.Action{
		{
			Name:      "Detach",
			Shortcut:  "X",
			Type:      action.ActionTypeAPI,
			Operation: "DetachInternetGateway",
			Confirm:   action.ConfirmDangerous,
		},
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteInternetGateway",
			Confirm:   action.ConfirmDangerous,
		},
	})

	action.RegisterExecutor("vpc", "internet-gateways", executeInternetGatewayAction)
}

func executeInternetGatewayAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "DetachInternetGateway":
		return executeDetachInternetGateway(ctx, resource)
	case "DeleteInternetGateway":
		return executeDeleteInternetGateway(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeDetachInternetGateway(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := appec2.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	igw, ok := resource.(*InternetGatewayResource)
	if !ok {
		return action.ActionResult{Success: false, Error: fmt.Errorf("invalid resource type")}
	}

	vpcID := igw.AttachedVpcId()
	if vpcID == "" {
		return action.ActionResult{Success: false, Error: fmt.Errorf("internet gateway is not attached to a VPC")}
	}

	igwID := resource.GetID()
	_, err = client.DetachInternetGateway(ctx, &ec2.DetachInternetGatewayInput{
		InternetGatewayId: &igwID,
		VpcId:             &vpcID,
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("detach internet gateway: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Detached internet gateway %s from VPC %s", igwID, vpcID),
	}
}

func executeDeleteInternetGateway(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := appec2.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	igwID := resource.GetID()
	_, err = client.DeleteInternetGateway(ctx, &ec2.DeleteInternetGatewayInput{
		InternetGatewayId: &igwID,
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete internet gateway: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleted internet gateway %s", igwID),
	}
}
