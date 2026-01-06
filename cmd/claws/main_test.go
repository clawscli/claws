package main

import (
	"slices"
	"testing"
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
