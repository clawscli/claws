package config

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// DemoAccountID is the masked account ID shown in demo mode
const DemoAccountID = "123456789012"

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

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithEC2IMDSRegion(),
	)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.region = cfg.Region
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

// FetchAvailableProfiles returns available AWS profiles from credentials and config files
func FetchAvailableProfiles() []string {
	profileSet := make(map[string]struct{})

	// Add "default" profile always
	profileSet["default"] = struct{}{}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return []string{"default"}
	}

	// Parse ~/.aws/credentials
	credentialsPath := filepath.Join(homeDir, ".aws", "credentials")
	parseProfilesFromFile(credentialsPath, profileSet, false)

	// Parse ~/.aws/config
	configPath := filepath.Join(homeDir, ".aws", "config")
	parseProfilesFromFile(configPath, profileSet, true)

	// Convert to sorted slice
	profiles := make([]string, 0, len(profileSet))
	for p := range profileSet {
		profiles = append(profiles, p)
	}
	slices.Sort(profiles)

	// Move "default" to the front
	for i, p := range profiles {
		if p == "default" && i > 0 {
			profiles = append([]string{"default"}, append(profiles[:i], profiles[i+1:]...)...)
			break
		}
	}

	return profiles
}

// parseProfilesFromFile parses profile names from AWS credentials or config file
func parseProfilesFromFile(path string, profiles map[string]struct{}, isConfig bool) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := line[1 : len(line)-1]
			// In config file, profiles are prefixed with "profile "
			if isConfig {
				if strings.HasPrefix(section, "profile ") {
					profiles[strings.TrimPrefix(section, "profile ")] = struct{}{}
				} else if section == "default" {
					profiles["default"] = struct{}{}
				}
			} else {
				// In credentials file, section name is the profile name
				profiles[section] = struct{}{}
			}
		}
	}
}
