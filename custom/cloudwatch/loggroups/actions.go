package loggroups

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"

	cwClient "github.com/clawscli/claws/custom/cloudwatch"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	action.Global.Register("cloudwatch", "log-groups", []action.Action{
		{
			Name:     action.ActionNameTailLogs,
			Shortcut: "t",
			Type:     action.ActionTypeExec,
			Command:  `aws logs tail "${ID}" --since 1h --follow`,
		},
		{
			Name:     action.ActionNameViewRecent1h,
			Shortcut: "1",
			Type:     action.ActionTypeExec,
			Command:  `aws logs tail "${ID}" --since 1h | less -R`,
		},
		{
			Name:     action.ActionNameViewRecent24h,
			Shortcut: "2",
			Type:     action.ActionTypeExec,
			Command:  `aws logs tail "${ID}" --since 24h | less -R`,
		},
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteLogGroup",
			Confirm:   action.ConfirmDangerous,
		},
	})

	action.RegisterExecutor("cloudwatch", "log-groups", executeLogGroupAction)
}

func executeLogGroupAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "DeleteLogGroup":
		return executeDeleteLogGroup(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeDeleteLogGroup(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := cwClient.GetLogsClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	logGroupName := resource.GetID()
	input := &cloudwatchlogs.DeleteLogGroupInput{
		LogGroupName: &logGroupName,
	}

	_, err = client.DeleteLogGroup(ctx, input)
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete log group: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleted log group %s", logGroupName),
	}
}
