package config

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// SelectionLoadOptions returns config load options based on the given ProfileSelection.
// This centralizes the logic for handling different credential modes:
//   - ModeSDKDefault: no extra options, let SDK use standard chain
//   - ModeEnvOnly: ignore ~/.aws files, use IMDS/environment only
//   - ModeNamedProfile: explicitly use that profile from ~/.aws files
func SelectionLoadOptions(sel ProfileSelection) []func(*config.LoadOptions) error {
	opts := []func(*config.LoadOptions) error{
		config.WithEC2IMDSRegion(),
	}
	switch sel.Mode {
	case ModeEnvOnly:
		opts = append(opts,
			config.WithSharedConfigFiles([]string{}),
			config.WithSharedCredentialsFiles([]string{}),
		)
	case ModeNamedProfile:
		opts = append(opts, config.WithSharedConfigProfile(sel.ProfileName))
	case ModeSDKDefault:
		// No extra options - let SDK use standard chain
	}
	return opts
}

// DemoAccountID is the masked account ID shown in demo mode
const DemoAccountID = "123456789012"

// CredentialMode represents how AWS credentials are resolved
type CredentialMode int

const (
	// ModeSDKDefault lets AWS SDK decide via standard credential chain.
	// Preserves existing AWS_PROFILE environment variable.
	ModeSDKDefault CredentialMode = iota

	// ModeNamedProfile explicitly uses a named profile from ~/.aws config.
	ModeNamedProfile

	// ModeEnvOnly ignores ~/.aws files, uses IMDS/environment/ECS/Lambda creds only.
	ModeEnvOnly
)

// String returns a display string for the credential mode
func (m CredentialMode) String() string {
	switch m {
	case ModeSDKDefault:
		return "SDK Default"
	case ModeNamedProfile:
		return "" // Profile name is shown separately
	case ModeEnvOnly:
		return "Env/IMDS Only"
	default:
		return "Unknown"
	}
}

// ProfileSelection represents the selected credential mode and optional profile name
type ProfileSelection struct {
	Mode        CredentialMode
	ProfileName string // Only used when Mode == ModeNamedProfile
}

// SDKDefault returns a selection for SDK default credential chain
func SDKDefault() ProfileSelection {
	return ProfileSelection{Mode: ModeSDKDefault}
}

// EnvOnly returns a selection for environment/IMDS credentials only
func EnvOnly() ProfileSelection {
	return ProfileSelection{Mode: ModeEnvOnly}
}

// NamedProfile returns a selection for a specific named profile
func NamedProfile(name string) ProfileSelection {
	return ProfileSelection{Mode: ModeNamedProfile, ProfileName: name}
}

// DisplayName returns the display name for this selection
func (s ProfileSelection) DisplayName() string {
	switch s.Mode {
	case ModeSDKDefault:
		return "SDK Default"
	case ModeEnvOnly:
		return "Env/IMDS Only"
	case ModeNamedProfile:
		return s.ProfileName
	default:
		return "Unknown"
	}
}

// IsSDKDefault returns true if this is SDK default mode
func (s ProfileSelection) IsSDKDefault() bool {
	return s.Mode == ModeSDKDefault
}

// IsEnvOnly returns true if this is env-only mode
func (s ProfileSelection) IsEnvOnly() bool {
	return s.Mode == ModeEnvOnly
}

// IsNamedProfile returns true if this is a named profile
func (s ProfileSelection) IsNamedProfile() bool {
	return s.Mode == ModeNamedProfile
}

// fetchAccountID fetches the AWS account ID using STS GetCallerIdentity.
// Returns empty string on error.
func fetchAccountID(ctx context.Context, cfg aws.Config) string {
	stsClient := sts.NewFromConfig(cfg)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil || identity.Account == nil {
		return ""
	}
	return *identity.Account
}

// Config holds global application configuration
type Config struct {
	mu        sync.RWMutex
	region    string
	selection ProfileSelection
	accountID string
	warnings  []string
	readOnly  bool
	demoMode  bool
}

var (
	global   *Config
	initOnce sync.Once
)

// Global returns the global config instance
func Global() *Config {
	initOnce.Do(func() {
		global = &Config{}
	})
	return global
}

// Region returns the current region
func (c *Config) Region() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.region
}

// SetRegion sets the current region
func (c *Config) SetRegion(region string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.region = region
}

// Selection returns the current profile selection
func (c *Config) Selection() ProfileSelection {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.selection
}

// SetSelection sets the profile selection
func (c *Config) SetSelection(sel ProfileSelection) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.selection = sel
}

// UseSDKDefault sets SDK default credential mode
func (c *Config) UseSDKDefault() {
	c.SetSelection(SDKDefault())
}

// UseEnvOnly sets environment-only credential mode
func (c *Config) UseEnvOnly() {
	c.SetSelection(EnvOnly())
}

// UseProfile sets a named profile
func (c *Config) UseProfile(name string) {
	c.SetSelection(NamedProfile(name))
}

// AccountID returns the current AWS account ID (masked in demo mode)
func (c *Config) AccountID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.demoMode {
		return DemoAccountID
	}
	return c.accountID
}

// SetDemoMode enables or disables demo mode
func (c *Config) SetDemoMode(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.demoMode = enabled
}

// DemoMode returns whether demo mode is enabled
func (c *Config) DemoMode() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.demoMode
}

// MaskAccountID masks an account ID if demo mode is enabled
func (c *Config) MaskAccountID(id string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.demoMode && id != "" {
		return DemoAccountID
	}
	return id
}

// Warnings returns any startup warnings
func (c *Config) Warnings() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.warnings
}

// ReadOnly returns whether the application is in read-only mode
func (c *Config) ReadOnly() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.readOnly
}

// SetReadOnly sets the read-only mode
func (c *Config) SetReadOnly(readOnly bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.readOnly = readOnly
}

// addWarning adds a warning message
func (c *Config) addWarning(msg string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.warnings = append(c.warnings, msg)
}

// Init initializes the config, detecting region and account ID from environment/IMDS
func (c *Config) Init(ctx context.Context) error {
	// Check external dependencies
	c.checkDependencies()

	c.mu.RLock()
	sel := c.selection
	c.mu.RUnlock()

	cfg, err := config.LoadDefaultConfig(ctx, SelectionLoadOptions(sel)...)
	if err != nil {
		return err
	}

	c.mu.Lock()
	if c.region == "" {
		c.region = cfg.Region
	}
	c.accountID = fetchAccountID(ctx, cfg)
	c.mu.Unlock()

	return nil
}

// RefreshForProfile re-fetches region and account ID for the current selection.
// Region is updated from the profile's default region if configured.
func (c *Config) RefreshForProfile(ctx context.Context) error {
	c.mu.RLock()
	sel := c.selection
	c.mu.RUnlock()

	cfg, err := config.LoadDefaultConfig(ctx, SelectionLoadOptions(sel)...)
	if err != nil {
		return err
	}

	c.mu.Lock()
	// Update region from profile's default (if set)
	if cfg.Region != "" {
		c.region = cfg.Region
	}
	c.accountID = fetchAccountID(ctx, cfg)
	c.mu.Unlock()

	return nil
}

// RefreshAccountID re-fetches the account ID for the current profile.
// Deprecated: Use RefreshForProfile instead to also update region.
func (c *Config) RefreshAccountID(ctx context.Context) error {
	return c.RefreshForProfile(ctx)
}

// checkDependencies checks for required external tools
func (c *Config) checkDependencies() {
	// Disabled: SSM plugin warning is too noisy for demo/general use
	// The action itself will fail gracefully if plugin is missing
}

// CommonRegions returns a list of common AWS regions
var CommonRegions = []string{
	"us-east-1",
	"us-east-2",
	"us-west-1",
	"us-west-2",
	"eu-west-1",
	"eu-west-2",
	"eu-central-1",
	"ap-northeast-1",
	"ap-northeast-2",
	"ap-southeast-1",
	"ap-southeast-2",
	"ap-south-1",
	"sa-east-1",
}

// FetchAvailableRegions fetches available regions from AWS using the current profile.
func FetchAvailableRegions(ctx context.Context) ([]string, error) {
	cfg, err := config.LoadDefaultConfig(ctx, SelectionLoadOptions(Global().Selection())...)
	if err != nil {
		return CommonRegions, nil // Fallback to common regions
	}

	client := ec2.NewFromConfig(cfg)
	output, err := client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{})
	if err != nil {
		return CommonRegions, nil // Fallback to common regions
	}

	regions := make([]string, 0, len(output.Regions))
	for _, r := range output.Regions {
		if r.RegionName != nil {
			regions = append(regions, *r.RegionName)
		}
	}
	return regions, nil
}
