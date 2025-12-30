package profile

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/ini.v1"

	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/log"
)

// ProfileData contains parsed profile information from ~/.aws files
type ProfileData struct {
	ID              string // config.SDKDefault().ID(), config.EnvOnly().ID(), or profile name
	Name            string
	Region          string
	Output          string
	RoleArn         string
	SourceProfile   string
	ExternalID      string
	MFASerial       string
	RoleSessionName string
	DurationSeconds string
	// SSO settings
	SSOStartURL  string
	SSORegion    string
	SSOAccountID string
	SSORoleName  string
	SSOSession   string
	// Credentials (masked in display)
	HasCredentials bool
	AccessKeyID    string // Will be masked
	// Source file info
	InConfig      bool
	InCredentials bool
	// Current status
	IsCurrent bool
}

// ProfileDAO provides data access for local AWS profiles
type ProfileDAO struct {
	dao.BaseDAO
}

// NewProfileDAO creates a new ProfileDAO
func NewProfileDAO(_ context.Context) (dao.DAO, error) {
	return &ProfileDAO{
		BaseDAO: dao.NewBaseDAO("local", "profile"),
	}, nil
}

// Supports returns whether this DAO supports the given operation.
// ProfileDAO is read-only (no Delete support).
func (d *ProfileDAO) Supports(op dao.Operation) bool {
	switch op {
	case dao.OpList, dao.OpGet:
		return true
	default:
		return false
	}
}

func (d *ProfileDAO) List(_ context.Context) ([]dao.Resource, error) {
	profiles, err := loadProfiles()
	if err != nil {
		return nil, err
	}

	sel := config.Global().Selection()

	// Capacity: 2 special entries + profiles
	resources := make([]dao.Resource, 0, len(profiles)+2)

	// 1. SDK Default - lets AWS SDK use standard credential chain
	resources = append(resources, NewProfileResource(&ProfileData{
		ID:        config.SDKDefault().ID(),
		Name:      config.SDKDefault().DisplayName(),
		IsCurrent: sel.IsSDKDefault(),
	}))

	// 2. Env/IMDS Only - ignores ~/.aws config, uses IMDS/environment
	resources = append(resources, NewProfileResource(&ProfileData{
		ID:        config.EnvOnly().ID(),
		Name:      config.EnvOnly().DisplayName(),
		IsCurrent: sel.IsEnvOnly(),
	}))

	// 3. Named profiles from ~/.aws (sorted for stable ordering)
	names := make([]string, 0, len(profiles))
	for name := range profiles {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		data := profiles[name]
		data.ID = name // Explicit ID for named profiles
		data.Name = name
		data.IsCurrent = sel.IsNamedProfile() && sel.ProfileName == name
		resources = append(resources, NewProfileResource(data))
	}

	return resources, nil
}

func (d *ProfileDAO) Get(_ context.Context, id string) (dao.Resource, error) {
	sel := config.Global().Selection()

	// Handle special selection modes
	switch id {
	case config.SDKDefault().DisplayName(), config.SDKDefault().ID():
		return NewProfileResource(&ProfileData{
			ID:        config.SDKDefault().ID(),
			Name:      config.SDKDefault().DisplayName(),
			IsCurrent: sel.IsSDKDefault(),
		}), nil
	case config.EnvOnly().DisplayName(), config.EnvOnly().ID():
		return NewProfileResource(&ProfileData{
			ID:        config.EnvOnly().ID(),
			Name:      config.EnvOnly().DisplayName(),
			IsCurrent: sel.IsEnvOnly(),
		}), nil
	}

	profiles, err := loadProfiles()
	if err != nil {
		return nil, err
	}

	data, ok := profiles[id]
	if !ok {
		return nil, fmt.Errorf("profile %q not found", id)
	}

	data.ID = id // Explicit ID for named profiles
	data.Name = id
	data.IsCurrent = sel.IsNamedProfile() && sel.ProfileName == id
	return NewProfileResource(data), nil
}

// Delete is not supported for local profiles.
func (d *ProfileDAO) Delete(_ context.Context, _ string) error {
	return fmt.Errorf("delete not supported for local profiles")
}

// ProfileResource represents a local AWS profile
type ProfileResource struct {
	dao.BaseResource
	Data *ProfileData
}

// NewProfileResource creates a new ProfileResource
func NewProfileResource(data *ProfileData) *ProfileResource {
	id := data.ID
	if id == "" {
		id = data.Name // Fallback for named profiles
	}
	return &ProfileResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: data.Name,
		},
		Data: data,
	}
}

// parseConfigSectionName extracts the profile name from a config file section.
// Returns the profile name and true if valid, or empty string and false if the section should be skipped.
func parseConfigSectionName(sectionName string) (string, bool) {
	if sectionName == "DEFAULT" {
		return "", false
	}
	// In config file, profiles are "profile xxx" except for "default"
	if strings.HasPrefix(sectionName, "profile ") {
		return strings.TrimPrefix(sectionName, "profile "), true
	}
	if sectionName == "default" {
		return "default", true
	}
	// Skip non-profile sections like sso-session
	return "", false
}

// getConfigPath returns the path to the AWS config file.
// Respects AWS_CONFIG_FILE environment variable, falls back to ~/.aws/config.
func getConfigPath() (string, error) {
	if path := os.Getenv("AWS_CONFIG_FILE"); path != "" {
		return path, nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(homeDir, ".aws", "config"), nil
}

// getCredentialsPath returns the path to the AWS credentials file.
// Respects AWS_SHARED_CREDENTIALS_FILE environment variable, falls back to ~/.aws/credentials.
func getCredentialsPath() (string, error) {
	if path := os.Getenv("AWS_SHARED_CREDENTIALS_FILE"); path != "" {
		return path, nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(homeDir, ".aws", "credentials"), nil
}

// loadProfiles parses ~/.aws/config and ~/.aws/credentials files.
// Respects AWS_CONFIG_FILE and AWS_SHARED_CREDENTIALS_FILE environment variables.
func loadProfiles() (map[string]*ProfileData, error) {
	profiles := make(map[string]*ProfileData)

	// Always include default
	profiles["default"] = &ProfileData{Name: "default"}

	// Parse config file
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}
	cfg, err := ini.Load(configPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Debug("failed to parse aws config", "path", configPath, "error", err)
	}
	if err == nil {
		for _, section := range cfg.Sections() {
			profileName, ok := parseConfigSectionName(section.Name())
			if !ok {
				continue
			}

			data, ok := profiles[profileName]
			if !ok {
				data = &ProfileData{Name: profileName}
				profiles[profileName] = data
			}

			data.InConfig = true
			data.Region = section.Key("region").String()
			data.Output = section.Key("output").String()
			data.RoleArn = section.Key("role_arn").String()
			data.SourceProfile = section.Key("source_profile").String()
			data.ExternalID = section.Key("external_id").String()
			data.MFASerial = section.Key("mfa_serial").String()
			data.RoleSessionName = section.Key("role_session_name").String()
			data.DurationSeconds = section.Key("duration_seconds").String()
			data.SSOStartURL = section.Key("sso_start_url").String()
			data.SSORegion = section.Key("sso_region").String()
			data.SSOAccountID = section.Key("sso_account_id").String()
			data.SSORoleName = section.Key("sso_role_name").String()
			data.SSOSession = section.Key("sso_session").String()
		}
	}

	// Parse credentials file
	credPath, err := getCredentialsPath()
	if err != nil {
		return nil, err
	}
	creds, err := ini.Load(credPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Debug("failed to parse aws credentials", "path", credPath, "error", err)
	}
	if err == nil {
		for _, section := range creds.Sections() {
			name := section.Name()
			if name == "DEFAULT" {
				continue
			}

			data, ok := profiles[name]
			if !ok {
				data = &ProfileData{Name: name}
				profiles[name] = data
			}

			data.InCredentials = true
			accessKey := section.Key("aws_access_key_id").String()
			if accessKey != "" {
				data.HasCredentials = true
				data.AccessKeyID = accessKey
			}
		}
	}

	return profiles, nil
}
