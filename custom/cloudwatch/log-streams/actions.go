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
	// Register actions for CloudWatch Log Streams
	action.Global.Register("cloudwatch", "log-streams", []action.Action{
		{
			Name:     action.ActionNameTailLogs,
			Shortcut: "t",
			Type:     action.ActionTypeExec,
			Command:  `aws logs tail "${LOG_GROUP}" --log-stream-names "${NAME}" --since 1h --follow`,
		},
		{
			Name:     action.ActionNameViewRecent1h,
			Shortcut: "1",
			Type:     action.ActionTypeExec,
			Command:  `aws logs tail "${LOG_GROUP}" --log-stream-names "${NAME}" --since 1h | less -R`,
		},
		{
			Name:     action.ActionNameViewRecent24h,
			Shortcut: "2",
			Type:     action.ActionTypeExec,
			Command:  `aws logs tail "${LOG_GROUP}" --log-stream-names "${NAME}" --since 24h | less -R`,
		},
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteLogStream",
			Confirm:   action.ConfirmDangerous,
		},
	})

	// Register executor
	action.RegisterExecutor("cloudwatch", "log-streams", executeLogStreamAction)
}

// executeLogStreamAction executes an action on a CloudWatch Log Stream
func executeLogStreamAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "DeleteLogStream":
		return executeDeleteLogStream(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func getCloudWatchLogsClient(ctx context.Context) (*cloudwatchlogs.Client, error) {
	return cwClient.GetLogsClient(ctx)
}

func executeDeleteLogStream(ctx context.Context, resource dao.Resource) action.ActionResult {
	ls, ok := resource.(*LogStreamResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	client, err := getCloudWatchLogsClient(ctx)
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
