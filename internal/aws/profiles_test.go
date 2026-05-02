package aws

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProfilesSkipsInvalidProfileNames(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	credentialsPath := filepath.Join(dir, "credentials")

	configData := []byte("[profile valid-profile]\nregion = us-east-1\n[profile bad; echo injected]\nregion = us-west-2\n")
	if err := os.WriteFile(configPath, configData, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	credentialsData := []byte("[valid-profile]\naws_access_key_id = AKIA1234567890ABCD\n[bad; echo injected]\naws_access_key_id = AKIA1234567890EFGH\n")
	if err := os.WriteFile(credentialsPath, credentialsData, 0o600); err != nil {
		t.Fatalf("write credentials: %v", err)
	}

	t.Setenv("AWS_CONFIG_FILE", configPath)
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", credentialsPath)

	profiles, err := LoadProfiles()
	if err != nil {
		t.Fatalf("LoadProfiles() returned error: %v", err)
	}

	for _, profile := range profiles {
		if profile.Name == "bad; echo injected" {
			t.Fatalf("LoadProfiles() returned invalid profile name: %+v", profile)
		}
	}
	if len(profiles) != 1 || profiles[0].Name != "valid-profile" {
		t.Fatalf("profiles = %+v, want only valid-profile", profiles)
	}
}
