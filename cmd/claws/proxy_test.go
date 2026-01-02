package main

import (
	"os"
	"testing"
)

func TestGetEnvWithFallback(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		keys     []string
		expected string
	}{
		{
			name:     "first key exists",
			envVars:  map[string]string{"TEST_VAR_A": "value_a"},
			keys:     []string{"TEST_VAR_A", "TEST_VAR_B"},
			expected: "value_a",
		},
		{
			name:     "second key exists",
			envVars:  map[string]string{"TEST_VAR_B": "value_b"},
			keys:     []string{"TEST_VAR_A", "TEST_VAR_B"},
			expected: "value_b",
		},
		{
			name:     "first key takes precedence",
			envVars:  map[string]string{"TEST_VAR_A": "value_a", "TEST_VAR_B": "value_b"},
			keys:     []string{"TEST_VAR_A", "TEST_VAR_B"},
			expected: "value_a",
		},
		{
			name:     "no keys exist",
			envVars:  map[string]string{},
			keys:     []string{"TEST_VAR_A", "TEST_VAR_B"},
			expected: "",
		},
		{
			name:     "empty value is skipped",
			envVars:  map[string]string{"TEST_VAR_A": "", "TEST_VAR_B": "value_b"},
			keys:     []string{"TEST_VAR_A", "TEST_VAR_B"},
			expected: "value_b",
		},
		{
			name:     "no keys provided",
			envVars:  map[string]string{},
			keys:     []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnvVars(t, tt.keys)
			setEnvVars(t, tt.envVars)

			result := getEnvWithFallback(tt.keys...)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestPropagateAllProxy(t *testing.T) {
	proxyVars := []string{"ALL_PROXY", "all_proxy", "HTTPS_PROXY", "https_proxy"}

	tests := []struct {
		name          string
		envVars       map[string]string
		expectSet     bool
		expectedValue string
	}{
		{
			name:          "ALL_PROXY propagates to HTTPS_PROXY",
			envVars:       map[string]string{"ALL_PROXY": "socks5h://proxy:1080"},
			expectSet:     true,
			expectedValue: "socks5h://proxy:1080",
		},
		{
			name:          "all_proxy (lowercase) propagates to HTTPS_PROXY",
			envVars:       map[string]string{"all_proxy": "socks5h://proxy:1080"},
			expectSet:     true,
			expectedValue: "socks5h://proxy:1080",
		},
		{
			name:          "ALL_PROXY takes precedence over all_proxy",
			envVars:       map[string]string{"ALL_PROXY": "http://upper", "all_proxy": "http://lower"},
			expectSet:     true,
			expectedValue: "http://upper",
		},
		{
			name:          "HTTPS_PROXY already set - no propagation",
			envVars:       map[string]string{"ALL_PROXY": "http://all", "HTTPS_PROXY": "http://existing"},
			expectSet:     true,
			expectedValue: "http://existing",
		},
		{
			name:          "https_proxy already set - no propagation",
			envVars:       map[string]string{"ALL_PROXY": "http://all", "https_proxy": "http://existing"},
			expectSet:     true,
			expectedValue: "http://existing",
		},
		{
			name:          "no ALL_PROXY - no action",
			envVars:       map[string]string{},
			expectSet:     false,
			expectedValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnvVars(t, proxyVars)
			setEnvVars(t, tt.envVars)

			propagateAllProxy()

			httpsProxy := getEnvWithFallback("HTTPS_PROXY", "https_proxy")
			if tt.expectSet {
				if httpsProxy != tt.expectedValue {
					t.Errorf("HTTPS_PROXY = %q, want %q", httpsProxy, tt.expectedValue)
				}
			} else {
				if httpsProxy != "" {
					t.Errorf("HTTPS_PROXY should not be set, got %q", httpsProxy)
				}
			}
		})
	}
}

func clearEnvVars(t *testing.T, keys []string) {
	t.Helper()
	for _, key := range keys {
		t.Setenv(key, "")
		os.Unsetenv(key)
	}
}

func setEnvVars(t *testing.T, vars map[string]string) {
	t.Helper()
	for key, value := range vars {
		t.Setenv(key, value)
	}
}
