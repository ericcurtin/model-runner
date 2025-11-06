# Resumable and Parallel Downloads

This document describes the implementation of resumable and parallel downloads for model files in the model-runner project.

## Overview

The model-runner now supports:
1. **Parallel Downloads**: Using 4 concurrent HTTP connections per model layer
2. **Resumable Downloads**: Automatic retry and resumption of interrupted downloads
3. **Enhanced Progress Reporting**: User-friendly progress messages indicating parallel connection usage

## Architecture

### Transport Stack

The download mechanism uses a layered transport architecture:

```
┌─────────────────────────────────────┐
│   go-containerregistry/remote       │  ← High-level API
└─────────────────────────────────────┘
               ↓
┌─────────────────────────────────────┐
│   Resumable Transport               │  ← Handles mid-stream failures
│   - Retries with exponential backoff│
│   - Uses HTTP Range + If-Range      │
│   - Max 3 retries, 200ms-5s backoff │
└─────────────────────────────────────┘
               ↓
┌─────────────────────────────────────┐
│   Parallel Transport                │  ← Downloads with 4 connections
│   - Splits files into chunks        │
│   - 1MB minimum chunk size          │
│   - Uses HTTP Range requests        │
└─────────────────────────────────────┘
               ↓
┌─────────────────────────────────────┐
│   HTTP Transport (net/http)         │  ← Standard HTTP client
└─────────────────────────────────────┘
```

### Key Components

#### 1. Registry Client (`pkg/distribution/registry/client.go`)

The registry client creates the transport stack using `createDefaultTransport()`:

```go
func createDefaultTransport() http.RoundTripper {
    // First, wrap with parallel downloading (4 connections)
    parallelTransport := parallel.New(
        remote.DefaultTransport,
        parallel.WithMaxConcurrentPerRequest(4),
        parallel.WithMinChunkSize(1024*1024), // 1MB chunks
    )
    
    // Then wrap with resumable transport for reliability
    return resumable.New(
        parallelTransport,
        resumable.WithMaxRetries(3),
    )
}
```

#### 2. Parallel Transport (`pkg/distribution/transport/parallel/`)

Features:
- Performs a HEAD request to check if server supports byte ranges
- Splits large files into chunks (minimum 1MB per chunk)
- Downloads chunks concurrently with 4 connections
- Uses temporary FIFO files to buffer chunk data
- Stitches chunks together transparently
- Falls back to single connection if server doesn't support ranges

**How it works:**
1. HEAD request checks for `Accept-Ranges: bytes` header
2. Calculates chunk boundaries based on file size
3. Issues concurrent GET requests with `Range: bytes=start-end` headers
4. Writes chunks to temporary files
5. Presents a single `io.ReadCloser` that reads from all chunks in order

#### 3. Resumable Transport (`pkg/distribution/transport/resumable/`)

Features:
- Wraps response bodies to detect mid-stream failures
- Automatically issues Range requests to resume from last byte received
- Uses `If-Range` headers with ETag or Last-Modified for safety
- Implements exponential backoff: 200ms → 400ms → 800ms → up to 5s
- Maximum 3 retry attempts per failure

**How it works:**
1. Wraps the response body with a resumable reader
2. If `Read()` returns an error (connection failure), starts retry logic
3. Issues a new request with `Range: bytes=N-` where N is the last byte received
4. Uses `If-Range` validator to ensure file hasn't changed
5. Continues reading from new response as if nothing happened

#### 4. Blob Storage (`pkg/distribution/internal/store/blobs.go`)

Enhanced to support incomplete file tracking:
- Creates files with `.incomplete` suffix during download
- Checks for existing `.incomplete` files on retry
- Opens in append mode to continue from where it left off
- Renames to final name only when complete

### Progress Reporting

The desktop client (`cmd/cli/desktop/desktop.go`) tracks:
- Per-layer progress (bytes downloaded vs total size)
- Aggregate progress across all layers
- Displays "using 4 parallel connections" to inform users

Example output:
```
Downloaded 2.5 GB of 5.0 GB (using 4 parallel connections)
```

## Benefits

### 1. Faster Downloads
- **4x potential speedup** for network-bound downloads
- Particularly effective for large model files (multi-GB)
- Maximizes available bandwidth by parallelizing requests

### 2. Reliability
- Automatic recovery from transient network failures
- No need to restart entire download after interruption
- Exponential backoff prevents overwhelming the server
- Validator checks (ETag/Last-Modified) ensure data integrity

### 3. Bandwidth Efficiency
- Resume from last byte received, not from start
- Parallel chunks use available bandwidth more efficiently
- Falls back gracefully if server doesn't support features

## Usage

No changes required - the enhancements work automatically:

```bash
# Downloads now use 4 parallel connections and automatically resume on interruption
docker model pull ai/gpt-oss

# If interrupted (Ctrl+C, network failure, system crash):
# [5GB of 10GB downloaded]

# Running again resumes from where it left off:
docker model pull ai/gpt-oss
# [Resumes from 5GB, downloads remaining 5GB]
```

## Implementation Details

### Chunk Size Selection

The parallel transport uses 1MB minimum chunk size:
- Small enough to benefit from parallelization
- Large enough to avoid excessive overhead
- Balances memory usage and performance

For a 10GB file:
- With 4 connections: ~2.5GB per connection
- Actual chunks: dynamically calculated based on file size

### Retry Strategy

The resumable transport uses exponential backoff:
```
Attempt 0: No delay (initial request)
Attempt 1: 200ms ± 20% jitter
Attempt 2: 400ms ± 20% jitter
Attempt 3: 800ms ± 20% jitter
...
Max:      5000ms ± 20% jitter
```

Jitter (±20%) prevents thundering herd when multiple clients retry simultaneously.

### Compatibility

**Requires:**
- Server support for `Accept-Ranges: bytes`
- Server support for `Range` and `If-Range` headers
- Content-Length header in response

**Falls back when:**
- Server doesn't advertise range support
- File is compressed (`Content-Encoding` present)
- File is too small (< 4MB for parallel)
- Server returns errors on range requests

## Testing

### Unit Tests
```bash
# Test parallel transport
go test ./pkg/distribution/transport/parallel/...

# Test resumable transport
go test ./pkg/distribution/transport/resumable/...

# Test registry client integration
go test ./pkg/distribution/registry/...
```

### Integration Tests

The repository includes integration tests that verify:
1. Parallel transport properly splits requests
2. Resumable transport retries on failures
3. Both transports work together correctly

### Manual Testing

To test with actual models:

```bash
# Build the project
make build

# Pull a large model
./model-runner model pull huggingface/large-model

# Interrupt during download (Ctrl+C)
# Then retry - should resume automatically
./model-runner model pull huggingface/large-model
```

## Troubleshooting

### Downloads Don't Use Parallel Connections

**Possible causes:**
1. File is too small (< 4MB)
2. Server doesn't support range requests
3. File is compressed (Content-Encoding header)

**Check:** Look for "using 4 parallel connections" in progress output

### Downloads Don't Resume

**Possible causes:**
1. Server doesn't support range requests
2. File changed between requests (ETag/Last-Modified changed)
3. Temporary failure limit exceeded (3 retries)

**Check:** Logs will show retry attempts and reasons for fallback

### Performance Issues

**If downloads are slow:**
1. Check network bandwidth utilization
2. Verify server supports concurrent connections
3. Check for rate limiting on server side
4. Consider increasing chunk size for very large files

## Future Enhancements

Potential improvements:
1. **Persistent resume state**: Store progress across process restarts
2. **Configurable connection count**: Allow users to specify number of parallel connections
3. **Adaptive chunk sizing**: Dynamically adjust based on network conditions
4. **Progress bar per connection**: Show 4 separate progress bars for each connection
5. **Delta resumption**: Resume only failed chunks rather than entire layers

## References

- Parallel Transport: `pkg/distribution/transport/parallel/transport.go`
- Resumable Transport: `pkg/distribution/transport/resumable/transport.go`
- Registry Client: `pkg/distribution/registry/client.go`
- Blob Storage: `pkg/distribution/internal/store/blobs.go`
- Progress Reporting: `pkg/distribution/internal/progress/reporter.go`
