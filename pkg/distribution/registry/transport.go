package registry

import (
	"crypto/tls"
	"net"
	"net/http"
	"strings"

	"github.com/google/go-containerregistry/pkg/v1/remote"
)

// insecureTransport wraps an http.RoundTripper to allow insecure connections
// to localhost and 127.0.0.1 registries.
//
// This is necessary for local development and testing, where developers often run
// registries without TLS certificates. The InsecureSkipVerify flag is only used
// for localhost addresses (localhost, 127.0.0.1, ::1, 127.x.x.x), which is safe
// because traffic never leaves the local machine.
//
// For production registries (non-localhost), TLS verification is always enforced.
type insecureTransport struct {
	inner http.RoundTripper
}

// RoundTrip implements http.RoundTripper
func (t *insecureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Check if this is a localhost registry
	if isLocalRegistry(req.URL.Host) {
		// For localhost registries, create a transport that skips TLS verification
		// and allows plaintext HTTP. This is safe for localhost because:
		// 1. Traffic never leaves the machine (no network exposure)
		// 2. Matches Docker daemon's behavior for local registries
		// 3. Required for development/testing with local registries
		transport := t.inner
		if httpTransport, ok := transport.(*http.Transport); ok {
			// Clone the transport to avoid modifying the original
			transport = httpTransport.Clone()
			clonedTransport := transport.(*http.Transport)

			// Skip TLS verification for localhost only (safe for local-only traffic)
			// nosemgrep: go.lang.security.audit.net.use-tls.use-tls
			if clonedTransport.TLSClientConfig == nil {
				clonedTransport.TLSClientConfig = &tls.Config{}
			}
			clonedTransport.TLSClientConfig.InsecureSkipVerify = true

			return clonedTransport.RoundTrip(req)
		}
	}

	// For non-localhost registries, use the default behavior
	return t.inner.RoundTrip(req)
}

// isLocalRegistry checks if the host is a localhost registry
func isLocalRegistry(host string) bool {
	// Extract just the hostname without port
	hostname, _, err := net.SplitHostPort(host)
	if err != nil {
		// If there's no port, use the host as-is
		hostname = host
	}

	// Check for localhost variants
	if hostname == "localhost" ||
		hostname == "127.0.0.1" ||
		hostname == "::1" ||
		strings.HasPrefix(hostname, "127.") {
		return true
	}

	return false
}

// NewDefaultTransport creates a transport suitable for use with container registries,
// including support for insecure localhost registries.
func NewDefaultTransport() http.RoundTripper {
	return &insecureTransport{
		inner: remote.DefaultTransport,
	}
}
