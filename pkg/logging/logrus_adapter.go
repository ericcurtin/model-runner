package logging

import (
	"io"

	"github.com/sirupsen/logrus"
)

// LogrusAdapter wraps a logrus logger to implement our Logger interface
type LogrusAdapter struct {
	logger *logrus.Logger
	entry  *logrus.Entry
}

// NewLogrusAdapter creates a new adapter from a logrus.Logger
func NewLogrusAdapter(logger *logrus.Logger) Logger {
	return &LogrusAdapter{
		logger: logger,
		entry:  logrus.NewEntry(logger),
	}
}

// NewLogrusAdapterFromEntry creates a new adapter from a logrus.Entry
func NewLogrusAdapterFromEntry(entry *logrus.Entry) Logger {
	return &LogrusAdapter{
		logger: entry.Logger,
		entry:  entry,
	}
}

// WithField creates a new logger with an additional field
func (l *LogrusAdapter) WithField(key string, value interface{}) Logger {
	return &LogrusAdapter{
		logger: l.logger,
		entry:  l.entry.WithField(key, value),
	}
}

// WithFields creates a new logger with additional fields
func (l *LogrusAdapter) WithFields(fields map[string]interface{}) Logger {
	return &LogrusAdapter{
		logger: l.logger,
		entry:  l.entry.WithFields(logrus.Fields(fields)),
	}
}

// WithError creates a new logger with an error field
func (l *LogrusAdapter) WithError(err error) Logger {
	return &LogrusAdapter{
		logger: l.logger,
		entry:  l.entry.WithError(err),
	}
}

// Debug logs a message at Debug level
func (l *LogrusAdapter) Debug(args ...interface{}) {
	l.entry.Debug(args...)
}

// Debugf logs a formatted message at Debug level
func (l *LogrusAdapter) Debugf(format string, args ...interface{}) {
	l.entry.Debugf(format, args...)
}

// Debugln logs a message at Debug level with a newline
func (l *LogrusAdapter) Debugln(args ...interface{}) {
	l.entry.Debugln(args...)
}

// Info logs a message at Info level
func (l *LogrusAdapter) Info(args ...interface{}) {
	l.entry.Info(args...)
}

// Infof logs a formatted message at Info level
func (l *LogrusAdapter) Infof(format string, args ...interface{}) {
	l.entry.Infof(format, args...)
}

// Infoln logs a message at Info level with a newline
func (l *LogrusAdapter) Infoln(args ...interface{}) {
	l.entry.Infoln(args...)
}

// Warn logs a message at Warn level
func (l *LogrusAdapter) Warn(args ...interface{}) {
	l.entry.Warn(args...)
}

// Warnf logs a formatted message at Warn level
func (l *LogrusAdapter) Warnf(format string, args ...interface{}) {
	l.entry.Warnf(format, args...)
}

// Warnln logs a message at Warn level with a newline
func (l *LogrusAdapter) Warnln(args ...interface{}) {
	l.entry.Warnln(args...)
}

// Warning logs a message at Warn level (alias for Warn)
func (l *LogrusAdapter) Warning(args ...interface{}) {
	l.entry.Warning(args...)
}

// Warningf logs a formatted message at Warn level (alias for Warnf)
func (l *LogrusAdapter) Warningf(format string, args ...interface{}) {
	l.entry.Warningf(format, args...)
}

// Warningln logs a message at Warn level with a newline (alias for Warnln)
func (l *LogrusAdapter) Warningln(args ...interface{}) {
	l.entry.Warningln(args...)
}

// Error logs a message at Error level
func (l *LogrusAdapter) Error(args ...interface{}) {
	l.entry.Error(args...)
}

// Errorf logs a formatted message at Error level
func (l *LogrusAdapter) Errorf(format string, args ...interface{}) {
	l.entry.Errorf(format, args...)
}

// Errorln logs a message at Error level with a newline
func (l *LogrusAdapter) Errorln(args ...interface{}) {
	l.entry.Errorln(args...)
}

// Fatal logs a message at Error level and exits
func (l *LogrusAdapter) Fatal(args ...interface{}) {
	l.entry.Fatal(args...)
}

// Fatalf logs a formatted message at Error level and exits
func (l *LogrusAdapter) Fatalf(format string, args ...interface{}) {
	l.entry.Fatalf(format, args...)
}

// Fatalln logs a message at Error level with a newline and exits
func (l *LogrusAdapter) Fatalln(args ...interface{}) {
	l.entry.Fatalln(args...)
}

// Panic logs a message at Error level and panics
func (l *LogrusAdapter) Panic(args ...interface{}) {
	l.entry.Panic(args...)
}

// Panicf logs a formatted message at Error level and panics
func (l *LogrusAdapter) Panicf(format string, args ...interface{}) {
	l.entry.Panicf(format, args...)
}

// Panicln logs a message at Error level with a newline and panics
func (l *LogrusAdapter) Panicln(args ...interface{}) {
	l.entry.Panicln(args...)
}

// Print logs a message at Info level (for compatibility)
func (l *LogrusAdapter) Print(args ...interface{}) {
	l.entry.Print(args...)
}

// Printf logs a formatted message at Info level (for compatibility)
func (l *LogrusAdapter) Printf(format string, args ...interface{}) {
	l.entry.Printf(format, args...)
}

// Println logs a message at Info level with a newline (for compatibility)
func (l *LogrusAdapter) Println(args ...interface{}) {
	l.entry.Println(args...)
}

// Writer returns a PipeWriter that writes to the logger at Info level
func (l *LogrusAdapter) Writer() *io.PipeWriter {
	return l.logger.Writer()
}
