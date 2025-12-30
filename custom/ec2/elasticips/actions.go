package elasticips

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	appec2 "github.com/clawscli/claws/custom/ec2"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	action.Global.Register("ec2", "elastic-ips", []action.Action{
		{
			Name:      "Release",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "ReleaseAddress",
			Confirm:   action.ConfirmDangerous,
		},
		{
			Name:      "Disassociate",
			Shortcut:  "x",
			Type:      action.ActionTypeAPI,
			Operation: "DisassociateAddress",
			Confirm:   action.ConfirmSimple,
		},
	})

	action.RegisterExecutor("ec2", "elastic-ips", executeElasticIPAction)
}

func executeElasticIPAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "ReleaseAddress":
		return executeReleaseAddress(ctx, resource)
	case "DisassociateAddress":
		return executeDisassociateAddress(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeReleaseAddress(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := appec2.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	allocationID := resource.GetID()
	_, err = client.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{
		AllocationId: &allocationID,
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("release address: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Released elastic IP %s", allocationID),
	}
}

func executeDisassociateAddress(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := appec2.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	// For disassociation, we need the association ID, not allocation ID
	eip, ok := resource.(*ElasticIPResource)
	if !ok {
		return action.ActionResult{Success: false, Error: fmt.Errorf("invalid resource type")}
	}

	associationID := eip.AssociationId()
	if associationID == "" {
		return action.ActionResult{Success: false, Error: fmt.Errorf("elastic IP is not associated")}
	}

	_, err = client.DisassociateAddress(ctx, &ec2.DisassociateAddressInput{
		AssociationId: &associationID,
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("disassociate address: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Disassociated elastic IP %s", resource.GetID()),
	}
}
