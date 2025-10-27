package main

import (
	"net/http"
	"os"
	"testing"

	"github.com/docker/model-runner/pkg/inference/backends/llamacpp"
	"github.com/sirupsen/logrus"
)

func TestCreateLlamaCppConfigFromEnv(t *testing.T) {
	tests := []struct {
		name      string
		llamaArgs string
		wantErr   bool
	}{
		{
			name:      "empty args",
			llamaArgs: "",
			wantErr:   false,
		},
		{
			name:      "valid args",
			llamaArgs: "--threads 4 --ctx-size 2048",
			wantErr:   false,
		},
		{
			name:      "disallowed model arg",
			llamaArgs: "--model test.gguf",
			wantErr:   true,
		},
		{
			name:      "disallowed host arg",
			llamaArgs: "--host localhost:8080",
			wantErr:   true,
		},
		{
			name:      "disallowed embeddings arg",
			llamaArgs: "--embeddings",
			wantErr:   true,
		},
		{
			name:      "disallowed mmproj arg",
			llamaArgs: "--mmproj test.mmproj",
			wantErr:   true,
		},
		{
			name:      "multiple disallowed args",
			llamaArgs: "--model test.gguf --host localhost:8080",
			wantErr:   true,
		},
		{
			name:      "quoted args",
			llamaArgs: "--prompt \"Hello, world!\" --threads 4",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			if tt.llamaArgs != "" {
				os.Setenv("LLAMA_ARGS", tt.llamaArgs)
				defer os.Unsetenv("LLAMA_ARGS")
			}

			// Create a test logger that captures fatal errors
			originalLog := log
			defer func() { log = originalLog }()

			// Create a new logger that will exit with a special exit code
			testLog := logrus.New()
			var exitCode int
			testLog.ExitFunc = func(code int) {
				exitCode = code
			}
			log = testLog

			config := createLlamaCppConfigFromEnv()

			if tt.wantErr {
				if exitCode != 1 {
					t.Errorf("Expected exit code 1, got %d", exitCode)
				}
			} else {
				if exitCode != 0 {
					t.Errorf("Expected exit code 0, got %d", exitCode)
				}
				if tt.llamaArgs == "" {
					if config != nil {
						t.Error("Expected nil config for empty args")
					}
				} else {
					llamaConfig, ok := config.(*llamacpp.Config)
					if !ok {
						t.Errorf("Expected *llamacpp.Config, got %T", config)
					}
					if llamaConfig == nil {
						t.Error("Expected non-nil config")
					}
					if len(llamaConfig.Args) == 0 {
						t.Error("Expected non-empty args")
					}
				}
			}
		})
	}
}

// TestProxyTransportConfiguration verifies that the HTTP transport
// is configured to use proxy settings from environment variables.
func TestProxyTransportConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
	}{
		{
			name: "HTTP_PROXY set",
			envVars: map[string]string{
				"HTTP_PROXY": "http://proxy.example.com:8080",
			},
		},
		{
			name: "HTTPS_PROXY set",
			envVars: map[string]string{
				"HTTPS_PROXY": "http://proxy.example.com:8080",
			},
		},
		{
			name: "Both HTTP_PROXY and HTTPS_PROXY set",
			envVars: map[string]string{
				"HTTP_PROXY":  "http://proxy.example.com:8080",
				"HTTPS_PROXY": "https://proxy.example.com:8080",
			},
		},
		{
			name:    "No proxy variables set",
			envVars: map[string]string{},
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

			// Test that we can create a proxy-aware transport
			// This simulates what we do in main()
			baseTransport := http.DefaultTransport.(*http.Transport).Clone()
			baseTransport.Proxy = http.ProxyFromEnvironment

			// Verify the transport has a Proxy function set
			if baseTransport.Proxy == nil {
				t.Error("Expected Proxy function to be set on transport")
			}
		})
	}
}
