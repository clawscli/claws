package view

import (
	"context"
	"testing"

	"github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/registry"
)

func TestFormatTags(t *testing.T) {
	tests := []struct {
		name   string
		tags   map[string]string
		maxLen int
		want   string
	}{
		{
			name:   "nil tags",
			tags:   nil,
			maxLen: 50,
			want:   "",
		},
		{
			name:   "single tag",
			tags:   map[string]string{"Env": "prod"},
			maxLen: 50,
			want:   "Env=prod",
		},
		{
			name:   "sorted output",
			tags:   map[string]string{"Z": "1", "A": "2"},
			maxLen: 50,
			want:   "A=2, Z=1",
		},
		{
			name:   "truncation",
			tags:   map[string]string{"VeryLongKey": "VeryLongValue"},
			maxLen: 10,
			want:   "VeryLongK…",
		},
		{
			name:   "control char sanitization",
			tags:   map[string]string{"Key": "val\x00ue\x1b[31m"},
			maxLen: 50,
			want:   "Key=value[31m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTags(tt.tags, tt.maxLen)
			if got != tt.want {
				t.Errorf("formatTags() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSanitizeTagValue(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"normal", "normal"},
		{"with\x00null", "withnull"},
		{"with\ttab", "withtab"},
		{"ansi\x1b[31mred", "ansi[31mred"},
		{"del\x7fchar", "delchar"},
		{"日本語", "日本語"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeTagValue(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeTagValue(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTagSearchView_parseTagFilters(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	tests := []struct {
		name       string
		tagFilter  string
		wantKey    string
		wantValues []string
		wantNil    bool
	}{
		{
			name:      "empty filter returns nil",
			tagFilter: "",
			wantNil:   true,
		},
		{
			name:       "key only",
			tagFilter:  "Environment",
			wantKey:    "Environment",
			wantValues: nil,
		},
		{
			name:       "key=value",
			tagFilter:  "Environment=production",
			wantKey:    "Environment",
			wantValues: []string{"production"},
		},
		{
			name:       "key with equals in value",
			tagFilter:  "Config=key=value",
			wantKey:    "Config",
			wantValues: []string{"key=value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewTagSearchView(ctx, reg, tt.tagFilter)
			filters := v.parseTagFilters()

			if tt.wantNil {
				if filters != nil {
					t.Errorf("parseTagFilters() = %v, want nil", filters)
				}
				return
			}

			if len(filters) != 1 {
				t.Fatalf("parseTagFilters() returned %d filters, want 1", len(filters))
			}

			filter := filters[0]
			if *filter.Key != tt.wantKey {
				t.Errorf("filter.Key = %q, want %q", *filter.Key, tt.wantKey)
			}

			if tt.wantValues == nil {
				if filter.Values != nil {
					t.Errorf("filter.Values = %v, want nil", filter.Values)
				}
			} else {
				if len(filter.Values) != len(tt.wantValues) {
					t.Errorf("filter.Values = %v, want %v", filter.Values, tt.wantValues)
				}
				for i, v := range tt.wantValues {
					if filter.Values[i] != v {
						t.Errorf("filter.Values[%d] = %q, want %q", i, filter.Values[i], v)
					}
				}
			}
		})
	}
}

func TestTagSearchView_applyFilter(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	v := NewTagSearchView(ctx, reg, "")
	v.resources = []taggedARN{
		{
			RawARN: "arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0",
			Region: "us-east-1",
			ARN:    &aws.ARN{Service: "ec2", ResourceType: "instance", ResourceID: "i-1234567890abcdef0"},
			Tags:   map[string]string{"Name": "web-server", "Environment": "production"},
		},
		{
			RawARN: "arn:aws:s3:::my-bucket",
			Region: "us-east-1",
			ARN:    &aws.ARN{Service: "s3", ResourceType: "", ResourceID: "my-bucket"},
			Tags:   map[string]string{"Name": "storage", "Environment": "development"},
		},
		{
			RawARN: "arn:aws:lambda:us-west-2:123456789012:function:my-func",
			Region: "us-west-2",
			ARN:    &aws.ARN{Service: "lambda", ResourceType: "function", ResourceID: "my-func"},
			Tags:   map[string]string{"Name": "processor"},
		},
	}

	tests := []struct {
		name       string
		filterText string
		wantCount  int
		wantIDs    []string
	}{
		{
			name:       "empty filter returns all",
			filterText: "",
			wantCount:  3,
		},
		{
			name:       "filter by service",
			filterText: "ec2",
			wantCount:  1,
			wantIDs:    []string{"i-1234567890abcdef0"},
		},
		{
			name:       "filter by region",
			filterText: "us-west-2",
			wantCount:  1,
			wantIDs:    []string{"my-func"},
		},
		{
			name:       "filter by tag value",
			filterText: "production",
			wantCount:  1,
			wantIDs:    []string{"i-1234567890abcdef0"},
		},
		{
			name:       "filter by tag key",
			filterText: "environment",
			wantCount:  2,
		},
		{
			name:       "filter by resource ID",
			filterText: "bucket",
			wantCount:  1,
			wantIDs:    []string{"my-bucket"},
		},
		{
			name:       "no match",
			filterText: "nonexistent",
			wantCount:  0,
		},
		{
			name:       "case insensitive",
			filterText: "LAMBDA",
			wantCount:  1,
			wantIDs:    []string{"my-func"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v.filterText = tt.filterText
			v.applyFilter()

			if len(v.filtered) != tt.wantCount {
				t.Errorf("applyFilter() filtered %d resources, want %d", len(v.filtered), tt.wantCount)
			}

			if tt.wantIDs != nil {
				for i, wantID := range tt.wantIDs {
					if i < len(v.filtered) && v.filtered[i].ARN.ResourceID != wantID {
						t.Errorf("filtered[%d].ResourceID = %q, want %q", i, v.filtered[i].ARN.ResourceID, wantID)
					}
				}
			}
		})
	}
}
