// Copyright 2025 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package remote

import (
	"context"
	"fmt"
	"io"
	"math"
	"time"

	v1 "github.com/docker/model-runner/pkg/go-containerregistry/pkg/v1"
)

const (
	maxRetries        = 3
	initialBackoff    = 1 * time.Second
	backoffMultiplier = 2.0
)

// ResumableReader wraps a reader with retry and resume capabilities.
type ResumableReader struct {
	ctx         context.Context
	fetcher     *fetcher
	hash        v1.Hash
	totalSize   int64
	offset      int64
	currentBody io.ReadCloser
	attempt     int
}

// NewResumableReader creates a new ResumableReader that can resume downloads from a given offset.
func NewResumableReader(ctx context.Context, f *fetcher, h v1.Hash, totalSize, offset int64) *ResumableReader {
	return &ResumableReader{
		ctx:       ctx,
		fetcher:   f,
		hash:      h,
		totalSize: totalSize,
		offset:    offset,
		attempt:   0,
	}
}

// Read implements io.Reader with automatic retry and resume on failures.
func (r *ResumableReader) Read(p []byte) (n int, err error) {
	// If we don't have a body yet, or previous body was closed, open a new one
	if r.currentBody == nil {
		if err := r.openBody(); err != nil {
			return 0, err
		}
	}

	n, err = r.currentBody.Read(p)
	if err != nil && err != io.EOF {
		// Check if we should retry
		if r.attempt < maxRetries {
			// Close the current body
			r.currentBody.Close()
			r.currentBody = nil

			// Update offset based on bytes read
			r.offset += int64(n)

			// Calculate backoff
			backoff := time.Duration(float64(initialBackoff) * math.Pow(backoffMultiplier, float64(r.attempt)))
			r.attempt++

			// Wait before retry
			select {
			case <-r.ctx.Done():
				return n, r.ctx.Err()
			case <-time.After(backoff):
			}

			// Check if server supports range requests for resume
			supportsRange, rangeErr := r.fetcher.supportsRangeRequests(r.ctx, r.hash)
			if rangeErr != nil {
				return n, fmt.Errorf("checking range support (attempt %d/%d): %w", r.attempt, maxRetries, err)
			}

			if !supportsRange {
				// Server doesn't support range requests, cannot resume
				return n, fmt.Errorf("download failed and server doesn't support range requests for resume (attempt %d/%d): %w", r.attempt, maxRetries, err)
			}

			// Try to reopen the body at the new offset
			if reopenErr := r.openBody(); reopenErr != nil {
				return n, fmt.Errorf("failed to resume download (attempt %d/%d): %w (original error: %v)", r.attempt, maxRetries, reopenErr, err)
			}

			// Recursively try reading again
			additionalN, readErr := r.Read(p[n:])
			return n + additionalN, readErr
		}

		// Max retries exceeded
		return n, fmt.Errorf("max retries (%d) exceeded: %w", maxRetries, err)
	}

	// Update offset for successful reads
	r.offset += int64(n)
	return n, err
}

// Close implements io.Closer.
func (r *ResumableReader) Close() error {
	if r.currentBody != nil {
		return r.currentBody.Close()
	}
	return nil
}

// openBody opens or reopens the HTTP connection at the current offset.
func (r *ResumableReader) openBody() error {
	body, err := r.fetcher.fetchBlobWithOffset(r.ctx, r.totalSize, r.hash, r.offset)
	if err != nil {
		return err
	}
	r.currentBody = body
	return nil
}
