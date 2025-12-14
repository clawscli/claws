package keypairs

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	appec2 "github.com/clawscli/claws/custom/ec2"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	action.Global.Register("ec2", "key-pairs", []action.Action{
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteKeyPair",
			Confirm:   true,
			Dangerous: true,
		},
	})

	action.RegisterExecutor("ec2", "key-pairs", executeKeyPairAction)
}

func executeKeyPairAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "DeleteKeyPair":
		return executeDeleteKeyPair(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeDeleteKeyPair(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := appec2.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	keyPairID := resource.GetID()
	_, err = client.DeleteKeyPair(ctx, &ec2.DeleteKeyPairInput{
		KeyPairId: &keyPairID,
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete key pair: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleted key pair %s", resource.GetName()),
	}
}
