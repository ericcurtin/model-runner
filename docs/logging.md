# Logging in Model Runner

Model Runner uses Go's standard library `log/slog` for structured logging with configurable log levels.

## Log Levels

The application supports the following log levels:
- **Debug**: Detailed diagnostic information (only shown when debug mode is enabled)
- **Info**: General informational messages (default)
- **Warn**: Warning messages for potentially problematic situations
- **Error**: Error messages for failures that don't stop the application
- **Fatal**: Critical errors that cause the application to exit

## Enabling Debug Logging

### For the Main Server

Set the `DEBUG` environment variable to `1` to enable debug-level logging:

```bash
DEBUG=1 ./model-runner
```

Or in Docker:

```bash
docker run -e DEBUG=1 docker/model-runner:latest
```

### For CLI Commands

The `docker model run` command supports a `--debug` flag:

```bash
docker model run --debug <model-name>
```

## Structured Logging

The logger supports structured logging with fields:

```go
// Add a single field
log.WithField("component", "scheduler").Info("Starting scheduler")

// Add multiple fields
log.WithFields(map[string]interface{}{
    "component": "backend",
    "model": "llama-3",
}).Info("Loading model")

// Add error context
log.WithError(err).Error("Failed to load model")
```

## Log Output Format

Logs are output in slog's text format:

```
time=2025-11-12T15:55:44.435Z level=INFO msg="Successfully initialized store" component=model-manager
time=2025-11-12T15:55:44.435Z level=DEBUG msg="Loading model from cache" model=llama-3 component=loader
time=2025-11-12T15:55:44.436Z level=WARN msg="Model cache miss, pulling from registry" model=llama-3
time=2025-11-12T15:55:44.437Z level=ERROR msg="Failed to connect to registry" error="connection timeout"
```

## Implementation Details

- The logging implementation is in `pkg/logging/`
- `SlogLogger` wraps `log/slog` and provides compatibility with the existing logging interface
- `LogrusAdapter` provides backward compatibility for code still using logrus
- The log level is set at startup based on the `DEBUG` environment variable
- Tests are available in `pkg/logging/slog_test.go`
