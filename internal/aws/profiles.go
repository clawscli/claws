package aws

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/ini.v1"

	"github.com/clawscli/claws/internal/log"
)

// ProfileInfo contains basic profile metadata from ~/.aws files.
type ProfileInfo struct {
	Name           string
	IsSSO          bool
	Region         string
	ProfileType    string // SSO, AssumeRole, Static, Default
	RoleArn        string
	SourceProfile  string
	SSOSession     string
	SSOStartURL    string
	SSORegion      string
	SSOAccountID   string
	SSORoleName    string
	HasCredentials bool
	AccessKeyID    string // masked
}

// LoadProfiles parses ~/.aws/config and ~/.aws/credentials files
// and returns a sorted list of profile information.
// Respects AWS_CONFIG_FILE and AWS_SHARED_CREDENTIALS_FILE environment variables.
func LoadProfiles() ([]ProfileInfo, error) {
	profileMap := make(map[string]*ProfileInfo)

	configPath := os.Getenv("AWS_CONFIG_FILE")
	if configPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		configPath = filepath.Join(homeDir, ".aws", "config")
	}

	cfg, err := ini.Load(configPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Debug("failed to parse aws config", "path", configPath, "error", err)
	}
	if err == nil {
		for _, section := range cfg.Sections() {
			name := section.Name()
			if name == "DEFAULT" {
				continue
			}

			var profileName string
			if after, found := strings.CutPrefix(name, "profile "); found {
				profileName = after
			} else if name == "default" {
				profileName = "default"
			} else {
				continue
			}

			ssoStartURL := section.Key("sso_start_url").String()
			ssoSession := section.Key("sso_session").String()
			roleArn := section.Key("role_arn").String()

			info := &ProfileInfo{
				Name:          profileName,
				IsSSO:         ssoStartURL != "" || ssoSession != "",
				Region:        section.Key("region").String(),
				RoleArn:       roleArn,
				SourceProfile: section.Key("source_profile").String(),
				SSOSession:    ssoSession,
				SSOStartURL:   ssoStartURL,
				SSORegion:     section.Key("sso_region").String(),
				SSOAccountID:  section.Key("sso_account_id").String(),
				SSORoleName:   section.Key("sso_role_name").String(),
			}
			info.ProfileType = determineProfileType(info)
			profileMap[profileName] = info
		}
	}

	credPath := os.Getenv("AWS_SHARED_CREDENTIALS_FILE")
	if credPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		credPath = filepath.Join(homeDir, ".aws", "credentials")
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

			accessKeyID := section.Key("aws_access_key_id").String()
			hasCredentials := accessKeyID != ""

			if info, exists := profileMap[name]; exists {
				info.HasCredentials = hasCredentials
				info.AccessKeyID = maskAccessKey(accessKeyID)
				info.ProfileType = determineProfileType(info)
			} else {
				info := &ProfileInfo{
					Name:           name,
					HasCredentials: hasCredentials,
					AccessKeyID:    maskAccessKey(accessKeyID),
				}
				info.ProfileType = determineProfileType(info)
				profileMap[name] = info
			}
		}
	}

	names := make([]string, 0, len(profileMap))
	for name := range profileMap {
		names = append(names, name)
	}
	sort.Strings(names)

	profiles := make([]ProfileInfo, 0, len(names))
	for _, name := range names {
		profiles = append(profiles, *profileMap[name])
	}
	return profiles, nil
}

func determineProfileType(info *ProfileInfo) string {
	if info.IsSSO {
		return "SSO"
	}
	if info.RoleArn != "" {
		return "AssumeRole"
	}
	if info.HasCredentials {
		return "Static"
	}
	return "Default"
}

func maskAccessKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}
