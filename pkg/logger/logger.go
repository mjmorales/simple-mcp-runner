// Package logger provides interfaces and types for logging
package logger

import (
	"context"
	"io"
)

// Logger interface defines the contract for logging.
type Logger interface {
	// Debug logs a debug message.
	Debug(msg string, args ...any)

	// Info logs an info message.
	Info(msg string, args ...any)

	// Warn logs a warning message.
	Warn(msg string, args ...any)

	// Error logs an error message.
	Error(msg string, args ...any)

	// Fatal logs a fatal message and exits.
	Fatal(msg string, args ...any)

	// WithContext returns a logger with context values.
	WithContext(ctx context.Context) Logger

	// WithError adds an error to the logger.
	WithError(err error) Logger

	// WithField adds a field to the logger.
	WithField(key string, value any) Logger

	// WithFields adds multiple fields to the logger.
	WithFields(fields map[string]any) Logger

	// IsDebugEnabled returns true if debug logging is enabled.
	IsDebugEnabled() bool
}

// LogLevel represents the log level.
type LogLevel string

const (
	// LevelDebug is the debug log level.
	LevelDebug LogLevel = "debug"
	// LevelInfo is the info log level.
	LevelInfo LogLevel = "info"
	// LevelWarn is the warn log level.
	LevelWarn LogLevel = "warn"
	// LevelError is the error log level.
	LevelError LogLevel = "error"
)

// LogFormat represents the log format.
type LogFormat string

const (
	// FormatText is the text log format.
	FormatText LogFormat = "text"
	// FormatJSON is the JSON log format.
	FormatJSON LogFormat = "json"
)

// Options configures a logger.
type Options struct {
	Level      LogLevel
	Format     LogFormat
	Output     io.Writer
	AddSource  bool
}

// DefaultOptions returns default logger options.
func DefaultOptions() Options {
	return Options{
		Level:      LevelInfo,
		Format:     FormatText,
		AddSource:  false,
	}
}

// Field represents a log field.
type Field struct {
	Key   string
	Value any
}

// Fields is a collection of log fields.
type Fields map[string]any

// LogEntry represents a log entry.
type LogEntry struct {
	Level   LogLevel
	Message string
	Fields  Fields
	Error   error
}

// Factory creates logger instances.
type Factory interface {
	// CreateLogger creates a new logger with the given options.
	CreateLogger(opts Options) (Logger, error)

	// CreateFromConfig creates a logger from configuration.
	CreateFromConfig(config LoggingConfig) (Logger, error)
}

// LoggingConfig contains logging configuration (matching config package).
type LoggingConfig struct {
	Level         string `yaml:"level,omitempty"`
	Format        string `yaml:"format,omitempty"`
	Output        string `yaml:"output,omitempty"`
	IncludeSource bool   `yaml:"include_source,omitempty"`
}

// NopLogger is a no-op logger implementation.
type NopLogger struct{}

// Debug implements the Logger interface.
func (l *NopLogger) Debug(msg string, args ...any) {}

// Info implements the Logger interface.
func (l *NopLogger) Info(msg string, args ...any) {}

// Warn implements the Logger interface.
func (l *NopLogger) Warn(msg string, args ...any) {}

// Error implements the Logger interface.
func (l *NopLogger) Error(msg string, args ...any) {}

// Fatal implements the Logger interface.
func (l *NopLogger) Fatal(msg string, args ...any) {}

// WithContext implements the Logger interface.
func (l *NopLogger) WithContext(ctx context.Context) Logger { return l }

// WithError implements the Logger interface.
func (l *NopLogger) WithError(err error) Logger { return l }

// WithField implements the Logger interface.
func (l *NopLogger) WithField(key string, value any) Logger { return l }

// WithFields implements the Logger interface.
func (l *NopLogger) WithFields(fields map[string]any) Logger { return l }

// IsDebugEnabled implements the Logger interface.
func (l *NopLogger) IsDebugEnabled() bool { return false }

// NewNopLogger creates a new no-op logger.
func NewNopLogger() Logger {
	return &NopLogger{}
}