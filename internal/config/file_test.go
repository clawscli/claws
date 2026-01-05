package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestDuration_MarshalUnmarshal(t *testing.T) {
	tests := []struct {
		name     string
		duration Duration
		want     string
	}{
		{"5s", Duration(5 * time.Second), "5s"},
		{"30s", Duration(30 * time.Second), "30s"},
		{"1m", Duration(1 * time.Minute), "1m0s"},
		{"zero", Duration(0), "0s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			data, err := yaml.Marshal(tt.duration)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}
			got := string(data)
			// yaml.Marshal adds newline
			if got != tt.want+"\n" {
				t.Errorf("Marshal = %q, want %q", got, tt.want+"\n")
			}

			// Unmarshal
			var d Duration
			if err := yaml.Unmarshal([]byte(tt.want), &d); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}
			if d != tt.duration {
				t.Errorf("Unmarshal = %v, want %v", d, tt.duration)
			}
		})
	}
}

func TestDuration_UnmarshalEmpty(t *testing.T) {
	var d Duration
	if err := yaml.Unmarshal([]byte(`""`), &d); err != nil {
		t.Fatalf("Unmarshal empty failed: %v", err)
	}
	if d != 0 {
		t.Errorf("Unmarshal empty = %v, want 0", d)
	}
}

func TestDuration_UnmarshalInvalid(t *testing.T) {
	var d Duration
	err := yaml.Unmarshal([]byte(`"invalid"`), &d)
	if err == nil {
		t.Error("Unmarshal invalid should fail")
	}
}

func TestDefaultFileConfig(t *testing.T) {
	cfg := DefaultFileConfig()

	if cfg.Timeouts.AWSInit.Duration() != DefaultAWSInitTimeout {
		t.Errorf("AWSInit = %v, want %v", cfg.Timeouts.AWSInit.Duration(), DefaultAWSInitTimeout)
	}
	if cfg.Timeouts.MultiRegionFetch.Duration() != DefaultMultiRegionFetchTimeout {
		t.Errorf("MultiRegionFetch = %v, want %v", cfg.Timeouts.MultiRegionFetch.Duration(), DefaultMultiRegionFetchTimeout)
	}
	if cfg.Timeouts.TagSearch.Duration() != DefaultTagSearchTimeout {
		t.Errorf("TagSearch = %v, want %v", cfg.Timeouts.TagSearch.Duration(), DefaultTagSearchTimeout)
	}
	if cfg.Timeouts.MetricsLoad.Duration() != DefaultMetricsLoadTimeout {
		t.Errorf("MetricsLoad = %v, want %v", cfg.Timeouts.MetricsLoad.Duration(), DefaultMetricsLoadTimeout)
	}
	if cfg.Concurrency.MaxFetches != DefaultMaxConcurrentFetches {
		t.Errorf("MaxFetches = %d, want %d", cfg.Concurrency.MaxFetches, DefaultMaxConcurrentFetches)
	}
	if cfg.Autosave.Enabled {
		t.Error("Autosave.Enabled should be false by default")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	// Use a temp dir that doesn't have config.yaml
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Should return defaults
	if cfg.AWSInitTimeout() != DefaultAWSInitTimeout {
		t.Errorf("AWSInitTimeout() = %v, want %v", cfg.AWSInitTimeout(), DefaultAWSInitTimeout)
	}
}

func TestLoad_Save_Roundtrip(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpDir)

	cfg := &FileConfig{}
	if err := cfg.SaveRegions([]string{"us-east-1", "us-west-2"}); err != nil {
		t.Fatalf("SaveRegions failed: %v", err)
	}
	if err := cfg.SaveProfiles([]string{"production"}); err != nil {
		t.Fatalf("SaveProfiles failed: %v", err)
	}

	configPath := filepath.Join(tmpDir, ".config", "claws", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	regions, profiles := loaded.GetStartup()
	if len(regions) != 2 || regions[0] != "us-east-1" || regions[1] != "us-west-2" {
		t.Errorf("GetStartup() regions = %v, want [us-east-1, us-west-2]", regions)
	}
	if len(profiles) != 1 || profiles[0] != "production" {
		t.Errorf("GetStartup() profiles = %v, want [production]", profiles)
	}
}

func TestFileConfig_ApplyDefaults(t *testing.T) {
	cfg := &FileConfig{}
	cfg.applyDefaults()

	if cfg.Timeouts.AWSInit.Duration() != DefaultAWSInitTimeout {
		t.Errorf("AWSInit = %v, want %v", cfg.Timeouts.AWSInit.Duration(), DefaultAWSInitTimeout)
	}
	if cfg.Concurrency.MaxFetches != DefaultMaxConcurrentFetches {
		t.Errorf("MaxFetches = %d, want %d", cfg.Concurrency.MaxFetches, DefaultMaxConcurrentFetches)
	}
}

func TestFileConfig_ApplyDefaults_NegativeValues(t *testing.T) {
	cfg := &FileConfig{
		Timeouts: TimeoutConfig{
			AWSInit:          Duration(-5 * time.Second),
			MultiRegionFetch: Duration(-1 * time.Minute),
		},
		Concurrency: ConcurrencyConfig{
			MaxFetches: -10,
		},
	}
	cfg.applyDefaults()

	if cfg.Timeouts.AWSInit.Duration() != DefaultAWSInitTimeout {
		t.Errorf("negative AWSInit should default, got %v", cfg.Timeouts.AWSInit.Duration())
	}
	if cfg.Timeouts.MultiRegionFetch.Duration() != DefaultMultiRegionFetchTimeout {
		t.Errorf("negative MultiRegionFetch should default, got %v", cfg.Timeouts.MultiRegionFetch.Duration())
	}
	if cfg.Concurrency.MaxFetches != DefaultMaxConcurrentFetches {
		t.Errorf("negative MaxFetches should default, got %d", cfg.Concurrency.MaxFetches)
	}
}

func TestFileConfig_SaveRegionsProfiles(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpDir)

	cfg := &FileConfig{}
	if err := cfg.SaveRegions([]string{"eu-west-1"}); err != nil {
		t.Fatalf("SaveRegions failed: %v", err)
	}
	if err := cfg.SaveProfiles([]string{"dev"}); err != nil {
		t.Fatalf("SaveProfiles failed: %v", err)
	}

	regions, profiles := cfg.GetStartup()
	if len(regions) != 1 || regions[0] != "eu-west-1" {
		t.Errorf("GetStartup() regions = %v, want [eu-west-1]", regions)
	}
	if len(profiles) != 1 || profiles[0] != "dev" {
		t.Errorf("GetStartup() profiles = %v, want [dev]", profiles)
	}
}

func TestFileConfig_Getters_ZeroValues(t *testing.T) {
	cfg := &FileConfig{}

	// Getters should return defaults when values are zero
	if cfg.AWSInitTimeout() != DefaultAWSInitTimeout {
		t.Errorf("AWSInitTimeout() = %v, want %v", cfg.AWSInitTimeout(), DefaultAWSInitTimeout)
	}
	if cfg.MultiRegionFetchTimeout() != DefaultMultiRegionFetchTimeout {
		t.Errorf("MultiRegionFetchTimeout() = %v, want %v", cfg.MultiRegionFetchTimeout(), DefaultMultiRegionFetchTimeout)
	}
	if cfg.TagSearchTimeout() != DefaultTagSearchTimeout {
		t.Errorf("TagSearchTimeout() = %v, want %v", cfg.TagSearchTimeout(), DefaultTagSearchTimeout)
	}
	if cfg.MetricsLoadTimeout() != DefaultMetricsLoadTimeout {
		t.Errorf("MetricsLoadTimeout() = %v, want %v", cfg.MetricsLoadTimeout(), DefaultMetricsLoadTimeout)
	}
	if cfg.MaxConcurrentFetches() != DefaultMaxConcurrentFetches {
		t.Errorf("MaxConcurrentFetches() = %d, want %d", cfg.MaxConcurrentFetches(), DefaultMaxConcurrentFetches)
	}
}

func TestLoad_PartialConfig(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".config", "claws")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	data := []byte("timeouts:\n  aws_init: 15s\n")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.AWSInitTimeout() != 15*time.Second {
		t.Errorf("AWSInitTimeout() = %v, want %v", cfg.AWSInitTimeout(), 15*time.Second)
	}
	if cfg.MultiRegionFetchTimeout() != DefaultMultiRegionFetchTimeout {
		t.Errorf("MultiRegionFetchTimeout() = %v, want %v", cfg.MultiRegionFetchTimeout(), DefaultMultiRegionFetchTimeout)
	}
	if cfg.MaxConcurrentFetches() != DefaultMaxConcurrentFetches {
		t.Errorf("MaxConcurrentFetches() = %d, want %d", cfg.MaxConcurrentFetches(), DefaultMaxConcurrentFetches)
	}
}

func TestThemeConfig_UnmarshalString(t *testing.T) {
	var cfg ThemeConfig
	if err := yaml.Unmarshal([]byte(`"nord"`), &cfg); err != nil {
		t.Fatalf("Unmarshal string failed: %v", err)
	}
	if cfg.Preset != "nord" {
		t.Errorf("Preset = %q, want %q", cfg.Preset, "nord")
	}
	if cfg.Primary != "" {
		t.Errorf("Primary should be empty, got %q", cfg.Primary)
	}
}

func TestThemeConfig_UnmarshalObject(t *testing.T) {
	yamlData := `
preset: dracula
primary: "#ff0000"
success: "#00ff00"
`
	var cfg ThemeConfig
	if err := yaml.Unmarshal([]byte(yamlData), &cfg); err != nil {
		t.Fatalf("Unmarshal object failed: %v", err)
	}
	if cfg.Preset != "dracula" {
		t.Errorf("Preset = %q, want %q", cfg.Preset, "dracula")
	}
	if cfg.Primary != "#ff0000" {
		t.Errorf("Primary = %q, want %q", cfg.Primary, "#ff0000")
	}
	if cfg.Success != "#00ff00" {
		t.Errorf("Success = %q, want %q", cfg.Success, "#00ff00")
	}
}

func TestThemeConfig_UnmarshalEmpty(t *testing.T) {
	var cfg ThemeConfig
	if err := yaml.Unmarshal([]byte(`{}`), &cfg); err != nil {
		t.Fatalf("Unmarshal empty failed: %v", err)
	}
	if cfg.Preset != "" {
		t.Errorf("Preset should be empty, got %q", cfg.Preset)
	}
}

func TestSave_PreservesExistingConfig(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".config", "claws")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	existing := `theme: nord
timeouts:
  aws_init: 10s
custom_key: custom_value
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(existing), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	cfg := &FileConfig{}
	if err := cfg.SaveRegions([]string{"us-west-2"}); err != nil {
		t.Fatalf("SaveRegions failed: %v", err)
	}
	if err := cfg.SaveProfiles([]string{"dev", "prod"}); err != nil {
		t.Fatalf("SaveProfiles failed: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	content := string(data)

	if !contains(content, "theme: nord") {
		t.Error("theme: nord was not preserved")
	}
	if !contains(content, "aws_init: 10s") {
		t.Error("timeouts.aws_init was not preserved")
	}
	if !contains(content, "custom_key: custom_value") {
		t.Error("custom_key was not preserved")
	}
	if !contains(content, "us-west-2") {
		t.Error("region us-west-2 was not saved")
	}
	if !contains(content, "dev") || !contains(content, "prod") {
		t.Error("profiles were not saved")
	}
}

func TestSave_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpDir)

	cfg := &FileConfig{}
	if err := cfg.SaveRegions([]string{"eu-west-1"}); err != nil {
		t.Fatalf("SaveRegions failed: %v", err)
	}
	if err := cfg.SaveProfiles([]string{"production"}); err != nil {
		t.Fatalf("SaveProfiles failed: %v", err)
	}

	configPath := filepath.Join(tmpDir, ".config", "claws", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	content := string(data)

	if !contains(content, "eu-west-1") {
		t.Error("region was not saved")
	}
	if !contains(content, "production") {
		t.Error("profile was not saved")
	}
	if contains(content, "theme") {
		t.Error("theme should not be in new minimal file")
	}
}

func TestSave_MultipleProfiles(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpDir)

	cfg := &FileConfig{}
	profiles := []string{"dev", "prod", "__env_only__", "__sdk_default__"}
	if err := cfg.SaveRegions([]string{"us-east-1", "us-west-2"}); err != nil {
		t.Fatalf("SaveRegions failed: %v", err)
	}
	if err := cfg.SaveProfiles(profiles); err != nil {
		t.Fatalf("SaveProfiles failed: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	regions, loadedProfiles := loaded.GetStartup()
	if len(regions) != 2 {
		t.Errorf("regions = %v, want 2 regions", regions)
	}
	if len(loadedProfiles) != 4 {
		t.Errorf("profiles = %v, want 4 profiles", loadedProfiles)
	}
	for i, want := range profiles {
		if loadedProfiles[i] != want {
			t.Errorf("profile[%d] = %q, want %q", i, loadedProfiles[i], want)
		}
	}
}

func TestSave_BackwardCompatProfile(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".config", "claws")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	existing := `startup:
  profile: legacy-profile
  regions:
    - us-east-1
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(existing), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	_, profiles := loaded.GetStartup()
	if len(profiles) != 1 || profiles[0] != "legacy-profile" {
		t.Errorf("profiles = %v, want [legacy-profile]", profiles)
	}
}

func TestSave_PreservesComments(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".config", "claws")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	existing := `# This is a comment on theme
theme: nord
# This is a comment on timeouts
timeouts:
  aws_init: 10s
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(existing), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	cfg := &FileConfig{}
	if err := cfg.SaveRegions([]string{"us-west-2"}); err != nil {
		t.Fatalf("SaveRegions failed: %v", err)
	}
	if err := cfg.SaveProfiles([]string{"dev"}); err != nil {
		t.Fatalf("SaveProfiles failed: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	content := string(data)

	if !contains(content, "# This is a comment on theme") {
		t.Error("comment on theme was not preserved")
	}
	if !contains(content, "# This is a comment on timeouts") {
		t.Error("comment on timeouts was not preserved")
	}
}

func TestSaveTheme(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".config", "claws")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	existing := `timeouts:
  aws_init: 10s
startup:
  regions:
    - us-east-1
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(existing), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	cfg := &FileConfig{}
	if err := cfg.SaveTheme("nord"); err != nil {
		t.Fatalf("SaveTheme failed: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	content := string(data)

	if !contains(content, "theme: nord") {
		t.Error("theme: nord was not saved")
	}
	if !contains(content, "aws_init: 10s") {
		t.Error("timeouts.aws_init was not preserved")
	}
	if !contains(content, "us-east-1") {
		t.Error("startup.regions was not preserved")
	}

	if cfg.GetTheme().Preset != "nord" {
		t.Errorf("GetTheme().Preset = %q, want %q", cfg.GetTheme().Preset, "nord")
	}
}

func TestSavePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".config", "claws")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	existing := `theme: nord
startup:
  regions:
    - us-east-1
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(existing), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	cfg := &FileConfig{}
	if err := cfg.SavePersistence(true); err != nil {
		t.Fatalf("SavePersistence(true) failed: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	content := string(data)

	if !contains(content, "autosave:") {
		t.Error("autosave: key was not created")
	}
	if !contains(content, "enabled: true") {
		t.Error("autosave.enabled: true was not saved")
	}
	if !contains(content, "theme: nord") {
		t.Error("theme was not preserved")
	}

	if !cfg.PersistenceEnabled() {
		t.Error("PersistenceEnabled() should be true")
	}

	if err := cfg.SavePersistence(false); err != nil {
		t.Fatalf("SavePersistence(false) failed: %v", err)
	}

	data, err = os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	content = string(data)

	if !contains(content, "enabled: false") {
		t.Error("autosave.enabled: false was not saved")
	}
	if cfg.PersistenceEnabled() {
		t.Error("PersistenceEnabled() should be false")
	}
}

func TestStartupConfig_GetProfiles(t *testing.T) {
	tests := []struct {
		name   string
		config StartupConfig
		want   []string
	}{
		{"new format", StartupConfig{Profiles: []string{"a", "b"}}, []string{"a", "b"}},
		{"old format", StartupConfig{Profile: "legacy"}, []string{"legacy"}},
		{"both prefers new", StartupConfig{Profile: "old", Profiles: []string{"new"}}, []string{"new"}},
		{"empty", StartupConfig{}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetProfiles()
			if len(got) != len(tt.want) {
				t.Errorf("GetProfiles() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("GetProfiles()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestConcurrentSaves(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpDir)

	cfg := &FileConfig{}

	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 20; j++ {
				_ = cfg.SaveRegions([]string{"us-east-1", "us-west-2"})
				_ = cfg.SaveProfiles([]string{"profile1", "profile2"})
				_ = cfg.SaveTheme("nord")
				_ = cfg.SavePersistence(true)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 5; i++ {
		<-done
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	regions, profiles := loaded.GetStartup()
	if len(regions) != 2 {
		t.Errorf("regions = %v, want 2 regions", regions)
	}
	if len(profiles) != 2 {
		t.Errorf("profiles = %v, want 2 profiles", profiles)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
