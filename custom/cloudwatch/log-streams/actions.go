package logstreams

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"

	cwClient "github.com/clawscli/claws/custom/cloudwatch"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	action.Global.Register("cloudwatch", "log-streams", []action.Action{
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteLogStream",
			Confirm:   action.ConfirmDangerous,
		},
	})

	action.RegisterExecutor("cloudwatch", "log-streams", executeLogStreamAction)
}

func executeLogStreamAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "DeleteLogStream":
		return executeDeleteLogStream(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeDeleteLogStream(ctx context.Context, resource dao.Resource) action.ActionResult {
	ls, ok := dao.UnwrapResource(resource).(*LogStreamResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	client, err := cwClient.GetLogsClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	logGroupName := ls.LogGroupName()
	streamName := ls.LogStreamName()

	input := &cloudwatchlogs.DeleteLogStreamInput{
		LogGroupName:  &logGroupName,
		LogStreamName: &streamName,
	}

	_, err = client.DeleteLogStream(ctx, input)
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete log stream: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleted log stream %s", streamName),
	}
}
