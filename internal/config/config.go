package config

import (
	"os"
	"sync"
)

// Profile resource ID constants for stable identification
const (
	// ProfileIDSDKDefault is the resource ID for SDK default credential mode
	ProfileIDSDKDefault = "__sdk_default__"
	// ProfileIDEnvOnly is the resource ID for env/IMDS-only credential mode
	ProfileIDEnvOnly = "__env_only__"
)

// ProfileSelectionFromID returns ProfileSelection for a resource ID.
func ProfileSelectionFromID(id string) ProfileSelection {
	switch id {
	case ProfileIDSDKDefault:
		return SDKDefault()
	case ProfileIDEnvOnly:
		return EnvOnly()
	default:
		return NamedProfile(id)
	}
}

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

// DisplayName returns the display name for this selection.
// For SDKDefault mode, includes AWS_PROFILE value if set.
func (s ProfileSelection) DisplayName() string {
	switch s.Mode {
	case ModeSDKDefault:
		if p := os.Getenv("AWS_PROFILE"); p != "" {
			return "SDK Default (AWS_PROFILE=" + p + ")"
		}
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

// ID returns the stable resource ID for this selection.
// This is the inverse of ProfileSelectionFromID.
func (s ProfileSelection) ID() string {
	switch s.Mode {
	case ModeSDKDefault:
		return ProfileIDSDKDefault
	case ModeEnvOnly:
		return ProfileIDEnvOnly
	case ModeNamedProfile:
		return s.ProfileName
	default:
		return ""
	}
}

// Config holds global application configuration
type Config struct {
	mu        sync.RWMutex
	region    string
	selection ProfileSelection
	accountID string
	warnings  []string
	readOnly  bool
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

// AccountID returns the current AWS account ID
func (c *Config) AccountID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.accountID
}

// SetAccountID sets the AWS account ID
func (c *Config) SetAccountID(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.accountID = id
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

// AddWarning adds a warning message
func (c *Config) AddWarning(msg string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.warnings = append(c.warnings, msg)
}
