package desktop

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDisplayProgress_ShowsOnlyLastCompletedLayer(t *testing.T) {
	// Simulate a pull with two layers
	layer1ID := "sha256:ed5fa30c487b1234567890abcdef1234567890abcdef1234567890abcdef1234"
	layer2ID := "sha256:609e2cb599f81234567890abcdef1234567890abcdef1234567890abcdef1234"

	// Create progress messages simulating a real pull
	progressMessages := []ProgressMessage{
		// Layer 1 downloading
		{Type: "progress", Layer: Layer{ID: layer1ID, Size: 105500000, Current: 1000000}},
		{Type: "progress", Layer: Layer{ID: layer1ID, Size: 105500000, Current: 50000000}},
		{Type: "progress", Layer: Layer{ID: layer1ID, Size: 105500000, Current: 105500000}}, // Complete
		// Layer 2 downloading
		{Type: "progress", Layer: Layer{ID: layer2ID, Size: 12620, Current: 4096}},
		{Type: "progress", Layer: Layer{ID: layer2ID, Size: 12620, Current: 12620}}, // Complete
		// Success message
		{Type: "success", Message: "Model pulled successfully"},
	}

	// Create input stream
	var input bytes.Buffer
	for _, msg := range progressMessages {
		data, err := json.Marshal(msg)
		require.NoError(t, err)
		input.WriteString(string(data) + "\n")
	}

	// Capture output
	var output bytes.Buffer
	printer := &testPrinter{
		output: &output,
		fd:     1, // Simulate terminal
		isTerm: true,
	}

	message, progressShown, err := DisplayProgress(&input, printer)

	// Verify results
	assert.NoError(t, err)
	assert.Equal(t, "Model pulled successfully", message)
	assert.True(t, progressShown, "Should indicate that progress was shown")

	// Verify output contains only the last layer's "Pull complete"
	outputStr := output.String()
	
	// Should contain the last layer's ID (shortened)
	shortLayer2ID := layer2ID[7:19] // sha256: prefix removed, first 12 chars
	assert.Contains(t, outputStr, shortLayer2ID, "Should show the last completed layer")
	assert.Contains(t, outputStr, "Pull complete", "Should show 'Pull complete' message")

	// Should NOT contain downloading progress or the first layer's completion
	// (We can't easily test this without parsing the JSONMessage format, but we can
	// verify that there's only one "Pull complete" in the output)
	pullCompleteCount := strings.Count(outputStr, "Pull complete")
	assert.Equal(t, 1, pullCompleteCount, "Should only show one 'Pull complete' message")
}

func TestDisplayProgress_NoProgressMessages(t *testing.T) {
	// Only success message, no progress
	progressMessages := []ProgressMessage{
		{Type: "success", Message: "Model pulled successfully"},
	}

	var input bytes.Buffer
	for _, msg := range progressMessages {
		data, err := json.Marshal(msg)
		require.NoError(t, err)
		input.WriteString(string(data) + "\n")
	}

	var output bytes.Buffer
	printer := &testPrinter{
		output: &output,
		fd:     1,
		isTerm: true,
	}

	message, progressShown, err := DisplayProgress(&input, printer)

	assert.NoError(t, err)
	assert.Equal(t, "Model pulled successfully", message)
	assert.False(t, progressShown, "Should indicate no progress was shown")
}

func TestDisplayProgress_ErrorMessage(t *testing.T) {
	progressMessages := []ProgressMessage{
		{Type: "progress", Layer: Layer{ID: "sha256:abc123", Size: 1000, Current: 500}},
		{Type: "error", Message: "Failed to pull layer"},
	}

	var input bytes.Buffer
	for _, msg := range progressMessages {
		data, err := json.Marshal(msg)
		require.NoError(t, err)
		input.WriteString(string(data) + "\n")
	}

	var output bytes.Buffer
	printer := &testPrinter{
		output: &output,
		fd:     1,
		isTerm: true,
	}

	_, _, err := DisplayProgress(&input, printer)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to pull layer")
}

// testPrinter is a test implementation of StatusPrinter
type testPrinter struct {
	output *bytes.Buffer
	fd     uintptr
	isTerm bool
}

func (p *testPrinter) Printf(format string, args ...any) {
	// Not used in DisplayProgress
}

func (p *testPrinter) Println(args ...any) {
	// Not used in DisplayProgress
}

func (p *testPrinter) PrintErrf(format string, args ...any) {
	// Not used in DisplayProgress
}

func (p *testPrinter) Write(data []byte) (n int, err error) {
	return p.output.Write(data)
}

func (p *testPrinter) GetFdInfo() (uintptr, bool) {
	return p.fd, p.isTerm
}
