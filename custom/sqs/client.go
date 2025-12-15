package sqs

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	appaws "github.com/clawscli/claws/internal/aws"
)

// GetClient returns an SQS client configured for the current context
func GetClient(ctx context.Context) (*sqs.Client, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, err
	}
	return sqs.NewFromConfig(cfg), nil
}
