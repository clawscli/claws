package config

import (
	"maps"
	"os"
	"regexp"
	"sync"
)

// Validation patterns
var (
	// regionPattern matches AWS region format: xx-xxxx-N (e.g., us-east-1, ap-northeast-1)
	regionPattern = regexp.MustCompile(`^[a-z]{2}(-[a-z]+-\d|-(gov|iso|isob)-[a-z]+-\d)$`)

	// accountIDPattern matches 12-digit AWS account IDs
	accountIDPattern = regexp.MustCompile(`^\d{12}$`)

	// profileNamePattern matches valid AWS profile names (alphanumeric, hyphen, underscore, period)
	profileNamePattern = regexp.MustCompile(`^[\w\-.]+$`)
)

type ValidationError struct {
	Field   string
	Value   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// IsValidRegion checks if the region name follows AWS region format.
// Valid examples: us-east-1, eu-west-2, ap-northeast-1, us-gov-west-1
func IsValidRegion(region string) bool {
	if region == "" || len(region) > 25 {
		return false
	}
	return regionPattern.MatchString(region)
}

// IsValidAccountID checks if the account ID is a 12-digit number.
// Used internally to validate STS-fetched account IDs, not for user input.
func IsValidAccountID(accountID string) bool {
	if accountID == "" {
		return true // Empty is allowed (not yet fetched)
	}
	return accountIDPattern.MatchString(accountID)
}

// IsValidProfileName checks if the profile name contains only valid characters.
// Valid characters: alphanumeric, hyphen, underscore, period
func IsValidProfileName(name string) bool {
	if name == "" || len(name) > 128 {
		return false
	}
	return profileNamePattern.MatchString(name)
}

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

type Config struct {
	mu         sync.RWMutex
	regions    []string
	selections []ProfileSelection
	accountIDs map[string]string
	warnings   []string
	readOnly   bool
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

func (c *Config) Region() string {
	return withRLock(&c.mu, func() string {
		if len(c.regions) == 0 {
			return ""
		}
		return c.regions[0]
	})
}

func (c *Config) Regions() []string {
	return withRLock(&c.mu, func() []string {
		return append([]string(nil), c.regions...)
	})
}

func (c *Config) SetRegion(region string) {
	doWithLock(&c.mu, func() { c.regions = []string{region} })
}

func (c *Config) SetRegions(regions []string) {
	doWithLock(&c.mu, func() { c.regions = append([]string(nil), regions...) })
}

func (c *Config) IsMultiRegion() bool {
	return withRLock(&c.mu, func() bool { return len(c.regions) > 1 })
}

func (c *Config) Selection() ProfileSelection {
	return withRLock(&c.mu, func() ProfileSelection {
		if len(c.selections) == 0 {
			return SDKDefault()
		}
		return c.selections[0]
	})
}

func (c *Config) Selections() []ProfileSelection {
	return withRLock(&c.mu, func() []ProfileSelection {
		if len(c.selections) == 0 {
			return []ProfileSelection{SDKDefault()}
		}
		return append([]ProfileSelection(nil), c.selections...)
	})
}

func (c *Config) SetSelection(sel ProfileSelection) {
	doWithLock(&c.mu, func() { c.selections = []ProfileSelection{sel} })
}

func (c *Config) SetSelections(sels []ProfileSelection) {
	doWithLock(&c.mu, func() { c.selections = append([]ProfileSelection(nil), sels...) })
}

func (c *Config) IsMultiProfile() bool {
	return withRLock(&c.mu, func() bool { return len(c.selections) > 1 })
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

func (c *Config) AccountID() string {
	return withRLock(&c.mu, func() string {
		key := ProfileIDSDKDefault
		if len(c.selections) > 0 {
			key = c.selections[0].ID()
		}
		return c.accountIDs[key]
	})
}

func (c *Config) AccountIDs() map[string]string {
	return withRLock(&c.mu, func() map[string]string {
		result := make(map[string]string, len(c.accountIDs))
		maps.Copy(result, c.accountIDs)
		return result
	})
}

func (c *Config) SetAccountID(id string) {
	doWithLock(&c.mu, func() {
		if c.accountIDs == nil {
			c.accountIDs = make(map[string]string)
		}
		key := ProfileIDSDKDefault
		if len(c.selections) > 0 {
			key = c.selections[0].ID()
		}
		c.accountIDs[key] = id
	})
}

func (c *Config) SetAccountIDs(ids map[string]string) {
	doWithLock(&c.mu, func() {
		c.accountIDs = make(map[string]string, len(ids))
		maps.Copy(c.accountIDs, ids)
	})
}

func (c *Config) SetAccountIDForProfile(profileID, accountID string) {
	doWithLock(&c.mu, func() {
		if c.accountIDs == nil {
			c.accountIDs = make(map[string]string)
		}
		c.accountIDs[profileID] = accountID
	})
}

func (c *Config) GetAccountIDForProfile(profileID string) string {
	return withRLock(&c.mu, func() string {
		if c.accountIDs == nil {
			return ""
		}
		return c.accountIDs[profileID]
	})
}

func (c *Config) Warnings() []string {
	return withRLock(&c.mu, func() []string { return c.warnings })
}

func (c *Config) ReadOnly() bool {
	return withRLock(&c.mu, func() bool { return c.readOnly })
}

func (c *Config) SetReadOnly(readOnly bool) {
	doWithLock(&c.mu, func() { c.readOnly = readOnly })
}

func (c *Config) AddWarning(msg string) {
	doWithLock(&c.mu, func() { c.warnings = append(c.warnings, msg) })
}
