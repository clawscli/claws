package s3

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	appaws "github.com/clawscli/claws/internal/aws"
)

// GetClient returns an S3 client configured for the current context
func GetClient(ctx context.Context) (*s3.Client, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, err
	}
	return s3.NewFromConfig(cfg), nil
}

// GetClientForRegion returns an S3 client configured for a specific region
func GetClientForRegion(ctx context.Context, region string) (*s3.Client, error) {
	cfg, err := appaws.NewConfigWithRegion(ctx, region)
	if err != nil {
		return nil, err
	}
	return s3.NewFromConfig(cfg), nil
}
