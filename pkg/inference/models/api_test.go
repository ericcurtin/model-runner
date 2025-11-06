package models

import (
	"testing"
	"time"

	"github.com/docker/model-runner/pkg/distribution/types"
)

// mockModel is a test implementation of types.Model
type mockModel struct {
	id         string
	tags       []string
	config     types.Config
	descriptor types.Descriptor
}

func (m *mockModel) ID() (string, error) {
	return m.id, nil
}

func (m *mockModel) Tags() []string {
	return m.tags
}

func (m *mockModel) Config() (types.Config, error) {
	return m.config, nil
}

func (m *mockModel) Descriptor() (types.Descriptor, error) {
	return m.descriptor, nil
}

func (m *mockModel) GGUFPaths() ([]string, error) {
	return nil, nil
}

func (m *mockModel) SafetensorsPaths() ([]string, error) {
	return nil, nil
}

func (m *mockModel) ConfigArchivePath() (string, error) {
	return "", nil
}

func (m *mockModel) MMPROJPath() (string, error) {
	return "", nil
}

func (m *mockModel) ChatTemplatePath() (string, error) {
	return "", nil
}

func TestToOpenAI(t *testing.T) {
	createdTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	contextSize := uint64(8192)

	tests := []struct {
		name     string
		model    *mockModel
		validate func(t *testing.T, result *OpenAIModel)
	}{
		{
			name: "basic model without metadata",
			model: &mockModel{
				id:     "sha256:abc123",
				tags:   []string{"llama3.2:latest"},
				config: types.Config{},
				descriptor: types.Descriptor{
					Created: &createdTime,
				},
			},
			validate: func(t *testing.T, result *OpenAIModel) {
				if result.ID != "llama3.2:latest" {
					t.Errorf("Expected ID 'llama3.2:latest', got '%s'", result.ID)
				}
				if result.Object != "model" {
					t.Errorf("Expected Object 'model', got '%s'", result.Object)
				}
				if result.OwnedBy != "docker" {
					t.Errorf("Expected OwnedBy 'docker', got '%s'", result.OwnedBy)
				}
				if result.Created != createdTime.Unix() {
					t.Errorf("Expected Created %d, got %d", createdTime.Unix(), result.Created)
				}
				// New fields should be empty/nil for models without metadata
				if result.Architecture != "" {
					t.Errorf("Expected empty Architecture, got '%s'", result.Architecture)
				}
				if result.ContextLength != nil {
					t.Errorf("Expected nil ContextLength, got %v", result.ContextLength)
				}
			},
		},
		{
			name: "model with full metadata",
			model: &mockModel{
				id:   "sha256:def456",
				tags: []string{"llama3:8b-instruct-q4"},
				config: types.Config{
					Format:       types.FormatGGUF,
					Quantization: "Q4_K_M",
					Parameters:   "8B",
					Architecture: "llama",
					ContextSize:  &contextSize,
				},
				descriptor: types.Descriptor{
					Created: &createdTime,
				},
			},
			validate: func(t *testing.T, result *OpenAIModel) {
				if result.ID != "llama3:8b-instruct-q4" {
					t.Errorf("Expected ID 'llama3:8b-instruct-q4', got '%s'", result.ID)
				}
				if result.Architecture != "llama" {
					t.Errorf("Expected Architecture 'llama', got '%s'", result.Architecture)
				}
				if result.ContextLength == nil || *result.ContextLength != 8192 {
					t.Errorf("Expected ContextLength 8192, got %v", result.ContextLength)
				}
				if result.Format != "gguf" {
					t.Errorf("Expected Format 'gguf', got '%s'", result.Format)
				}
				if result.Quantization != "Q4_K_M" {
					t.Errorf("Expected Quantization 'Q4_K_M', got '%s'", result.Quantization)
				}
				if result.Parameters != "8B" {
					t.Errorf("Expected Parameters '8B', got '%s'", result.Parameters)
				}
			},
		},
		{
			name: "model with partial metadata",
			model: &mockModel{
				id:   "sha256:ghi789",
				tags: []string{"mistral:7b"},
				config: types.Config{
					Architecture: "mistral",
					Parameters:   "7B",
					// No format, quantization, or context size
				},
				descriptor: types.Descriptor{
					Created: &createdTime,
				},
			},
			validate: func(t *testing.T, result *OpenAIModel) {
				if result.Architecture != "mistral" {
					t.Errorf("Expected Architecture 'mistral', got '%s'", result.Architecture)
				}
				if result.Parameters != "7B" {
					t.Errorf("Expected Parameters '7B', got '%s'", result.Parameters)
				}
				if result.Format != "" {
					t.Errorf("Expected empty Format, got '%s'", result.Format)
				}
				if result.Quantization != "" {
					t.Errorf("Expected empty Quantization, got '%s'", result.Quantization)
				}
				if result.ContextLength != nil {
					t.Errorf("Expected nil ContextLength, got %v", result.ContextLength)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ToOpenAI(tt.model)
			if err != nil {
				t.Fatalf("ToOpenAI() error = %v", err)
			}
			tt.validate(t, result)
		})
	}
}

func TestToOpenAIList(t *testing.T) {
	createdTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	contextSize := uint64(4096)

	models := []types.Model{
		&mockModel{
			id:   "sha256:model1",
			tags: []string{"model1:latest"},
			config: types.Config{
				Architecture: "llama",
				Parameters:   "7B",
			},
			descriptor: types.Descriptor{
				Created: &createdTime,
			},
		},
		&mockModel{
			id:   "sha256:model2",
			tags: []string{"model2:v1"},
			config: types.Config{
				Format:       types.FormatGGUF,
				Quantization: "Q8_0",
				ContextSize:  &contextSize,
			},
			descriptor: types.Descriptor{
				Created: &createdTime,
			},
		},
	}

	result, err := ToOpenAIList(models)
	if err != nil {
		t.Fatalf("ToOpenAIList() error = %v", err)
	}

	if result.Object != "list" {
		t.Errorf("Expected Object 'list', got '%s'", result.Object)
	}

	if len(result.Data) != 2 {
		t.Fatalf("Expected 2 models, got %d", len(result.Data))
	}

	// Validate first model
	if result.Data[0].ID != "model1:latest" {
		t.Errorf("Expected first model ID 'model1:latest', got '%s'", result.Data[0].ID)
	}
	if result.Data[0].Architecture != "llama" {
		t.Errorf("Expected first model Architecture 'llama', got '%s'", result.Data[0].Architecture)
	}

	// Validate second model
	if result.Data[1].ID != "model2:v1" {
		t.Errorf("Expected second model ID 'model2:v1', got '%s'", result.Data[1].ID)
	}
	if result.Data[1].Format != "gguf" {
		t.Errorf("Expected second model Format 'gguf', got '%s'", result.Data[1].Format)
	}
}

func TestToOpenAIList_EmptyList(t *testing.T) {
	result, err := ToOpenAIList([]types.Model{})
	if err != nil {
		t.Fatalf("ToOpenAIList() error = %v", err)
	}

	if result.Object != "list" {
		t.Errorf("Expected Object 'list', got '%s'", result.Object)
	}

	if len(result.Data) != 0 {
		t.Errorf("Expected empty Data slice, got %d items", len(result.Data))
	}
}
