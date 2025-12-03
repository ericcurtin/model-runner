//go:build !windows

package readline

import (
	"os"
	"testing"
)

func TestGetEditor(t *testing.T) {
	tests := []struct {
		name     string
		visual   string
		editor   string
		expected string
	}{
		{
			name:     "VISUAL environment variable set",
			visual:   "nano",
			editor:   "",
			expected: "nano",
		},
		{
			name:     "EDITOR environment variable set",
			visual:   "",
			editor:   "emacs",
			expected: "emacs",
		},
		{
			name:     "VISUAL takes precedence over EDITOR",
			visual:   "code",
			editor:   "nano",
			expected: "code",
		},
		{
			name:     "default to vi when no environment variables set",
			visual:   "",
			editor:   "",
			expected: "vi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original values
			origVisual := os.Getenv("VISUAL")
			origEditor := os.Getenv("EDITOR")

			// Set test values
			if tt.visual != "" {
				os.Setenv("VISUAL", tt.visual)
			} else {
				os.Unsetenv("VISUAL")
			}

			if tt.editor != "" {
				os.Setenv("EDITOR", tt.editor)
			} else {
				os.Unsetenv("EDITOR")
			}

			// Run test
			result := getEditor()

			// Restore original values
			if origVisual != "" {
				os.Setenv("VISUAL", origVisual)
			} else {
				os.Unsetenv("VISUAL")
			}

			if origEditor != "" {
				os.Setenv("EDITOR", origEditor)
			} else {
				os.Unsetenv("EDITOR")
			}

			if result != tt.expected {
				t.Errorf("getEditor() = %q, want %q", result, tt.expected)
			}
		})
	}
}
