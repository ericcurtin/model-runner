package desktop

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStatusPrinter is a simple implementation of StatusPrinter for testing
type mockStatusPrinter struct {
	output *bytes.Buffer
}

func (m *mockStatusPrinter) Printf(format string, args ...any) {
	fmt.Fprintf(m.output, format, args...)
}

func (m *mockStatusPrinter) Println(args ...any) {
	fmt.Fprintln(m.output, args...)
}

func (m *mockStatusPrinter) PrintErrf(format string, args ...any) {
	fmt.Fprintf(m.output, format, args...)
}

func (m *mockStatusPrinter) Write(p []byte) (n int, err error) {
	return m.output.Write(p)
}

func (m *mockStatusPrinter) GetFdInfo() (uintptr, bool) {
	// Return non-terminal to use simple progress display
	return 0, false
}

func TestWriteDockerProgress_SkipsNearComplete(t *testing.T) {
	tests := []struct {
		name          string
		current       uint64
		size          uint64
		expectSkipped bool
		description   string
	}{
		{
			name:          "early progress",
			current:       10 * 1024 * 1024,  // 10MB
			size:          100 * 1024 * 1024, // 100MB
			expectSkipped: false,
			description:   "10% complete - should show",
		},
		{
			name:          "mid progress",
			current:       50 * 1024 * 1024,  // 50MB
			size:          100 * 1024 * 1024, // 100MB
			expectSkipped: false,
			description:   "50% complete - should show",
		},
		{
			name:          "high progress",
			current:       90 * 1024 * 1024,  // 90MB
			size:          100 * 1024 * 1024, // 100MB
			expectSkipped: false,
			description:   "90% complete - should show",
		},
		{
			name:          "very close to complete",
			current:       99900000,  // 99.9MB
			size:          100000000, // 100MB
			expectSkipped: true,
			description:   "99.9% complete - should skip (>99.5% threshold)",
		},
		{
			name:          "almost complete like in issue",
			current:       91690000, // 91.69MB
			size:          91730000, // 91.73MB
			expectSkipped: true,
			description:   "99.956% complete - should skip (like the issue)",
		},
		{
			name:          "small file near complete",
			current:       4050,  // 4.05kB
			size:          12620, // 12.62kB
			expectSkipped: false,
			description:   "32% complete - should show (like the issue, but this is not near complete)",
		},
		{
			name:          "complete",
			current:       100 * 1024 * 1024,
			size:          100 * 1024 * 1024,
			expectSkipped: false, // Should show "Pull complete"
			description:   "100% complete - should show Pull complete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			msg := &ProgressMessage{
				Type:  "progress",
				Total: tt.size,
				Layer: Layer{
					ID:      "sha256:test1234567890abcdef",
					Size:    tt.size,
					Current: tt.current,
				},
			}
			layerStatus := make(map[string]string)

			err := writeDockerProgress(&buf, msg, layerStatus)
			require.NoError(t, err)

			output := buf.String()

			if tt.expectSkipped {
				assert.Empty(t, output, "Expected output to be empty for %s", tt.description)
			} else {
				assert.NotEmpty(t, output, "Expected output to not be empty for %s", tt.description)

				// Parse the JSON output to verify it's valid
				if output != "" {
					lines := strings.Split(strings.TrimSpace(output), "\n")
					for _, line := range lines {
						if line == "" {
							continue
						}
						var dockerMsg jsonmessage.JSONMessage
						err := json.Unmarshal([]byte(line), &dockerMsg)
						require.NoError(t, err, "Failed to parse JSON output for %s", tt.description)

						// Verify the status is either Downloading or Pull complete
						assert.Contains(t, []string{"Downloading", "Pull complete", "Waiting"}, dockerMsg.Status)
					}
				}
			}
		})
	}
}

func TestWriteDockerProgress_IssueScenario(t *testing.T) {
	// Test the specific scenario from the issue where we have:
	// Layer 1: downloading normally, then almost complete, then complete
	// Layer 2: small progress, then near complete, then complete

	var buf bytes.Buffer
	layerStatus := make(map[string]string)

	// Track all messages that were written
	var messages []string

	// Layer 1: normal progress (should be shown)
	msg1 := &ProgressMessage{
		Type:  "progress",
		Total: 91730000,
		Layer: Layer{
			ID:      "sha256:384a89bd054c",
			Size:    91730000,
			Current: 10000000, // 10.9% complete
		},
	}
	buf.Reset()
	err := writeDockerProgress(&buf, msg1, layerStatus)
	require.NoError(t, err)
	if buf.Len() > 0 {
		messages = append(messages, "Layer1: 10MB/91.73MB - Downloading")
	}

	// Layer 1: almost complete (should be skipped due to 99.9% threshold)
	msg2 := &ProgressMessage{
		Type:  "progress",
		Total: 91730000,
		Layer: Layer{
			ID:      "sha256:384a89bd054c",
			Size:    91730000,
			Current: 91690000, // 99.956% complete
		},
	}
	buf.Reset()
	err = writeDockerProgress(&buf, msg2, layerStatus)
	require.NoError(t, err)
	if buf.Len() > 0 {
		messages = append(messages, "Layer1: 91.69MB/91.73MB - Downloading (SHOULD BE SKIPPED)")
	}

	// Layer 1: complete (should show "Pull complete")
	msg3 := &ProgressMessage{
		Type:  "progress",
		Total: 91730000,
		Layer: Layer{
			ID:      "sha256:384a89bd054c",
			Size:    91730000,
			Current: 91730000, // 100% complete
		},
	}
	buf.Reset()
	err = writeDockerProgress(&buf, msg3, layerStatus)
	require.NoError(t, err)
	if buf.Len() > 0 {
		messages = append(messages, "Layer1: Complete")
	}

	// Layer 2: small progress (32% complete, should be shown)
	msg4 := &ProgressMessage{
		Type:  "progress",
		Total: 104350000,
		Layer: Layer{
			ID:      "sha256:609e2cb599f8",
			Size:    12620,
			Current: 4050, // 32% complete
		},
	}
	buf.Reset()
	err = writeDockerProgress(&buf, msg4, layerStatus)
	require.NoError(t, err)
	if buf.Len() > 0 {
		messages = append(messages, "Layer2: 4.05kB/12.62kB - Downloading")
	}

	// Layer 2: near complete (should be skipped due to 99.9% threshold)
	msg5 := &ProgressMessage{
		Type:  "progress",
		Total: 104350000,
		Layer: Layer{
			ID:      "sha256:609e2cb599f8",
			Size:    12620,
			Current: 12600, // 99.84% complete
		},
	}
	buf.Reset()
	err = writeDockerProgress(&buf, msg5, layerStatus)
	require.NoError(t, err)
	if buf.Len() > 0 {
		messages = append(messages, "Layer2: 12.60kB/12.62kB - Downloading (SHOULD BE SKIPPED)")
	}

	// Layer 2: complete (should show "Pull complete")
	msg6 := &ProgressMessage{
		Type:  "progress",
		Total: 104350000,
		Layer: Layer{
			ID:      "sha256:609e2cb599f8",
			Size:    12620,
			Current: 12620, // 100% complete
		},
	}
	buf.Reset()
	err = writeDockerProgress(&buf, msg6, layerStatus)
	require.NoError(t, err)
	if buf.Len() > 0 {
		messages = append(messages, "Layer2: Complete")
	}

	// Verify the expected messages
	expectedMessages := []string{
		"Layer1: 10MB/91.73MB - Downloading",
		"Layer1: Complete",
		"Layer2: 4.05kB/12.62kB - Downloading",
		"Layer2: Complete",
	}

	assert.Equal(t, expectedMessages, messages, "Should skip near-complete progress messages")
}

func TestWriteDockerProgress_AlreadyComplete(t *testing.T) {
	// Test that we don't show "Pull complete" twice for the same layer
	var buf bytes.Buffer
	msg := &ProgressMessage{
		Type:  "progress",
		Total: 100 * 1024 * 1024,
		Layer: Layer{
			ID:      "sha256:test1234567890abcdef",
			Size:    100 * 1024 * 1024,
			Current: 100 * 1024 * 1024,
		},
	}
	layerStatus := make(map[string]string)

	// First call should show "Pull complete"
	err := writeDockerProgress(&buf, msg, layerStatus)
	require.NoError(t, err)
	output1 := buf.String()
	assert.NotEmpty(t, output1)
	assert.Contains(t, output1, "Pull complete")

	// Second call should be skipped (empty output)
	buf.Reset()
	err = writeDockerProgress(&buf, msg, layerStatus)
	require.NoError(t, err)
	output2 := buf.String()
	assert.Empty(t, output2, "Second call should be skipped")
}
