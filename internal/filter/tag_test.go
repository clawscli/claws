package filter

import (
	"testing"
)

func TestMatchesTagFilter(t *testing.T) {
	tests := []struct {
		name   string
		tags   map[string]string
		filter string
		want   bool
	}{
		{
			name:   "nil tags",
			tags:   nil,
			filter: "key",
			want:   false,
		},
		{
			name:   "empty filter with tags",
			tags:   map[string]string{"key": "value"},
			filter: "",
			want:   true,
		},
		{
			name:   "empty filter with empty tags",
			tags:   map[string]string{},
			filter: "",
			want:   false,
		},
		{
			name:   "exact match success",
			tags:   map[string]string{"env": "prod"},
			filter: "env=prod",
			want:   true,
		},
		{
			name:   "exact match case insensitive key",
			tags:   map[string]string{"Environment": "prod"},
			filter: "environment=prod",
			want:   true,
		},
		{
			name:   "exact match case insensitive value",
			tags:   map[string]string{"env": "Production"},
			filter: "env=production",
			want:   true,
		},
		{
			name:   "exact match failure",
			tags:   map[string]string{"env": "prod"},
			filter: "env=dev",
			want:   false,
		},
		{
			name:   "key exists success",
			tags:   map[string]string{"Name": "my-instance"},
			filter: "Name",
			want:   true,
		},
		{
			name:   "key exists case insensitive",
			tags:   map[string]string{"Name": "my-instance"},
			filter: "name",
			want:   true,
		},
		{
			name:   "key exists failure",
			tags:   map[string]string{"Name": "my-instance"},
			filter: "env",
			want:   false,
		},
		{
			name:   "partial match success",
			tags:   map[string]string{"Name": "my-production-server"},
			filter: "Name~prod",
			want:   true,
		},
		{
			name:   "partial match case insensitive",
			tags:   map[string]string{"Name": "MY-PRODUCTION-SERVER"},
			filter: "Name~prod",
			want:   true,
		},
		{
			name:   "partial match failure",
			tags:   map[string]string{"Name": "my-dev-server"},
			filter: "Name~prod",
			want:   false,
		},
		{
			name:   "partial match key not found",
			tags:   map[string]string{"env": "prod"},
			filter: "Name~prod",
			want:   false,
		},
		{
			name:   "malformed partial match - trailing tilde",
			tags:   map[string]string{"key": "value"},
			filter: "key~",
			want:   true,
		},
		{
			name:   "malformed partial match - only tilde",
			tags:   map[string]string{"key": "value"},
			filter: "~",
			want:   false,
		},
		{
			name:   "malformed exact match - trailing equals",
			tags:   map[string]string{"key": "value"},
			filter: "key=",
			want:   false,
		},
		{
			name:   "malformed exact match - only equals",
			tags:   map[string]string{"key": "value"},
			filter: "=",
			want:   false,
		},
		{
			name:   "exact match key not found",
			tags:   map[string]string{"other": "value"},
			filter: "key=value",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchesTagFilter(tt.tags, tt.filter)
			if got != tt.want {
				t.Errorf("MatchesTagFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchesTagFilters(t *testing.T) {
	tests := []struct {
		name       string
		tags       map[string]string
		tagFilters []string
		want       bool
	}{
		{
			name:       "all filters match",
			tags:       map[string]string{"env": "prod", "app": "claws"},
			tagFilters: []string{"env=prod", "app"},
			want:       true,
		},
		{
			name:       "one missing filter fails",
			tags:       map[string]string{"env": "prod", "app": "claws"},
			tagFilters: []string{"env=prod", "tier"},
			want:       false,
		},
		{
			name:       "key only plus exact match",
			tags:       map[string]string{"environment": "production"},
			tagFilters: []string{"environment", "environment=production"},
			want:       true,
		},
		{
			name:       "partial plus exact match",
			tags:       map[string]string{"name": "claws-production"},
			tagFilters: []string{"name~prod", "name=claws-production"},
			want:       true,
		},
		{
			name:       "empty slice matches nil tags",
			tags:       nil,
			tagFilters: nil,
			want:       true,
		},
		{
			name:       "nil tags with non-empty filters fails",
			tags:       nil,
			tagFilters: []string{"env=prod"},
			want:       false,
		},
		{
			name:       "comma in value is literal",
			tags:       map[string]string{"description": "alpha,beta"},
			tagFilters: []string{"description=alpha,beta"},
			want:       true,
		},
		{
			name:       "space in value is literal",
			tags:       map[string]string{"description": "alpha beta"},
			tagFilters: []string{"description=alpha beta"},
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchesTagFilters(tt.tags, tt.tagFilters)
			if got != tt.want {
				t.Errorf("MatchesTagFilters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCycleIndex(t *testing.T) {
	tests := []struct {
		name    string
		current int
		length  int
		reverse bool
		want    int
	}{
		{"forward from 0", 0, 3, false, 1},
		{"forward from 1", 1, 3, false, 2},
		{"forward wrap", 2, 3, false, 0},
		{"reverse from 2", 2, 3, true, 1},
		{"reverse from 1", 1, 3, true, 0},
		{"reverse wrap", 0, 3, true, 2},
		{"zero length", 0, 0, false, 0},
		{"negative length", 0, -1, false, 0},
		{"single element forward", 0, 1, false, 0},
		{"single element reverse", 0, 1, true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CycleIndex(tt.current, tt.length, tt.reverse)
			if got != tt.want {
				t.Errorf("CycleIndex(%d, %d, %v) = %d, want %d",
					tt.current, tt.length, tt.reverse, got, tt.want)
			}
		})
	}
}
