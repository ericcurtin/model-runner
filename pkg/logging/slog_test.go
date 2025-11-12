package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestSlogLogger_DebugLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := NewSlogLogger(slog.LevelDebug, &buf)

	logger.Debug("debug message")
	logger.Info("info message")

	output := buf.String()
	
	// Both debug and info messages should appear
	if !strings.Contains(output, "debug message") {
		t.Error("Expected debug message to appear in output")
	}
	if !strings.Contains(output, "info message") {
		t.Error("Expected info message to appear in output")
	}
}

func TestSlogLogger_InfoLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := NewSlogLogger(slog.LevelInfo, &buf)

	logger.Debug("debug message")
	logger.Info("info message")

	output := buf.String()
	
	// Debug message should NOT appear, but info should
	if strings.Contains(output, "debug message") {
		t.Error("Debug message should not appear when log level is Info")
	}
	if !strings.Contains(output, "info message") {
		t.Error("Expected info message to appear in output")
	}
}

func TestSlogLogger_WithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewSlogLogger(slog.LevelInfo, &buf)

	childLogger := logger.WithFields(map[string]interface{}{
		"component": "test",
		"request_id": "123",
	})

	childLogger.Info("test message")

	output := buf.String()
	
	if !strings.Contains(output, "component=test") {
		t.Error("Expected component field in output")
	}
	if !strings.Contains(output, "request_id=123") {
		t.Error("Expected request_id field in output")
	}
	if !strings.Contains(output, "test message") {
		t.Error("Expected test message in output")
	}
}

func TestSlogLogger_WithField(t *testing.T) {
	var buf bytes.Buffer
	logger := NewSlogLogger(slog.LevelInfo, &buf)

	childLogger := logger.WithField("user", "alice")
	childLogger.Info("user logged in")

	output := buf.String()
	
	if !strings.Contains(output, "user=alice") {
		t.Error("Expected user field in output")
	}
	if !strings.Contains(output, "user logged in") {
		t.Error("Expected message in output")
	}
}

func TestSlogLogger_WithError(t *testing.T) {
	var buf bytes.Buffer
	logger := NewSlogLogger(slog.LevelInfo, &buf)

	err := &testError{"something went wrong"}
	childLogger := logger.WithError(err)
	childLogger.Error("operation failed")

	output := buf.String()
	
	if !strings.Contains(output, "something went wrong") {
		t.Error("Expected error message in output")
	}
	if !strings.Contains(output, "operation failed") {
		t.Error("Expected error message in output")
	}
}

func TestSlogLogger_FatalExitFunc(t *testing.T) {
	var buf bytes.Buffer
	logger := NewSlogLogger(slog.LevelInfo, &buf)

	exitCalled := false
	exitCode := 0
	logger.SetExitFunc(func(code int) {
		exitCalled = true
		exitCode = code
	})

	logger.Fatal("fatal error")

	if !exitCalled {
		t.Error("Expected exit function to be called")
	}
	if exitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", exitCode)
	}
	
	output := buf.String()
	if !strings.Contains(output, "fatal error") {
		t.Error("Expected fatal error message in output")
	}
}

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
