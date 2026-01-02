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
	proxyVars := []string{
		"ALL_PROXY", "all_proxy",
		"HTTP_PROXY", "http_proxy",
		"HTTPS_PROXY", "https_proxy",
		"NO_PROXY", "no_proxy",
	}

	tests := []struct {
		name           string
		envVars        map[string]string
		wantHTTPS      string
		wantHTTP       string
		wantNoProxy    string
		noProxyChanged bool
	}{
		{
			name:           "ALL_PROXY propagates to both HTTP and HTTPS",
			envVars:        map[string]string{"ALL_PROXY": "socks5h://proxy:1080"},
			wantHTTPS:      "socks5h://proxy:1080",
			wantHTTP:       "socks5h://proxy:1080",
			wantNoProxy:    "169.254.169.254,169.254.170.2,169.254.170.23",
			noProxyChanged: true,
		},
		{
			name:           "all_proxy (lowercase) propagates",
			envVars:        map[string]string{"all_proxy": "socks5h://proxy:1080"},
			wantHTTPS:      "socks5h://proxy:1080",
			wantHTTP:       "socks5h://proxy:1080",
			wantNoProxy:    "169.254.169.254,169.254.170.2,169.254.170.23",
			noProxyChanged: true,
		},
		{
			name:           "ALL_PROXY takes precedence over all_proxy",
			envVars:        map[string]string{"ALL_PROXY": "http://upper", "all_proxy": "http://lower"},
			wantHTTPS:      "http://upper",
			wantHTTP:       "http://upper",
			wantNoProxy:    "169.254.169.254,169.254.170.2,169.254.170.23",
			noProxyChanged: true,
		},
		{
			name:           "HTTPS_PROXY already set - only HTTP propagated",
			envVars:        map[string]string{"ALL_PROXY": "http://all", "HTTPS_PROXY": "http://existing"},
			wantHTTPS:      "http://existing",
			wantHTTP:       "http://all",
			wantNoProxy:    "169.254.169.254,169.254.170.2,169.254.170.23",
			noProxyChanged: true,
		},
		{
			name:           "HTTP_PROXY already set - only HTTPS propagated, NO_PROXY unchanged",
			envVars:        map[string]string{"ALL_PROXY": "http://all", "HTTP_PROXY": "http://existing"},
			wantHTTPS:      "http://all",
			wantHTTP:       "http://existing",
			wantNoProxy:    "",
			noProxyChanged: false,
		},
		{
			name:           "both already set - no propagation",
			envVars:        map[string]string{"ALL_PROXY": "http://all", "HTTP_PROXY": "http://h", "HTTPS_PROXY": "http://s"},
			wantHTTPS:      "http://s",
			wantHTTP:       "http://h",
			wantNoProxy:    "",
			noProxyChanged: false,
		},
		{
			name:           "no ALL_PROXY - no action",
			envVars:        map[string]string{},
			wantHTTPS:      "",
			wantHTTP:       "",
			wantNoProxy:    "",
			noProxyChanged: false,
		},
		{
			name:           "existing NO_PROXY is preserved and extended",
			envVars:        map[string]string{"ALL_PROXY": "http://proxy", "NO_PROXY": "localhost,127.0.0.1"},
			wantHTTPS:      "http://proxy",
			wantHTTP:       "http://proxy",
			wantNoProxy:    "localhost,127.0.0.1,169.254.169.254,169.254.170.2,169.254.170.23",
			noProxyChanged: true,
		},
		{
			name:           "NO_PROXY with partial AWS endpoints - only missing added",
			envVars:        map[string]string{"ALL_PROXY": "http://proxy", "NO_PROXY": "169.254.169.254"},
			wantHTTPS:      "http://proxy",
			wantHTTP:       "http://proxy",
			wantNoProxy:    "169.254.169.254,169.254.170.2,169.254.170.23",
			noProxyChanged: true,
		},
		{
			name:           "NO_PROXY already has all AWS endpoints - no change",
			envVars:        map[string]string{"ALL_PROXY": "http://proxy", "NO_PROXY": "169.254.169.254,169.254.170.2,169.254.170.23"},
			wantHTTPS:      "http://proxy",
			wantHTTP:       "http://proxy",
			wantNoProxy:    "169.254.169.254,169.254.170.2,169.254.170.23",
			noProxyChanged: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnvVars(t, proxyVars)
			setEnvVars(t, tt.envVars)

			propagateAllProxy()

			if got := getEnvWithFallback("HTTPS_PROXY", "https_proxy"); got != tt.wantHTTPS {
				t.Errorf("HTTPS_PROXY = %q, want %q", got, tt.wantHTTPS)
			}
			if got := getEnvWithFallback("HTTP_PROXY", "http_proxy"); got != tt.wantHTTP {
				t.Errorf("HTTP_PROXY = %q, want %q", got, tt.wantHTTP)
			}
			if got := getEnvWithFallback("NO_PROXY", "no_proxy"); got != tt.wantNoProxy {
				t.Errorf("NO_PROXY = %q, want %q", got, tt.wantNoProxy)
			}
		})
	}
}

func TestSplitNoProxy(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"", nil},
		{"localhost", []string{"localhost"}},
		{"a,b,c", []string{"a", "b", "c"}},
		{"a, b, c", []string{"a", "b", "c"}},
		{" a , b , c ", []string{"a", "b", "c"}},
		{"a,,b", []string{"a", "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := splitNoProxy(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("splitNoProxy(%q) = %v, want %v", tt.input, got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitNoProxy(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
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
