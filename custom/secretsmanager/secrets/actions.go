package secrets

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"

	smClient "github.com/clawscli/claws/custom/secretsmanager"
	"github.com/clawscli/claws/internal/action"
	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	action.Global.Register("secretsmanager", "secrets", []action.Action{
		{
			Name:     "View Value",
			Shortcut: "v",
			Type:     action.ActionTypeExec,
			Command:  `aws secretsmanager get-secret-value --secret-id "${ID}" --query 'SecretString' --output text | less`,
			Confirm:  action.ConfirmSimple,
		},
		{
			Name:     "Describe (JSON)",
			Shortcut: "j",
			Type:     action.ActionTypeExec,
			Command:  `aws secretsmanager describe-secret --secret-id "${ID}" | less -R`,
		},
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteSecret",
			Confirm:   action.ConfirmDangerous,
		},
	})

	// Register executor
	action.RegisterExecutor("secretsmanager", "secrets", executeSecretAction)
}

// executeSecretAction executes an action on a secret
func executeSecretAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "DeleteSecret":
		return executeDeleteSecret(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func getSecretsManagerClient(ctx context.Context) (*secretsmanager.Client, error) {
	return smClient.GetClient(ctx)
}

func executeDeleteSecret(ctx context.Context, resource dao.Resource) action.ActionResult {
	secret, ok := resource.(*SecretResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	client, err := getSecretsManagerClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	secretId := secret.GetID()
	input := &secretsmanager.DeleteSecretInput{
		SecretId:                   &secretId,
		ForceDeleteWithoutRecovery: appaws.BoolPtr(false), // Safe delete with 30-day recovery window
	}

	_, err = client.DeleteSecret(ctx, input)
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete secret: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Secret %s scheduled for deletion (30-day recovery window)", secretId),
	}
}
