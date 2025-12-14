package repositories

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/clawscli/claws/internal/action"
	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	action.Global.Register("ecr", "repositories", []action.Action{
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteRepository",
			Confirm:   true,
			Dangerous: true,
		},
	})

	action.RegisterExecutor("ecr", "repositories", executeRepositoryAction)
}

func executeRepositoryAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "DeleteRepository":
		return executeDeleteRepository(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func getECRClient(ctx context.Context) (*ecr.Client, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, err
	}
	return ecr.NewFromConfig(cfg), nil
}

func executeDeleteRepository(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := getECRClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	repoName := resource.GetName()
	_, err = client.DeleteRepository(ctx, &ecr.DeleteRepositoryInput{
		RepositoryName: &repoName,
		Force:          true,
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete repository: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleted repository %s", repoName),
	}
}
