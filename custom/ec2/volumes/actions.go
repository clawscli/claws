package volumes

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	appec2 "github.com/clawscli/claws/custom/ec2"
	"github.com/clawscli/claws/internal/action"
	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	action.Global.Register("ec2", "volumes", []action.Action{
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteVolume",
			Confirm:   true,
			Dangerous: true,
		},
		{
			Name:      "Create Snapshot",
			Shortcut:  "s",
			Type:      action.ActionTypeAPI,
			Operation: "CreateSnapshot",
		},
		{
			Name:      "Detach",
			Shortcut:  "d",
			Type:      action.ActionTypeAPI,
			Operation: "DetachVolume",
			Confirm:   true,
		},
	})

	action.RegisterExecutor("ec2", "volumes", executeVolumeAction)
}

func executeVolumeAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "DeleteVolume":
		return executeDeleteVolume(ctx, resource)
	case "CreateSnapshot":
		return executeCreateSnapshot(ctx, resource)
	case "DetachVolume":
		return executeDetachVolume(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeDeleteVolume(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := appec2.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	volumeID := resource.GetID()
	_, err = client.DeleteVolume(ctx, &ec2.DeleteVolumeInput{
		VolumeId: &volumeID,
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete volume: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleted volume %s", volumeID),
	}
}

func executeCreateSnapshot(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := appec2.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	volumeID := resource.GetID()
	output, err := client.CreateSnapshot(ctx, &ec2.CreateSnapshotInput{
		VolumeId: &volumeID,
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("create snapshot: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Created snapshot %s from volume %s", appaws.Str(output.SnapshotId), volumeID),
	}
}

func executeDetachVolume(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := appec2.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	volumeID := resource.GetID()
	_, err = client.DetachVolume(ctx, &ec2.DetachVolumeInput{
		VolumeId: &volumeID,
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("detach volume: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Detached volume %s", volumeID),
	}
}
