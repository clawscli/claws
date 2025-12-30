package instances

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	appec2 "github.com/clawscli/claws/custom/ec2"
	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	action.Global.Register("ec2", "instances", []action.Action{
		{
			Name:      "Start",
			Shortcut:  "R",
			Type:      action.ActionTypeAPI,
			Operation: "StartInstances",
			Confirm:   action.ConfirmSimple,
		},
		{
			Name:      "Stop",
			Shortcut:  "S",
			Type:      action.ActionTypeAPI,
			Operation: "StopInstances",
			Confirm:   action.ConfirmSimple,
		},
		{
			Name:      "Reboot",
			Shortcut:  "B",
			Type:      action.ActionTypeAPI,
			Operation: "RebootInstances",
			Confirm:   action.ConfirmSimple,
		},
		{
			Name:      "Terminate",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "TerminateInstances",
			Confirm:   action.ConfirmDangerous,
		},
		{
			Name:     "SSM Session",
			Shortcut: "x",
			Type:     action.ActionTypeExec,
			Command:  "aws ssm start-session --target ${ID}",
		},
	})

	action.RegisterExecutor("ec2", "instances", executeInstanceAction)
}

func executeInstanceAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "StartInstances":
		return executeStartInstance(ctx, resource)
	case "StopInstances":
		return executeStopInstance(ctx, resource)
	case "RebootInstances":
		return executeRebootInstance(ctx, resource)
	case "TerminateInstances":
		return executeTerminateInstance(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeStartInstance(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := appec2.GetClient(ctx)
	if err != nil {
		return action.FailResult(err)
	}

	instanceID := resource.GetID()
	_, err = client.StartInstances(ctx, &ec2.StartInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return action.FailResultf(err, "start instance %s", instanceID)
	}

	return action.SuccessResult(fmt.Sprintf("Started instance %s", instanceID))
}

func executeStopInstance(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := appec2.GetClient(ctx)
	if err != nil {
		return action.FailResult(err)
	}

	instanceID := resource.GetID()
	_, err = client.StopInstances(ctx, &ec2.StopInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return action.FailResultf(err, "stop instance %s", instanceID)
	}

	return action.SuccessResult(fmt.Sprintf("Stopped instance %s", instanceID))
}

func executeRebootInstance(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := appec2.GetClient(ctx)
	if err != nil {
		return action.FailResult(err)
	}

	instanceID := resource.GetID()
	_, err = client.RebootInstances(ctx, &ec2.RebootInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return action.FailResultf(err, "reboot instance %s", instanceID)
	}

	return action.SuccessResult(fmt.Sprintf("Rebooted instance %s", instanceID))
}

func executeTerminateInstance(ctx context.Context, resource dao.Resource) action.ActionResult {
	client, err := appec2.GetClient(ctx)
	if err != nil {
		return action.FailResult(err)
	}

	instanceID := resource.GetID()
	_, err = client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return action.FailResultf(err, "terminate instance %s", instanceID)
	}

	return action.SuccessResult(fmt.Sprintf("Terminated instance %s", instanceID))
}
