package subscriptions

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/sns"

	snsClient "github.com/clawscli/claws/custom/sns"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	action.Global.Register("sns", "subscriptions", []action.Action{
		{
			Name:         "Unsubscribe",
			Shortcut:     "D",
			Type:         action.ActionTypeAPI,
			Operation:    "Unsubscribe",
			Confirm:      action.ConfirmDangerous,
			ConfirmToken: action.ConfirmTokenName,
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
	return snsClient.GetClient(ctx)
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
