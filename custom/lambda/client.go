package lambda

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/lambda"
	appaws "github.com/clawscli/claws/internal/aws"
)

// GetClient returns a Lambda client configured for the current context
func GetClient(ctx context.Context) (*lambda.Client, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, err
	}
	return lambda.NewFromConfig(cfg), nil
}
