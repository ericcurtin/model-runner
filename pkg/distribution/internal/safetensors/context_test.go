package safetensors

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractContextSizeFromConfig(t *testing.T) {
	tests := []struct {
		name           string
		configContent  map[string]interface{}
		expectedResult uint64
	}{
		{
			name: "max_position_embeddings",
			configContent: map[string]interface{}{
				"max_position_embeddings": 2048.0,
				"model_type":              "llama",
			},
			expectedResult: 2048,
		},
		{
			name: "n_positions",
			configContent: map[string]interface{}{
				"n_positions": 1024.0,
				"model_type":  "gpt2",
			},
			expectedResult: 1024,
		},
		{
			name: "max_length",
			configContent: map[string]interface{}{
				"max_length": 512.0,
				"model_type": "t5",
			},
			expectedResult: 512,
		},
		{
			name: "n_ctx",
			configContent: map[string]interface{}{
				"n_ctx":      4096.0,
				"model_type": "custom",
			},
			expectedResult: 4096,
		},
		{
			name: "no_context_field",
			configContent: map[string]interface{}{
				"model_type":  "test",
				"vocab_size":  32000.0,
				"hidden_size": 768.0,
			},
			expectedResult: 0,
		},
		{
			name: "invalid_negative_value",
			configContent: map[string]interface{}{
				"max_position_embeddings": -1.0,
			},
			expectedResult: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.json")

			// Write config to file
			data, err := json.Marshal(tt.configContent)
			if err != nil {
				t.Fatalf("Failed to marshal config: %v", err)
			}
			if err := os.WriteFile(configPath, data, 0644); err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			// Test extraction
			result := extractContextSizeFromConfig(configPath)
			if result != tt.expectedResult {
				t.Errorf("Expected context size %d, got %d", tt.expectedResult, result)
			}
		})
	}
}

func TestExtractContextSizeFromConfig_FileNotFound(t *testing.T) {
	result := extractContextSizeFromConfig("/nonexistent/config.json")
	if result != 0 {
		t.Errorf("Expected 0 for non-existent file, got %d", result)
	}
}

func TestExtractContextSizeFromConfig_InvalidJSON(t *testing.T) {
	// Create temporary file with invalid JSON
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	if err := os.WriteFile(configPath, []byte("not valid json{"), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	result := extractContextSizeFromConfig(configPath)
	if result != 0 {
		t.Errorf("Expected 0 for invalid JSON, got %d", result)
	}
}
