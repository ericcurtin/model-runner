package main

import (
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

func TestGetAllowedOrigins(t *testing.T) {
	tests := []struct {
		name              string
		dmrOrigins        string
		modelRunnerPort   string
		expectedOrigins   []string
		expectNil         bool
	}{
		{
			name:            "Unix socket mode - no DMR_ORIGINS set",
			dmrOrigins:      "",
			modelRunnerPort: "",
			expectNil:       true,
		},
		{
			name:            "TCP mode - no DMR_ORIGINS set (secure default)",
			dmrOrigins:      "",
			modelRunnerPort: "12434",
			expectedOrigins: []string{
				"http://localhost",
				"http://127.0.0.1",
				"https://localhost",
				"https://127.0.0.1",
			},
		},
		{
			name:            "DMR_ORIGINS set to single origin",
			dmrOrigins:      "http://example.com",
			modelRunnerPort: "12434",
			expectedOrigins: []string{"http://example.com"},
		},
		{
			name:            "DMR_ORIGINS set to multiple origins",
			dmrOrigins:      "http://example.com,https://example.org",
			modelRunnerPort: "12434",
			expectedOrigins: []string{"http://example.com", "https://example.org"},
		},
		{
			name:            "DMR_ORIGINS with whitespace",
			dmrOrigins:      " http://example.com , https://example.org ",
			modelRunnerPort: "12434",
			expectedOrigins: []string{"http://example.com", "https://example.org"},
		},
		{
			name:            "DMR_ORIGINS set to wildcard",
			dmrOrigins:      "*",
			modelRunnerPort: "12434",
			expectedOrigins: []string{"*"},
		},
		{
			name:            "DMR_ORIGINS set but empty (no origins allowed)",
			dmrOrigins:      "   ",
			modelRunnerPort: "12434",
			expectNil:       true,
		},
		{
			name:            "DMR_ORIGINS overrides default even in Unix socket mode",
			dmrOrigins:      "http://example.com",
			modelRunnerPort: "",
			expectedOrigins: []string{"http://example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment
			os.Unsetenv("DMR_ORIGINS")
			os.Unsetenv("MODEL_RUNNER_PORT")

			// Set up environment
			if tt.dmrOrigins != "" {
				os.Setenv("DMR_ORIGINS", tt.dmrOrigins)
			}
			if tt.modelRunnerPort != "" {
				os.Setenv("MODEL_RUNNER_PORT", tt.modelRunnerPort)
			}

			// Test
			result := getAllowedOrigins()

			if tt.expectNil {
				if result != nil {
					t.Errorf("Expected nil, got %v", result)
				}
			} else {
				if result == nil {
					t.Error("Expected non-nil result")
				} else if len(result) != len(tt.expectedOrigins) {
					t.Errorf("Expected %d origins, got %d", len(tt.expectedOrigins), len(result))
				} else {
					for i, expected := range tt.expectedOrigins {
						if result[i] != expected {
							t.Errorf("Expected origin[%d] = %s, got %s", i, expected, result[i])
						}
					}
				}
			}

			// Clean up
			os.Unsetenv("DMR_ORIGINS")
			os.Unsetenv("MODEL_RUNNER_PORT")
		})
	}
}
