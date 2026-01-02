package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	DefaultAWSInitTimeout          = 5 * time.Second
	DefaultMultiRegionFetchTimeout = 30 * time.Second
	DefaultTagSearchTimeout        = 30 * time.Second
	DefaultMetricsLoadTimeout      = 30 * time.Second
	DefaultMaxConcurrentFetches    = 50
)

func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".config", "claws"), nil
}

func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

type TimeoutConfig struct {
	AWSInit          Duration `yaml:"aws_init,omitempty"`
	MultiRegionFetch Duration `yaml:"multi_region_fetch,omitempty"`
	TagSearch        Duration `yaml:"tag_search,omitempty"`
	MetricsLoad      Duration `yaml:"metrics_load,omitempty"`
}

type ConcurrencyConfig struct {
	MaxFetches int `yaml:"max_fetches,omitempty"`
}

type PersistenceConfig struct {
	Enabled bool `yaml:"enabled"`
}

type StartupConfig struct {
	Regions []string `yaml:"regions,omitempty"`
	Profile string   `yaml:"profile,omitempty"`
}

type FileConfig struct {
	Timeouts    TimeoutConfig     `yaml:"timeouts,omitempty"`
	Concurrency ConcurrencyConfig `yaml:"concurrency,omitempty"`
	Persistence PersistenceConfig `yaml:"persistence"`
	Startup     StartupConfig     `yaml:"startup,omitempty"`
}

// Duration wraps time.Duration for YAML marshal/unmarshal as string (e.g., "5s", "30s")
type Duration time.Duration

func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}

func (d Duration) MarshalYAML() (interface{}, error) {
	return time.Duration(d).String(), nil
}

func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err != nil {
		return err
	}
	if s == "" {
		*d = 0
		return nil
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	*d = Duration(dur)
	return nil
}

func DefaultFileConfig() *FileConfig {
	return &FileConfig{
		Timeouts: TimeoutConfig{
			AWSInit:          Duration(DefaultAWSInitTimeout),
			MultiRegionFetch: Duration(DefaultMultiRegionFetchTimeout),
			TagSearch:        Duration(DefaultTagSearchTimeout),
			MetricsLoad:      Duration(DefaultMetricsLoadTimeout),
		},
		Concurrency: ConcurrencyConfig{
			MaxFetches: DefaultMaxConcurrentFetches,
		},
		Persistence: PersistenceConfig{
			Enabled: false,
		},
	}
}

var (
	fileConfig     *FileConfig
	fileConfigOnce sync.Once
	fileConfigMu   sync.RWMutex
)

func File() *FileConfig {
	fileConfigOnce.Do(func() {
		cfg, err := Load()
		if err != nil {
			cfg = DefaultFileConfig()
		}
		fileConfig = cfg
	})
	return fileConfig
}

func ReloadFile() error {
	cfg, err := Load()
	if err != nil {
		return err
	}
	fileConfigMu.Lock()
	fileConfig = cfg
	fileConfigMu.Unlock()
	return nil
}

func Load() (*FileConfig, error) {
	path, err := ConfigPath()
	if err != nil {
		return DefaultFileConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultFileConfig(), nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := DefaultFileConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg.applyDefaults()
	return cfg, nil
}

func (c *FileConfig) Save() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

func (c *FileConfig) applyDefaults() {
	if c.Timeouts.AWSInit == 0 {
		c.Timeouts.AWSInit = Duration(DefaultAWSInitTimeout)
	}
	if c.Timeouts.MultiRegionFetch == 0 {
		c.Timeouts.MultiRegionFetch = Duration(DefaultMultiRegionFetchTimeout)
	}
	if c.Timeouts.TagSearch == 0 {
		c.Timeouts.TagSearch = Duration(DefaultTagSearchTimeout)
	}
	if c.Timeouts.MetricsLoad == 0 {
		c.Timeouts.MetricsLoad = Duration(DefaultMetricsLoadTimeout)
	}
	if c.Concurrency.MaxFetches == 0 {
		c.Concurrency.MaxFetches = DefaultMaxConcurrentFetches
	}
}

func (c *FileConfig) AWSInitTimeout() time.Duration {
	fileConfigMu.RLock()
	defer fileConfigMu.RUnlock()
	if c.Timeouts.AWSInit == 0 {
		return DefaultAWSInitTimeout
	}
	return c.Timeouts.AWSInit.Duration()
}

func (c *FileConfig) MultiRegionFetchTimeout() time.Duration {
	fileConfigMu.RLock()
	defer fileConfigMu.RUnlock()
	if c.Timeouts.MultiRegionFetch == 0 {
		return DefaultMultiRegionFetchTimeout
	}
	return c.Timeouts.MultiRegionFetch.Duration()
}

func (c *FileConfig) TagSearchTimeout() time.Duration {
	fileConfigMu.RLock()
	defer fileConfigMu.RUnlock()
	if c.Timeouts.TagSearch == 0 {
		return DefaultTagSearchTimeout
	}
	return c.Timeouts.TagSearch.Duration()
}

func (c *FileConfig) MetricsLoadTimeout() time.Duration {
	fileConfigMu.RLock()
	defer fileConfigMu.RUnlock()
	if c.Timeouts.MetricsLoad == 0 {
		return DefaultMetricsLoadTimeout
	}
	return c.Timeouts.MetricsLoad.Duration()
}

func (c *FileConfig) MaxConcurrentFetches() int {
	fileConfigMu.RLock()
	defer fileConfigMu.RUnlock()
	if c.Concurrency.MaxFetches == 0 {
		return DefaultMaxConcurrentFetches
	}
	return c.Concurrency.MaxFetches
}

func (c *FileConfig) PersistenceEnabled() bool {
	fileConfigMu.RLock()
	defer fileConfigMu.RUnlock()
	return c.Persistence.Enabled
}

func (c *FileConfig) SetStartup(regions []string, profile string) {
	fileConfigMu.Lock()
	defer fileConfigMu.Unlock()
	c.Startup.Regions = regions
	c.Startup.Profile = profile
}

func (c *FileConfig) GetStartup() (regions []string, profile string) {
	fileConfigMu.RLock()
	defer fileConfigMu.RUnlock()
	return c.Startup.Regions, c.Startup.Profile
}
