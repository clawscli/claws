package cloudwatch

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	appaws "github.com/clawscli/claws/internal/aws"
)

// GetLogsClient returns a CloudWatch Logs client configured for the current context
func GetLogsClient(ctx context.Context) (*cloudwatchlogs.Client, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, err
	}
	return cloudwatchlogs.NewFromConfig(cfg), nil
}
