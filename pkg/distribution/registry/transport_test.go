package registry

import (
	"net/http"
	"testing"
)

func TestIsLocalRegistry(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		expected bool
	}{
		{"localhost with port", "localhost:5000", true},
		{"localhost without port", "localhost", true},
		{"127.0.0.1 with port", "127.0.0.1:5000", true},
		{"127.0.0.1 without port", "127.0.0.1", true},
		{"127.x.x.x network", "127.0.0.2:5000", true},
		{"::1 IPv6", "[::1]:5000", true},
		{"remote registry", "registry.hub.docker.com", false},
		{"remote with port", "registry.example.com:443", false},
		{"gcr.io", "gcr.io", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLocalRegistry(tt.host)
			if result != tt.expected {
				t.Errorf("isLocalRegistry(%q) = %v, want %v", tt.host, result, tt.expected)
			}
		})
	}
}

func TestInsecureTransport(t *testing.T) {
	transport := NewDefaultTransport()
	
	// Verify it's wrapped
	if _, ok := transport.(*insecureTransport); !ok {
		t.Errorf("NewDefaultTransport() should return *insecureTransport, got %T", transport)
	}
	
	// Verify the inner transport exists
	insecure := transport.(*insecureTransport)
	if insecure.inner == nil {
		t.Error("insecureTransport.inner should not be nil")
	}
	
	// Verify it implements http.RoundTripper
	var _ http.RoundTripper = transport
}

func TestDefaultTransportIsInsecure(t *testing.T) {
	// Verify that DefaultTransport uses the insecure transport
	if _, ok := DefaultTransport.(*insecureTransport); !ok {
		t.Errorf("DefaultTransport should be *insecureTransport, got %T", DefaultTransport)
	}
}
