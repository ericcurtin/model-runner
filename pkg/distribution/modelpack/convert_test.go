package modelpack

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/docker/model-runner/pkg/distribution/types"
	"github.com/opencontainers/go-digest"
)

func TestIsModelPackMediaType(t *testing.T) {
	tests := []struct {
		name      string
		mediaType string
		expected  bool
	}{
		{
			name:      "CNCF v1 config",
			mediaType: "application/vnd.cncf.model.config.v1+json",
			expected:  true,
		},
		{
			name:      "CNCF future version",
			mediaType: "application/vnd.cncf.model.config.v2+json",
			expected:  true,
		},
		{
			name:      "CNCF weight media type",
			mediaType: "application/vnd.cncf.model.weight.v1.raw",
			expected:  true,
		},
		{
			name:      "Docker format",
			mediaType: "application/vnd.docker.ai.model.config.v0.1+json",
			expected:  false,
		},
		{
			name:      "Generic JSON",
			mediaType: "application/json",
			expected:  false,
		},
		{
			name:      "Empty string",
			mediaType: "",
			expected:  false,
		},
		{
			name:      "OCI image config",
			mediaType: "application/vnd.oci.image.config.v1+json",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsModelPackMediaType(tt.mediaType)
			if result != tt.expected {
				t.Errorf("IsModelPackMediaType(%q) = %v, want %v", tt.mediaType, result, tt.expected)
			}
		})
	}
}

func TestConvertToDockerConfig(t *testing.T) {
	t.Run("full config conversion", func(t *testing.T) {
		created := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
		knowledgeCutoff := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
		reasoning := true
		toolUsage := true

		mpConfig := Model{
			Descriptor: ModelDescriptor{
				CreatedAt:   &created,
				Authors:     []string{"Author1", "Author2"},
				Family:      "llama",
				Name:        "llama3-8b-instruct",
				DocURL:      "https://example.com/docs",
				SourceURL:   "https://example.com/source",
				DatasetsURL: []string{"https://example.com/dataset1", "https://example.com/dataset2"},
				Version:     "1.0.0",
				Revision:    "abc123",
				Vendor:      "TestVendor",
				Licenses:    []string{"MIT", "Apache-2.0"},
				Title:       "Llama 3 8B Instruct",
				Description: "A test model for testing",
			},
			Config: ModelConfig{
				Architecture: "transformer",
				Format:       "gguf",
				ParamSize:    "8B",
				Precision:    "fp16",
				Quantization: "Q4_K_M",
				Capabilities: &ModelCapabilities{
					InputTypes:      []string{"text"},
					OutputTypes:     []string{"text"},
					KnowledgeCutoff: &knowledgeCutoff,
					Reasoning:       &reasoning,
					ToolUsage:       &toolUsage,
					Languages:       []string{"en", "zh"},
				},
			},
			ModelFS: ModelFS{
				Type:    "layers",
				DiffIDs: []digest.Digest{"sha256:abc123def456abc123def456abc123def456abc123def456abc123def456abc1"},
			},
		}

		rawConfig, err := json.Marshal(mpConfig)
		if err != nil {
			t.Fatalf("Failed to marshal test config: %v", err)
		}

		dockerConfig, err := ConvertToDockerConfig(rawConfig)
		if err != nil {
			t.Fatalf("ConvertToDockerConfig failed: %v", err)
		}

		// Verify direct field mappings
		if dockerConfig.Config.Format != types.FormatGGUF {
			t.Errorf("Format = %v, want %v", dockerConfig.Config.Format, types.FormatGGUF)
		}
		if dockerConfig.Config.Architecture != "transformer" {
			t.Errorf("Architecture = %q, want %q", dockerConfig.Config.Architecture, "transformer")
		}
		if dockerConfig.Config.Quantization != "Q4_K_M" {
			t.Errorf("Quantization = %q, want %q", dockerConfig.Config.Quantization, "Q4_K_M")
		}
		if dockerConfig.Config.Parameters != "8B" {
			t.Errorf("Parameters = %q, want %q", dockerConfig.Config.Parameters, "8B")
		}
		if dockerConfig.Config.Size != "0" {
			t.Errorf("Size = %q, want %q", dockerConfig.Config.Size, "0")
		}

		// Verify descriptor
		if dockerConfig.Descriptor.Created == nil {
			t.Error("Descriptor.Created should not be nil")
		} else if !dockerConfig.Descriptor.Created.Equal(created) {
			t.Errorf("Descriptor.Created = %v, want %v", dockerConfig.Descriptor.Created, created)
		}

		// Verify RootFS
		if dockerConfig.RootFS.Type != "layers" {
			t.Errorf("RootFS.Type = %q, want %q", dockerConfig.RootFS.Type, "layers")
		}
		if len(dockerConfig.RootFS.DiffIDs) != 1 {
			t.Errorf("RootFS.DiffIDs length = %d, want 1", len(dockerConfig.RootFS.DiffIDs))
		}
		// Note: Extended metadata (ModelPack field) is no longer preserved since
		// types.Config no longer has a ModelPack field. Native format support (Option B)
		// handles ModelPack configs directly without conversion.
	})

	t.Run("minimal config", func(t *testing.T) {
		mpConfig := Model{
			Config: ModelConfig{
				Format: "gguf",
			},
			ModelFS: ModelFS{
				Type:    "layers",
				DiffIDs: []digest.Digest{"sha256:abc123"},
			},
		}

		rawConfig, _ := json.Marshal(mpConfig)
		dockerConfig, err := ConvertToDockerConfig(rawConfig)
		if err != nil {
			t.Fatalf("ConvertToDockerConfig failed for minimal config: %v", err)
		}

		if dockerConfig.Config.Format != types.FormatGGUF {
			t.Errorf("Format = %v, want %v", dockerConfig.Config.Format, types.FormatGGUF)
		}
	})

	t.Run("empty config", func(t *testing.T) {
		mpConfig := Model{}
		rawConfig, _ := json.Marshal(mpConfig)

		dockerConfig, err := ConvertToDockerConfig(rawConfig)
		if err != nil {
			t.Fatalf("ConvertToDockerConfig failed for empty config: %v", err)
		}

		if dockerConfig.Config.Format != "" {
			t.Errorf("Format should be empty, got %v", dockerConfig.Config.Format)
		}
		if dockerConfig.RootFS.Type != "layers" {
			t.Errorf("RootFS.Type should default to 'layers', got %q", dockerConfig.RootFS.Type)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		_, err := ConvertToDockerConfig([]byte("invalid json"))
		if err == nil {
			t.Error("Expected error for invalid JSON, got nil")
		}
	})

	t.Run("empty input", func(t *testing.T) {
		_, err := ConvertToDockerConfig([]byte(""))
		if err == nil {
			t.Error("Expected error for empty input, got nil")
		}
	})
}

func TestConvertFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected types.Format
	}{
		{"gguf", types.FormatGGUF},
		{"GGUF", types.FormatGGUF},
		{"GgUf", types.FormatGGUF},
		{"safetensors", types.FormatSafetensors},
		{"SafeTensors", types.FormatSafetensors},
		{"SAFETENSORS", types.FormatSafetensors},
		{"onnx", types.Format("onnx")},
		{"pytorch", types.Format("pytorch")},
		{"", types.Format("")},
		{"unknown", types.Format("unknown")},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertFormat(tt.input)
			if result != tt.expected {
				t.Errorf("convertFormat(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestConvertDiffIDs(t *testing.T) {
	t.Run("valid digests", func(t *testing.T) {
		digests := []digest.Digest{
			"sha256:abc123def456abc123def456abc123def456abc123def456abc123def456abc1",
			"sha256:123456789012345678901234567890123456789012345678901234567890abcd",
		}

		result := convertDiffIDs(digests)
		if len(result) != 2 {
			t.Errorf("Expected 2 hashes, got %d", len(result))
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		result := convertDiffIDs([]digest.Digest{})
		if result != nil {
			t.Errorf("Expected nil for empty slice, got %v", result)
		}
	})

	t.Run("nil slice", func(t *testing.T) {
		result := convertDiffIDs(nil)
		if result != nil {
			t.Errorf("Expected nil for nil slice, got %v", result)
		}
	})

	t.Run("invalid digest skipped", func(t *testing.T) {
		digests := []digest.Digest{
			"sha256:abc123def456abc123def456abc123def456abc123def456abc123def456abc1",
			"invalid-digest-format", // This should be skipped
			"sha256:123456789012345678901234567890123456789012345678901234567890abcd",
		}

		result := convertDiffIDs(digests)
		// Should only have 2 valid hashes, invalid one skipped
		if len(result) != 2 {
			t.Errorf("Expected 2 valid hashes (invalid skipped), got %d", len(result))
		}
	})
}

// Note: TestExtractExtendedMetadata was removed because the extractExtendedMetadata
// function was removed. With Option B (native format support), ModelPack configs
// are handled directly without conversion to Docker format.

func TestNormalizeRootFSType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"layers", "layers"},
		{"", "layers"},
		{"rootfs", "rootfs"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeRootFSType(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeRootFSType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestMapLayerMediaType tests layer media type mapping
func TestMapLayerMediaType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// ModelPack GGUF related media types
		{
			name:     "ModelPack weight gguf v1",
			input:    "application/vnd.cncf.model.weight.v1.gguf",
			expected: "application/vnd.docker.ai.gguf.v3",
		},
		{
			name:     "ModelPack weight gguf no version",
			input:    "application/vnd.cncf.model.weight.gguf",
			expected: "application/vnd.docker.ai.gguf.v3",
		},
		// ModelPack safetensors related
		{
			name:     "ModelPack weight safetensors",
			input:    "application/vnd.cncf.model.weight.v1.safetensors",
			expected: "application/vnd.docker.ai.safetensors",
		},
		// Docker format passthrough
		{
			name:     "Docker GGUF passthrough",
			input:    "application/vnd.docker.ai.gguf.v3",
			expected: "application/vnd.docker.ai.gguf.v3",
		},
		{
			name:     "Docker safetensors passthrough",
			input:    "application/vnd.docker.ai.safetensors",
			expected: "application/vnd.docker.ai.safetensors",
		},
		// Other types unchanged
		{
			name:     "generic octet-stream",
			input:    "application/octet-stream",
			expected: "application/octet-stream",
		},
		{
			name:     "ModelPack doc layer unchanged",
			input:    "application/vnd.cncf.model.doc.v1",
			expected: "application/vnd.cncf.model.doc.v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapLayerMediaType(tt.input)
			if got != tt.expected {
				t.Errorf("MapLayerMediaType(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestIsModelPackConfig tests detecting ModelPack format from raw config bytes
func TestIsModelPackConfig(t *testing.T) {
	// Prepare test ModelPack format config (has paramSize field)
	modelPackConfig := `{
		"descriptor": {"createdAt": "2025-01-15T10:30:00Z"},
		"config": {"paramSize": "8B", "format": "gguf"}
	}`

	// Docker format config (uses parameters instead of paramSize)
	dockerConfig := `{
		"config": {"parameters": "8B", "format": "gguf"},
		"descriptor": {"created": "2025-01-15T10:30:00Z"}
	}`

	tests := []struct {
		name     string
		input    []byte
		expected bool
	}{
		{
			name:     "ModelPack config with paramSize",
			input:    []byte(modelPackConfig),
			expected: true,
		},
		{
			name:     "Docker config with parameters",
			input:    []byte(dockerConfig),
			expected: false,
		},
		{
			name:     "empty JSON object",
			input:    []byte("{}"),
			expected: false,
		},
		{
			name:     "invalid JSON",
			input:    []byte("not json"),
			expected: false,
		},
		{
			name:     "nil input",
			input:    nil,
			expected: false,
		},
		{
			name:     "empty input",
			input:    []byte(""),
			expected: false,
		},
		{
			name:     "config with createdAt field",
			input:    []byte(`{"descriptor": {"createdAt": "2025-01-01T00:00:00Z"}}`),
			expected: true,
		},
		{
			name:     "config with modelfs field",
			input:    []byte(`{"modelfs": {"type": "layers", "diffIds": []}}`),
			expected: true,
		},
		{
			name:     "false positive prevention - paramSize as value",
			input:    []byte(`{"config": {"description": "paramSize is 8B"}}`),
			expected: false,
		},
		{
			name:     "false positive prevention - createdAt as value",
			input:    []byte(`{"descriptor": {"note": "createdAt was yesterday"}}`),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsModelPackConfig(tt.input)
			if got != tt.expected {
				t.Errorf("IsModelPackConfig() = %v, want %v", got, tt.expected)
			}
		})
	}
}
