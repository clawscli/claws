package instances

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/rds"

	rdsClient "github.com/clawscli/claws/custom/rds"
	"github.com/clawscli/claws/internal/action"
	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	// Register actions for RDS instances
	action.Global.Register("rds", "instances", []action.Action{
		{
			Name:      "Start",
			Shortcut:  "R",
			Type:      action.ActionTypeAPI,
			Operation: "StartDBInstance",
			Confirm:   action.ConfirmSimple,
		},
		{
			Name:      "Stop",
			Shortcut:  "S",
			Type:      action.ActionTypeAPI,
			Operation: "StopDBInstance",
			Confirm:   action.ConfirmSimple,
		},
		{
			Name:      "Reboot",
			Shortcut:  "B",
			Type:      action.ActionTypeAPI,
			Operation: "RebootDBInstance",
			Confirm:   action.ConfirmSimple,
		},
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteDBInstance",
			Confirm:   action.ConfirmDangerous,
		},
	})

	// Register executor
	action.RegisterExecutor("rds", "instances", executeInstanceAction)
}

// executeInstanceAction executes an action on an RDS instance
func executeInstanceAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "StartDBInstance":
		return executeStartInstance(ctx, resource)
	case "StopDBInstance":
		return executeStopInstance(ctx, resource)
	case "RebootDBInstance":
		return executeRebootInstance(ctx, resource)
	case "DeleteDBInstance":
		return executeDeleteInstance(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeStartInstance(ctx context.Context, resource dao.Resource) action.ActionResult {
	instance, ok := resource.(*InstanceResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	client, err := rdsClient.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	identifier := instance.GetID()
	input := &rds.StartDBInstanceInput{
		DBInstanceIdentifier: &identifier,
	}

	_, err = client.StartDBInstance(ctx, input)
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("start db instance: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Starting DB instance %s", identifier),
	}
}

func executeStopInstance(ctx context.Context, resource dao.Resource) action.ActionResult {
	instance, ok := resource.(*InstanceResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	client, err := rdsClient.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	identifier := instance.GetID()
	input := &rds.StopDBInstanceInput{
		DBInstanceIdentifier: &identifier,
	}

	_, err = client.StopDBInstance(ctx, input)
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("stop db instance: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Stopping DB instance %s", identifier),
	}
}

func executeRebootInstance(ctx context.Context, resource dao.Resource) action.ActionResult {
	instance, ok := resource.(*InstanceResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	client, err := rdsClient.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	identifier := instance.GetID()
	input := &rds.RebootDBInstanceInput{
		DBInstanceIdentifier: &identifier,
	}

	_, err = client.RebootDBInstance(ctx, input)
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("reboot db instance: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Rebooting DB instance %s", identifier),
	}
}

func executeDeleteInstance(ctx context.Context, resource dao.Resource) action.ActionResult {
	instance, ok := resource.(*InstanceResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	client, err := rdsClient.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	identifier := instance.GetID()
	skipFinalSnapshot := true
	input := &rds.DeleteDBInstanceInput{
		DBInstanceIdentifier:   &identifier,
		SkipFinalSnapshot:      &skipFinalSnapshot,
		DeleteAutomatedBackups: appaws.BoolPtr(true),
	}

	_, err = client.DeleteDBInstance(ctx, input)
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete db instance: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleting DB instance %s", identifier),
	}
}
