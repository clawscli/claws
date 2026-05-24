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

func TestLoadProfilesExpandsSSOSessionSettings(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	credentialsPath := filepath.Join(dir, "credentials")

	configData := []byte(`[profile dev]
sso_session = dev-session
sso_account_id = 123456789012
sso_role_name = AdministratorAccess
region = ap-northeast-1

[sso-session dev-session]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
sso_registration_scopes = sso:account:access
`)
	if err := os.WriteFile(configPath, configData, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(credentialsPath, nil, 0o600); err != nil {
		t.Fatalf("write credentials: %v", err)
	}

	t.Setenv("AWS_CONFIG_FILE", configPath)
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", credentialsPath)

	profiles, err := LoadProfiles()
	if err != nil {
		t.Fatalf("LoadProfiles() returned error: %v", err)
	}
	if len(profiles) != 1 {
		t.Fatalf("profiles length = %d, want 1: %+v", len(profiles), profiles)
	}
	profile := profiles[0]
	if !profile.IsSSO {
		t.Fatal("profile should be detected as SSO")
	}
	if profile.SSOStartURL != "https://example.awsapps.com/start" {
		t.Errorf("SSOStartURL = %q", profile.SSOStartURL)
	}
	if profile.SSORegion != "us-east-1" {
		t.Errorf("SSORegion = %q", profile.SSORegion)
	}
	if profile.SSOScopes != "sso:account:access" {
		t.Errorf("SSOScopes = %q", profile.SSOScopes)
	}
	if profile.SSOAccountID != "123456789012" {
		t.Errorf("SSOAccountID = %q", profile.SSOAccountID)
	}
	if profile.SSORoleName != "AdministratorAccess" {
		t.Errorf("SSORoleName = %q", profile.SSORoleName)
	}
}
