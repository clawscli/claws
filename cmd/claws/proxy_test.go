package main

import (
	"os"
	"testing"
)

func TestPropagateAllProxy(t *testing.T) {
	proxyVars := []string{
		"ALL_PROXY",
		"HTTP_PROXY",
		"HTTPS_PROXY",
		"NO_PROXY",
	}

	tests := []struct {
		name      string
		envVars   map[string]string
		wantHTTPS string
		wantHTTP  string
	}{
		{
			name:      "ALL_PROXY propagates to both HTTP and HTTPS",
			envVars:   map[string]string{"ALL_PROXY": "socks5h://proxy:1080"},
			wantHTTPS: "socks5h://proxy:1080",
			wantHTTP:  "socks5h://proxy:1080",
		},
		{
			name:      "HTTPS_PROXY already set - only HTTP propagated",
			envVars:   map[string]string{"ALL_PROXY": "http://all", "HTTPS_PROXY": "http://existing"},
			wantHTTPS: "http://existing",
			wantHTTP:  "http://all",
		},
		{
			name:      "HTTP_PROXY already set - only HTTPS propagated",
			envVars:   map[string]string{"ALL_PROXY": "http://all", "HTTP_PROXY": "http://existing"},
			wantHTTPS: "http://all",
			wantHTTP:  "http://existing",
		},
		{
			name:      "both already set - no propagation",
			envVars:   map[string]string{"ALL_PROXY": "http://all", "HTTP_PROXY": "http://h", "HTTPS_PROXY": "http://s"},
			wantHTTPS: "http://s",
			wantHTTP:  "http://h",
		},
		{
			name:      "no ALL_PROXY - no action",
			envVars:   map[string]string{},
			wantHTTPS: "",
			wantHTTP:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnvVars(t, proxyVars)
			setEnvVars(t, tt.envVars)

			propagateAllProxy()

			if got := os.Getenv("HTTPS_PROXY"); got != tt.wantHTTPS {
				t.Errorf("HTTPS_PROXY = %q, want %q", got, tt.wantHTTPS)
			}
			if got := os.Getenv("HTTP_PROXY"); got != tt.wantHTTP {
				t.Errorf("HTTP_PROXY = %q, want %q", got, tt.wantHTTP)
			}
		})
	}
}

func TestConfigureNoProxy(t *testing.T) {
	proxyVars := []string{
		"HTTP_PROXY",
		"HTTPS_PROXY",
		"NO_PROXY",
	}

	tests := []struct {
		name        string
		envVars     map[string]string
		wantNoProxy string
	}{
		{
			name:        "no proxy set - no action",
			envVars:     map[string]string{},
			wantNoProxy: "",
		},
		{
			name:        "HTTP_PROXY set - adds IMDS",
			envVars:     map[string]string{"HTTP_PROXY": "http://proxy:8080"},
			wantNoProxy: "169.254.169.254",
		},
		{
			name:        "HTTPS_PROXY set - adds IMDS",
			envVars:     map[string]string{"HTTPS_PROXY": "http://proxy:8080"},
			wantNoProxy: "169.254.169.254",
		},
		{
			name:        "both proxies set - adds IMDS",
			envVars:     map[string]string{"HTTP_PROXY": "http://h", "HTTPS_PROXY": "http://s"},
			wantNoProxy: "169.254.169.254",
		},
		{
			name:        "existing NO_PROXY preserved and extended",
			envVars:     map[string]string{"HTTP_PROXY": "http://proxy", "NO_PROXY": "localhost,127.0.0.1"},
			wantNoProxy: "localhost,127.0.0.1,169.254.169.254",
		},
		{
			name:        "NO_PROXY already has IMDS - no change",
			envVars:     map[string]string{"HTTP_PROXY": "http://proxy", "NO_PROXY": "169.254.169.254"},
			wantNoProxy: "169.254.169.254",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnvVars(t, proxyVars)
			setEnvVars(t, tt.envVars)

			configureNoProxy()

			if got := os.Getenv("NO_PROXY"); got != tt.wantNoProxy {
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
