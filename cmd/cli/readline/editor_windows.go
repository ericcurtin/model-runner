//go:build windows

package readline

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// getEditor returns the preferred text editor from environment variables.
// It checks VISUAL first, then EDITOR, and falls back to "notepad" as default on Windows.
func getEditor() string {
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	return "notepad"
}

// OpenInEditor opens a temporary file with the given content in the user's
// preferred text editor. It returns the edited content after the editor closes.
// The function handles restoring terminal mode before launching the editor
// and setting it back to raw mode after the editor closes.
func OpenInEditor(fd uintptr, termios any, currentContent string) (string, error) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "model-prompt-*.txt")
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Write current content to the file
	if _, err := tmpFile.WriteString(currentContent); err != nil {
		tmpFile.Close()
		return "", err
	}
	if err := tmpFile.Close(); err != nil {
		return "", err
	}

	// Restore terminal to normal mode before launching editor
	s, ok := termios.(*State)
	if !ok {
		return "", fmt.Errorf("invalid state type")
	}
	if err := UnsetRawMode(fd, s); err != nil {
		return "", err
	}

	// Get the editor and launch it
	editor := getEditor()
	editorParts := strings.Fields(editor)
	editorCmd := editorParts[0]
	editorArgs := append(editorParts[1:], tmpPath)

	cmd := exec.Command(editorCmd, editorArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Try to restore raw mode even if editor failed
		_, _ = SetRawMode(fd)
		return "", err
	}

	// Restore raw mode after editor closes
	if _, err := SetRawMode(fd); err != nil {
		return "", err
	}

	// Read the edited content
	content, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", err
	}

	// Trim trailing newlines that editors typically add
	result := strings.TrimSuffix(string(content), "\n")
	return result, nil
}
