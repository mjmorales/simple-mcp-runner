// Package logger provides structured logging for the MCP server
package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"time"
)

// Logger wraps slog.Logger with application-specific functionality.
type Logger struct {
	*slog.Logger
	level slog.Level
}

// Options configures the logger.
type Options struct {
	Level      string
	Output     io.Writer
	JSONOutput bool
	AddSource  bool
}

// DefaultOptions returns default logger options.
func DefaultOptions() Options {
	return Options{
		Level:      "info",
		Output:     os.Stderr,
		JSONOutput: false,
		AddSource:  false,
	}
}

// New creates a new logger instance.
func New(opts Options) (*Logger, error) {
	level, err := parseLevel(opts.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level %q: %w", opts.Level, err)
	}

	var handler slog.Handler
	handlerOpts := &slog.HandlerOptions{
		Level:     level,
		AddSource: opts.AddSource,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize time format
			if a.Key == slog.TimeKey {
				if t, ok := a.Value.Any().(time.Time); ok {
					a.Value = slog.StringValue(t.Format("2006-01-02T15:04:05.000Z07:00"))
				}
			}
			// Add caller info for errors
			if a.Key == slog.SourceKey {
				if src, ok := a.Value.Any().(*slog.Source); ok {
					a.Value = slog.StringValue(fmt.Sprintf("%s:%d", trimPath(src.File), src.Line))
				}
			}
			return a
		},
	}

	if opts.JSONOutput {
		handler = slog.NewJSONHandler(opts.Output, handlerOpts)
	} else {
		handler = slog.NewTextHandler(opts.Output, handlerOpts)
	}

	return &Logger{
		Logger: slog.New(handler),
		level:  level,
	}, nil
}

// WithContext returns a logger with context values.
func (l *Logger) WithContext(ctx context.Context) *Logger {
	// Extract request ID or trace ID from context if available
	attrs := []slog.Attr{}

	if reqID := ctx.Value("request_id"); reqID != nil {
		attrs = append(attrs, slog.String("request_id", fmt.Sprint(reqID)))
	}

	if traceID := ctx.Value("trace_id"); traceID != nil {
		attrs = append(attrs, slog.String("trace_id", fmt.Sprint(traceID)))
	}

	if len(attrs) == 0 {
		return l
	}

	// Convert attrs to any slice
	args := make([]any, len(attrs))
	for i, attr := range attrs {
		args[i] = attr
	}

	return &Logger{
		Logger: l.With(args...),
		level:  l.level,
	}
}

// WithError adds an error to the logger.
func (l *Logger) WithError(err error) *Logger {
	return &Logger{
		Logger: l.With(slog.String("error", err.Error())),
		level:  l.level,
	}
}

// WithField adds a field to the logger.
func (l *Logger) WithField(key string, value any) *Logger {
	return &Logger{
		Logger: l.With(slog.Any(key, value)),
		level:  l.level,
	}
}

// WithFields adds multiple fields to the logger.
func (l *Logger) WithFields(fields map[string]any) *Logger {
	attrs := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		attrs = append(attrs, k, v)
	}
	return &Logger{
		Logger: l.With(attrs...),
		level:  l.level,
	}
}

// IsDebugEnabled returns true if debug logging is enabled.
func (l *Logger) IsDebugEnabled() bool {
	return l.level <= slog.LevelDebug
}

// Fatal logs at error level and exits.
func (l *Logger) Fatal(msg string, args ...any) {
	l.Error(msg, args...)
	os.Exit(1)
}

// parseLevel converts a string level to slog.Level.
func parseLevel(level string) (slog.Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unknown level: %s", level)
	}
}

// trimPath removes the project path prefix for cleaner source locations.
func trimPath(path string) string {
	// Get the directory of the current file
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return path
	}

	// Find project root (where go.mod is)
	parts := strings.Split(file, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == "simple-mcp-runner" {
			prefix := strings.Join(parts[:i+1], "/") + "/"
			return strings.TrimPrefix(path, prefix)
		}
	}

	return path
}

// Default logger instance.
var defaultLogger *Logger

func init() {
	var err error
	defaultLogger, err = New(DefaultOptions())
	if err != nil {
		panic(fmt.Sprintf("failed to create default logger: %v", err))
	}
}

// Default returns the default logger instance.
func Default() *Logger {
	return defaultLogger
}

// SetDefault sets the default logger instance.
func SetDefault(l *Logger) {
	defaultLogger = l
}
