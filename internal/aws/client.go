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
type selectionOverrideKey struct{}

func WithRegionOverride(ctx context.Context, region string) context.Context {
	return context.WithValue(ctx, regionOverrideKey{}, region)
}

func GetRegionFromContext(ctx context.Context) string {
	if r, ok := ctx.Value(regionOverrideKey{}).(string); ok {
		return r
	}
	return ""
}

func WithSelectionOverride(ctx context.Context, sel appconfig.ProfileSelection) context.Context {
	return context.WithValue(ctx, selectionOverrideKey{}, sel)
}

func GetSelectionFromContext(ctx context.Context) (appconfig.ProfileSelection, bool) {
	if s, ok := ctx.Value(selectionOverrideKey{}).(appconfig.ProfileSelection); ok {
		return s, true
	}
	return appconfig.ProfileSelection{}, false
}

func NewConfig(ctx context.Context) (aws.Config, error) {
	sel := appconfig.Global().Selection()
	if ctxSel, ok := GetSelectionFromContext(ctx); ok {
		sel = ctxSel
	}
	opts := SelectionLoadOptions(sel)

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

func NewConfigWithRegion(ctx context.Context, region string) (aws.Config, error) {
	sel := appconfig.Global().Selection()
	if ctxSel, ok := GetSelectionFromContext(ctx); ok {
		sel = ctxSel
	}
	opts := SelectionLoadOptions(sel)
	opts = append(opts, config.WithRegion(region))

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("load AWS config for region %s: %w", region, err)
	}
	return cfg, nil
}
