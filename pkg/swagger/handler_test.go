package swagger

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandler_ServeHTTP_Root(t *testing.T) {
	handler := NewHandler()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected Content-Type to contain text/html, got %s", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Docker Model Runner") {
		t.Error("expected body to contain 'Docker Model Runner'")
	}
	if !strings.Contains(body, "swagger-ui") {
		t.Error("expected body to contain 'swagger-ui'")
	}
}

func TestHandler_ServeHTTP_IndexHTML(t *testing.T) {
	handler := NewHandler()

	req := httptest.NewRequest(http.MethodGet, "/index.html", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected Content-Type to contain text/html, got %s", contentType)
	}
}

func TestHandler_ServeHTTP_OpenAPISpec(t *testing.T) {
	handler := NewHandler()

	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/yaml" {
		t.Errorf("expected Content-Type 'application/yaml', got %s", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "openapi: 3.0.3") {
		t.Error("expected body to contain OpenAPI version")
	}
	if !strings.Contains(body, "Docker Model Runner API") {
		t.Error("expected body to contain API title")
	}
}

func TestHandler_ServeHTTP_NotFound(t *testing.T) {
	handler := NewHandler()

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestOpenAPISpecContainsAllAPIs(t *testing.T) {
	spec := string(openapiSpec)

	// Check for OpenAI API endpoints
	if !strings.Contains(spec, "/v1/chat/completions") {
		t.Error("expected OpenAPI spec to contain /v1/chat/completions endpoint")
	}
	if !strings.Contains(spec, "/v1/completions") {
		t.Error("expected OpenAPI spec to contain /v1/completions endpoint")
	}
	if !strings.Contains(spec, "/v1/embeddings") {
		t.Error("expected OpenAPI spec to contain /v1/embeddings endpoint")
	}

	// Check for Anthropic API endpoints
	if !strings.Contains(spec, "/anthropic/v1/messages") {
		t.Error("expected OpenAPI spec to contain /anthropic/v1/messages endpoint")
	}

	// Check for Ollama API endpoints
	if !strings.Contains(spec, "/api/chat") {
		t.Error("expected OpenAPI spec to contain /api/chat endpoint")
	}
	if !strings.Contains(spec, "/api/generate") {
		t.Error("expected OpenAPI spec to contain /api/generate endpoint")
	}
	if !strings.Contains(spec, "/api/tags") {
		t.Error("expected OpenAPI spec to contain /api/tags endpoint")
	}

	// Check for tags
	if !strings.Contains(spec, "OpenAI API") {
		t.Error("expected OpenAPI spec to contain 'OpenAI API' tag")
	}
	if !strings.Contains(spec, "Anthropic API") {
		t.Error("expected OpenAPI spec to contain 'Anthropic API' tag")
	}
	if !strings.Contains(spec, "Ollama API") {
		t.Error("expected OpenAPI spec to contain 'Ollama API' tag")
	}
}
