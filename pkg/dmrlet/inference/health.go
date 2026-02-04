// Package inference provides inference container management for dmrlet.
package inference

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const (
	// DefaultHealthTimeout is the maximum time to wait for a container to become ready.
	DefaultHealthTimeout = 5 * time.Minute
	// DefaultHealthInterval is the interval between health checks.
	DefaultHealthInterval = 2 * time.Second
)

// HealthChecker checks the health of inference containers.
type HealthChecker struct {
	client  *http.Client
	timeout time.Duration
}

// NewHealthChecker creates a new health checker.
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		timeout: DefaultHealthTimeout,
	}
}

// WaitForReady waits for the inference server to become ready.
func (h *HealthChecker) WaitForReady(ctx context.Context, port int) error {
	healthURL := fmt.Sprintf("http://localhost:%d/health", port)
	modelsURL := fmt.Sprintf("http://localhost:%d/v1/models", port)

	deadline := time.Now().Add(h.timeout)
	ticker := time.NewTicker(DefaultHealthInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("health check timeout after %v", h.timeout)
			}

			// Try /health endpoint first
			if h.checkEndpoint(healthURL) {
				return nil
			}

			// Fall back to /v1/models (OpenAI-compatible)
			if h.checkEndpoint(modelsURL) {
				return nil
			}
		}
	}
}

// checkEndpoint checks if an endpoint returns a successful response.
func (h *HealthChecker) checkEndpoint(url string) bool {
	resp, err := h.client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// CheckHealth performs a single health check.
func (h *HealthChecker) CheckHealth(port int) bool {
	healthURL := fmt.Sprintf("http://localhost:%d/health", port)
	return h.checkEndpoint(healthURL)
}
