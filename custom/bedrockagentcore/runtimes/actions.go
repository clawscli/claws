package runtimes

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/bedrockagentcorecontrol"

	"github.com/clawscli/claws/internal/action"
	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	action.Global.Register("bedrock-agentcore", "runtimes", []action.Action{
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteAgentRuntime",
			Confirm:   action.ConfirmDangerous,
		},
	})

	action.RegisterExecutor("bedrock-agentcore", "runtimes", executeRuntimeAction)
}

func executeRuntimeAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "DeleteAgentRuntime":
		return executeDeleteAgentRuntime(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeDeleteAgentRuntime(ctx context.Context, resource dao.Resource) action.ActionResult {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}
	client := bedrockagentcorecontrol.NewFromConfig(cfg)

	runtimeID := resource.GetID()
	input := &bedrockagentcorecontrol.DeleteAgentRuntimeInput{
		AgentRuntimeId: &runtimeID,
	}

	_, err = client.DeleteAgentRuntime(ctx, input)
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete agent runtime: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleted agent runtime %s", runtimeID),
	}
}
