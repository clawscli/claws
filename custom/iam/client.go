package iam

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	appaws "github.com/clawscli/claws/internal/aws"
)

// GetClient returns an IAM client configured for the current context
func GetClient(ctx context.Context) (*iam.Client, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, err
	}
	return iam.NewFromConfig(cfg), nil
}
