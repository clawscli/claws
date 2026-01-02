package queues

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"

	sqsClient "github.com/clawscli/claws/custom/sqs"
	"github.com/clawscli/claws/internal/action"
	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
)

func init() {
	// Register actions for SQS queues
	action.Global.Register("sqs", "queues", []action.Action{
		{
			Name:      "Purge Queue",
			Shortcut:  "p",
			Type:      action.ActionTypeAPI,
			Operation: "PurgeQueue",
			Confirm:   action.ConfirmSimple,
		},
		{
			Name:      "Send Test Message",
			Shortcut:  "s",
			Type:      action.ActionTypeAPI,
			Operation: "SendTestMessage",
			Confirm:   action.ConfirmSimple,
		},
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteQueue",
			Confirm:   action.ConfirmDangerous,
		},
	})

	// Register executor
	action.RegisterExecutor("sqs", "queues", executeQueueAction)
}

// executeQueueAction executes an action on an SQS queue
func executeQueueAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "PurgeQueue":
		return executePurgeQueue(ctx, resource)
	case "SendTestMessage":
		return executeSendTestMessage(ctx, resource)
	case "DeleteQueue":
		return executeDeleteQueue(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func getSQSClient(ctx context.Context) (*sqs.Client, error) {
	return sqsClient.GetClient(ctx)
}

func executePurgeQueue(ctx context.Context, resource dao.Resource) action.ActionResult {
	queue, ok := resource.(*QueueResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	client, err := getSQSClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	queueUrl := queue.URL
	queueName := queue.GetName()

	input := &sqs.PurgeQueueInput{
		QueueUrl: &queueUrl,
	}

	_, err = client.PurgeQueue(ctx, input)
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("purge queue: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Purged all messages from %s", queueName),
	}
}

func executeSendTestMessage(ctx context.Context, resource dao.Resource) action.ActionResult {
	queue, ok := resource.(*QueueResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	client, err := getSQSClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	queueUrl := queue.URL
	queueName := queue.GetName()
	messageBody := `{"test": true, "source": "claws"}`

	input := &sqs.SendMessageInput{
		QueueUrl:    &queueUrl,
		MessageBody: &messageBody,
	}

	// Add message group ID for FIFO queues
	if queue.IsFIFO() {
		groupId := "claws-test"
		dedupId := fmt.Sprintf("claws-test-%d", time.Now().UnixNano())
		input.MessageGroupId = &groupId
		input.MessageDeduplicationId = &dedupId
	}

	output, err := client.SendMessage(ctx, input)
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("send message: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Sent test message to %s (ID: %s)", queueName, appaws.Str(output.MessageId)),
	}
}

func executeDeleteQueue(ctx context.Context, resource dao.Resource) action.ActionResult {
	queue, ok := resource.(*QueueResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	client, err := getSQSClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	queueUrl := queue.URL
	queueName := queue.GetName()

	input := &sqs.DeleteQueueInput{
		QueueUrl: &queueUrl,
	}

	_, err = client.DeleteQueue(ctx, input)
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete queue: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleted queue %s", queueName),
	}
}
