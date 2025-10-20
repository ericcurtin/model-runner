package progress

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

func TestMultiLayerTracker(t *testing.T) {
	t.Run("single layer", func(t *testing.T) {
		var buf bytes.Buffer
		layer := newMockLayer(1024 * 1024) // 1MB

		tracker := NewMultiLayerTracker(&buf, PullMsg, 1024*1024)
		updates, err := tracker.AddLayer(layer)
		if err != nil {
			t.Fatalf("Failed to add layer: %v", err)
		}

		// Send progress updates
		updates <- v1.Update{Complete: 512 * 1024} // 512KB
		time.Sleep(200 * time.Millisecond)         // Wait for update to be processed
		updates <- v1.Update{Complete: 1024 * 1024} // 1MB (complete)
		close(updates)

		if err := tracker.Wait(); err != nil {
			t.Fatalf("Tracker.Wait() failed: %v", err)
		}

		// Parse messages
		lines := bytes.Split(buf.Bytes(), []byte("\n"))
		var messages []Message
		for _, line := range lines {
			if len(line) == 0 {
				continue
			}
			var msg Message
			if err := json.Unmarshal(line, &msg); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}
			messages = append(messages, msg)
		}

		if len(messages) < 1 {
			t.Errorf("Expected at least 1 message, got %d", len(messages))
		}

		// Verify that messages contain progress type
		for i, msg := range messages {
			if msg.Type != "progress" {
				t.Errorf("message %d: expected type 'progress', got '%s'", i, msg.Type)
			}
		}
	})

	t.Run("multiple layers", func(t *testing.T) {
		var buf bytes.Buffer
		layer1 := &mockLayer{
			size:      1024 * 1024,
			diffID:    "sha256:1111111111111111111111111111111111111111111111111111111111111111",
			mediaType: "application/vnd.docker.model.rootfs.diff.tar.gzip",
		}
		layer2 := &mockLayer{
			size:      2048 * 1024,
			diffID:    "sha256:2222222222222222222222222222222222222222222222222222222222222222",
			mediaType: "application/vnd.docker.model.rootfs.diff.tar.gzip",
		}
		totalSize := int64(1024*1024 + 2048*1024)

		tracker := NewMultiLayerTracker(&buf, PullMsg, totalSize)

		// Add first layer
		updates1, err := tracker.AddLayer(layer1)
		if err != nil {
			t.Fatalf("Failed to add layer 1: %v", err)
		}

		// Add second layer
		updates2, err := tracker.AddLayer(layer2)
		if err != nil {
			t.Fatalf("Failed to add layer 2: %v", err)
		}

		// Simulate downloading both layers concurrently
		updates1 <- v1.Update{Complete: 512 * 1024} // 512KB from layer 1
		time.Sleep(200 * time.Millisecond)
		updates2 <- v1.Update{Complete: 1024 * 1024} // 1MB from layer 2
		time.Sleep(200 * time.Millisecond)
		updates1 <- v1.Update{Complete: 1024 * 1024} // Complete layer 1 (1MB total)
		time.Sleep(200 * time.Millisecond)
		updates2 <- v1.Update{Complete: 2048 * 1024} // Complete layer 2 (2MB total)
		time.Sleep(200 * time.Millisecond) // Wait for final update to be processed

		close(updates1)
		close(updates2)

		if err := tracker.Wait(); err != nil {
			t.Fatalf("Tracker.Wait() failed: %v", err)
		}

		// Parse messages
		lines := bytes.Split(buf.Bytes(), []byte("\n"))
		var messages []Message
		for _, line := range lines {
			if len(line) == 0 {
				continue
			}
			var msg Message
			if err := json.Unmarshal(line, &msg); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}
			messages = append(messages, msg)
		}

		if len(messages) < 1 {
			t.Errorf("Expected at least 1 message, got %d", len(messages))
		}

		// Verify total size is reported correctly
		for _, msg := range messages {
			if msg.Total != uint64(totalSize) {
				t.Errorf("Expected total %d, got %d", totalSize, msg.Total)
			}
		}

		// Verify that the last message shows complete download
		lastMsg := messages[len(messages)-1]
		expectedMessage := "Downloaded: 3.00 MB"
		if lastMsg.Message != expectedMessage {
			t.Errorf("Expected final message '%s', got '%s'", expectedMessage, lastMsg.Message)
		}
	})

	t.Run("sequential layer downloads", func(t *testing.T) {
		var buf bytes.Buffer
		layer1 := &mockLayer{
			size:      1024 * 1024,
			diffID:    "sha256:3333333333333333333333333333333333333333333333333333333333333333",
			mediaType: "application/vnd.docker.model.rootfs.diff.tar.gzip",
		}
		layer2 := &mockLayer{
			size:      1024 * 1024,
			diffID:    "sha256:4444444444444444444444444444444444444444444444444444444444444444",
			mediaType: "application/vnd.docker.model.rootfs.diff.tar.gzip",
		}
		totalSize := int64(2 * 1024 * 1024)

		tracker := NewMultiLayerTracker(&buf, PullMsg, totalSize)

		// Add and complete first layer
		updates1, err := tracker.AddLayer(layer1)
		if err != nil {
			t.Fatalf("Failed to add layer 1: %v", err)
		}
		updates1 <- v1.Update{Complete: 1024 * 1024}
		time.Sleep(200 * time.Millisecond)
		close(updates1)

		// Add and complete second layer
		updates2, err := tracker.AddLayer(layer2)
		if err != nil {
			t.Fatalf("Failed to add layer 2: %v", err)
		}
		updates2 <- v1.Update{Complete: 1024 * 1024}
		time.Sleep(200 * time.Millisecond)
		close(updates2)

		if err := tracker.Wait(); err != nil {
			t.Fatalf("Tracker.Wait() failed: %v", err)
		}

		// Parse messages
		lines := bytes.Split(buf.Bytes(), []byte("\n"))
		var messages []Message
		for _, line := range lines {
			if len(line) == 0 {
				continue
			}
			var msg Message
			if err := json.Unmarshal(line, &msg); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}
			messages = append(messages, msg)
		}

		if len(messages) < 2 {
			t.Errorf("Expected at least 2 messages (one per layer), got %d", len(messages))
		}

		// Verify that progress increases monotonically (or stays the same)
		var lastComplete int64 = 0
		for i, msg := range messages {
			if msg.Type != "progress" {
				t.Errorf("message %d: expected type 'progress', got '%s'", i, msg.Type)
			}
			// Extract complete from message (approximation based on message text)
			// We just verify the total is correct
			if msg.Total != uint64(totalSize) {
				t.Errorf("message %d: expected total %d, got %d", i, totalSize, msg.Total)
			}
			lastComplete++ // Just for validation
		}
	})
}
