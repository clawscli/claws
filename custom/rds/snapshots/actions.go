package snapshots

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/rds"

	rdsClient "github.com/clawscli/claws/custom/rds"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	action.Global.Register("rds", "snapshots", []action.Action{
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteDBSnapshot",
			Confirm:   action.ConfirmDangerous,
		},
	})

	action.RegisterExecutor("rds", "snapshots", executeSnapshotAction)
}

func executeSnapshotAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "DeleteDBSnapshot":
		return executeDeleteDBSnapshot(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func getRDSClient(ctx context.Context) (*rds.Client, error) {
	return rdsClient.GetClient(ctx)
}

func executeDeleteDBSnapshot(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := getRDSClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	snapshotID := resource.GetID()
	_, err = client.DeleteDBSnapshot(ctx, &rds.DeleteDBSnapshotInput{
		DBSnapshotIdentifier: &snapshotID,
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete db snapshot: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleted DB snapshot %s", snapshotID),
	}
}
