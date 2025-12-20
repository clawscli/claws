package config

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// ProfileLoadOptions returns config load options based on the given profile.
// This centralizes the logic for handling different profile modes:
//   - "" (empty): SDK default behavior (respects AWS_PROFILE env, falls back to default)
//   - UseEnvironmentCredentials: ignore ~/.aws files, use IMDS/environment only
//   - any other value: explicitly use that profile from ~/.aws files
func ProfileLoadOptions(profile string) []func(*config.LoadOptions) error {
	if profile == UseEnvironmentCredentials {
		return []func(*config.LoadOptions) error{
			config.WithSharedConfigFiles([]string{}),
			config.WithSharedCredentialsFiles([]string{}),
		}
	}
	if profile != "" {
		return []func(*config.LoadOptions) error{
			config.WithSharedConfigProfile(profile),
		}
	}
	return nil
}

// DemoAccountID is the masked account ID shown in demo mode
const DemoAccountID = "123456789012"

// UseEnvironmentCredentials is a special value to ignore ~/.aws config and use environment credentials
// (instance profile, ECS task role, Lambda execution role, environment variables, etc.)
const UseEnvironmentCredentials = "__environment__"

// EnvironmentCredentialsDisplayName is the display name for the environment credentials option
const EnvironmentCredentialsDisplayName = "(Environment)"

// Config holds global application configuration
type Config struct {
	mu        sync.RWMutex
	region    string
	profile   string
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

// Profile returns the current AWS profile
func (c *Config) Profile() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.profile
}

// SetProfile sets the current AWS profile
func (c *Config) SetProfile(profile string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.profile = profile
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
	profile := c.profile
	c.mu.RUnlock()

	opts := []func(*config.LoadOptions) error{
		config.WithEC2IMDSRegion(),
	}
	opts = append(opts, ProfileLoadOptions(profile)...)

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return err
	}

	c.mu.Lock()
	if c.region == "" {
		c.region = cfg.Region
	}
	c.mu.Unlock()

	// Get account ID from STS
	stsClient := sts.NewFromConfig(cfg)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err == nil && identity.Account != nil {
		c.mu.Lock()
		c.accountID = *identity.Account
		c.mu.Unlock()
	}

	return nil
}

// RefreshAccountID re-fetches the account ID for the current profile
func (c *Config) RefreshAccountID(ctx context.Context) error {
	c.mu.RLock()
	profile := c.profile
	c.mu.RUnlock()

	opts := []func(*config.LoadOptions) error{
		config.WithEC2IMDSRegion(),
	}
	opts = append(opts, ProfileLoadOptions(profile)...)

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return err
	}

	stsClient := sts.NewFromConfig(cfg)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		c.mu.Lock()
		c.accountID = ""
		c.mu.Unlock()
		return err
	}

	if identity.Account != nil {
		c.mu.Lock()
		c.accountID = *identity.Account
		c.mu.Unlock()
	}

	return nil
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

// FetchAvailableRegions fetches available regions from AWS
func FetchAvailableRegions(ctx context.Context) ([]string, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithEC2IMDSRegion(),
	)
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
