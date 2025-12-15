package sns

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sns"
	appaws "github.com/clawscli/claws/internal/aws"
)

// GetClient returns an SNS client configured for the current context
func GetClient(ctx context.Context) (*sns.Client, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, err
	}
	return sns.NewFromConfig(cfg), nil
}
