package cfn

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	appaws "github.com/clawscli/claws/internal/aws"
)

// GetClient returns a CloudFormation client configured for the current context
func GetClient(ctx context.Context) (*cloudformation.Client, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, err
	}
	return cloudformation.NewFromConfig(cfg), nil
}
