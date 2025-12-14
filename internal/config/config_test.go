package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfig_RegionGetSet(t *testing.T) {
	cfg := &Config{}

	// Initial value should be empty
	if cfg.Region() != "" {
		t.Errorf("Region() = %q, want empty string", cfg.Region())
	}

	// Set and get
	cfg.SetRegion("us-east-1")
	if cfg.Region() != "us-east-1" {
		t.Errorf("Region() = %q, want %q", cfg.Region(), "us-east-1")
	}

	// Update
	cfg.SetRegion("eu-west-1")
	if cfg.Region() != "eu-west-1" {
		t.Errorf("Region() = %q, want %q", cfg.Region(), "eu-west-1")
	}
}

func TestConfig_ProfileGetSet(t *testing.T) {
	cfg := &Config{}

	// Initial value should be empty
	if cfg.Profile() != "" {
		t.Errorf("Profile() = %q, want empty string", cfg.Profile())
	}

	// Set and get
	cfg.SetProfile("production")
	if cfg.Profile() != "production" {
		t.Errorf("Profile() = %q, want %q", cfg.Profile(), "production")
	}
}

func TestConfig_AccountID(t *testing.T) {
	cfg := &Config{accountID: "123456789012"}

	if cfg.AccountID() != "123456789012" {
		t.Errorf("AccountID() = %q, want %q", cfg.AccountID(), "123456789012")
	}
}

func TestConfig_ReadOnlyGetSet(t *testing.T) {
	cfg := &Config{}

	// Initial value should be false
	if cfg.ReadOnly() {
		t.Error("ReadOnly() = true, want false")
	}

	// Set to true
	cfg.SetReadOnly(true)
	if !cfg.ReadOnly() {
		t.Error("ReadOnly() = false, want true")
	}

	// Set back to false
	cfg.SetReadOnly(false)
	if cfg.ReadOnly() {
		t.Error("ReadOnly() = true, want false")
	}
}

func TestConfig_Warnings(t *testing.T) {
	cfg := &Config{}

	// Initial should be empty
	if len(cfg.Warnings()) != 0 {
		t.Errorf("Warnings() = %v, want empty slice", cfg.Warnings())
	}

	// Add warnings
	cfg.addWarning("warning 1")
	cfg.addWarning("warning 2")

	warnings := cfg.Warnings()
	if len(warnings) != 2 {
		t.Errorf("Warnings() has %d items, want 2", len(warnings))
	}
	if warnings[0] != "warning 1" {
		t.Errorf("Warnings()[0] = %q, want %q", warnings[0], "warning 1")
	}
	if warnings[1] != "warning 2" {
		t.Errorf("Warnings()[1] = %q, want %q", warnings[1], "warning 2")
	}
}

func TestGlobal(t *testing.T) {
	// Should return non-nil config
	cfg := Global()
	if cfg == nil {
		t.Fatal("Global() returned nil")
	}

	// Should return same instance on subsequent calls
	cfg2 := Global()
	if cfg != cfg2 {
		t.Error("Global() should return same instance")
	}
}

func TestCommonRegions(t *testing.T) {
	if len(CommonRegions) == 0 {
		t.Error("CommonRegions should not be empty")
	}

	// Check some expected regions are present
	expectedRegions := []string{"us-east-1", "us-west-2", "eu-west-1", "ap-northeast-1"}
	for _, expected := range expectedRegions {
		found := false
		for _, region := range CommonRegions {
			if region == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("CommonRegions should contain %q", expected)
		}
	}
}

func TestParseProfilesFromFile_Credentials(t *testing.T) {
	// Create a temporary credentials file
	tempDir := t.TempDir()
	credPath := filepath.Join(tempDir, "credentials")

	content := `[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

[dev]
aws_access_key_id = AKIAI44QH8DHBEXAMPLE
aws_secret_access_key = je7MtGbClwBF/2Zp9Utk/h3yCo8nvbEXAMPLEKEY

[prod]
aws_access_key_id = AKIAJ7KCZUAFVBEXAMPLE
aws_secret_access_key = aJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
`
	if err := os.WriteFile(credPath, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	profiles := make(map[string]struct{})
	parseProfilesFromFile(credPath, profiles, false)

	expected := []string{"default", "dev", "prod"}
	for _, name := range expected {
		if _, ok := profiles[name]; !ok {
			t.Errorf("Expected profile %q not found", name)
		}
	}
}

func TestParseProfilesFromFile_Config(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")

	content := `[default]
region = us-east-1
output = json

[profile staging]
region = us-west-2

[profile production]
region = eu-west-1
role_arn = arn:aws:iam::123456789012:role/Admin
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	profiles := make(map[string]struct{})
	parseProfilesFromFile(configPath, profiles, true)

	expected := []string{"default", "staging", "production"}
	for _, name := range expected {
		if _, ok := profiles[name]; !ok {
			t.Errorf("Expected profile %q not found", name)
		}
	}
}

func TestParseProfilesFromFile_NonExistent(t *testing.T) {
	profiles := make(map[string]struct{})
	parseProfilesFromFile("/nonexistent/path/file", profiles, false)

	// Should not panic and profiles should be empty
	if len(profiles) != 0 {
		t.Errorf("profiles should be empty for non-existent file")
	}
}

func TestFetchAvailableProfiles(t *testing.T) {
	profiles := FetchAvailableProfiles()

	// Should at least return "default"
	if len(profiles) == 0 {
		t.Error("FetchAvailableProfiles() should return at least one profile")
	}

	// "default" should be first
	if profiles[0] != "default" {
		t.Errorf("profiles[0] = %q, want %q", profiles[0], "default")
	}
}
