package parameters

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ssm"

	"github.com/clawscli/claws/internal/action"
	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	action.Global.Register("ssm", "parameters", []action.Action{
		{
			Name:     "View Value",
			Shortcut: "v",
			Type:     action.ActionTypeExec,
			Command:  `aws ssm get-parameter --name "${ID}" --with-decryption --query 'Parameter.Value' --output text | less -R`,
		},
		{
			Name:     "View History",
			Shortcut: "h",
			Type:     action.ActionTypeExec,
			Command:  `aws ssm get-parameter-history --name "${ID}" --with-decryption | less -R`,
		},
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteParameter",
			Confirm:   action.ConfirmDangerous,
		},
	})

	action.RegisterExecutor("ssm", "parameters", executeParameterAction)
}

func executeParameterAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "DeleteParameter":
		return executeDeleteParameter(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeDeleteParameter(ctx context.Context, resource dao.Resource) action.ActionResult {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}
	client := ssm.NewFromConfig(cfg)

	paramName := resource.GetID()
	input := &ssm.DeleteParameterInput{
		Name: &paramName,
	}

	_, err = client.DeleteParameter(ctx, input)
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete parameter: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleted parameter %s", paramName),
	}
}
