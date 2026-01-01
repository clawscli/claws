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

// RefreshContext re-fetches region and account ID for the current profile selection(s).
func RefreshContext(ctx context.Context) error {
	selections := appconfig.Global().Selections()
	if len(selections) == 0 {
		selections = []appconfig.ProfileSelection{appconfig.SDKDefault()}
	}

	// Update global region if single selection
	if !appconfig.Global().IsMultiRegion() {
		sel := selections[0]
		cfg, err := config.LoadDefaultConfig(ctx, SelectionLoadOptions(sel)...)
		if err == nil && cfg.Region != "" {
			appconfig.Global().SetRegion(cfg.Region)
		}
	}

	var wg sync.WaitGroup
	accountIDs := make(map[string]string)
	var mu sync.Mutex
	errChan := make(chan error, len(selections))

	for _, sel := range selections {
		wg.Add(1)
		go func(s appconfig.ProfileSelection) {
			defer wg.Done()
			cfg, err := config.LoadDefaultConfig(ctx, SelectionLoadOptions(s)...)
			if err != nil {
				errChan <- err
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

	// Collect errors, but proceed if at least some succeeded
	if len(accountIDs) > 0 {
		appconfig.Global().SetAccountIDs(accountIDs)
	}

	// Return first error if any occurred during config load
	for err := range errChan {
		return err
	}

	return nil
}

func RefreshContextData(ctx context.Context) (region string, accountIDs map[string]string, err error) {
	selections := appconfig.Global().Selections()
	if len(selections) == 0 {
		selections = []appconfig.ProfileSelection{appconfig.SDKDefault()}
	}

	if !appconfig.Global().IsMultiRegion() {
		sel := selections[0]
		cfg, loadErr := config.LoadDefaultConfig(ctx, SelectionLoadOptions(sel)...)
		if loadErr == nil && cfg.Region != "" {
			region = cfg.Region
		}
	}

	var wg sync.WaitGroup
	accountIDs = make(map[string]string)
	var mu sync.Mutex
	errChan := make(chan error, len(selections))

	for _, sel := range selections {
		wg.Add(1)
		go func(s appconfig.ProfileSelection) {
			defer wg.Done()
			cfg, loadErr := config.LoadDefaultConfig(ctx, SelectionLoadOptions(s)...)
			if loadErr != nil {
				errChan <- loadErr
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

	for e := range errChan {
		err = e
		break
	}

	return region, accountIDs, err
}
