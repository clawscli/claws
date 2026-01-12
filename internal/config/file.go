package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	apperrors "github.com/clawscli/claws/internal/errors"
	"github.com/clawscli/claws/internal/log"
)

const (
	DefaultAWSInitTimeout          = 5 * time.Second
	DefaultMultiRegionFetchTimeout = 30 * time.Second
	DefaultTagSearchTimeout        = 30 * time.Second
	DefaultMetricsLoadTimeout      = 30 * time.Second
	DefaultLogFetchTimeout         = 10 * time.Second
	DefaultDocsSearchTimeout       = 10 * time.Second
	DefaultMetricsWindow           = 15 * time.Minute
	DefaultMaxConcurrentFetches    = 50
	DefaultMaxStackSize            = 100
	DefaultAIMaxToolCallsPerQuery  = 50
)

var (
	customConfigPath string
	configPathMu     sync.RWMutex
)

// expandTilde expands ~ to user home directory.
func expandTilde(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("expand ~: %w", err)
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

// SetConfigPath sets custom config file path. Must be called before File().
// Returns error if file doesn't exist or isn't readable.
func SetConfigPath(path string) error {
	expanded, err := expandTilde(path)
	if err != nil {
		return apperrors.Wrap(err, "config file", "path", path)
	}
	if _, err := os.Stat(expanded); err != nil {
		return apperrors.Wrap(err, "config file", "path", expanded)
	}
	configPathMu.Lock()
	customConfigPath = expanded
	configPathMu.Unlock()
	return nil
}

// GetConfigPath returns the current custom config path (empty if using default).
func GetConfigPath() string {
	configPathMu.RLock()
	defer configPathMu.RUnlock()
	return customConfigPath
}

func ConfigDir() (string, error) {
	configPathMu.RLock()
	custom := customConfigPath
	configPathMu.RUnlock()

	if custom != "" {
		return filepath.Dir(custom), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".config", "claws"), nil
}

func ConfigPath() (string, error) {
	configPathMu.RLock()
	custom := customConfigPath
	configPathMu.RUnlock()

	if custom != "" {
		return custom, nil
	}

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
	LogFetch         Duration `yaml:"log_fetch,omitempty"`
	DocsSearch       Duration `yaml:"docs_search,omitempty"`
}

type CloudWatchConfig struct {
	Window Duration `yaml:"window,omitempty"`
}

type ConcurrencyConfig struct {
	MaxFetches int `yaml:"max_fetches,omitempty"`
}

type PersistenceConfig struct {
	Enabled bool `yaml:"enabled"`
}

type StartupConfig struct {
	View     string   `yaml:"view,omitempty"` // "dashboard", "services", or "service/resource" (e.g., "ec2", "rds/snapshots")
	Regions  []string `yaml:"regions,omitempty"`
	Profile  string   `yaml:"profile,omitempty"`  // Deprecated: for backward compat (read-only)
	Profiles []string `yaml:"profiles,omitempty"` // New format: multiple profile IDs
}

// GetProfiles returns profile IDs (new format preferred, fallback to old).
// Returns a copy to prevent race conditions with concurrent writes.
func (s StartupConfig) GetProfiles() []string {
	if len(s.Profiles) > 0 {
		return append([]string(nil), s.Profiles...)
	}
	if s.Profile != "" {
		return []string{s.Profile}
	}
	return nil
}

type NavigationConfig struct {
	MaxStackSize int `yaml:"max_stack_size,omitempty"`
}

type AIConfig struct {
	Profile              string `yaml:"profile,omitempty"`
	Region               string `yaml:"region,omitempty"`
	Model                string `yaml:"model,omitempty"`
	MaxSessions          int    `yaml:"max_sessions,omitempty"`
	MaxTokens            int    `yaml:"max_tokens,omitempty"`
	ThinkingBudget       *int   `yaml:"thinking_budget,omitempty"`
	MaxToolRounds        int    `yaml:"max_tool_rounds,omitempty"`
	MaxToolCallsPerQuery int    `yaml:"max_tool_calls_per_query,omitempty"`
	SaveSessions         *bool  `yaml:"save_sessions,omitempty"`
}

// ThemeConfig holds theme configuration.
// Can be specified as:
//   - A preset name string: "dark", "light", "nord", "dracula", "gruvbox", "catppuccin"
//   - An object with optional preset and color overrides
type ThemeConfig struct {
	Preset          string `yaml:"preset,omitempty"`
	Primary         string `yaml:"primary,omitempty"`
	Secondary       string `yaml:"secondary,omitempty"`
	Accent          string `yaml:"accent,omitempty"`
	Text            string `yaml:"text,omitempty"`
	TextBright      string `yaml:"text_bright,omitempty"`
	TextDim         string `yaml:"text_dim,omitempty"`
	TextMuted       string `yaml:"text_muted,omitempty"`
	Success         string `yaml:"success,omitempty"`
	Warning         string `yaml:"warning,omitempty"`
	Danger          string `yaml:"danger,omitempty"`
	Info            string `yaml:"info,omitempty"`
	Pending         string `yaml:"pending,omitempty"`
	Border          string `yaml:"border,omitempty"`
	BorderHighlight string `yaml:"border_highlight,omitempty"`
	Background      string `yaml:"background,omitempty"`
	BackgroundAlt   string `yaml:"background_alt,omitempty"`
	Selection       string `yaml:"selection,omitempty"`
	SelectionText   string `yaml:"selection_text,omitempty"`
	TableHeader     string `yaml:"table_header,omitempty"`
	TableHeaderText string `yaml:"table_header_text,omitempty"`
	TableBorder     string `yaml:"table_border,omitempty"`
	BadgeForeground string `yaml:"badge_foreground,omitempty"`
	BadgeBackground string `yaml:"badge_background,omitempty"`
}

func (t *ThemeConfig) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		t.Preset = node.Value
		return nil
	}

	type rawThemeConfig ThemeConfig
	return node.Decode((*rawThemeConfig)(t))
}

type FileConfig struct {
	mu                  sync.RWMutex      `yaml:"-"`
	persistenceOverride *bool             `yaml:"-"`
	Timeouts            TimeoutConfig     `yaml:"timeouts,omitempty"`
	Concurrency         ConcurrencyConfig `yaml:"concurrency,omitempty"`
	CloudWatch          CloudWatchConfig  `yaml:"cloudwatch,omitempty"`
	Autosave            PersistenceConfig `yaml:"autosave,omitempty"`
	Startup             StartupConfig     `yaml:"startup,omitempty"`
	Theme               ThemeConfig       `yaml:"theme,omitempty"`
	Navigation          NavigationConfig  `yaml:"navigation,omitempty"`
	AI                  AIConfig          `yaml:"ai,omitempty"`
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
			LogFetch:         Duration(DefaultLogFetchTimeout),
			DocsSearch:       Duration(DefaultDocsSearchTimeout),
		},
		Concurrency: ConcurrencyConfig{
			MaxFetches: DefaultMaxConcurrentFetches,
		},
		CloudWatch: CloudWatchConfig{
			Window: Duration(DefaultMetricsWindow),
		},
		Navigation: NavigationConfig{
			MaxStackSize: DefaultMaxStackSize,
		},
	}
}

var (
	fileConfig     *FileConfig
	fileConfigOnce sync.Once
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

func (c *FileConfig) applyDefaults() {
	if c.Timeouts.AWSInit <= 0 {
		c.Timeouts.AWSInit = Duration(DefaultAWSInitTimeout)
	}
	if c.Timeouts.MultiRegionFetch <= 0 {
		c.Timeouts.MultiRegionFetch = Duration(DefaultMultiRegionFetchTimeout)
	}
	if c.Timeouts.TagSearch <= 0 {
		c.Timeouts.TagSearch = Duration(DefaultTagSearchTimeout)
	}
	if c.Timeouts.MetricsLoad <= 0 {
		c.Timeouts.MetricsLoad = Duration(DefaultMetricsLoadTimeout)
	}
	if c.Timeouts.LogFetch <= 0 {
		c.Timeouts.LogFetch = Duration(DefaultLogFetchTimeout)
	}
	if c.Timeouts.DocsSearch <= 0 {
		c.Timeouts.DocsSearch = Duration(DefaultDocsSearchTimeout)
	}
	if c.CloudWatch.Window <= 0 {
		c.CloudWatch.Window = Duration(DefaultMetricsWindow)
	}
	if c.Concurrency.MaxFetches <= 0 {
		c.Concurrency.MaxFetches = DefaultMaxConcurrentFetches
	}
	if c.Navigation.MaxStackSize <= 0 {
		c.Navigation.MaxStackSize = DefaultMaxStackSize
	}
}

func (c *FileConfig) AWSInitTimeout() time.Duration {
	return withRLock(&c.mu, func() time.Duration {
		if c.Timeouts.AWSInit == 0 {
			return DefaultAWSInitTimeout
		}
		return c.Timeouts.AWSInit.Duration()
	})
}

func (c *FileConfig) MultiRegionFetchTimeout() time.Duration {
	return withRLock(&c.mu, func() time.Duration {
		if c.Timeouts.MultiRegionFetch == 0 {
			return DefaultMultiRegionFetchTimeout
		}
		return c.Timeouts.MultiRegionFetch.Duration()
	})
}

func (c *FileConfig) TagSearchTimeout() time.Duration {
	return withRLock(&c.mu, func() time.Duration {
		if c.Timeouts.TagSearch == 0 {
			return DefaultTagSearchTimeout
		}
		return c.Timeouts.TagSearch.Duration()
	})
}

func (c *FileConfig) MetricsLoadTimeout() time.Duration {
	return withRLock(&c.mu, func() time.Duration {
		if c.Timeouts.MetricsLoad == 0 {
			return DefaultMetricsLoadTimeout
		}
		return c.Timeouts.MetricsLoad.Duration()
	})
}

func (c *FileConfig) LogFetchTimeout() time.Duration {
	return withRLock(&c.mu, func() time.Duration {
		if c.Timeouts.LogFetch == 0 {
			return DefaultLogFetchTimeout
		}
		return c.Timeouts.LogFetch.Duration()
	})
}

func (c *FileConfig) DocsSearchTimeout() time.Duration {
	return withRLock(&c.mu, func() time.Duration {
		if c.Timeouts.DocsSearch == 0 {
			return DefaultDocsSearchTimeout
		}
		return c.Timeouts.DocsSearch.Duration()
	})
}

func (c *FileConfig) MaxConcurrentFetches() int {
	return withRLock(&c.mu, func() int {
		if c.Concurrency.MaxFetches == 0 {
			return DefaultMaxConcurrentFetches
		}
		return c.Concurrency.MaxFetches
	})
}

func (c *FileConfig) MetricsWindow() time.Duration {
	return withRLock(&c.mu, func() time.Duration {
		if c.CloudWatch.Window == 0 {
			return DefaultMetricsWindow
		}
		return c.CloudWatch.Window.Duration()
	})
}

// MaxStackSize returns the maximum navigation stack size.
func (c *FileConfig) MaxStackSize() int {
	return withRLock(&c.mu, func() int {
		if c.Navigation.MaxStackSize <= 0 {
			return DefaultMaxStackSize
		}
		return c.Navigation.MaxStackSize
	})
}

func (c *FileConfig) PersistenceEnabled() bool {
	return withRLock(&c.mu, func() bool {
		if c.persistenceOverride != nil {
			return *c.persistenceOverride
		}
		return c.Autosave.Enabled
	})
}

func (c *FileConfig) SetPersistenceEnabled(enabled bool) {
	doWithLock(&c.mu, func() { c.persistenceOverride = &enabled })
}

func (c *FileConfig) GetStartup() ([]string, []string) {
	type result struct {
		regions  []string
		profiles []string
	}
	r := withRLock(&c.mu, func() result {
		return result{
			append([]string(nil), c.Startup.Regions...),
			c.Startup.GetProfiles(),
		}
	})
	return r.regions, r.profiles
}

// GetStartupView returns the configured startup view ("dashboard", "services", or service/resource).
func (c *FileConfig) GetStartupView() string {
	return withRLock(&c.mu, func() string {
		return c.Startup.View
	})
}

func (c *FileConfig) GetTheme() ThemeConfig {
	return withRLock(&c.mu, func() ThemeConfig { return c.Theme })
}

const DefaultAIModel = "global.anthropic.claude-haiku-4-5-20251001-v1:0"
const DefaultAIMaxSessions = 100
const DefaultAIMaxTokens = 16000
const DefaultAIThinkingBudget = 8000
const DefaultAIMaxToolRounds = 15

func (c *FileConfig) GetAIProfile() string {
	return withRLock(&c.mu, func() string {
		return c.AI.Profile
	})
}

func (c *FileConfig) GetAIRegion() string {
	return withRLock(&c.mu, func() string {
		return c.AI.Region
	})
}

func (c *FileConfig) GetAIModel() string {
	return withRLock(&c.mu, func() string {
		if c.AI.Model == "" {
			return DefaultAIModel
		}
		return c.AI.Model
	})
}

func (c *FileConfig) GetAIMaxSessions() int {
	return withRLock(&c.mu, func() int {
		if c.AI.MaxSessions <= 0 {
			return DefaultAIMaxSessions
		}
		return c.AI.MaxSessions
	})
}

func (c *FileConfig) GetAIMaxTokens() int {
	return withRLock(&c.mu, func() int {
		if c.AI.MaxTokens <= 0 {
			return DefaultAIMaxTokens
		}
		return c.AI.MaxTokens
	})
}

func (c *FileConfig) GetAIThinkingBudget() int {
	return withRLock(&c.mu, func() int {
		if c.AI.ThinkingBudget == nil {
			return DefaultAIThinkingBudget
		}
		v := *c.AI.ThinkingBudget
		if v < 0 {
			log.Warn("ai.thinking_budget is negative, treating as disabled", "value", v)
			return 0
		}
		return v
	})
}

func (c *FileConfig) GetAIMaxToolRounds() int {
	return withRLock(&c.mu, func() int {
		if c.AI.MaxToolRounds <= 0 {
			return DefaultAIMaxToolRounds
		}
		return c.AI.MaxToolRounds
	})
}

func (c *FileConfig) GetAIMaxToolCallsPerQuery() int {
	return withRLock(&c.mu, func() int {
		if c.AI.MaxToolCallsPerQuery <= 0 {
			return DefaultAIMaxToolCallsPerQuery
		}
		return c.AI.MaxToolCallsPerQuery
	})
}

func (c *FileConfig) GetAISaveSessions() bool {
	return withRLock(&c.mu, func() bool {
		if c.AI.SaveSessions == nil {
			return false
		}
		return *c.AI.SaveSessions
	})
}

func (c *FileConfig) SaveRegions(regions []string) error {
	if len(regions) == 0 {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.Startup.Regions = append([]string(nil), regions...)

	return c.patchConfigLocked(func(mapping *yaml.Node) {
		startupNode := findOrCreateMappingKey(mapping, "startup")
		ensureMappingNode(startupNode)
		setSequenceValue(startupNode, "regions", regions)
	})
}

func (c *FileConfig) SaveProfiles(profiles []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Startup.Profiles = append([]string(nil), profiles...)
	c.Startup.Profile = ""

	return c.patchConfigLocked(func(mapping *yaml.Node) {
		startupNode := findOrCreateMappingKey(mapping, "startup")
		ensureMappingNode(startupNode)
		setSequenceValue(startupNode, "profiles", profiles)
		removeKey(startupNode, "profile")
	})
}

func (c *FileConfig) SaveTheme(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Theme.Preset = name

	return c.patchConfigLocked(func(mapping *yaml.Node) {
		setScalarValue(mapping, "theme", name)
	})
}

func (c *FileConfig) SavePersistence(enabled bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Autosave.Enabled = enabled
	c.persistenceOverride = nil

	return c.patchConfigLocked(func(mapping *yaml.Node) {
		autosaveNode := findOrCreateMappingKey(mapping, "autosave")
		ensureMappingNode(autosaveNode)
		setBoolValue(autosaveNode, "enabled", enabled)
	})
}

func (c *FileConfig) patchConfigLocked(patchFn func(mapping *yaml.Node)) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			data = []byte("{}")
		} else {
			return fmt.Errorf("read config: %w", err)
		}
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		root = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode}}}
	}
	mapping := root.Content[0]
	if mapping.Kind != yaml.MappingNode {
		mapping = &yaml.Node{Kind: yaml.MappingNode}
		root.Content[0] = mapping
	}

	patchFn(mapping)

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&root); err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("close encoder: %w", err)
	}

	return atomicWrite(path, buf.Bytes())
}

func ensureMappingNode(node *yaml.Node) {
	if node.Kind != yaml.MappingNode {
		node.Kind = yaml.MappingNode
		node.Content = nil
	}
}

func findOrCreateMappingKey(mapping *yaml.Node, key string) *yaml.Node {
	for i := 0; i < len(mapping.Content)-1; i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}

	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: key}
	valueNode := &yaml.Node{Kind: yaml.MappingNode}
	mapping.Content = append(mapping.Content, keyNode, valueNode)
	return valueNode
}

func setSequenceValue(mapping *yaml.Node, key string, values []string) {
	var seqNode *yaml.Node
	for i := 0; i < len(mapping.Content)-1; i += 2 {
		if mapping.Content[i].Value == key {
			seqNode = mapping.Content[i+1]
			break
		}
	}

	if len(values) == 0 {
		removeKey(mapping, key)
		return
	}

	if seqNode == nil {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: key}
		seqNode = &yaml.Node{Kind: yaml.SequenceNode}
		mapping.Content = append(mapping.Content, keyNode, seqNode)
	}

	seqNode.Kind = yaml.SequenceNode
	seqNode.Content = make([]*yaml.Node, len(values))
	for i, v := range values {
		seqNode.Content[i] = &yaml.Node{Kind: yaml.ScalarNode, Value: v}
	}
}

func setScalarValue(mapping *yaml.Node, key string, value string) {
	if value == "" {
		removeKey(mapping, key)
		return
	}

	for i := 0; i < len(mapping.Content)-1; i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content[i+1].Kind = yaml.ScalarNode
			mapping.Content[i+1].Value = value
			mapping.Content[i+1].Content = nil
			return
		}
	}

	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: key}
	valueNode := &yaml.Node{Kind: yaml.ScalarNode, Value: value}
	mapping.Content = append(mapping.Content, keyNode, valueNode)
}

func setBoolValue(mapping *yaml.Node, key string, value bool) {
	strVal := "false"
	if value {
		strVal = "true"
	}

	for i := 0; i < len(mapping.Content)-1; i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content[i+1].Kind = yaml.ScalarNode
			mapping.Content[i+1].Tag = "!!bool"
			mapping.Content[i+1].Value = strVal
			mapping.Content[i+1].Content = nil
			return
		}
	}

	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: key}
	valueNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: strVal}
	mapping.Content = append(mapping.Content, keyNode, valueNode)
}

func removeKey(mapping *yaml.Node, key string) {
	for i := 0; i < len(mapping.Content)-1; i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content = append(mapping.Content[:i], mapping.Content[i+2:]...)
			return
		}
	}
}

func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, ".config.yaml.tmp.*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename config file: %w", err)
	}
	return nil
}
