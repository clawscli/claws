package services

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs"

	ecsClient "github.com/clawscli/claws/custom/ecs"
	"github.com/clawscli/claws/internal/action"
	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	// Register actions for ECS services
	action.Global.Register("ecs", "services", []action.Action{
		{
			Name:      "Scale Up",
			Shortcut:  "+",
			Type:      action.ActionTypeAPI,
			Operation: "ScaleUp",
			Confirm:   action.ConfirmSimple,
		},
		{
			Name:      "Scale Down",
			Shortcut:  "-",
			Type:      action.ActionTypeAPI,
			Operation: "ScaleDown",
			Confirm:   action.ConfirmSimple,
		},
		{
			Name:      "Force Deploy",
			Shortcut:  "f",
			Type:      action.ActionTypeAPI,
			Operation: "ForceNewDeployment",
			Confirm:   action.ConfirmSimple,
		},
		{
			Name:      "Enable Exec",
			Shortcut:  "x",
			Type:      action.ActionTypeAPI,
			Operation: "EnableExecuteCommand",
			Confirm:   action.ConfirmSimple,
		},
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteService",
			Confirm:   action.ConfirmDangerous,
		},
	})

	// Register executor
	action.RegisterExecutor("ecs", "services", executeServiceAction)
}

// executeServiceAction executes an action on an ECS service
func executeServiceAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "ScaleUp":
		return executeScale(ctx, resource, 1)
	case "ScaleDown":
		return executeScale(ctx, resource, -1)
	case "ForceNewDeployment":
		return executeForceNewDeployment(ctx, resource)
	case "EnableExecuteCommand":
		return executeEnableExecuteCommand(ctx, resource)
	case "DeleteService":
		return executeDeleteService(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func executeScale(ctx context.Context, resource dao.Resource, delta int32) action.ActionResult {
	svc, ok := resource.(*ServiceResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	client, err := ecsClient.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	clusterName := appaws.ExtractResourceName(svc.ClusterArn())
	serviceName := svc.GetName()
	currentCount := svc.DesiredCount()
	newCount := currentCount + delta

	if newCount < 0 {
		newCount = 0
	}

	input := &ecs.UpdateServiceInput{
		Cluster:      &clusterName,
		Service:      &serviceName,
		DesiredCount: &newCount,
	}

	output, err := client.UpdateService(ctx, input)
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("update service: %w", err)}
	}

	actualCount := int32(0)
	if output.Service != nil {
		actualCount = output.Service.DesiredCount
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Scaled %s: %d → %d tasks", serviceName, currentCount, actualCount),
	}
}

func executeForceNewDeployment(ctx context.Context, resource dao.Resource) action.ActionResult {
	svc, ok := resource.(*ServiceResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	client, err := ecsClient.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	clusterName := appaws.ExtractResourceName(svc.ClusterArn())
	serviceName := svc.GetName()

	input := &ecs.UpdateServiceInput{
		Cluster:            &clusterName,
		Service:            &serviceName,
		ForceNewDeployment: true,
	}

	_, err = client.UpdateService(ctx, input)
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("force new deployment: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Force deployment initiated for %s", serviceName),
	}
}

func executeEnableExecuteCommand(ctx context.Context, resource dao.Resource) action.ActionResult {
	svc, ok := resource.(*ServiceResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	client, err := ecsClient.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	clusterName := appaws.ExtractResourceName(svc.ClusterArn())
	serviceName := svc.GetName()
	enableExec := true

	input := &ecs.UpdateServiceInput{
		Cluster:              &clusterName,
		Service:              &serviceName,
		EnableExecuteCommand: &enableExec,
		ForceNewDeployment:   true, // Required to apply the change to running tasks
	}

	_, err = client.UpdateService(ctx, input)
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("enable execute command: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Enabled ECS Exec for %s (new tasks will have exec enabled)", serviceName),
	}
}

func executeDeleteService(ctx context.Context, resource dao.Resource) action.ActionResult {
	svc, ok := resource.(*ServiceResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	client, err := ecsClient.GetClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	clusterName := appaws.ExtractResourceName(svc.ClusterArn())
	serviceName := svc.GetName()
	force := true

	input := &ecs.DeleteServiceInput{
		Cluster: &clusterName,
		Service: &serviceName,
		Force:   &force,
	}

	_, err = client.DeleteService(ctx, input)
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete service: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleted service %s", serviceName),
	}
}
