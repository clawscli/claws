package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"

	appconfig "github.com/clawscli/claws/internal/config"
)

// InitContext initializes AWS context by loading config and fetching account ID.
// Updates the global config with region (if not already set) and account ID.
func InitContext(ctx context.Context) error {
	sel := appconfig.Global().Selection()

	cfg, err := config.LoadDefaultConfig(ctx, SelectionLoadOptions(sel)...)
	if err != nil {
		return err
	}

	// Set region if not already set
	if appconfig.Global().Region() == "" {
		appconfig.Global().SetRegion(cfg.Region)
	}

	// Fetch and set account ID
	accountID := FetchAccountID(ctx, cfg)
	appconfig.Global().SetAccountID(accountID)

	return nil
}

// RefreshContext re-fetches region and account ID for the current profile selection.
func RefreshContext(ctx context.Context) error {
	sel := appconfig.Global().Selection()

	cfg, err := config.LoadDefaultConfig(ctx, SelectionLoadOptions(sel)...)
	if err != nil {
		return err
	}

	if cfg.Region != "" && !appconfig.Global().IsMultiRegion() {
		appconfig.Global().SetRegion(cfg.Region)
	}

	accountID := FetchAccountID(ctx, cfg)
	appconfig.Global().SetAccountID(accountID)

	return nil
}
