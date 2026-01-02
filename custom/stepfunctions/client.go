package stepfunctions

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sfn"

	appaws "github.com/clawscli/claws/internal/aws"
)

// GetClient returns a Step Functions client configured for the current context
func GetClient(ctx context.Context) (*sfn.Client, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, err
	}
	return sfn.NewFromConfig(cfg), nil
}
