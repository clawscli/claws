package subscriptions

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/clawscli/claws/internal/action"
	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	action.Global.Register("sns", "subscriptions", []action.Action{
		{
			Name:      "Unsubscribe",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "Unsubscribe",
			Confirm:   true,
			Dangerous: true,
		},
	})

	action.RegisterExecutor("sns", "subscriptions", executeSubscriptionAction)
}

func executeSubscriptionAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "Unsubscribe":
		return executeUnsubscribe(ctx, resource)
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

func executeUnsubscribe(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := getSNSClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	subscriptionArn := resource.GetARN()
	_, err = client.Unsubscribe(ctx, &sns.UnsubscribeInput{
		SubscriptionArn: &subscriptionArn,
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("unsubscribe: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Unsubscribed %s", resource.GetID()),
	}
}
