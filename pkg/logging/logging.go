package logging

import (
	"io"
)

// Logger is a flexible logging interface that can be implemented by both logrus and slog-based loggers
type Logger interface {
	// WithField creates a new logger with an additional field
	WithField(key string, value interface{}) Logger
	// WithFields creates a new logger with additional fields
	WithFields(fields map[string]interface{}) Logger
	// WithError creates a new logger with an error field
	WithError(err error) Logger

	// Standard logging methods
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Printf(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Warningf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	Panicf(format string, args ...interface{})

	Debug(args ...interface{})
	Info(args ...interface{})
	Print(args ...interface{})
	Warn(args ...interface{})
	Warning(args ...interface{})
	Error(args ...interface{})
	Fatal(args ...interface{})
	Panic(args ...interface{})

	Debugln(args ...interface{})
	Infoln(args ...interface{})
	Println(args ...interface{})
	Warnln(args ...interface{})
	Warningln(args ...interface{})
	Errorln(args ...interface{})
	Fatalln(args ...interface{})
	Panicln(args ...interface{})

	// Writer returns a PipeWriter that writes to the logger
	Writer() *io.PipeWriter
}
