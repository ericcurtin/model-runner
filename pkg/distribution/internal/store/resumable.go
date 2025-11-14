package store

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/docker/model-runner/pkg/distribution/internal/progress"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

// ResumableLayer wraps a v1.Layer and adds resumable download capability
type ResumableLayer struct {
	v1.Layer
	store      *LocalStore
	httpClient *http.Client
	blobURL    string
	authToken  string
}

// NewResumableLayer creates a resumable layer wrapper
func NewResumableLayer(
	layer v1.Layer,
	store *LocalStore,
	httpClient *http.Client,
	blobURL string,
	authToken string,
) *ResumableLayer {
	return &ResumableLayer{
		Layer:      layer,
		store:      store,
		httpClient: httpClient,
		blobURL:    blobURL,
		authToken:  authToken,
	}
}

// Compressed returns an io.ReadCloser for the compressed layer contents
// Note: Resume logic is handled in DownloadAndDecompress, this just returns the full layer
func (rl *ResumableLayer) Compressed() (io.ReadCloser, error) {
	return rl.Layer.Compressed()
}

// DownloadAndDecompress downloads the layer with resume support and decompresses it
func (rl *ResumableLayer) DownloadAndDecompress(updates chan<- v1.Update) (bool, v1.Hash, error) {
	diffID, err := rl.DiffID()
	if err != nil {
		return false, v1.Hash{}, fmt.Errorf("get diff ID: %w", err)
	}

	// Check if we already have this blob
	hasBlob, err := rl.store.hasBlob(diffID)
	if err != nil {
		return false, v1.Hash{}, fmt.Errorf("check blob existence: %w", err)
	}
	if hasBlob {
		return false, diffID, nil
	}

	// Get the compressed digest
	compressedDigest, err := rl.Digest()
	if err != nil {
		return false, v1.Hash{}, fmt.Errorf("get compressed digest: %w", err)
	}

	// Get path for storing compressed data
	compressedPath, err := rl.store.blobPath(compressedDigest)
	if err != nil {
		return false, v1.Hash{}, fmt.Errorf("get compressed path: %w", err)
	}
	// Use a different suffix for compressed incomplete files to avoid conflicts
	compressedIncompletePath := compressedPath + ".compressed.incomplete"

	// Check for existing incomplete file
	var offset int64
	if stat, err := os.Stat(compressedIncompletePath); err == nil {
		offset = stat.Size()
	}

	// Get compressed reader with resume support if we have an offset
	var compressedReader io.ReadCloser
	if offset > 0 && rl.httpClient != nil && rl.blobURL != "" {
		// Try to resume with HTTP Range
		req, err := http.NewRequest("GET", rl.blobURL, nil)
		if err == nil {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
			if rl.authToken != "" {
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", rl.authToken))
			}

			resp, err := rl.httpClient.Do(req)
			if err == nil {
				if resp.StatusCode == http.StatusPartialContent {
					// Successfully resumed!
					compressedReader = resp.Body
				} else {
					// Server doesn't support range, start fresh
					resp.Body.Close()
					offset = 0
					os.Remove(compressedIncompletePath)
					// Fall through to get full layer
				}
			}
		}
	}

	// If we couldn't resume, get the full layer
	if compressedReader == nil {
		var err error
		compressedReader, err = rl.Layer.Compressed()
		if err != nil {
			return false, v1.Hash{}, fmt.Errorf("get compressed reader: %w", err)
		}
	}
	defer compressedReader.Close()

	// Wrap compressed reader with progress reporting for download
	var progressReader io.Reader = compressedReader
	if updates != nil {
		progressReader = progress.NewReader(compressedReader, updates)
	}

	// Open file for writing (append if offset > 0)
	var compressedFile *os.File
	if offset > 0 {
		compressedFile, err = os.OpenFile(compressedIncompletePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			return false, v1.Hash{}, fmt.Errorf("open compressed file for append: %w", err)
		}
	} else {
		compressedFile, err = createFile(compressedIncompletePath)
		if err != nil {
			return false, v1.Hash{}, fmt.Errorf("create compressed file: %w", err)
		}
	}

	// Download compressed data with progress reporting
	written, err := io.Copy(compressedFile, progressReader)
	if err != nil {
		compressedFile.Close()
		// Keep incomplete file for resume
		return false, v1.Hash{}, fmt.Errorf("download compressed (offset=%d, wrote=%d): %w", offset, written, err)
	}
	compressedFile.Close()

	// Decompress the complete file
	compressedFile, err = os.Open(compressedIncompletePath)
	if err != nil {
		return false, v1.Hash{}, fmt.Errorf("open for decompression: %w", err)
	}

	// Try to decompress - if it fails, the data might already be uncompressed
	gzipReader, err := gzip.NewReader(compressedFile)
	var reader io.Reader
	if err != nil {
		// Data is not gzipped, use it directly
		// Need to reopen the file since gzip.NewReader consumed some bytes
		compressedFile.Close()
		compressedFile, err = os.Open(compressedIncompletePath)
		if err != nil {
			return false, v1.Hash{}, fmt.Errorf("reopen for direct read: %w", err)
		}
		defer compressedFile.Close()
		reader = compressedFile
	} else {
		// gzipReader wraps compressedFile, so close them in proper order
		defer gzipReader.Close()
		defer compressedFile.Close()
		reader = gzipReader
	}

	// Write data (no progress wrapping here since download progress was already reported)
	if err := rl.store.WriteBlob(diffID, reader); err != nil {
		return false, v1.Hash{}, fmt.Errorf("write blob: %w", err)
	}

	// Clean up compressed file
	os.Remove(compressedIncompletePath)

	return true, diffID, nil
}

