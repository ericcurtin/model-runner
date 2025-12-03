package anthropic

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteAnthropicError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		errorType  string
		message    string
		wantBody   string
	}{
		{
			name:       "invalid request error",
			statusCode: http.StatusBadRequest,
			errorType:  "invalid_request_error",
			message:    "Missing required field: model",
			wantBody:   `{"type":"error","error":{"type":"invalid_request_error","message":"Missing required field: model"}}`,
		},
		{
			name:       "not found error",
			statusCode: http.StatusNotFound,
			errorType:  "not_found_error",
			message:    "Model not found: test-model",
			wantBody:   `{"type":"error","error":{"type":"not_found_error","message":"Model not found: test-model"}}`,
		},
		{
			name:       "internal error",
			statusCode: http.StatusInternalServerError,
			errorType:  "internal_error",
			message:    "An internal error occurred",
			wantBody:   `{"type":"error","error":{"type":"internal_error","message":"An internal error occurred"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			h := &Handler{}
			h.writeAnthropicError(rec, tt.statusCode, tt.errorType, tt.message)

			if rec.Code != tt.statusCode {
				t.Errorf("expected status %d, got %d", tt.statusCode, rec.Code)
			}

			if contentType := rec.Header().Get("Content-Type"); contentType != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", contentType)
			}

			body := strings.TrimSpace(rec.Body.String())
			if body != tt.wantBody {
				t.Errorf("expected body %s, got %s", tt.wantBody, body)
			}
		})
	}
}

func TestRouteHandlers(t *testing.T) {
	t.Parallel()

	h := &Handler{
		router: http.NewServeMux(),
	}

	routes := h.routeHandlers()

	expectedRoutes := []string{
		"POST " + APIPrefix + "/v1/messages",
		"POST " + APIPrefix + "/v1/messages/count_tokens",
	}

	for _, route := range expectedRoutes {
		if _, exists := routes[route]; !exists {
			t.Errorf("expected route %s to be registered", route)
		}
	}

	if len(routes) != len(expectedRoutes) {
		t.Errorf("expected %d routes, got %d", len(expectedRoutes), len(routes))
	}
}

func TestAPIPrefix(t *testing.T) {
	t.Parallel()

	if APIPrefix != "/anthropic" {
		t.Errorf("expected APIPrefix to be /anthropic, got %s", APIPrefix)
	}
}
