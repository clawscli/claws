package aws

import (
	"context"
	"sync"

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

// RefreshContextData re-fetches region and account ID for the current profile selection(s).
// Returns the data without modifying global state, allowing the caller to apply changes.
// Concurrency is limited by config.File().MaxConcurrentFetches(). Returns partial results and first error on failure.
func RefreshContextData(ctx context.Context) (region string, accountIDs map[string]string, err error) {
	selections := appconfig.Global().Selections()
	if len(selections) == 0 {
		selections = []appconfig.ProfileSelection{appconfig.SDKDefault()}
	}

	if !appconfig.Global().IsMultiRegion() {
		sel := selections[0]
		cfg, cfgErr := config.LoadDefaultConfig(ctx, SelectionLoadOptions(sel)...)
		if cfgErr == nil && cfg.Region != "" {
			region = cfg.Region
		}
	}

	var wg sync.WaitGroup
	accountIDs = make(map[string]string)
	var mu sync.Mutex
	errChan := make(chan error, len(selections))
	sem := make(chan struct{}, appconfig.File().MaxConcurrentFetches())

	for _, sel := range selections {
		wg.Add(1)
		go func(s appconfig.ProfileSelection) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			cfg, cfgErr := config.LoadDefaultConfig(ctx, SelectionLoadOptions(s)...)
			if cfgErr != nil {
				errChan <- cfgErr
				return
			}
			id := FetchAccountID(ctx, cfg)
			mu.Lock()
			accountIDs[s.ID()] = id
			mu.Unlock()
		}(sel)
	}

	wg.Wait()
	close(errChan)

	// Collect first error if any (channel is closed, so this won't block)
	select {
	case err = <-errChan:
	default:
	}

	return region, accountIDs, err
}
