package store

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/model-runner/pkg/distribution/internal/progress"

	v1 "github.com/google/go-containerregistry/pkg/v1"
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
// If an incomplete file exists from a previous interrupted download, it will attempt to resume
// by skipping bytes already written. Note: This requires the reader to support reading from the
// beginning; if the skip fails, the incomplete file is removed and download starts from scratch.
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
	incompletePath := incompletePath(path)

	// Check if incomplete file exists from a previous download attempt
	var f *os.File
	var bytesToSkip int64
	if info, err := os.Stat(incompletePath); err == nil && info.Size() > 0 {
		// Incomplete file exists, open for append
		bytesToSkip = info.Size()
		f, err = os.OpenFile(incompletePath, os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return fmt.Errorf("open incomplete blob file: %w", err)
		}
	} else {
		// No incomplete file or it's empty, create new one
		f, err = createFile(incompletePath)
		if err != nil {
			return fmt.Errorf("create blob file: %w", err)
		}
	}
	defer f.Close()

	// If we're resuming, we need to skip bytes already written.
	// Since the reader provides the full blob, we discard the bytes we already have.
	if bytesToSkip > 0 {
		discarded, err := io.CopyN(io.Discard, r, bytesToSkip)
		if err != nil || discarded != bytesToSkip {
			// If we can't skip the expected bytes, the incomplete file may be corrupt
			// or the reader may not provide the full content. Start over.
			f.Close()
			os.Remove(incompletePath)
			f, err = createFile(incompletePath)
			if err != nil {
				return fmt.Errorf("create blob file after skip failure: %w", err)
			}
			// Note: we don't add another defer f.Close() here because we already have one
			// The old file handle is already closed above
		}
	}

	// Copy the remaining data
	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("copy blob %q to store: %w", diffID.String(), err)
	}

	f.Close() // Rename will fail on Windows if the file is still open.
	if err := os.Rename(incompletePath, path); err != nil {
		return fmt.Errorf("rename blob file: %w", err)
	}

	// Clean up incomplete file on success (it's now renamed to final path)
	os.Remove(incompletePath)
	return nil
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
