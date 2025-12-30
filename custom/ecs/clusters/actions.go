package clusters

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs"

	ecsClient "github.com/clawscli/claws/custom/ecs"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	// Register actions for ECS clusters
	action.Global.Register("ecs", "clusters", []action.Action{
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteCluster",
			Confirm:   action.ConfirmDangerous,
		},
	})

	// Register executor
	action.RegisterExecutor("ecs", "clusters", executeClusterAction)
}

// executeClusterAction executes an action on an ECS cluster
func executeClusterAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "DeleteCluster":
		return executeDeleteCluster(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeDeleteCluster(ctx context.Context, resource dao.Resource) action.ActionResult {
	cluster, ok := resource.(*ClusterResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	// Check if cluster has running tasks or services
	if cluster.RunningTasksCount() > 0 || cluster.ActiveServicesCount() > 0 {
		return action.ActionResult{
			Success: false,
			Error: fmt.Errorf("cluster has %d running tasks and %d active services; stop them first",
				cluster.RunningTasksCount(), cluster.ActiveServicesCount()),
		}
	}

	client, err := ecsClient.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	clusterName := cluster.GetName()

	input := &ecs.DeleteClusterInput{
		Cluster: &clusterName,
	}

	_, err = client.DeleteCluster(ctx, input)
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete cluster: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleted cluster %s", clusterName),
	}
}
