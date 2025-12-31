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
	Name  string
	IsSSO bool
}

// LoadProfiles parses ~/.aws/config and ~/.aws/credentials files
// and returns a sorted list of profile information.
// Respects AWS_CONFIG_FILE and AWS_SHARED_CREDENTIALS_FILE environment variables.
func LoadProfiles() ([]ProfileInfo, error) {
	type profileData struct {
		name  string
		isSSO bool
	}
	profileMap := make(map[string]*profileData)

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

			isSSO := section.Key("sso_start_url").String() != "" ||
				section.Key("sso_session").String() != ""

			profileMap[profileName] = &profileData{name: profileName, isSSO: isSSO}
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
			if _, exists := profileMap[name]; !exists {
				profileMap[name] = &profileData{name: name, isSSO: false}
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
		data := profileMap[name]
		profiles = append(profiles, ProfileInfo{
			Name:  name,
			IsSSO: data.isSSO,
		})
	}
	return profiles, nil
}
