package view

import (
	"slices"
	"testing"
)

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		str     string
		pattern string
		want    bool
	}{
		{"AgentCoreStackdev", "agecrstdev", true},
		{"AgentCoreStackdev", "agent", true},
		{"AgentCoreStackdev", "acd", true},
		{"AgentCoreStackdev", "xyz", false},
		{"AgentCoreStackdev", "deva", false},
		{"i-1234567890abcdef0", "i1234", true},
		{"i-1234567890abcdef0", "abcdef", true},
		{"production", "prod", true},
		{"production", "pdn", true},
		{"", "a", false},
		{"abc", "", true},
		// uppercase pattern - case insensitive
		{"production", "PROD", true},
		{"AgentCoreStackdev", "ACD", true},
		{"web-server", "WEB", true},
	}

	for _, tt := range tests {
		t.Run(tt.str+"_"+tt.pattern, func(t *testing.T) {
			got := fuzzyMatch(tt.str, tt.pattern)
			if got != tt.want {
				t.Errorf("fuzzyMatch(%q, %q) = %v, want %v", tt.str, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestMatchNamesWithFallback(t *testing.T) {
	tests := []struct {
		name    string
		names   []string
		pattern string
		want    []string
	}{
		{
			name:    "empty pattern returns all sorted",
			names:   []string{"web-server", "db-server", "cache"},
			pattern: "",
			want:    []string{"cache", "db-server", "web-server"},
		},
		{
			name:    "prefix match single",
			names:   []string{"web-server", "db-server", "cache"},
			pattern: "web",
			want:    []string{"web-server"},
		},
		{
			name:    "prefix match multiple",
			names:   []string{"web-server", "web-api", "db-server"},
			pattern: "web",
			want:    []string{"web-api", "web-server"},
		},
		{
			name:    "fuzzy fallback when no prefix",
			names:   []string{"web-server", "db-server", "cache"},
			pattern: "server",
			want:    []string{"db-server", "web-server"},
		},
		{
			name:    "fuzzy match pattern",
			names:   []string{"web-server", "db-server", "cache"},
			pattern: "wsr",
			want:    []string{"web-server"},
		},
		{
			name:    "case insensitive prefix lowercase pattern",
			names:   []string{"Web-Server", "DB-Server", "Cache"},
			pattern: "web",
			want:    []string{"Web-Server"},
		},
		{
			name:    "case insensitive prefix uppercase pattern",
			names:   []string{"web-server", "web-api", "db-server"},
			pattern: "WEB",
			want:    []string{"web-api", "web-server"},
		},
		{
			name:    "no match returns empty",
			names:   []string{"web-server", "db-server"},
			pattern: "xyz",
			want:    nil,
		},
		{
			name:    "empty names",
			names:   []string{},
			pattern: "web",
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchNamesWithFallback(tt.names, tt.pattern)
			if !slices.Equal(got, tt.want) {
				t.Errorf("matchNamesWithFallback(%v, %q) = %v, want %v",
					tt.names, tt.pattern, got, tt.want)
			}
		})
	}
}
