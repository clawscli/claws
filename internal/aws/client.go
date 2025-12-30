package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"

	appconfig "github.com/clawscli/claws/internal/config"
)

// CostExplorerRegion is the only region where Cost Explorer API is available.
const CostExplorerRegion = "us-east-1"

type regionOverrideKey struct{}

// WithRegionOverride returns a context with region override for multi-region queries
func WithRegionOverride(ctx context.Context, region string) context.Context {
	return context.WithValue(ctx, regionOverrideKey{}, region)
}

// GetRegionFromContext returns region from context override, or empty string if not set
func GetRegionFromContext(ctx context.Context) string {
	if r, ok := ctx.Value(regionOverrideKey{}).(string); ok {
		return r
	}
	return ""
}

// NewConfig creates a new AWS config with the application's region and profile settings.
// This is the preferred way to create AWS configs in DAOs.
func NewConfig(ctx context.Context) (aws.Config, error) {
	opts := SelectionLoadOptions(appconfig.Global().Selection())

	region := GetRegionFromContext(ctx)
	if region == "" {
		region = appconfig.Global().Region()
	}
	if region != "" {
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
	opts := SelectionLoadOptions(appconfig.Global().Selection())
	opts = append(opts, config.WithRegion(region))

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("load AWS config for region %s: %w", region, err)
	}
	return cfg, nil
}
