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
	if cfg.Persistence.Enabled {
		t.Error("Persistence.Enabled should be false by default")
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

	// Create config with custom values
	cfg := &FileConfig{
		Timeouts: TimeoutConfig{
			AWSInit:          Duration(10 * time.Second),
			MultiRegionFetch: Duration(60 * time.Second),
			TagSearch:        Duration(45 * time.Second),
			MetricsLoad:      Duration(20 * time.Second),
		},
		Concurrency: ConcurrencyConfig{
			MaxFetches: 100,
		},
		Persistence: PersistenceConfig{
			Enabled: true,
		},
		Startup: StartupConfig{
			Regions: []string{"us-east-1", "us-west-2"},
			Profile: "production",
		},
	}

	// Save
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	configPath := filepath.Join(tmpDir, ".config", "claws", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	// Load and verify
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.AWSInitTimeout() != 10*time.Second {
		t.Errorf("AWSInitTimeout() = %v, want %v", loaded.AWSInitTimeout(), 10*time.Second)
	}
	if loaded.MultiRegionFetchTimeout() != 60*time.Second {
		t.Errorf("MultiRegionFetchTimeout() = %v, want %v", loaded.MultiRegionFetchTimeout(), 60*time.Second)
	}
	if loaded.TagSearchTimeout() != 45*time.Second {
		t.Errorf("TagSearchTimeout() = %v, want %v", loaded.TagSearchTimeout(), 45*time.Second)
	}
	if loaded.MetricsLoadTimeout() != 20*time.Second {
		t.Errorf("MetricsLoadTimeout() = %v, want %v", loaded.MetricsLoadTimeout(), 20*time.Second)
	}
	if loaded.MaxConcurrentFetches() != 100 {
		t.Errorf("MaxConcurrentFetches() = %d, want %d", loaded.MaxConcurrentFetches(), 100)
	}
	if !loaded.PersistenceEnabled() {
		t.Error("PersistenceEnabled() should be true")
	}

	regions, profile := loaded.GetStartup()
	if len(regions) != 2 || regions[0] != "us-east-1" || regions[1] != "us-west-2" {
		t.Errorf("GetStartup() regions = %v, want [us-east-1, us-west-2]", regions)
	}
	if profile != "production" {
		t.Errorf("GetStartup() profile = %q, want %q", profile, "production")
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

func TestFileConfig_SetStartup(t *testing.T) {
	cfg := &FileConfig{}

	cfg.SetStartup([]string{"eu-west-1"}, "dev")

	regions, profile := cfg.GetStartup()
	if len(regions) != 1 || regions[0] != "eu-west-1" {
		t.Errorf("GetStartup() regions = %v, want [eu-west-1]", regions)
	}
	if profile != "dev" {
		t.Errorf("GetStartup() profile = %q, want %q", profile, "dev")
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

	// Create config dir
	configDir := filepath.Join(tmpDir, ".config", "claws")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	// Write partial config (only timeouts.aws_init)
	configPath := filepath.Join(configDir, "config.yaml")
	data := []byte("timeouts:\n  aws_init: 15s\n")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Load should fill in defaults for missing values
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.AWSInitTimeout() != 15*time.Second {
		t.Errorf("AWSInitTimeout() = %v, want %v", cfg.AWSInitTimeout(), 15*time.Second)
	}
	// Other values should be defaults
	if cfg.MultiRegionFetchTimeout() != DefaultMultiRegionFetchTimeout {
		t.Errorf("MultiRegionFetchTimeout() = %v, want %v", cfg.MultiRegionFetchTimeout(), DefaultMultiRegionFetchTimeout)
	}
	if cfg.MaxConcurrentFetches() != DefaultMaxConcurrentFetches {
		t.Errorf("MaxConcurrentFetches() = %d, want %d", cfg.MaxConcurrentFetches(), DefaultMaxConcurrentFetches)
	}
}
