package aws

import (
	"strings"
	"testing"

	"github.com/clawscli/claws/internal/config"
)

func TestBuildSubprocessEnv(t *testing.T) {
	baseEnv := []string{
		"HOME=/home/user",
		"PATH=/usr/bin",
		"AWS_PROFILE=existing",
		"AWS_REGION=us-west-2",
		"AWS_DEFAULT_REGION=us-west-2",
	}

	tests := []struct {
		name       string
		sel        config.ProfileSelection
		region     string
		wantEnv    map[string]string
		wantAbsent []string
	}{
		{
			name:   "SDKDefault preserves AWS_PROFILE",
			sel:    config.SDKDefault(),
			region: "",
			wantEnv: map[string]string{
				"AWS_PROFILE": "existing",
			},
			wantAbsent: []string{},
		},
		{
			name:   "SDKDefault with region override",
			sel:    config.SDKDefault(),
			region: "eu-west-1",
			wantEnv: map[string]string{
				"AWS_PROFILE":        "existing",
				"AWS_REGION":         "eu-west-1",
				"AWS_DEFAULT_REGION": "eu-west-1",
			},
			wantAbsent: []string{},
		},
		{
			name:   "NamedProfile sets AWS_PROFILE",
			sel:    config.NamedProfile("production"),
			region: "",
			wantEnv: map[string]string{
				"AWS_PROFILE": "production",
			},
			wantAbsent: []string{},
		},
		{
			name:   "EnvOnly removes AWS_PROFILE and sets null files",
			sel:    config.EnvOnly(),
			region: "",
			wantEnv: map[string]string{
				"AWS_CONFIG_FILE":             "/dev/null",
				"AWS_SHARED_CREDENTIALS_FILE": "/dev/null",
			},
			wantAbsent: []string{"AWS_PROFILE"},
		},
		{
			name:   "EnvOnly with region",
			sel:    config.EnvOnly(),
			region: "ap-northeast-1",
			wantEnv: map[string]string{
				"AWS_CONFIG_FILE":             "/dev/null",
				"AWS_SHARED_CREDENTIALS_FILE": "/dev/null",
				"AWS_REGION":                  "ap-northeast-1",
				"AWS_DEFAULT_REGION":          "ap-northeast-1",
			},
			wantAbsent: []string{"AWS_PROFILE"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildSubprocessEnv(baseEnv, tt.sel, tt.region)

			// Build map for easy lookup
			envMap := make(map[string]string)
			for _, e := range result {
				parts := strings.SplitN(e, "=", 2)
				if len(parts) == 2 {
					envMap[parts[0]] = parts[1]
				}
			}

			// Check expected values
			for key, want := range tt.wantEnv {
				got, ok := envMap[key]
				if !ok {
					t.Errorf("expected %s to be set", key)
					continue
				}
				if got != want {
					t.Errorf("%s = %q, want %q", key, got, want)
				}
			}

			// Check absent values
			for _, key := range tt.wantAbsent {
				if _, ok := envMap[key]; ok {
					t.Errorf("expected %s to be absent", key)
				}
			}

			// Verify base env vars are preserved
			if envMap["HOME"] != "/home/user" {
				t.Error("HOME should be preserved")
			}
			if envMap["PATH"] != "/usr/bin" {
				t.Error("PATH should be preserved")
			}
		})
	}
}

func TestBuildSubprocessEnv_NilBaseEnv(t *testing.T) {
	// Should not panic with nil base env
	result := BuildSubprocessEnv(nil, config.SDKDefault(), "")
	if result == nil {
		t.Error("BuildSubprocessEnv should return non-nil slice")
	}
}
