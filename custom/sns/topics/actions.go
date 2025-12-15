package topics

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/clawscli/claws/internal/action"
	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	action.Global.Register("sns", "topics", []action.Action{
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteTopic",
			Confirm:   true,
			Dangerous: true,
		},
	})

	action.RegisterExecutor("sns", "topics", executeTopicAction)
}

func executeTopicAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "DeleteTopic":
		return executeDeleteTopic(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func getSNSClient(ctx context.Context) (*sns.Client, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, err
	}
	return sns.NewFromConfig(cfg), nil
}

func executeDeleteTopic(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := getSNSClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	topicArn := resource.GetARN()
	_, err = client.DeleteTopic(ctx, &sns.DeleteTopicInput{
		TopicArn: &topicArn,
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete topic: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleted topic %s", resource.GetName()),
	}
}
