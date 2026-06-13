package main

import (
	"slices"
	"testing"

	"github.com/clawscli/claws/internal/config"
)

func TestParseFlags_Profiles(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name:     "comma separated",
			args:     []string{"-p", "dev,prod"},
			expected: []string{"dev", "prod"},
		},
		{
			name:     "repeated flags",
			args:     []string{"-p", "dev", "-p", "prod"},
			expected: []string{"dev", "prod"},
		},
		{
			name:     "mixed comma and repeated",
			args:     []string{"-p", "dev,staging", "-p", "prod"},
			expected: []string{"dev", "staging", "prod"},
		},
		{
			name:     "empty values filtered",
			args:     []string{"-p", "dev, , prod"},
			expected: []string{"dev", "prod"},
		},
		{
			name:     "duplicates removed",
			args:     []string{"-p", "dev,dev", "-p", "dev"},
			expected: []string{"dev"},
		},
		{
			name:     "whitespace trimmed",
			args:     []string{"-p", " dev , prod "},
			expected: []string{"dev", "prod"},
		},
		{
			name:     "long form flag",
			args:     []string{"--profile", "dev,prod"},
			expected: []string{"dev", "prod"},
		},
		{
			name:     "no profiles",
			args:     []string{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := parseFlagsFromArgs(tt.args)

			if !slices.Equal(opts.profiles, tt.expected) {
				t.Errorf("profiles = %v, want %v", opts.profiles, tt.expected)
			}
		})
	}
}

func TestParseFlags_Regions(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name:     "comma separated",
			args:     []string{"-r", "us-east-1,ap-northeast-1"},
			expected: []string{"us-east-1", "ap-northeast-1"},
		},
		{
			name:     "repeated flags",
			args:     []string{"-r", "us-east-1", "-r", "ap-northeast-1"},
			expected: []string{"us-east-1", "ap-northeast-1"},
		},
		{
			name:     "duplicates removed",
			args:     []string{"-r", "us-east-1,us-east-1", "-r", "us-east-1"},
			expected: []string{"us-east-1"},
		},
		{
			name:     "long form flag",
			args:     []string{"--region", "us-east-1,eu-west-1"},
			expected: []string{"us-east-1", "eu-west-1"},
		},
		{
			name:     "no regions",
			args:     []string{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := parseFlagsFromArgs(tt.args)

			if !slices.Equal(opts.regions, tt.expected) {
				t.Errorf("regions = %v, want %v", opts.regions, tt.expected)
			}
		})
	}
}

func TestParseFlags_Combined(t *testing.T) {
	opts := parseFlagsFromArgs([]string{"-p", "dev,prod", "-r", "us-east-1,ap-northeast-1", "-ro"})

	expectedProfiles := []string{"dev", "prod"}
	expectedRegions := []string{"us-east-1", "ap-northeast-1"}

	if !slices.Equal(opts.profiles, expectedProfiles) {
		t.Errorf("profiles = %v, want %v", opts.profiles, expectedProfiles)
	}
	if !slices.Equal(opts.regions, expectedRegions) {
		t.Errorf("regions = %v, want %v", opts.regions, expectedRegions)
	}
	if !opts.readOnly {
		t.Error("readOnly should be true")
	}
}

func TestParseFlags_ConfigFile(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{"short flag", []string{"-c", "/path/to/config.yaml"}, "/path/to/config.yaml"},
		{"long flag", []string{"--config", "/custom/config.yaml"}, "/custom/config.yaml"},
		{"with other flags", []string{"-p", "dev", "-c", "/config.yaml", "-r", "us-east-1"}, "/config.yaml"},
		{"no config", []string{"-p", "dev"}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := parseFlagsFromArgs(tt.args)
			if opts.configFile != tt.expected {
				t.Errorf("configFile = %q, want %q", opts.configFile, tt.expected)
			}
		})
	}
}

func TestParseFlags_EnvCreds(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "short flag", args: []string{"-e"}},
		{name: "long flag", args: []string{"--env"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := parseFlagsFromArgs(tt.args)
			if !opts.envCreds {
				t.Error("envCreds should be true")
			}
		})
	}
}

func TestParseFlags_Filter(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{"short flag", []string{"-f", "bastion"}, "bastion"},
		{"long flag", []string{"--filter", "bastion"}, "bastion"},
		{"with service", []string{"-s", "ec2", "-f", "bastion"}, "bastion"},
		{"whitespace trimmed", []string{"-f", "  bastion  "}, "bastion"},
		{"no filter", []string{"-s", "ec2"}, ""},
		{"missing value", []string{"-f"}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := parseFlagsFromArgs(tt.args)
			if opts.filter != tt.expected {
				t.Errorf("filter = %q, want %q", opts.filter, tt.expected)
			}
		})
	}
}

func TestParseFlags_Tag(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{"key=value", []string{"--tag", "Role=bastion"}, []string{"Role=bastion"}},
		{"key only", []string{"--tag", "Role"}, []string{"Role"}},
		{"partial match", []string{"--tag", "Name~web"}, []string{"Name~web"}},
		{"with service", []string{"-s", "ec2", "--tag", "Env=prod"}, []string{"Env=prod"}},
		{"whitespace trimmed", []string{"--tag", "  Env=prod  "}, []string{"Env=prod"}},
		{"comma literal", []string{"--tag", "Name=api,primary"}, []string{"Name=api,primary"}},
		{"space literal", []string{"--tag", "Owner=Team A"}, []string{"Owner=Team A"}},
		{"duplicates removed case-insensitively", []string{"--tag", "Env=prod", "--tag", "env=prod", "--tag", "ENV=PROD"}, []string{"Env=prod"}},
		{"no tag", []string{"-s", "ec2"}, nil},
		{"missing value", []string{"--tag"}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := parseFlagsFromArgs(tt.args)
			if !slices.Equal(opts.tags, tt.expected) {
				t.Errorf("tags = %v, want %v", opts.tags, tt.expected)
			}
		})
	}
}

func TestParseFlags_FilterAndTagCombined(t *testing.T) {
	opts := parseFlagsFromArgs([]string{"-s", "ec2", "-f", "bastion", "--tag", "Role=bastion"})

	if opts.service != "ec2" {
		t.Errorf("service = %q, want %q", opts.service, "ec2")
	}
	if opts.filter != "bastion" {
		t.Errorf("filter = %q, want %q", opts.filter, "bastion")
	}
	if !slices.Equal(opts.tags, []string{"Role=bastion"}) {
		t.Errorf("tags = %v, want %v", opts.tags, []string{"Role=bastion"})
	}
}

func TestBuildStartupPath_Tags(t *testing.T) {
	tests := []struct {
		name           string
		opts           cliOptions
		startup        config.StartupConfig
		wantTags       []string
		wantFilter     string
		wantResourceID string
	}{
		{
			name:     "cli tags override config tags",
			opts:     cliOptions{service: "ec2", tags: []string{"Role=bastion", "Env=prod"}},
			startup:  config.StartupConfig{Tags: []string{"saved=tag"}},
			wantTags: []string{"Role=bastion", "Env=prod"},
		},
		{
			name:     "config tags used when cli tags absent",
			opts:     cliOptions{service: "ec2"},
			startup:  config.StartupConfig{Tags: []string{"saved=tag"}},
			wantTags: []string{"saved=tag"},
		},
		{
			name:           "cli filter and tags override config values",
			opts:           cliOptions{service: "ec2", filter: "bastion", tags: []string{"Role=bastion"}, resourceID: " i-123 "},
			startup:        config.StartupConfig{Filter: "saved-filter", Tags: []string{"saved=tag"}},
			wantTags:       []string{"Role=bastion"},
			wantFilter:     "bastion",
			wantResourceID: "i-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileCfg := &config.FileConfig{Startup: tt.startup}
			path := buildStartupPath("ec2", "", tt.opts, fileCfg)

			if got := path.Tags; !slices.Equal(got, tt.wantTags) {
				t.Fatalf("tags = %v, want %v", got, tt.wantTags)
			}
			if tt.wantFilter != "" && path.Filter != tt.wantFilter {
				t.Fatalf("filter = %q, want %q", path.Filter, tt.wantFilter)
			}
			if tt.wantResourceID != "" && path.ResourceID != tt.wantResourceID {
				t.Fatalf("resourceID = %q, want %q", path.ResourceID, tt.wantResourceID)
			}
			if path.Service != "ec2" {
				t.Fatalf("service = %q, want %q", path.Service, "ec2")
			}
			if path.ResourceType != "" {
				t.Fatalf("resourceType = %q, want empty", path.ResourceType)
			}
		})
	}
}

func TestApplyStartupConfig_ProfilePrecedence(t *testing.T) {
	tests := []struct {
		name        string
		opts        cliOptions
		startup     []string
		wantProfile []string
	}{
		{
			name:        "saved startup profiles used when no CLI override",
			opts:        cliOptions{},
			startup:     []string{"saved"},
			wantProfile: []string{"saved"},
		},
		{
			name:        "profile flag overrides saved startup profiles",
			opts:        cliOptions{profiles: []string{"cli"}},
			startup:     []string{"saved"},
			wantProfile: []string{"cli"},
		},
		{
			name:        "env flag overrides profile flag and saved startup profiles",
			opts:        cliOptions{envCreds: true, profiles: []string{"cli"}},
			startup:     []string{"saved"},
			wantProfile: []string{config.ProfileIDEnvOnly},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileCfg := &config.FileConfig{Startup: config.StartupConfig{Profiles: tt.startup}}
			cfg := &config.Config{}

			applyStartupConfig(tt.opts, fileCfg, cfg)

			if got := selectionIDs(cfg.Selections()); !slices.Equal(got, tt.wantProfile) {
				t.Errorf("selections = %v, want %v", got, tt.wantProfile)
			}
		})
	}
}

func TestApplyStartupConfig_EnvOverrideDoesNotMutateSavedProfiles(t *testing.T) {
	fileCfg := &config.FileConfig{Startup: config.StartupConfig{Profiles: []string{"personal"}}}
	cfg := &config.Config{}

	applyStartupConfig(cliOptions{envCreds: true}, fileCfg, cfg)

	if got := cfg.Selection().ID(); got != config.ProfileIDEnvOnly {
		t.Fatalf("selection = %q, want env-only", got)
	}
	_, savedProfiles := fileCfg.GetStartup()
	if !slices.Equal(savedProfiles, []string{"personal"}) {
		t.Fatalf("saved profiles = %v, want [personal]", savedProfiles)
	}

	nextCfg := &config.Config{}
	applyStartupConfig(cliOptions{}, fileCfg, nextCfg)

	if got := nextCfg.Selection().ID(); got != "personal" {
		t.Errorf("next launch selection = %q, want personal", got)
	}
}

func selectionIDs(selections []config.ProfileSelection) []string {
	ids := make([]string, len(selections))
	for i, sel := range selections {
		ids[i] = sel.ID()
	}
	return ids
}
