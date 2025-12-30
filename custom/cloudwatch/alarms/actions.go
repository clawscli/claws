package alarms

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"

	cwClient "github.com/clawscli/claws/custom/cloudwatch"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	action.Global.Register("cloudwatch", "alarms", []action.Action{
		{
			Name:      "Enable",
			Shortcut:  "E",
			Type:      action.ActionTypeAPI,
			Operation: "EnableAlarmActions",
			Confirm:   action.ConfirmSimple,
		},
		{
			Name:      "Disable",
			Shortcut:  "X",
			Type:      action.ActionTypeAPI,
			Operation: "DisableAlarmActions",
			Confirm:   action.ConfirmSimple,
		},
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteAlarms",
			Confirm:   action.ConfirmDangerous,
		},
	})

	action.RegisterExecutor("cloudwatch", "alarms", executeAlarmAction)
}

func executeAlarmAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "EnableAlarmActions":
		return executeEnableAlarm(ctx, resource)
	case "DisableAlarmActions":
		return executeDisableAlarm(ctx, resource)
	case "DeleteAlarms":
		return executeDeleteAlarm(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func getClient(ctx context.Context) (*cloudwatch.Client, error) {
	return cwClient.GetClient(ctx)
}

func executeEnableAlarm(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := getClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	alarmName := resource.GetID()
	_, err = client.EnableAlarmActions(ctx, &cloudwatch.EnableAlarmActionsInput{
		AlarmNames: []string{alarmName},
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("enable alarm actions: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Enabled actions for alarm %s", alarmName),
	}
}

func executeDisableAlarm(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := getClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	alarmName := resource.GetID()
	_, err = client.DisableAlarmActions(ctx, &cloudwatch.DisableAlarmActionsInput{
		AlarmNames: []string{alarmName},
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("disable alarm actions: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Disabled actions for alarm %s", alarmName),
	}
}

func executeDeleteAlarm(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := getClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	alarmName := resource.GetID()
	_, err = client.DeleteAlarms(ctx, &cloudwatch.DeleteAlarmsInput{
		AlarmNames: []string{alarmName},
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete alarm: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleted alarm %s", alarmName),
	}
}
