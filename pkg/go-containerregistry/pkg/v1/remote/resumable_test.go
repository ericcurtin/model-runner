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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/docker/model-runner/pkg/go-containerregistry/pkg/name"
	v1 "github.com/docker/model-runner/pkg/go-containerregistry/pkg/v1"
)

// TestFetchBlobWithOffset tests resumable downloads with Range requests
func TestFetchBlobWithOffset(t *testing.T) {
	testCases := []struct {
		name           string
		offset         int64
		serverSupports bool
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "No offset - full download",
			offset:         0,
			serverSupports: true,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "With offset - server supports Range",
			offset:         20,
			serverSupports: true,
			expectedStatus: http.StatusPartialContent,
			expectError:    false,
		},
		{
			name:           "With offset - server doesn't support Range",
			offset:         20,
			serverSupports: false,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test data
			testData := []byte("This is test blob data for resumable download testing")
			hash, _, err := v1.SHA256(strings.NewReader(string(testData)))
			if err != nil {
				t.Fatalf("Failed to calculate hash: %v", err)
			}

			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/v2/" {
					w.WriteHeader(http.StatusOK)
					return
				}

				rangeHeader := r.Header.Get("Range")
				
				if tc.serverSupports && rangeHeader != "" {
					// Server supports Range requests
					w.Header().Set("Accept-Ranges", "bytes")
					w.WriteHeader(http.StatusPartialContent)
					// Return data starting from offset
					w.Write(testData[tc.offset:])
				} else {
					// Server doesn't support Range or no Range requested
					w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testData)))
					w.WriteHeader(http.StatusOK)
					w.Write(testData)
				}
			}))
			defer server.Close()

			// Create registry and repository
			reg, err := name.NewRegistry(strings.TrimPrefix(server.URL, "http://"), name.Insecure)
			if err != nil {
				t.Fatalf("Failed to create registry: %v", err)
			}

			repo := reg.Repo("test/repo")
			ctx := context.Background()

			// Create fetcher
			opts, err := makeOptions()
			if err != nil {
				t.Fatalf("Failed to create options: %v", err)
			}
			
			f, err := makeFetcher(ctx, repo, opts)
			if err != nil {
				t.Fatalf("Failed to create fetcher: %v", err)
			}

			// Test fetchBlobWithOffset
			rc, err := f.fetchBlobWithOffset(ctx, int64(len(testData)), hash, tc.offset)
			if tc.expectError {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			defer rc.Close()

			// Read and verify data
			data, err := io.ReadAll(rc)
			if err != nil {
				t.Fatalf("Failed to read data: %v", err)
			}

			// Verify we got the right data
			expectedData := testData
			if tc.serverSupports && tc.offset > 0 {
				expectedData = testData[tc.offset:]
			}
			
			if string(data) != string(expectedData) {
				t.Errorf("Data mismatch: got %q, want %q", string(data), string(expectedData))
			}
		})
	}
}

// TestSupportsRangeRequests tests the supportsRangeRequests method
func TestSupportsRangeRequests(t *testing.T) {
	testCases := []struct {
		name           string
		acceptRanges   string
		expectedResult bool
	}{
		{
			name:           "Server supports bytes range",
			acceptRanges:   "bytes",
			expectedResult: true,
		},
		{
			name:           "Server doesn't support range",
			acceptRanges:   "none",
			expectedResult: false,
		},
		{
			name:           "No Accept-Ranges header",
			acceptRanges:   "",
			expectedResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hash, _, err := v1.SHA256(strings.NewReader("test"))
			if err != nil {
				t.Fatalf("Failed to calculate hash: %v", err)
			}

			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/v2/" {
					w.WriteHeader(http.StatusOK)
					return
				}

				if r.Method == http.MethodHead {
					if tc.acceptRanges != "" {
						w.Header().Set("Accept-Ranges", tc.acceptRanges)
					}
					w.Header().Set("Content-Length", "100")
					w.Header().Set("Docker-Content-Digest", hash.String())
					w.WriteHeader(http.StatusOK)
				}
			}))
			defer server.Close()

			// Create registry and repository
			reg, err := name.NewRegistry(strings.TrimPrefix(server.URL, "http://"), name.Insecure)
			if err != nil {
				t.Fatalf("Failed to create registry: %v", err)
			}

			repo := reg.Repo("test/repo")
			ctx := context.Background()

			// Create fetcher
			opts, err := makeOptions()
			if err != nil {
				t.Fatalf("Failed to create options: %v", err)
			}
			
			f, err := makeFetcher(ctx, repo, opts)
			if err != nil {
				t.Fatalf("Failed to create fetcher: %v", err)
			}

			// Test supportsRangeRequests
			supports, err := f.supportsRangeRequests(ctx, hash)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if supports != tc.expectedResult {
				t.Errorf("Expected %v, got %v", tc.expectedResult, supports)
			}
		})
	}
}
