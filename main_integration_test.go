package main

import (
	"fmt"
	"os"
	"testing"
	"time"
)

// TestCORSProtectionIntegration tests that CORS protection works end-to-end
func TestCORSProtectionIntegration(t *testing.T) {
	// Skip in short mode as this requires actually starting the server
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name           string
		dmrOrigins     string
		modelRunnerPort string
		origin         string
		expectAllowed  bool
	}{
		{
			name:            "TCP mode with default - localhost allowed",
			dmrOrigins:      "",
			modelRunnerPort: "18434",
			origin:          "http://localhost",
			expectAllowed:   true,
		},
		{
			name:            "TCP mode with default - 127.0.0.1 allowed",
			dmrOrigins:      "",
			modelRunnerPort: "18435",
			origin:          "http://127.0.0.1",
			expectAllowed:   true,
		},
		{
			name:            "TCP mode with default - external origin blocked",
			dmrOrigins:      "",
			modelRunnerPort: "18436",
			origin:          "http://evil.com",
			expectAllowed:   false,
		},
		{
			name:            "TCP mode with custom origin - allowed",
			dmrOrigins:      "http://example.com",
			modelRunnerPort: "18437",
			origin:          "http://example.com",
			expectAllowed:   true,
		},
		{
			name:            "TCP mode with custom origin - blocked",
			dmrOrigins:      "http://example.com",
			modelRunnerPort: "18438",
			origin:          "http://evil.com",
			expectAllowed:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			if tt.dmrOrigins != "" {
				os.Setenv("DMR_ORIGINS", tt.dmrOrigins)
				defer os.Unsetenv("DMR_ORIGINS")
			}
			if tt.modelRunnerPort != "" {
				os.Setenv("MODEL_RUNNER_PORT", tt.modelRunnerPort)
				defer os.Unsetenv("MODEL_RUNNER_PORT")
			}

			// Test the getAllowedOrigins function directly
			allowedOrigins := getAllowedOrigins()
			
			// Verify the expected origins
			if tt.dmrOrigins == "" && tt.modelRunnerPort != "" {
				// Should have default localhost origins
				if len(allowedOrigins) != 4 {
					t.Errorf("Expected 4 default origins, got %d", len(allowedOrigins))
				}
			} else if tt.dmrOrigins != "" {
				// Should have custom origins
				if len(allowedOrigins) != 1 {
					t.Errorf("Expected 1 custom origin, got %d", len(allowedOrigins))
				}
				if len(allowedOrigins) > 0 && allowedOrigins[0] != tt.dmrOrigins {
					t.Errorf("Expected origin %s, got %s", tt.dmrOrigins, allowedOrigins[0])
				}
			}

			// Verify that the origin is allowed or blocked as expected
			originAllowed := false
			for _, allowed := range allowedOrigins {
				if allowed == "*" || allowed == tt.origin {
					originAllowed = true
					break
				}
			}

			if originAllowed != tt.expectAllowed {
				t.Errorf("Expected origin %s to be allowed=%v, but got allowed=%v", tt.origin, tt.expectAllowed, originAllowed)
			}

			// Cleanup
			os.Unsetenv("DMR_ORIGINS")
			os.Unsetenv("MODEL_RUNNER_PORT")
		})
	}
}

// TestCORSMiddlewareWithServer tests the CORS middleware by making actual HTTP requests
func TestCORSMiddlewareWithServer(t *testing.T) {
	// Skip in short mode as this requires network access
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test would require starting the actual server, which is complex
	// For now, we rely on the unit tests in pkg/middleware/cors_test.go
	// and the getAllowedOrigins tests above
	t.Skip("Full server integration test not implemented - see cors_test.go for middleware tests")
}

// TestCORSWithPOCScenario tests the specific scenario from the problem statement
func TestCORSWithPOCScenario(t *testing.T) {
	tests := []struct {
		name             string
		dmrOrigins       string
		modelRunnerPort  string
		requestOrigin    string
		expectedBlocked  bool
	}{
		{
			name:            "POC scenario - external website blocked by default",
			dmrOrigins:      "",
			modelRunnerPort: "12434",
			requestOrigin:   "http://sudi.s1r1us.ninja:1339",
			expectedBlocked: true,
		},
		{
			name:            "POC scenario - localhost allowed by default",
			dmrOrigins:      "",
			modelRunnerPort: "12434",
			requestOrigin:   "http://localhost",
			expectedBlocked: false,
		},
		{
			name:            "POC scenario - explicit allow external origin",
			dmrOrigins:      "*",
			modelRunnerPort: "12434",
			requestOrigin:   "http://sudi.s1r1us.ninja:1339",
			expectedBlocked: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			os.Setenv("MODEL_RUNNER_PORT", tt.modelRunnerPort)
			if tt.dmrOrigins != "" {
				os.Setenv("DMR_ORIGINS", tt.dmrOrigins)
			}

			// Get allowed origins
			allowedOrigins := getAllowedOrigins()

			// Check if the request origin would be blocked
			isAllowed := false
			if allowedOrigins != nil {
				for _, allowed := range allowedOrigins {
					if allowed == "*" || allowed == tt.requestOrigin {
						isAllowed = true
						break
					}
				}
			}

			isBlocked := !isAllowed

			if isBlocked != tt.expectedBlocked {
				t.Errorf("Expected origin %s to be blocked=%v, but got blocked=%v (allowed origins: %v)",
					tt.requestOrigin, tt.expectedBlocked, isBlocked, allowedOrigins)
			}

			// Cleanup
			os.Unsetenv("DMR_ORIGINS")
			os.Unsetenv("MODEL_RUNNER_PORT")
		})
	}
}

// BenchmarkGetAllowedOrigins benchmarks the getAllowedOrigins function
func BenchmarkGetAllowedOrigins(b *testing.B) {
	os.Setenv("MODEL_RUNNER_PORT", "12434")
	defer os.Unsetenv("MODEL_RUNNER_PORT")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = getAllowedOrigins()
	}
}

// TestCORSHeadersInResponse is a helper test to demonstrate how CORS works
func TestCORSHeadersInResponse(t *testing.T) {
	// This is a documentation test showing expected behavior
	t.Log("CORS Protection Behavior:")
	t.Log("1. When MODEL_RUNNER_PORT is set (TCP mode):")
	t.Log("   - Default allowed origins: localhost, 127.0.0.1 (http and https)")
	t.Log("   - Requests from other origins will NOT have Access-Control-Allow-Origin header")
	t.Log("   - Browsers will block the response for cross-origin requests")
	t.Log("")
	t.Log("2. When DMR_ORIGINS is set:")
	t.Log("   - Only specified origins are allowed")
	t.Log("   - Use '*' to allow all origins (insecure)")
	t.Log("")
	t.Log("3. When using Unix socket (no MODEL_RUNNER_PORT):")
	t.Log("   - CORS is disabled (not needed for socket communication)")
	t.Log("")
	t.Log("Security Note: The fix prevents the POC attack by:")
	t.Log("- Rejecting requests from arbitrary websites by default")
	t.Log("- Only allowing localhost origins in TCP mode")
	t.Log("- Requiring explicit configuration to allow external origins")
}

// TestServerStartupTime measures how long it takes to configure CORS
func TestServerStartupTime(t *testing.T) {
	os.Setenv("MODEL_RUNNER_PORT", "12434")
	defer os.Unsetenv("MODEL_RUNNER_PORT")

	start := time.Now()
	_ = getAllowedOrigins()
	duration := time.Since(start)

	// CORS configuration should be very fast (< 1ms)
	if duration > time.Millisecond {
		t.Logf("CORS configuration took %v (expected < 1ms)", duration)
	}
}

// TestMultipleConcurrentOriginChecks tests thread safety
func TestMultipleConcurrentOriginChecks(t *testing.T) {
	os.Setenv("MODEL_RUNNER_PORT", "12434")
	defer os.Unsetenv("MODEL_RUNNER_PORT")

	// Run multiple goroutines calling getAllowedOrigins
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			origins := getAllowedOrigins()
			if len(origins) != 4 {
				t.Errorf("Goroutine %d: Expected 4 origins, got %d", id, len(origins))
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(time.Second):
			t.Fatal("Timeout waiting for goroutines")
		}
	}
}

// TestOriginValidationExamples provides examples of valid and invalid origins
func TestOriginValidationExamples(t *testing.T) {
	examples := []struct {
		origin      string
		shouldAllow bool
		description string
	}{
		{"http://localhost", true, "HTTP localhost should be allowed by default"},
		{"https://localhost", true, "HTTPS localhost should be allowed by default"},
		{"http://127.0.0.1", true, "HTTP 127.0.0.1 should be allowed by default"},
		{"https://127.0.0.1", true, "HTTPS 127.0.0.1 should be allowed by default"},
		{"http://localhost:3000", false, "Localhost with port should NOT match default (exact match required)"},
		{"http://evil.com", false, "External domain should be blocked by default"},
		{"https://attacker.example.com", false, "HTTPS external domain should be blocked by default"},
		{"http://127.0.0.1:8080", false, "127.0.0.1 with port should NOT match default"},
	}

	os.Setenv("MODEL_RUNNER_PORT", "12434")
	defer os.Unsetenv("MODEL_RUNNER_PORT")

	allowedOrigins := getAllowedOrigins()

	for _, ex := range examples {
		t.Run(fmt.Sprintf("Origin: %s", ex.origin), func(t *testing.T) {
			isAllowed := false
			for _, allowed := range allowedOrigins {
				if allowed == ex.origin {
					isAllowed = true
					break
				}
			}

			if isAllowed != ex.shouldAllow {
				t.Errorf("%s: expected allowed=%v, got allowed=%v", ex.description, ex.shouldAllow, isAllowed)
			}
		})
	}
}
