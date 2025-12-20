package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	appconfig "github.com/clawscli/claws/internal/config"
)

// NewConfig creates a new AWS config with the application's region and profile settings.
// This is the preferred way to create AWS configs in DAOs.
func NewConfig(ctx context.Context) (aws.Config, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithEC2IMDSRegion(),
	}
	opts = append(opts, appconfig.ProfileLoadOptions(appconfig.Global().Profile())...)

	if region := appconfig.Global().Region(); region != "" {
		opts = append(opts, config.WithRegion(region))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("load AWS config: %w", err)
	}
	return cfg, nil
}

// NewConfigWithRegion creates a new AWS config with a specific region override.
// Use this when you need to make API calls to a specific region (e.g., S3 bucket operations).
func NewConfigWithRegion(ctx context.Context, region string) (aws.Config, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(region),
	}
	opts = append(opts, appconfig.ProfileLoadOptions(appconfig.Global().Profile())...)

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("load AWS config for region %s: %w", region, err)
	}
	return cfg, nil
}
