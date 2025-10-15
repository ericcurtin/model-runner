package commands

import (
	"testing"

	"github.com/docker/model-runner/cmd/cli/pkg/ollama"
)

func TestOllamaModelNameExtraction(t *testing.T) {
	testCases := []struct {
		name          string
		inputModel    string
		expectedModel string
	}{
		{
			name:          "library model with tag",
			inputModel:    "ollama.com/library/smollm:135m",
			expectedModel: "library/smollm:135m",
		},
		{
			name:          "user model with tag",
			inputModel:    "ollama.com/user/custom-model:latest",
			expectedModel: "user/custom-model:latest",
		},
		{
			name:          "simple model",
			inputModel:    "ollama.com/llama2",
			expectedModel: "llama2",
		},
		{
			name:          "model with namespace",
			inputModel:    "ollama.com/library/llama2:7b",
			expectedModel: "library/llama2:7b",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ollama.ExtractModelName(tc.inputModel)
			if result != tc.expectedModel {
				t.Errorf("ExtractModelName(%q) = %q, expected %q", tc.inputModel, result, tc.expectedModel)
			}
		})
	}
}
