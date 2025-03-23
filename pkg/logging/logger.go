// Package logging provides structured logging capabilities for the application.
package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"
)

// LogLevel represents the logging level.
type LogLevel string

const (
	// DebugLevel logs detailed debugging information.
	DebugLevel LogLevel = "debug"
	// InfoLevel logs informational messages.
	InfoLevel LogLevel = "info"
	// WarnLevel logs warning messages.
	WarnLevel LogLevel = "warn"
	// ErrorLevel logs error messages.
	ErrorLevel LogLevel = "error"
)

// Logger is a wrapper around slog.Logger that provides structured logging.
type Logger struct {
	*slog.Logger
}

// Config holds the configuration for the logger.
type Config struct {
	// Level is the minimum log level to output.
	Level LogLevel
	// Output is where the logs will be written to.
	Output io.Writer
	// ServiceName is the name of the service that is logging.
	ServiceName string
	// Environment is the environment the service is running in (e.g., "production", "development").
	Environment string
}

// DefaultConfig returns a default logger configuration.
func DefaultConfig() Config {
	return Config{
		Level:       InfoLevel,
		Output:      os.Stdout,
		ServiceName: "stathera",
		Environment: "development",
	}
}

// New creates a new structured logger with the given configuration.
func New(cfg Config) *Logger {
	var level slog.Level
	switch cfg.Level {
	case DebugLevel:
		level = slog.LevelDebug
	case InfoLevel:
		level = slog.LevelInfo
	case WarnLevel:
		level = slog.LevelWarn
	case ErrorLevel:
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Create a JSON handler with the configured level
	handler := slog.NewJSONHandler(cfg.Output, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize timestamp format
			if a.Key == slog.TimeKey {
				if t, ok := a.Value.Any().(time.Time); ok {
					return slog.String(slog.TimeKey, t.Format(time.RFC3339))
				}
			}
			return a
		},
	})

	// Create a logger with the handler and add default attributes
	logger := slog.New(handler).With(
		slog.String("service", cfg.ServiceName),
		slog.String("environment", cfg.Environment),
	)

	return &Logger{Logger: logger}
}

// WithContext returns a new Logger with context values added to the logger.
func (l *Logger) WithContext(ctx context.Context) *Logger {
	// Extract values from context and add them to the logger
	// This is a placeholder - in a real implementation, you would extract
	// values like request ID, user ID, etc. from the context
	return l
}

// WithField adds a field to the logger.
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{Logger: l.With(slog.Any(key, value))}
}

// WithFields adds multiple fields to the logger.
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	logger := l.Logger
	for k, v := range fields {
		logger = logger.With(slog.Any(k, v))
	}
	return &Logger{Logger: logger}
}

// WithError adds an error to the logger.
func (l *Logger) WithError(err error) *Logger {
	if err == nil {
		return l
	}
	return &Logger{Logger: l.With(slog.String("error", err.Error()))}
}

// Debug logs a debug message.
func (l *Logger) Debug(msg string, args ...interface{}) {
	l.Logger.Debug(msg, toSlogArgs(args)...)
}

// Info logs an info message.
func (l *Logger) Info(msg string, args ...interface{}) {
	l.Logger.Info(msg, toSlogArgs(args)...)
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string, args ...interface{}) {
	l.Logger.Warn(msg, toSlogArgs(args)...)
}

// Error logs an error message.
func (l *Logger) Error(msg string, args ...interface{}) {
	l.Logger.Error(msg, toSlogArgs(args)...)
}

// toSlogArgs converts a slice of interface{} to a slice of slog.Attr.
// It expects the args to be in key-value pairs.
func toSlogArgs(args []interface{}) []any {
	if len(args) == 0 {
		return nil
	}

	// If odd number of args, add an empty string to make it even
	if len(args)%2 != 0 {
		args = append(args, "")
	}

	slogArgs := make([]any, 0, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok {
			key = "unknown"
		}
		slogArgs = append(slogArgs, slog.Any(key, args[i+1]))
	}

	return slogArgs
}
