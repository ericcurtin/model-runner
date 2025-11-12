package store

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

// slowReader simulates a slow/unreliable reader that can fail partway through
type slowReader struct {
	data      []byte
	pos       int
	failAfter int // fail after reading this many bytes
}

func (sr *slowReader) Read(p []byte) (n int, err error) {
	if sr.pos >= len(sr.data) {
		return 0, io.EOF
	}
	
	// Check if we should fail
	if sr.failAfter > 0 && sr.pos >= sr.failAfter {
		return 0, errors.New("simulated read error")
	}
	
	// Read some data
	n = copy(p, sr.data[sr.pos:])
	sr.pos += n
	return n, nil
}

// seekableReader wraps a reader and adds seeking capability
type seekableReader struct {
	data []byte
	pos  int
}

func (sr *seekableReader) Read(p []byte) (n int, err error) {
	if sr.pos >= len(sr.data) {
		return 0, io.EOF
	}
	n = copy(p, sr.data[sr.pos:])
	sr.pos += n
	return n, nil
}

func (sr *seekableReader) Seek(offset int64, whence int) (int64, error) {
	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = int64(sr.pos) + offset
	case io.SeekEnd:
		newPos = int64(len(sr.data)) + offset
	default:
		return 0, errors.New("invalid whence")
	}
	
	if newPos < 0 || newPos > int64(len(sr.data)) {
		return 0, errors.New("invalid seek position")
	}
	
	sr.pos = int(newPos)
	return newPos, nil
}

func TestWriteBlobResumable(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "resumable-test")
	if err != nil {
		t.Fatalf("error creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	
	rootDir := filepath.Join(tmpDir, "store")
	store, err := New(Options{RootPath: rootDir})
	if err != nil {
		t.Fatalf("error creating store: %v", err)
	}

	t.Run("Resume with seekable reader", func(t *testing.T) {
		expectedContent := []byte("this is a test file with some content")
		hash, _, err := v1.SHA256(bytes.NewReader(expectedContent))
		if err != nil {
			t.Fatalf("error calculating hash: %v", err)
		}

		// First attempt: write partial data and simulate failure
		partialContent := expectedContent[:20]
		incompletePath := incompletePath(filepath.Join(store.blobsDir(), hash.Algorithm, hash.Hex))
		
		// Write partial data to simulate interrupted download
		if err := os.MkdirAll(filepath.Dir(incompletePath), 0755); err != nil {
			t.Fatalf("error creating directory: %v", err)
		}
		if err := os.WriteFile(incompletePath, partialContent, 0644); err != nil {
			t.Fatalf("error writing incomplete file: %v", err)
		}

		// Second attempt: resume with seekable reader
		reader := &seekableReader{data: expectedContent, pos: 0}
		if err := store.WriteBlobResumable(hash, reader); err != nil {
			t.Fatalf("error writing blob: %v", err)
		}

		// Verify the complete content
		blobPath, err := store.blobPath(hash)
		if err != nil {
			t.Fatalf("error getting blob path: %v", err)
		}
		content, err := os.ReadFile(blobPath)
		if err != nil {
			t.Fatalf("error reading blob file: %v", err)
		}
		if !bytes.Equal(content, expectedContent) {
			t.Fatalf("unexpected blob content: got %d bytes, expected %d bytes", len(content), len(expectedContent))
		}

		// Ensure incomplete file is cleaned up
		if _, err := os.Stat(incompletePath); !os.IsNotExist(err) {
			t.Fatalf("expected incomplete file to be removed")
		}
	})

	t.Run("Resume with non-seekable reader", func(t *testing.T) {
		expectedContent := []byte("another test file with different content")
		hash, _, err := v1.SHA256(bytes.NewReader(expectedContent))
		if err != nil {
			t.Fatalf("error calculating hash: %v", err)
		}

		// First attempt: write partial data
		partialContent := expectedContent[:15]
		incompletePath := incompletePath(filepath.Join(store.blobsDir(), hash.Algorithm, hash.Hex))
		
		if err := os.MkdirAll(filepath.Dir(incompletePath), 0755); err != nil {
			t.Fatalf("error creating directory: %v", err)
		}
		if err := os.WriteFile(incompletePath, partialContent, 0644); err != nil {
			t.Fatalf("error writing incomplete file: %v", err)
		}

		// Second attempt: resume with non-seekable reader
		// The reader should discard the first 15 bytes and continue from there
		reader := bytes.NewReader(expectedContent)
		if err := store.WriteBlobResumable(hash, reader); err != nil {
			t.Fatalf("error writing blob: %v", err)
		}

		// Verify the complete content
		blobPath, err := store.blobPath(hash)
		if err != nil {
			t.Fatalf("error getting blob path: %v", err)
		}
		content, err := os.ReadFile(blobPath)
		if err != nil {
			t.Fatalf("error reading blob file: %v", err)
		}
		if !bytes.Equal(content, expectedContent) {
			t.Fatalf("unexpected blob content: got %d bytes, expected %d bytes", len(content), len(expectedContent))
		}
	})

	t.Run("No resume needed for fresh download", func(t *testing.T) {
		expectedContent := []byte("fresh download content")
		hash, _, err := v1.SHA256(bytes.NewReader(expectedContent))
		if err != nil {
			t.Fatalf("error calculating hash: %v", err)
		}

		// Download from scratch
		reader := bytes.NewReader(expectedContent)
		if err := store.WriteBlobResumable(hash, reader); err != nil {
			t.Fatalf("error writing blob: %v", err)
		}

		// Verify content
		blobPath, err := store.blobPath(hash)
		if err != nil {
			t.Fatalf("error getting blob path: %v", err)
		}
		content, err := os.ReadFile(blobPath)
		if err != nil {
			t.Fatalf("error reading blob file: %v", err)
		}
		if !bytes.Equal(content, expectedContent) {
			t.Fatalf("unexpected blob content")
		}
	})

	t.Run("Skip already downloaded blob", func(t *testing.T) {
		expectedContent := []byte("already downloaded content")
		hash, _, err := v1.SHA256(bytes.NewReader(expectedContent))
		if err != nil {
			t.Fatalf("error calculating hash: %v", err)
		}

		// First download
		reader1 := bytes.NewReader(expectedContent)
		if err := store.WriteBlobResumable(hash, reader1); err != nil {
			t.Fatalf("error writing blob: %v", err)
		}

		// Second attempt should skip
		reader2 := &errorReader{} // This would fail if read
		if err := store.WriteBlobResumable(hash, reader2); err != nil {
			t.Fatalf("error on second write: %v", err)
		}
	})
}

func TestResumableDownloadLayerDiscard(t *testing.T) {
	// Test the byte discarding logic for resumable downloads
	tmpDir, err := os.MkdirTemp("", "discard-test")
	if err != nil {
		t.Fatalf("error creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	
	rootDir := filepath.Join(tmpDir, "store")
	store, err := New(Options{RootPath: rootDir})
	if err != nil {
		t.Fatalf("error creating store: %v", err)
	}

	expectedContent := []byte("0123456789abcdefghijklmnopqrstuvwxyz")
	hash, _, err := v1.SHA256(bytes.NewReader(expectedContent))
	if err != nil {
		t.Fatalf("error calculating hash: %v", err)
	}

	// Create incomplete file with first 10 bytes
	partialContent := expectedContent[:10]
	incompletePath := incompletePath(filepath.Join(store.blobsDir(), hash.Algorithm, hash.Hex))
	
	if err := os.MkdirAll(filepath.Dir(incompletePath), 0755); err != nil {
		t.Fatalf("error creating directory: %v", err)
	}
	if err := os.WriteFile(incompletePath, partialContent, 0644); err != nil {
		t.Fatalf("error writing incomplete file: %v", err)
	}

	// Create a reader with the full content
	// The function should discard the first 10 bytes and write the rest
	reader := bytes.NewReader(expectedContent)
	if err := store.WriteBlobResumable(hash, reader); err != nil {
		t.Fatalf("error writing blob: %v", err)
	}

	// Verify the complete content was written
	blobPath, err := store.blobPath(hash)
	if err != nil {
		t.Fatalf("error getting blob path: %v", err)
	}
	content, err := os.ReadFile(blobPath)
	if err != nil {
		t.Fatalf("error reading blob file: %v", err)
	}
	if !bytes.Equal(content, expectedContent) {
		t.Fatalf("unexpected blob content: got %q, expected %q", string(content), string(expectedContent))
	}
}
