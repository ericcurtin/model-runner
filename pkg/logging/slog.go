package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
)

// InfoLevel returns slog.LevelInfo
func InfoLevel() slog.Level {
	return slog.LevelInfo
}

// SlogLogger is a wrapper around slog.Logger that implements the logrus.FieldLogger interface
// This allows gradual migration from logrus to slog while maintaining compatibility
type SlogLogger struct {
	logger   *slog.Logger
	fields   map[string]interface{}
	exitFunc func(int) // For testing purposes, to override os.Exit behavior
}

// NewSlogLogger creates a new slog-based logger with the specified level
func NewSlogLogger(level slog.Level, writer io.Writer) *SlogLogger {
	if writer == nil {
		writer = os.Stderr
	}
	
	handler := slog.NewTextHandler(writer, &slog.HandlerOptions{
		Level: level,
	})
	
	return &SlogLogger{
		logger:   slog.New(handler),
		fields:   make(map[string]interface{}),
		exitFunc: os.Exit,
	}
}

// NewSlogLoggerFromLogger creates a SlogLogger from an existing slog.Logger
func NewSlogLoggerFromLogger(logger *slog.Logger) *SlogLogger {
	return &SlogLogger{
		logger:   logger,
		fields:   make(map[string]interface{}),
		exitFunc: os.Exit,
	}
}

// SetExitFunc sets the function to call on fatal errors (for testing)
func (s *SlogLogger) SetExitFunc(fn func(int)) {
	s.exitFunc = fn
}

// buildArgs converts fields to slog attributes and appends message
func (s *SlogLogger) buildArgs(args ...interface{}) []interface{} {
	// Convert fields map to slog attributes
	slogArgs := make([]interface{}, 0, len(s.fields)*2+len(args))
	for k, v := range s.fields {
		slogArgs = append(slogArgs, k, v)
	}
	slogArgs = append(slogArgs, args...)
	return slogArgs
}

// Debug logs a message at Debug level
func (s *SlogLogger) Debug(args ...interface{}) {
	s.logger.Debug(fmt.Sprint(args...), s.buildArgs()...)
}

// Debugf logs a formatted message at Debug level
func (s *SlogLogger) Debugf(format string, args ...interface{}) {
	s.logger.Debug(fmt.Sprintf(format, args...), s.buildArgs()...)
}

// Debugln logs a message at Debug level with a newline
func (s *SlogLogger) Debugln(args ...interface{}) {
	s.logger.Debug(fmt.Sprintln(args...), s.buildArgs()...)
}

// Info logs a message at Info level
func (s *SlogLogger) Info(args ...interface{}) {
	s.logger.Info(fmt.Sprint(args...), s.buildArgs()...)
}

// Infof logs a formatted message at Info level
func (s *SlogLogger) Infof(format string, args ...interface{}) {
	s.logger.Info(fmt.Sprintf(format, args...), s.buildArgs()...)
}

// Infoln logs a message at Info level with a newline
func (s *SlogLogger) Infoln(args ...interface{}) {
	s.logger.Info(fmt.Sprintln(args...), s.buildArgs()...)
}

// Warn logs a message at Warn level
func (s *SlogLogger) Warn(args ...interface{}) {
	s.logger.Warn(fmt.Sprint(args...), s.buildArgs()...)
}

// Warnf logs a formatted message at Warn level
func (s *SlogLogger) Warnf(format string, args ...interface{}) {
	s.logger.Warn(fmt.Sprintf(format, args...), s.buildArgs()...)
}

// Warnln logs a message at Warn level with a newline
func (s *SlogLogger) Warnln(args ...interface{}) {
	s.logger.Warn(fmt.Sprintln(args...), s.buildArgs()...)
}

// Warning logs a message at Warn level (alias for Warn)
func (s *SlogLogger) Warning(args ...interface{}) {
	s.Warn(args...)
}

// Warningf logs a formatted message at Warn level (alias for Warnf)
func (s *SlogLogger) Warningf(format string, args ...interface{}) {
	s.Warnf(format, args...)
}

// Warningln logs a message at Warn level with a newline (alias for Warnln)
func (s *SlogLogger) Warningln(args ...interface{}) {
	s.Warnln(args...)
}

// Error logs a message at Error level
func (s *SlogLogger) Error(args ...interface{}) {
	s.logger.Error(fmt.Sprint(args...), s.buildArgs()...)
}

// Errorf logs a formatted message at Error level
func (s *SlogLogger) Errorf(format string, args ...interface{}) {
	s.logger.Error(fmt.Sprintf(format, args...), s.buildArgs()...)
}

// Errorln logs a message at Error level with a newline
func (s *SlogLogger) Errorln(args ...interface{}) {
	s.logger.Error(fmt.Sprintln(args...), s.buildArgs()...)
}

// Fatal logs a message at Error level and exits (slog doesn't have Fatal, so we use Error + exit)
func (s *SlogLogger) Fatal(args ...interface{}) {
	s.logger.Error(fmt.Sprint(args...), s.buildArgs()...)
	s.exitFunc(1)
}

// Fatalf logs a formatted message at Error level and exits
func (s *SlogLogger) Fatalf(format string, args ...interface{}) {
	s.logger.Error(fmt.Sprintf(format, args...), s.buildArgs()...)
	s.exitFunc(1)
}

// Fatalln logs a message at Error level with a newline and exits
func (s *SlogLogger) Fatalln(args ...interface{}) {
	s.logger.Error(fmt.Sprintln(args...), s.buildArgs()...)
	s.exitFunc(1)
}

// Panic logs a message at Error level and panics
func (s *SlogLogger) Panic(args ...interface{}) {
	msg := fmt.Sprint(args...)
	s.logger.Error(msg, s.buildArgs()...)
	panic(msg)
}

// Panicf logs a formatted message at Error level and panics
func (s *SlogLogger) Panicf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	s.logger.Error(msg, s.buildArgs()...)
	panic(msg)
}

// Panicln logs a message at Error level with a newline and panics
func (s *SlogLogger) Panicln(args ...interface{}) {
	msg := fmt.Sprintln(args...)
	s.logger.Error(msg, s.buildArgs()...)
	panic(msg)
}

// WithField creates a new logger with an additional field
func (s *SlogLogger) WithField(key string, value interface{}) Logger {
	newFields := make(map[string]interface{}, len(s.fields)+1)
	for k, v := range s.fields {
		newFields[k] = v
	}
	newFields[key] = value
	
	// Create a new logger with the additional field
	attrs := make([]slog.Attr, 0, len(newFields))
	for k, v := range newFields {
		attrs = append(attrs, slog.Any(k, v))
	}
	
	newLogger := s.logger.With(s.convertFieldsToArgs(newFields)...)
	
	return &SlogLogger{
		logger:   newLogger,
		fields:   newFields,
		exitFunc: s.exitFunc,
	}
}

// WithFields creates a new logger with additional fields
func (s *SlogLogger) WithFields(fields map[string]interface{}) Logger {
	newFields := make(map[string]interface{}, len(s.fields)+len(fields))
	for k, v := range s.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}
	
	newLogger := s.logger.With(s.convertFieldsToArgs(newFields)...)
	
	return &SlogLogger{
		logger:   newLogger,
		fields:   newFields,
		exitFunc: s.exitFunc,
	}
}

// WithError creates a new logger with an error field
func (s *SlogLogger) WithError(err error) Logger {
	return s.WithField("error", err)
}

// Writer returns a PipeWriter that writes to the logger at Info level
func (s *SlogLogger) Writer() *io.PipeWriter {
	reader, writer := io.Pipe()
	
	go func() {
		scanner := io.Reader(reader)
		buf := make([]byte, 4096)
		for {
			n, err := scanner.Read(buf)
			if n > 0 {
				s.Info(string(buf[:n]))
			}
			if err != nil {
				break
			}
		}
	}()
	
	return writer
}

// convertFieldsToArgs converts a map of fields to slog arguments
func (s *SlogLogger) convertFieldsToArgs(fields map[string]interface{}) []interface{} {
	args := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return args
}

// Print logs a message at Info level (for compatibility)
func (s *SlogLogger) Print(args ...interface{}) {
	s.Info(args...)
}

// Printf logs a formatted message at Info level (for compatibility)
func (s *SlogLogger) Printf(format string, args ...interface{}) {
	s.Infof(format, args...)
}

// Println logs a message at Info level with a newline (for compatibility)
func (s *SlogLogger) Println(args ...interface{}) {
	s.Infoln(args...)
}

// Trace logs a message at Debug level (slog doesn't have Trace, map to Debug)
func (s *SlogLogger) Trace(args ...interface{}) {
	s.Debug(args...)
}

// Tracef logs a formatted message at Debug level
func (s *SlogLogger) Tracef(format string, args ...interface{}) {
	s.Debugf(format, args...)
}

// Traceln logs a message at Debug level with a newline
func (s *SlogLogger) Traceln(args ...interface{}) {
	s.Debugln(args...)
}
