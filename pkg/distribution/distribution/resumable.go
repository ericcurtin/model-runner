package distribution

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/name"
)

// resumableImage wraps a v1.Image to provide layers with resumable download support
type resumableImage struct {
	v1.Image
	reference   name.Reference
	transport   http.RoundTripper
	storePath   string
}

// Layers returns the ordered collection of filesystem layers that comprise this image, with resume support.
func (ri *resumableImage) Layers() ([]v1.Layer, error) {
	layers, err := ri.Image.Layers()
	if err != nil {
		return nil, err
	}

	// Wrap each layer with resumable support
	resumableLayers := make([]v1.Layer, len(layers))
	for i, layer := range layers {
		resumableLayers[i] = &resumableLayer{
			Layer:     layer,
			reference: ri.reference,
			transport: ri.transport,
			storePath: ri.storePath,
		}
	}

	return resumableLayers, nil
}

// resumableLayer wraps a v1.Layer to support resumable downloads using HTTP Range requests
type resumableLayer struct {
	v1.Layer
	reference name.Reference
	transport http.RoundTripper
	storePath string
}

// Uncompressed returns an io.ReadCloser for the uncompressed layer contents,
// with support for resuming from a partial download.
func (rl *resumableLayer) Uncompressed() (io.ReadCloser, error) {
	// Get the layer digest to find the incomplete file path
	digest, err := rl.Layer.Digest()
	if err != nil {
		return nil, fmt.Errorf("getting layer digest: %w", err)
	}

	// Construct the incomplete file path (matching the store's path structure)
	incompletePath := fmt.Sprintf("%s/blobs/%s/%s.incomplete", rl.storePath, digest.Algorithm, digest.Hex)

	// Check if there's a partial download
	var offset int64
	if fi, err := os.Stat(incompletePath); err == nil {
		offset = fi.Size()

		// Get the expected layer size
		size, sizeErr := rl.Layer.Size()
		if sizeErr != nil {
			// Can't determine size, start fresh
			_ = os.Remove(incompletePath)
			offset = 0
		} else if offset >= size {
			// Partial file is complete or corrupt, start over
			_ = os.Remove(incompletePath)
			offset = 0
		}
	}

	if offset > 0 {
		// Attempt to resume download using HTTP Range request
		rc, err := rl.resumeUncompressed(offset)
		if err != nil {
			// If resume fails, fall back to downloading from scratch
			_ = os.Remove(incompletePath)
			return rl.Layer.Uncompressed()
		}
		return rc, nil
	}

	// No partial download, proceed normally
	return rl.Layer.Uncompressed()
}

// resumeUncompressed attempts to resume the download from the given offset using HTTP Range requests
func (rl *resumableLayer) resumeUncompressed(offset int64) (io.ReadCloser, error) {
	digest, err := rl.Layer.Digest()
	if err != nil {
		return nil, fmt.Errorf("getting layer digest: %w", err)
	}

	// Construct the blob URL
	blobURL := fmt.Sprintf("%s://%s/v2/%s/blobs/%s",
		rl.reference.Context().Registry.Scheme(),
		rl.reference.Context().Registry.RegistryStr(),
		rl.reference.Context().RepositoryStr(),
		digest.String())

	// Create HTTP request with Range header
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, blobURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set Range header to request from offset to end
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))

	// Make the request using the transport (which should handle authentication)
	resp, err := rl.transport.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("making range request: %w", err)
	}

	// Check response status
	// HTTP 206 Partial Content means the server supports range requests
	// HTTP 200 OK means the server doesn't support ranges (we'll get full content)
	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// If we got HTTP 200 (full content), we need to skip to the offset
	if resp.StatusCode == http.StatusOK {
		return &skipReadCloser{
			ReadCloser: resp.Body,
			skip:       offset,
		}, nil
	}

	// HTTP 206 - server supports range requests, we can use the body directly
	return resp.Body, nil
}

// skipReadCloser wraps a ReadCloser and skips the first skip bytes on first read
type skipReadCloser struct {
	io.ReadCloser
	skip    int64
	skipped bool
}

func (s *skipReadCloser) Read(p []byte) (n int, err error) {
	if !s.skipped && s.skip > 0 {
		// Discard the first skip bytes
		_, err := io.CopyN(io.Discard, s.ReadCloser, s.skip)
		if err != nil {
			return 0, fmt.Errorf("skipping to offset: %w", err)
		}
		s.skipped = true
	}
	return s.ReadCloser.Read(p)
}
