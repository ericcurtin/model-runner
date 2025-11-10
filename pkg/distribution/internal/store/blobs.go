package store

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/model-runner/pkg/distribution/internal/progress"

	v1 "github.com/docker/model-runner/pkg/go-containerregistry/pkg/v1"
)

const (
	blobsDir = "blobs"
)

var allowedAlgorithms = map[string]int{
	"sha256": 64,
	"sha512": 128,
}

func isSafeAlgorithm(a string) (int, bool) {
	hexLength, ok := allowedAlgorithms[a]
	return hexLength, ok
}

func isSafeHex(hexLength int, s string) bool {
	if len(s) != hexLength {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// validateHash ensures the hash components are safe for filesystem paths
func validateHash(hash v1.Hash) error {
	hexLength, ok := isSafeAlgorithm(hash.Algorithm)
	if !ok {
		return fmt.Errorf("invalid hash algorithm: %q not in allowlist", hash.Algorithm)
	}
	if !isSafeHex(hexLength, hash.Hex) {
		return fmt.Errorf("invalid hash hex: contains non-hexadecimal characters or invalid length")
	}
	return nil
}

// blobDir returns the path to the blobs directory
func (s *LocalStore) blobsDir() string {
	return filepath.Join(s.rootPath, blobsDir)
}

// blobPath returns the path to the blob for the given hash.
func (s *LocalStore) blobPath(hash v1.Hash) (string, error) {
	if err := validateHash(hash); err != nil {
		return "", fmt.Errorf("unsafe hash: %w", err)
	}

	path := filepath.Join(s.rootPath, blobsDir, hash.Algorithm, hash.Hex)

	cleanRootPath := filepath.Clean(s.rootPath)
	cleanPath := filepath.Clean(path)
	relPath, err := filepath.Rel(cleanRootPath, cleanPath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return "", fmt.Errorf("path traversal attempt detected: %s", path)
	}

	return cleanPath, nil
}

type blob interface {
	DiffID() (v1.Hash, error)
	Uncompressed() (io.ReadCloser, error)
}

// ResumableBlob extends the blob interface to support resumable downloads.
type ResumableBlob interface {
	blob
	// CompressedWithOffset returns a reader starting from the given offset.
	CompressedWithOffset(offset int64) (io.ReadCloser, error)
	// SupportsRangeRequests checks if the remote server supports HTTP Range requests.
	SupportsRangeRequests() (bool, error)
}

// tryResumable attempts to extract a ResumableBlob from a layer.
// It returns nil if the layer doesn't support resumable downloads.
func tryResumable(layer blob) ResumableBlob {
	// Try direct type assertion
	if r, ok := layer.(ResumableBlob); ok {
		return r
	}
	
	// Try to unwrap compressedLayerExtender or other wrappers
	type unwrapper interface {
		Unwrap() blob
	}
	if u, ok := layer.(unwrapper); ok {
		return tryResumable(u.Unwrap())
	}
	
	return nil
}

// writeLayer writes the layer blob to the store.
// It returns true when a new blob was created and the blob's DiffID.
func (s *LocalStore) writeLayer(layer blob, updates chan<- v1.Update) (bool, v1.Hash, error) {
	hash, err := layer.DiffID()
	if err != nil {
		return false, v1.Hash{}, fmt.Errorf("get file hash: %w", err)
	}
	hasBlob, err := s.hasBlob(hash)
	if err != nil {
		return false, v1.Hash{}, fmt.Errorf("check blob existence: %w", err)
	}
	if hasBlob {
		// TODO: write something to the progress channel (we probably need to redo progress reporting a little bit)
		return false, hash, nil
	}

	// Check if layer supports resumable downloads
	if resumableLayer := tryResumable(layer); resumableLayer != nil {
		created, err := s.writeBlobResumable(hash, resumableLayer, updates)
		return created, hash, err
	}

	// Fall back to regular download
	lr, err := layer.Uncompressed()
	if err != nil {
		return false, v1.Hash{}, fmt.Errorf("get blob contents: %w", err)
	}
	defer lr.Close()
	r := progress.NewReader(lr, updates)

	if err := s.WriteBlob(hash, r); err != nil {
		return false, hash, err
	}
	return true, hash, nil
}

// WriteBlob writes the blob to the store, reporting progress to the given channel.
// If the blob is already in the store, it is a no-op and the blob is not consumed from the reader.
func (s *LocalStore) WriteBlob(diffID v1.Hash, r io.Reader) error {
	hasBlob, err := s.hasBlob(diffID)
	if err != nil {
		return fmt.Errorf("check blob existence: %w", err)
	}
	if hasBlob {
		return nil
	}

	path, err := s.blobPath(diffID)
	if err != nil {
		return fmt.Errorf("get blob path: %w", err)
	}
	f, err := createFile(incompletePath(path))
	if err != nil {
		return fmt.Errorf("create blob file: %w", err)
	}
	defer os.Remove(incompletePath(path))
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("copy blob %q to store: %w", diffID.String(), err)
	}

	f.Close() // Rename will fail on Windows if the file is still open.
	if err := os.Rename(incompletePath(path), path); err != nil {
		return fmt.Errorf("rename blob file: %w", err)
	}
	return nil
}

// writeBlobResumable writes a blob to the store with resume support and retry logic.
// It returns true when a new blob was created.
func (s *LocalStore) writeBlobResumable(diffID v1.Hash, layer ResumableBlob, updates chan<- v1.Update) (bool, error) {
	const maxRetries = 3
	const initialBackoffSec = 1
	const backoffMultiplier = 2

	path, err := s.blobPath(diffID)
	if err != nil {
		return false, fmt.Errorf("get blob path: %w", err)
	}
	
	incompletePath := incompletePath(path)
	
	// Check for existing incomplete file
	var offset int64 = 0
	var supportsRange bool
	if info, statErr := os.Stat(incompletePath); statErr == nil {
		offset = info.Size()
		
		// Check if server supports range requests
		supportsRange, err = layer.SupportsRangeRequests()
		if err != nil {
			// If we can't check range support, log and try anyway
			fmt.Printf("Warning: failed to check range request support: %v\n", err)
			supportsRange = false
		}
		
		if !supportsRange {
			// Server doesn't support range requests, remove incomplete file and start fresh
			fmt.Printf("Server doesn't support range requests, starting download from scratch\n")
			if removeErr := os.Remove(incompletePath); removeErr != nil {
				fmt.Printf("Warning: failed to remove incomplete file: %v\n", removeErr)
			}
			offset = 0
		} else {
			fmt.Printf("Resuming download from offset %d bytes\n", offset)
		}
	}
	
	// Retry loop with exponential backoff
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Calculate backoff
			backoffDuration := time.Duration(initialBackoffSec * (1 << uint(attempt-1)) * backoffMultiplier) * time.Second
			fmt.Printf("Retry attempt %d/%d after %v\n", attempt, maxRetries, backoffDuration)
			time.Sleep(backoffDuration)
			
			// Re-check incomplete file size in case it changed
			if info, statErr := os.Stat(incompletePath); statErr == nil {
				offset = info.Size()
			}
		}
		
		// Open or create the incomplete file
		var f *os.File
		if offset > 0 {
			// Open for appending
			f, err = os.OpenFile(incompletePath, os.O_WRONLY|os.O_APPEND, 0666)
		} else {
			// Create new file
			f, err = createFile(incompletePath)
		}
		if err != nil {
			lastErr = fmt.Errorf("open/create incomplete file: %w", err)
			continue
		}
		
		// Get the reader (with offset if resuming)
		var lr io.ReadCloser
		if offset > 0 && supportsRange {
			lr, err = layer.CompressedWithOffset(offset)
		} else {
			lr, err = layer.Uncompressed()
		}
		if err != nil {
			f.Close()
			lastErr = fmt.Errorf("get blob contents: %w", err)
			continue
		}
		
		// Wrap with progress reporter
		r := progress.NewReader(lr, updates)
		
		// Copy data
		_, copyErr := io.Copy(f, r)
		lr.Close()
		f.Close()
		
		if copyErr != nil {
			lastErr = fmt.Errorf("copy blob data: %w", copyErr)
			// Don't remove incomplete file - keep it for resume
			continue
		}
		
		// Success! Rename the file
		if err := os.Rename(incompletePath, path); err != nil {
			return false, fmt.Errorf("rename blob file: %w", err)
		}
		
		return true, nil
	}
	
	// All retries failed
	return false, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

// removeBlob removes the blob with the given hash from the store.
func (s *LocalStore) removeBlob(hash v1.Hash) error {
	path, err := s.blobPath(hash)
	if err != nil {
		return fmt.Errorf("get blob path: %w", err)
	}
	return os.Remove(path)
}

func (s *LocalStore) hasBlob(hash v1.Hash) (bool, error) {
	path, err := s.blobPath(hash)
	if err != nil {
		return false, fmt.Errorf("get blob path: %w", err)
	}
	if _, err := os.Stat(path); err == nil {
		return true, nil
	}
	return false, nil
}

// createFile is a wrapper around os.Create that creates any parent directories as needed.
func createFile(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
		return nil, fmt.Errorf("create parent directory %q: %w", filepath.Dir(path), err)
	}
	return os.Create(path)
}

// incompletePath returns the path to the incomplete file for the given path.
func incompletePath(path string) string {
	return path + ".incomplete"
}

// writeConfigFile writes the model config JSON file to the blob store and reports whether the file was newly created.
func (s *LocalStore) writeConfigFile(mdl v1.Image) (bool, error) {
	hash, err := mdl.ConfigName()
	if err != nil {
		return false, fmt.Errorf("get digest: %w", err)
	}
	hasBlob, err := s.hasBlob(hash)
	if err != nil {
		return false, fmt.Errorf("check config existence: %w", err)
	}
	if hasBlob {
		return false, nil
	}

	path, err := s.blobPath(hash)
	if err != nil {
		return false, fmt.Errorf("get blob path: %w", err)
	}

	rcf, err := mdl.RawConfigFile()
	if err != nil {
		return false, fmt.Errorf("get raw manifest: %w", err)
	}
	if err := writeFile(path, rcf); err != nil {
		return false, err
	}
	return true, nil
}
