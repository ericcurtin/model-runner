package model

import "testing"

func TestIsOllamaModel(t *testing.T) {
	tests := []struct {
		name      string
		modelName string
		want      bool
	}{
		{
			name:      "ollama model with full path",
			modelName: "ollama.com/library/smollm:135m",
			want:      true,
		},
		{
			name:      "ollama model simple",
			modelName: "ollama.com/llama2",
			want:      true,
		},
		{
			name:      "docker hub model",
			modelName: "docker.io/library/llama2",
			want:      false,
		},
		{
			name:      "simple model name",
			modelName: "llama2",
			want:      false,
		},
		{
			name:      "huggingface model",
			modelName: "hf.co/meta-llama/Llama-2-7b-hf",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsOllamaModel(tt.modelName); got != tt.want {
				t.Errorf("IsOllamaModel(%q) = %v, want %v", tt.modelName, got, tt.want)
			}
		})
	}
}

func TestStripOllamaPrefix(t *testing.T) {
	tests := []struct {
		name      string
		modelName string
		want      string
	}{
		{
			name:      "ollama model with full path",
			modelName: "ollama.com/library/smollm:135m",
			want:      "library/smollm:135m",
		},
		{
			name:      "ollama model simple",
			modelName: "ollama.com/llama2",
			want:      "llama2",
		},
		{
			name:      "non-ollama model",
			modelName: "docker.io/library/llama2",
			want:      "docker.io/library/llama2",
		},
		{
			name:      "simple model name",
			modelName: "llama2",
			want:      "llama2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StripOllamaPrefix(tt.modelName); got != tt.want {
				t.Errorf("StripOllamaPrefix(%q) = %q, want %q", tt.modelName, got, tt.want)
			}
		})
	}
}
