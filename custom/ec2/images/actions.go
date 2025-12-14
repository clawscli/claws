package images

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	appec2 "github.com/clawscli/claws/custom/ec2"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	action.Global.Register("ec2", "images", []action.Action{
		{
			Name:      "Deregister",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeregisterImage",
			Confirm:   true,
			Dangerous: true,
		},
	})

	action.RegisterExecutor("ec2", "images", executeImageAction)
}

func executeImageAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "DeregisterImage":
		return executeDeregisterImage(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeDeregisterImage(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := appec2.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	imageID := resource.GetID()
	_, err = client.DeregisterImage(ctx, &ec2.DeregisterImageInput{
		ImageId: &imageID,
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("deregister image: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deregistered image %s", imageID),
	}
}
