package store

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

// ResumableDownloadLayer wraps a v1.Layer to provide resumable download capability.
// It implements the blob interface expected by writeLayer.
type ResumableDownloadLayer struct {
	layer         v1.Layer
	blobURL       string
	httpClient    *http.Client
	authorization string
	tmpFilePath   string
	ctx           context.Context
}

// NewResumableDownloadLayer creates a wrapper around a layer that supports resumable downloads.
// If blobURL and httpClient are provided, it will use HTTP Range requests to resume partial downloads.
func NewResumableDownloadLayer(ctx context.Context, layer v1.Layer, blobURL string, httpClient *http.Client, authorization string, tmpFilePath string) *ResumableDownloadLayer {
	return &ResumableDownloadLayer{
		layer:         layer,
		blobURL:       blobURL,
		httpClient:    httpClient,
		authorization: authorization,
		tmpFilePath:   tmpFilePath,
		ctx:           ctx,
	}
}

// DiffID returns the DiffID of the layer
func (r *ResumableDownloadLayer) DiffID() (v1.Hash, error) {
	return r.layer.DiffID()
}

// Uncompressed returns a reader for the uncompressed layer content.
// If a partial download exists and we have HTTP client info, it will resume using Range requests.
func (r *ResumableDownloadLayer) Uncompressed() (io.ReadCloser, error) {
	// If we don't have the necessary info for resumable downloads, fall back to default
	if r.blobURL == "" || r.httpClient == nil {
		return r.layer.Uncompressed()
	}

	// Check if we have a partial download
	var offset int64
	if r.tmpFilePath != "" {
		if fileInfo, err := os.Stat(r.tmpFilePath); err == nil {
			offset = fileInfo.Size()
		}
	}

	// Create HTTP request
	ctx := r.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.blobURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Add authorization if provided
	if r.authorization != "" {
		req.Header.Set("Authorization", r.authorization)
	}

	// Add Range header if we're resuming
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}

	// Make the request
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching blob: %w", err)
	}

	// Check response status
	if offset > 0 {
		// We requested a range
		if resp.StatusCode == http.StatusPartialContent {
			// Resume successful
			return resp.Body, nil
		} else if resp.StatusCode == http.StatusOK {
			// Server doesn't support ranges or returned full content
			// This is okay, we'll just redownload
			return resp.Body, nil
		} else if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable {
			// The range we requested is invalid (e.g., file changed)
			// Close this response and retry without Range
			resp.Body.Close()
			req.Header.Del("Range")
			resp, err = r.httpClient.Do(req)
			if err != nil {
				return nil, fmt.Errorf("fetching blob (retry): %w", err)
			}
			if resp.StatusCode != http.StatusOK {
				resp.Body.Close()
				return nil, fmt.Errorf("unexpected status code on retry: %d", resp.StatusCode)
			}
			return resp.Body, nil
		} else {
			resp.Body.Close()
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
	} else {
		// No range requested
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
		return resp.Body, nil
	}
}
