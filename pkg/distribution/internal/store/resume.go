package store

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	v1 "github.com/docker/model-runner/pkg/go-containerregistry/pkg/v1"
)

// ResumableTransport wraps an HTTP transport and adds Range header support for resuming downloads
type ResumableTransport struct {
	base      http.RoundTripper
	store     *LocalStore
	offsetsMu sync.RWMutex
	offsets   map[string]int64 // Maps blob digest to resume offset
	resumedMu sync.RWMutex
	resumed   map[string]bool // Tracks which blobs actually resumed successfully
}

// NewResumableTransport creates a new resumable transport
func NewResumableTransport(base http.RoundTripper, store *LocalStore) *ResumableTransport {
	return &ResumableTransport{
		base:    base,
		store:   store,
		offsets: make(map[string]int64),
		resumed: make(map[string]bool),
	}
}

// SetResumeOffset sets the resume offset for a blob
func (rt *ResumableTransport) SetResumeOffset(digest v1.Hash, offset int64) {
	rt.offsetsMu.Lock()
	defer rt.offsetsMu.Unlock()
	rt.offsets[digest.String()] = offset
	// Reset resumed status
	rt.resumedMu.Lock()
	rt.resumed[digest.String()] = false
	rt.resumedMu.Unlock()
}

// GetResumeOffset gets the resume offset for a blob
func (rt *ResumableTransport) GetResumeOffset(digest v1.Hash) int64 {
	rt.offsetsMu.RLock()
	defer rt.offsetsMu.RUnlock()
	return rt.offsets[digest.String()]
}

// DidResume returns true if the blob was actually resumed (server returned 206)
func (rt *ResumableTransport) DidResume(digest v1.Hash) bool {
	rt.resumedMu.RLock()
	defer rt.resumedMu.RUnlock()
	return rt.resumed[digest.String()]
}

// ClearResumeOffset clears the resume offset for a blob
func (rt *ResumableTransport) ClearResumeOffset(digest v1.Hash) {
	rt.offsetsMu.Lock()
	defer rt.offsetsMu.Unlock()
	delete(rt.offsets, digest.String())
	rt.resumedMu.Lock()
	delete(rt.resumed, digest.String())
	rt.resumedMu.Unlock()
}

// RoundTrip implements http.RoundTripper
func (rt *ResumableTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Check if this is a blob request that should be resumed
	// Blob requests look like: /v2/{name}/blobs/{digest}
	if req.Method == http.MethodGet && strings.Contains(req.URL.Path, "/blobs/sha256:") {
		// Extract digest from URL
		parts := strings.Split(req.URL.Path, "/blobs/")
		if len(parts) == 2 {
			digestStr := parts[1]
			// Remove any query parameters
			if idx := strings.Index(digestStr, "?"); idx != -1 {
				digestStr = digestStr[:idx]
			}
			
			// Check if we have a resume offset for this digest
			if strings.HasPrefix(digestStr, "sha256:") {
				hash, err := v1.NewHash(digestStr)
				if err == nil {
					offset := rt.GetResumeOffset(hash)
					if offset > 0 {
						// Add Range header to resume download
						req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
						
						// Make the request
						resp, err := rt.base.RoundTrip(req)
						if err != nil {
							return resp, err
						}
						
						// Check if server supported the range request
						if resp.StatusCode == http.StatusPartialContent {
							// Success! Mark as resumed
							rt.resumedMu.Lock()
							rt.resumed[hash.String()] = true
							rt.resumedMu.Unlock()
							return resp, nil
						} else if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable || resp.StatusCode == http.StatusOK {
							// Server doesn't support range or returned full content
							// Close this response and retry without Range header
							resp.Body.Close()
							req.Header.Del("Range")
							// Fall through to regular request below
						} else {
							// Some other status, return as-is
							return resp, nil
						}
					}
				}
			}
		}
	}

	return rt.base.RoundTrip(req)
}

// getIncompleteFileSize returns the size of the incomplete file for a given hash, or 0 if it doesn't exist
func (s *LocalStore) getIncompleteFileSize(hash v1.Hash) (int64, error) {
	path, err := s.blobPath(hash)
	if err != nil {
		return 0, err
	}
	
	incompletePath := incompletePath(path)
	stat, err := os.Stat(incompletePath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	
	return stat.Size(), nil
}
