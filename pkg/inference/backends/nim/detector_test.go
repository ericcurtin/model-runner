package nim

import (
	"testing"
)

func TestIsNIMReference(t *testing.T) {
	tests := []struct {
		name      string
		reference string
		expected  bool
	}{
		{
			name:      "Full NIM reference with nvcr.io",
			reference: "nvcr.io/nim/google/gemma-3-1b-it:latest",
			expected:  true,
		},
		{
			name:      "NIM reference without registry",
			reference: "nim/google/gemma-3-1b-it:latest",
			expected:  true,
		},
		{
			name:      "Regular HuggingFace model",
			reference: "ai/smollm2",
			expected:  false,
		},
		{
			name:      "Regular Docker Hub model",
			reference: "docker/model:latest",
			expected:  false,
		},
		{
			name:      "Different nvcr.io image (not NIM)",
			reference: "nvcr.io/nvidia/cuda:latest",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNIMReference(tt.reference)
			if result != tt.expected {
				t.Errorf("IsNIMReference(%q) = %v, want %v", tt.reference, result, tt.expected)
			}
		})
	}
}
