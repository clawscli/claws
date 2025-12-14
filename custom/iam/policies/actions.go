package policies

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	appiam "github.com/clawscli/claws/custom/iam"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	action.Global.Register("iam", "policies", []action.Action{
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeletePolicy",
			Confirm:   true,
			Dangerous: true,
		},
	})

	action.RegisterExecutor("iam", "policies", executePolicyAction)
}

func executePolicyAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "DeletePolicy":
		return executeDeletePolicy(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeDeletePolicy(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := appiam.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	policyArn := resource.GetARN()
	_, err = client.DeletePolicy(ctx, &iam.DeletePolicyInput{
		PolicyArn: &policyArn,
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete policy: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleted policy %s", resource.GetName()),
	}
}
