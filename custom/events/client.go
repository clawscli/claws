package events

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/eventbridge"

	appaws "github.com/clawscli/claws/internal/aws"
)

// GetClient returns an EventBridge client configured for the current context
func GetClient(ctx context.Context) (*eventbridge.Client, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, err
	}
	return eventbridge.NewFromConfig(cfg), nil
}
