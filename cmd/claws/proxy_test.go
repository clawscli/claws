package main

import (
	"os"
	"testing"
)

func TestPropagateAllProxy(t *testing.T) {
	// Note: Only uppercase ALL_PROXY is supported.
	// Lowercase all_proxy is intentionally not supported to match
	// the AWS SDK behavior and keep the implementation simple.
	// Go's net/http.ProxyFromEnvironment also only checks uppercase
	// for HTTP_PROXY and HTTPS_PROXY on non-CGI environments.

	proxyVars := []string{
		"ALL_PROXY",
		"HTTP_PROXY",
		"HTTPS_PROXY",
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
		{
			name:      "lowercase all_proxy not supported",
			envVars:   map[string]string{"all_proxy": "http://proxy:8080"},
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

func clearEnvVars(t *testing.T, keys []string) {
	t.Helper()
	for _, key := range keys {
		os.Unsetenv(key)
	}
}

func setEnvVars(t *testing.T, vars map[string]string) {
	t.Helper()
	for key, value := range vars {
		t.Setenv(key, value)
	}
}
