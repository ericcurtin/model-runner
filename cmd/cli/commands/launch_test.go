package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetModelRunnerURL(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected string
	}{
		{
			name:     "default URL",
			envValue: "",
			expected: "http://localhost:12434/engines/llama.cpp/v1",
		},
		{
			name:     "custom URL without trailing slash",
			envValue: "http://localhost:13434",
			expected: "http://localhost:13434/engines/llama.cpp/v1",
		},
		{
			name:     "custom URL with trailing slash",
			envValue: "http://localhost:13434/",
			expected: "http://localhost:13434/engines/llama.cpp/v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore environment
			originalValue := os.Getenv("MODEL_RUNNER_HOST")
			defer os.Setenv("MODEL_RUNNER_HOST", originalValue)

			if tt.envValue != "" {
				os.Setenv("MODEL_RUNNER_HOST", tt.envValue)
			} else {
				os.Unsetenv("MODEL_RUNNER_HOST")
			}

			result := getModelRunnerURL()
			if result != tt.expected {
				t.Errorf("getModelRunnerURL() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestLaunchConfigPath(t *testing.T) {
	path, err := launchConfigPath()
	if err != nil {
		t.Fatalf("launchConfigPath() error = %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	expected := filepath.Join(home, ".docker", "model-cli", "launch.json")
	if path != expected {
		t.Errorf("launchConfigPath() = %q, want %q", path, expected)
	}
}

func TestLoadSaveLaunchConfig(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	// Test loading empty config
	cfg, err := loadLaunchConfig()
	if err != nil {
		t.Fatalf("loadLaunchConfig() error = %v", err)
	}
	if cfg == nil || cfg.Integrations == nil {
		t.Fatal("loadLaunchConfig() returned nil config or nil Integrations map")
	}
	if len(cfg.Integrations) != 0 {
		t.Errorf("loadLaunchConfig() returned non-empty Integrations: %v", cfg.Integrations)
	}

	// Test saving and loading integration
	err = saveLaunchIntegration("claude", []string{"gemma3", "smollm2"})
	if err != nil {
		t.Fatalf("saveLaunchIntegration() error = %v", err)
	}

	integration, err := loadLaunchIntegration("claude")
	if err != nil {
		t.Fatalf("loadLaunchIntegration() error = %v", err)
	}
	if len(integration.Models) != 2 {
		t.Errorf("loadLaunchIntegration() models count = %d, want 2", len(integration.Models))
	}
	if integration.Models[0] != "gemma3" {
		t.Errorf("loadLaunchIntegration() models[0] = %q, want %q", integration.Models[0], "gemma3")
	}

	// Test case insensitivity
	integration2, err := loadLaunchIntegration("CLAUDE")
	if err != nil {
		t.Fatalf("loadLaunchIntegration(CLAUDE) error = %v", err)
	}
	if len(integration2.Models) != 2 {
		t.Errorf("loadLaunchIntegration(CLAUDE) models count = %d, want 2", len(integration2.Models))
	}
}

func TestFilterLaunchItems(t *testing.T) {
	items := []launchSelectItem{
		{Name: "claude", Description: "Claude Code"},
		{Name: "codex", Description: "Codex"},
		{Name: "opencode", Description: "OpenCode"},
		{Name: "openwebui", Description: "Open WebUI"},
		{Name: "anythingllm", Description: "AnythingLLM"},
	}

	tests := []struct {
		name     string
		filter   string
		expected int
	}{
		{
			name:     "no filter",
			filter:   "",
			expected: 5,
		},
		{
			name:     "filter 'open'",
			filter:   "open",
			expected: 2, // opencode, openwebui
		},
		{
			name:     "filter 'claude'",
			filter:   "claude",
			expected: 1,
		},
		{
			name:     "filter 'code'",
			filter:   "code",
			expected: 2, // codex, opencode
		},
		{
			name:     "filter 'xyz' (no match)",
			filter:   "xyz",
			expected: 0,
		},
		{
			name:     "case insensitive filter",
			filter:   "CLAUDE",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterLaunchItems(items, tt.filter)
			if len(result) != tt.expected {
				t.Errorf("filterLaunchItems() returned %d items, want %d", len(result), tt.expected)
			}
		})
	}
}

func TestIntegrationsRegistry(t *testing.T) {
	// Verify all expected integrations are registered
	expectedIntegrations := []string{"anythingllm", "claude", "codex", "opencode", "openwebui"}

	for _, name := range expectedIntegrations {
		if _, ok := integrations[name]; !ok {
			t.Errorf("integration %q not found in registry", name)
		}
	}

	// Verify integration count
	if len(integrations) != len(expectedIntegrations) {
		t.Errorf("expected %d integrations, got %d", len(expectedIntegrations), len(integrations))
	}
}

func TestIntegrationsImplementRunner(t *testing.T) {
	// Verify all integrations implement Runner interface and have String()
	for name, runner := range integrations {
		if runner == nil {
			t.Errorf("integration %q is nil", name)
			continue
		}

		displayName := runner.String()
		if displayName == "" {
			t.Errorf("integration %q has empty String()", name)
		}
	}
}

func TestCheckCodexVersion(t *testing.T) {
	// Skip if codex is not installed
	err := checkCodexVersion()
	if err != nil && err.Error() == "codex is not installed, install with: npm install -g @openai/codex" {
		t.Skip("codex is not installed, skipping version check test")
	}
	// If codex is installed, this test should either pass or fail with a version error
}

func TestClaudeArgs(t *testing.T) {
	claude := &Claude{}

	tests := []struct {
		name     string
		model    string
		expected []string
	}{
		{
			name:     "with model",
			model:    "gemma3",
			expected: []string{"--model", "gemma3"},
		},
		{
			name:     "empty model",
			model:    "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := claude.args(tt.model)
			if len(result) != len(tt.expected) {
				t.Errorf("claude.args(%q) = %v, want %v", tt.model, result, tt.expected)
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("claude.args(%q)[%d] = %q, want %q", tt.model, i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestCodexArgs(t *testing.T) {
	codex := &Codex{}

	tests := []struct {
		name     string
		model    string
		expected []string
	}{
		{
			name:     "with model",
			model:    "gemma3",
			expected: []string{"--oss", "-m", "gemma3"},
		},
		{
			name:     "empty model",
			model:    "",
			expected: []string{"--oss"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := codex.args(tt.model)
			if len(result) != len(tt.expected) {
				t.Errorf("codex.args(%q) = %v, want %v", tt.model, result, tt.expected)
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("codex.args(%q)[%d] = %q, want %q", tt.model, i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestIntegrationStrings(t *testing.T) {
	tests := []struct {
		name     string
		runner   Runner
		expected string
	}{
		{name: "AnythingLLM", runner: &AnythingLLM{}, expected: "AnythingLLM"},
		{name: "Claude", runner: &Claude{}, expected: "Claude Code"},
		{name: "Codex", runner: &Codex{}, expected: "Codex"},
		{name: "OpenCode", runner: &OpenCode{}, expected: "OpenCode"},
		{name: "OpenWebUI", runner: &OpenWebUI{}, expected: "Open WebUI"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.runner.String()
			if result != tt.expected {
				t.Errorf("%s.String() = %q, want %q", tt.name, result, tt.expected)
			}
		})
	}
}

func TestOpenCodePaths(t *testing.T) {
	opencode := &OpenCode{}
	// Just verify it doesn't panic
	_ = opencode.Paths()
}

func TestOpenCodeModels(t *testing.T) {
	opencode := &OpenCode{}
	// Just verify it doesn't panic and returns nil or a slice
	models := opencode.Models()
	if models != nil {
		t.Logf("OpenCode.Models() returned %d models", len(models))
	}
}
