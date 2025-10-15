package ollama

import (
	"testing"
)

func TestExtractModelName(t *testing.T) {
	testCases := []struct {
		name     string
		fullName string
		expected string
	}{
		{
			name:     "library model with tag",
			fullName: "ollama.com/library/smollm:135m",
			expected: "library/smollm:135m",
		},
		{
			name:     "user model with tag",
			fullName: "ollama.com/user/custom-model:latest",
			expected: "user/custom-model:latest",
		},
		{
			name:     "simple model",
			fullName: "ollama.com/model",
			expected: "model",
		},
		{
			name:     "model without prefix",
			fullName: "library/smollm:135m",
			expected: "library/smollm:135m",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ExtractModelName(tc.fullName)
			if result != tc.expected {
				t.Errorf("ExtractModelName(%q) = %q, expected %q", tc.fullName, result, tc.expected)
			}
		})
	}
}
