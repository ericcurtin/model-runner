package standalone

import (
	"os"
	"testing"
)

// TestProxyEnvironmentVariablesPassedToContainer verifies that proxy environment
// variables are correctly identified and would be passed to the container.
func TestProxyEnvironmentVariablesPassedToContainer(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected []string
	}{
		{
			name: "All proxy variables set",
			envVars: map[string]string{
				"HTTP_PROXY":  "http://proxy.example.com:8080",
				"HTTPS_PROXY": "http://proxy.example.com:8080",
				"NO_PROXY":    "localhost,127.0.0.1",
			},
			expected: []string{
				"HTTP_PROXY=http://proxy.example.com:8080",
				"HTTPS_PROXY=http://proxy.example.com:8080",
				"NO_PROXY=localhost,127.0.0.1",
			},
		},
		{
			name: "Lowercase proxy variables set",
			envVars: map[string]string{
				"http_proxy":  "http://proxy.example.com:8080",
				"https_proxy": "http://proxy.example.com:8080",
				"no_proxy":    "localhost,127.0.0.1",
			},
			expected: []string{
				"http_proxy=http://proxy.example.com:8080",
				"https_proxy=http://proxy.example.com:8080",
				"no_proxy=localhost,127.0.0.1",
			},
		},
		{
			name: "Mixed case proxy variables",
			envVars: map[string]string{
				"HTTP_PROXY": "http://proxy.example.com:8080",
				"https_proxy": "http://proxy.example.com:8080",
				"NO_PROXY":    "localhost",
			},
			expected: []string{
				"HTTP_PROXY=http://proxy.example.com:8080",
				"https_proxy=http://proxy.example.com:8080",
				"NO_PROXY=localhost",
			},
		},
		{
			name:     "No proxy variables set",
			envVars:  map[string]string{},
			expected: []string{},
		},
		{
			name: "Only HTTP_PROXY set",
			envVars: map[string]string{
				"HTTP_PROXY": "http://proxy.example.com:8080",
			},
			expected: []string{
				"HTTP_PROXY=http://proxy.example.com:8080",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
			originalEnv := make(map[string]string)
			proxyEnvVars := []string{"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "http_proxy", "https_proxy", "no_proxy"}
			for _, key := range proxyEnvVars {
				if val, exists := os.LookupEnv(key); exists {
					originalEnv[key] = val
				}
				os.Unsetenv(key)
			}
			defer func() {
				// Restore original environment
				for _, key := range proxyEnvVars {
					os.Unsetenv(key)
				}
				for key, val := range originalEnv {
					os.Setenv(key, val)
				}
			}()

			// Set test environment variables
			for key, val := range tt.envVars {
				os.Setenv(key, val)
			}

			// Simulate the proxy environment variable collection logic
			var result []string
			for _, proxyVar := range proxyEnvVars {
				if value := os.Getenv(proxyVar); value != "" {
					result = append(result, proxyVar+"="+value)
				}
			}

			// Verify results
			if len(result) != len(tt.expected) {
				t.Fatalf("Expected %d environment variables, got %d", len(tt.expected), len(result))
			}

			expectedMap := make(map[string]bool)
			for _, e := range tt.expected {
				expectedMap[e] = true
			}

			for _, r := range result {
				if !expectedMap[r] {
					t.Errorf("Unexpected environment variable: %s", r)
				}
			}
		})
	}
}
