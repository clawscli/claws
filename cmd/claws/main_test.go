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
