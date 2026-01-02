package tables

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	ddbClient "github.com/clawscli/claws/custom/dynamodb"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	// Register actions for DynamoDB tables
	action.Global.Register("dynamodb", "tables", []action.Action{
		{
			Name:      "Scale Up RCU",
			Shortcut:  "r",
			Type:      action.ActionTypeAPI,
			Operation: "ScaleUpRCU",
			Confirm:   action.ConfirmSimple,
		},
		{
			Name:      "Scale Up WCU",
			Shortcut:  "w",
			Type:      action.ActionTypeAPI,
			Operation: "ScaleUpWCU",
			Confirm:   action.ConfirmSimple,
		},
		{
			Name:      "Switch to On-Demand",
			Shortcut:  "o",
			Type:      action.ActionTypeAPI,
			Operation: "SwitchToOnDemand",
			Confirm:   action.ConfirmSimple,
		},
		{
			Name:      "Switch to Provisioned",
			Shortcut:  "p",
			Type:      action.ActionTypeAPI,
			Operation: "SwitchToProvisioned",
			Confirm:   action.ConfirmSimple,
		},
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteTable",
			Confirm:   action.ConfirmDangerous,
		},
	})

	// Register executor
	action.RegisterExecutor("dynamodb", "tables", executeTableAction)
}

// executeTableAction executes an action on a DynamoDB table
func executeTableAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "ScaleUpRCU":
		return executeScaleCapacity(ctx, resource, true, false)
	case "ScaleUpWCU":
		return executeScaleCapacity(ctx, resource, false, true)
	case "SwitchToOnDemand":
		return executeSwitchToOnDemand(ctx, resource)
	case "SwitchToProvisioned":
		return executeSwitchToProvisioned(ctx, resource)
	case "DeleteTable":
		return executeDeleteTable(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func getDynamoDBClient(ctx context.Context) (*dynamodb.Client, error) {
	return ddbClient.GetClient(ctx)
}

func executeScaleCapacity(ctx context.Context, resource dao.Resource, scaleRCU, scaleWCU bool) action.ActionResult {
	table, ok := resource.(*TableResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	// Check if table is in provisioned mode
	if table.BillingMode() == "PAY_PER_REQUEST" {
		return action.ActionResult{Success: false, Error: fmt.Errorf("table is in on-demand mode, cannot scale capacity")}
	}

	client, err := getDynamoDBClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	tableName := table.GetName()
	currentRCU := table.ReadCapacity()
	currentWCU := table.WriteCapacity()

	newRCU := currentRCU
	newWCU := currentWCU

	// Scale by 50% or minimum +5
	if scaleRCU {
		delta := currentRCU / 2
		if delta < 5 {
			delta = 5
		}
		newRCU = currentRCU + delta
	}
	if scaleWCU {
		delta := currentWCU / 2
		if delta < 5 {
			delta = 5
		}
		newWCU = currentWCU + delta
	}

	input := &dynamodb.UpdateTableInput{
		TableName: &tableName,
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  &newRCU,
			WriteCapacityUnits: &newWCU,
		},
	}

	_, err = client.UpdateTable(ctx, input)
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("update table: %w", err)}
	}

	if scaleRCU {
		return action.ActionResult{
			Success: true,
			Message: fmt.Sprintf("Scaled %s RCU: %d → %d", tableName, currentRCU, newRCU),
		}
	}
	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Scaled %s WCU: %d → %d", tableName, currentWCU, newWCU),
	}
}

func executeSwitchToOnDemand(ctx context.Context, resource dao.Resource) action.ActionResult {
	table, ok := resource.(*TableResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	// Check if already on-demand
	if table.BillingMode() == "PAY_PER_REQUEST" {
		return action.ActionResult{Success: false, Error: fmt.Errorf("table is already in on-demand mode")}
	}

	client, err := getDynamoDBClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	tableName := table.GetName()

	input := &dynamodb.UpdateTableInput{
		TableName:   &tableName,
		BillingMode: types.BillingModePayPerRequest,
	}

	_, err = client.UpdateTable(ctx, input)
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("update table: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Switched %s to on-demand billing mode", tableName),
	}
}

func executeSwitchToProvisioned(ctx context.Context, resource dao.Resource) action.ActionResult {
	table, ok := resource.(*TableResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	// Check if already provisioned
	if table.BillingMode() != "PAY_PER_REQUEST" {
		return action.ActionResult{Success: false, Error: fmt.Errorf("table is already in provisioned mode")}
	}

	client, err := getDynamoDBClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	tableName := table.GetName()

	// Default to 5 RCU/WCU when switching from on-demand
	var rcu, wcu int64 = 5, 5

	input := &dynamodb.UpdateTableInput{
		TableName:   &tableName,
		BillingMode: types.BillingModeProvisioned,
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  &rcu,
			WriteCapacityUnits: &wcu,
		},
	}

	// If table has GSIs, set their provisioned throughput too
	gsis := table.GlobalSecondaryIndexes()
	if len(gsis) > 0 {
		var gsiUpdates []types.GlobalSecondaryIndexUpdate
		for _, gsi := range gsis {
			if gsi.IndexName != nil {
				gsiUpdates = append(gsiUpdates, types.GlobalSecondaryIndexUpdate{
					Update: &types.UpdateGlobalSecondaryIndexAction{
						IndexName: gsi.IndexName,
						ProvisionedThroughput: &types.ProvisionedThroughput{
							ReadCapacityUnits:  &rcu,
							WriteCapacityUnits: &wcu,
						},
					},
				})
			}
		}
		input.GlobalSecondaryIndexUpdates = gsiUpdates
	}

	_, err = client.UpdateTable(ctx, input)
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("update table: %w", err)}
	}

	msg := fmt.Sprintf("Switched %s to provisioned mode (RCU: %d, WCU: %d)", tableName, rcu, wcu)
	if len(gsis) > 0 {
		msg += fmt.Sprintf(" with %d GSI(s)", len(gsis))
	}
	return action.ActionResult{
		Success: true,
		Message: msg,
	}
}

func executeDeleteTable(ctx context.Context, resource dao.Resource) action.ActionResult {
	table, ok := resource.(*TableResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	client, err := getDynamoDBClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	tableName := table.GetName()

	input := &dynamodb.DeleteTableInput{
		TableName: &tableName,
	}

	_, err = client.DeleteTable(ctx, input)
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete table: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleted table %s", tableName),
	}
}
