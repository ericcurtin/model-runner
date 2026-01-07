package models

import (
	"encoding/json"
	"testing"

	"github.com/docker/model-runner/pkg/distribution/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		expected Model
	}{
		{
			name: "full model with all config fields",
			jsonData: `{
				"id": "sha256:abc123",
				"tags": ["ai/smollm2:latest", "ai/smollm2:1.7B-instruct-q4_K_M"],
				"created": 1704067200,
				"config": {
					"format": "gguf",
					"quantization": "Q4_K_M",
					"parameters": "1.7B",
					"architecture": "llama",
					"size": "1.7B",
					"context_size": 8192
				}
			}`,
			expected: Model{
				ID:      "sha256:abc123",
				Tags:    []string{"ai/smollm2:latest", "ai/smollm2:1.7B-instruct-q4_K_M"},
				Created: 1704067200,
				Config: &types.Config{
					Format:       "gguf",
					Quantization: "Q4_K_M",
					Parameters:   "1.7B",
					Architecture: "llama",
					Size:         "1.7B",
					ContextSize:  int32Ptr(8192),
				},
			},
		},
		{
			name: "model with minimal config",
			jsonData: `{
				"id": "sha256:def456",
				"created": 1704067200,
				"config": {
					"format": "safetensors"
				}
			}`,
			expected: Model{
				ID:      "sha256:def456",
				Tags:    nil,
				Created: 1704067200,
				Config: &types.Config{
					Format: "safetensors",
				},
			},
		},
		{
			name: "model with empty config",
			jsonData: `{
				"id": "sha256:ghi789",
				"created": 1704067200,
				"config": {}
			}`,
			expected: Model{
				ID:      "sha256:ghi789",
				Tags:    nil,
				Created: 1704067200,
				Config:  &types.Config{},
			},
		},
		{
			name: "model with gguf metadata",
			jsonData: `{
				"id": "sha256:jkl012",
				"tags": ["ai/testmodel:latest"],
				"created": 1704067200,
				"config": {
					"format": "gguf",
					"architecture": "llama",
					"gguf": {
						"llama.context_length": "4096",
						"llama.embedding_length": "2048"
					}
				}
			}`,
			expected: Model{
				ID:      "sha256:jkl012",
				Tags:    []string{"ai/testmodel:latest"},
				Created: 1704067200,
				Config: &types.Config{
					Format:       "gguf",
					Architecture: "llama",
					GGUF: map[string]string{
						"llama.context_length":   "4096",
						"llama.embedding_length": "2048",
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var model Model
			err := json.Unmarshal([]byte(tc.jsonData), &model)
			require.NoError(t, err)

			assert.Equal(t, tc.expected.ID, model.ID)
			assert.Equal(t, tc.expected.Tags, model.Tags)
			assert.Equal(t, tc.expected.Created, model.Created)

			// Verify config is properly unmarshaled
			require.NotNil(t, model.Config)
			expectedConfig := tc.expected.Config.(*types.Config)
			actualConfig, ok := model.Config.(*types.Config)
			require.True(t, ok, "Config should be *types.Config")

			assert.Equal(t, expectedConfig.Format, actualConfig.Format)
			assert.Equal(t, expectedConfig.Quantization, actualConfig.Quantization)
			assert.Equal(t, expectedConfig.Parameters, actualConfig.Parameters)
			assert.Equal(t, expectedConfig.Architecture, actualConfig.Architecture)
			assert.Equal(t, expectedConfig.Size, actualConfig.Size)
			assert.Equal(t, expectedConfig.GGUF, actualConfig.GGUF)

			if expectedConfig.ContextSize != nil {
				require.NotNil(t, actualConfig.ContextSize)
				assert.Equal(t, *expectedConfig.ContextSize, *actualConfig.ContextSize)
			} else {
				assert.Nil(t, actualConfig.ContextSize)
			}
		})
	}
}

func TestModelUnmarshalJSONArray(t *testing.T) {
	// This test simulates what the CLI does when listing models
	jsonData := `[
		{
			"id": "sha256:abc123",
			"tags": ["ai/model1:latest"],
			"created": 1704067200,
			"config": {
				"format": "gguf",
				"quantization": "Q4_K_M",
				"architecture": "llama"
			}
		},
		{
			"id": "sha256:def456",
			"tags": ["ai/model2:latest"],
			"created": 1704067300,
			"config": {
				"format": "safetensors",
				"architecture": "mistral"
			}
		}
	]`

	var models []Model
	err := json.Unmarshal([]byte(jsonData), &models)
	require.NoError(t, err)

	require.Len(t, models, 2)

	// Verify first model
	assert.Equal(t, "sha256:abc123", models[0].ID)
	assert.Equal(t, []string{"ai/model1:latest"}, models[0].Tags)
	config0, ok := models[0].Config.(*types.Config)
	require.True(t, ok)
	assert.Equal(t, types.FormatGGUF, config0.Format)
	assert.Equal(t, "Q4_K_M", config0.Quantization)
	assert.Equal(t, "llama", config0.Architecture)

	// Verify second model
	assert.Equal(t, "sha256:def456", models[1].ID)
	assert.Equal(t, []string{"ai/model2:latest"}, models[1].Tags)
	config1, ok := models[1].Config.(*types.Config)
	require.True(t, ok)
	assert.Equal(t, types.FormatSafetensors, config1.Format)
	assert.Equal(t, "mistral", config1.Architecture)
}

func TestModelJSONRoundTrip(t *testing.T) {
	// Test that marshaling and unmarshaling preserves data
	original := Model{
		ID:      "sha256:roundtrip123",
		Tags:    []string{"ai/testmodel:v1", "ai/testmodel:latest"},
		Created: 1704067200,
		Config: &types.Config{
			Format:       "gguf",
			Quantization: "Q8_0",
			Parameters:   "7B",
			Architecture: "llama",
			Size:         "7B",
			ContextSize:  int32Ptr(4096),
			GGUF: map[string]string{
				"llama.context_length": "4096",
			},
		},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(original)
	require.NoError(t, err)

	// Unmarshal back
	var unmarshaled Model
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	// Verify all fields match
	assert.Equal(t, original.ID, unmarshaled.ID)
	assert.Equal(t, original.Tags, unmarshaled.Tags)
	assert.Equal(t, original.Created, unmarshaled.Created)

	originalConfig := original.Config.(*types.Config)
	unmarshaledConfig, ok := unmarshaled.Config.(*types.Config)
	require.True(t, ok)

	assert.Equal(t, originalConfig.Format, unmarshaledConfig.Format)
	assert.Equal(t, originalConfig.Quantization, unmarshaledConfig.Quantization)
	assert.Equal(t, originalConfig.Parameters, unmarshaledConfig.Parameters)
	assert.Equal(t, originalConfig.Architecture, unmarshaledConfig.Architecture)
	assert.Equal(t, originalConfig.Size, unmarshaledConfig.Size)
	assert.Equal(t, originalConfig.GGUF, unmarshaledConfig.GGUF)
	require.NotNil(t, unmarshaledConfig.ContextSize)
	assert.Equal(t, *originalConfig.ContextSize, *unmarshaledConfig.ContextSize)
}

func TestModelUnmarshalJSONInvalidData(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
	}{
		{
			name:     "invalid JSON",
			jsonData: `{invalid}`,
		},
		{
			name:     "wrong type for id",
			jsonData: `{"id": 123, "config": {}}`,
		},
		{
			name:     "wrong type for tags",
			jsonData: `{"id": "test", "tags": "not-an-array", "config": {}}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var model Model
			err := json.Unmarshal([]byte(tc.jsonData), &model)
			assert.Error(t, err)
		})
	}
}

// Helper function to create int32 pointers
func int32Ptr(i int32) *int32 {
	return &i
}
