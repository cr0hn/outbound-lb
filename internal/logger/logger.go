// Package logger provides structured logging using log/slog.
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
)

// LevelTrace is more verbose than debug, for detailed tracing.
const LevelTrace = slog.Level(-8)

var (
	defaultLogger *slog.Logger
	levelVar      = new(slog.LevelVar)
	currentFormat string
	output        io.Writer
	mu            sync.RWMutex
)

// Init initializes the global logger with the specified level and format.
func Init(level, format string) {
	mu.Lock()
	defer mu.Unlock()

	if defaultLogger == nil {
		output = os.Stdout
	}
	currentFormat = format
	levelVar.Set(parseLevel(level))
	defaultLogger = newLogger(format, output)
}

// parseLevel converts a string level to slog.Level.
func parseLevel(level string) slog.Level {
	switch level {
	case "trace":
		return LevelTrace
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// newLogger creates a new logger with the current levelVar.
func newLogger(format string, w io.Writer) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: levelVar,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				level := a.Value.Any().(slog.Level)
				if level == LevelTrace {
					a.Value = slog.StringValue("TRACE")
				}
			}
			return a
		},
	}

	var handler slog.Handler
	if format == "json" {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}

	return slog.New(handler)
}

// New creates a new logger with the specified configuration.
func New(level, format string, w io.Writer) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "trace":
		lvl = LevelTrace
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: lvl,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				level := a.Value.Any().(slog.Level)
				if level == LevelTrace {
					a.Value = slog.StringValue("TRACE")
				}
			}
			return a
		},
	}

	var handler slog.Handler
	if format == "json" {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}

	return slog.New(handler)
}

// Reconfigure changes the log level and/or format at runtime.
func Reconfigure(level, format string) {
	mu.Lock()
	defer mu.Unlock()

	levelVar.Set(parseLevel(level))

	// Recreate handler if format changed
	if format != currentFormat {
		currentFormat = format
		defaultLogger = newLogger(format, output)
	}

	Info("logger_reconfigured", "level", level, "format", format)
}

// Default returns the default logger, initializing it if necessary.
func Default() *slog.Logger {
	mu.RLock()
	logger := defaultLogger
	mu.RUnlock()

	if logger == nil {
		Init("info", "json")
		mu.RLock()
		logger = defaultLogger
		mu.RUnlock()
	}
	return logger
}

// Trace logs at trace level (more verbose than debug).
func Trace(msg string, args ...any) {
	Default().Log(context.Background(), LevelTrace, msg, args...)
}

// Debug logs at debug level.
func Debug(msg string, args ...any) {
	Default().Debug(msg, args...)
}

// Info logs at info level.
func Info(msg string, args ...any) {
	Default().Info(msg, args...)
}

// Warn logs at warn level.
func Warn(msg string, args ...any) {
	Default().Warn(msg, args...)
}

// Error logs at error level.
func Error(msg string, args ...any) {
	Default().Error(msg, args...)
}

// TraceContext logs at trace level with context.
func TraceContext(ctx context.Context, msg string, args ...any) {
	Default().Log(ctx, LevelTrace, msg, args...)
}

// DebugContext logs at debug level with context.
func DebugContext(ctx context.Context, msg string, args ...any) {
	Default().DebugContext(ctx, msg, args...)
}

// InfoContext logs at info level with context.
func InfoContext(ctx context.Context, msg string, args ...any) {
	Default().InfoContext(ctx, msg, args...)
}

// WarnContext logs at warn level with context.
func WarnContext(ctx context.Context, msg string, args ...any) {
	Default().WarnContext(ctx, msg, args...)
}

// ErrorContext logs at error level with context.
func ErrorContext(ctx context.Context, msg string, args ...any) {
	Default().ErrorContext(ctx, msg, args...)
}

// With returns a new logger with the given attributes.
func With(args ...any) *slog.Logger {
	return Default().With(args...)
}

// WithGroup returns a new logger with the given group name.
func WithGroup(name string) *slog.Logger {
	return Default().WithGroup(name)
}

// LogRequest logs a proxy request with standard fields.
func LogRequest(method, host, sourceIP, outboundIP string, status int, duration int64, bytesIn, bytesOut int64) {
	Default().Info("request",
		"method", method,
		"host", host,
		"source_ip", sourceIP,
		"outbound_ip", outboundIP,
		"status", status,
		"duration_ms", duration,
		"bytes_in", bytesIn,
		"bytes_out", bytesOut,
	)
}

// LogBalancerSelection logs IP selection by the balancer.
func LogBalancerSelection(host, selectedIP string, candidateCount int) {
	Default().Debug("balancer_selection",
		"host", host,
		"selected_ip", selectedIP,
		"candidates", candidateCount,
	)
}

// LogConnectionLimit logs when a connection limit is reached.
func LogConnectionLimit(limitType, ip string, current, max int) {
	Default().Warn("connection_limit_reached",
		"limit_type", limitType,
		"ip", ip,
		"current", current,
		"max", max,
	)
}

// LogError logs an error with context.
func LogError(operation string, err error, args ...any) {
	allArgs := append([]any{"operation", operation, "error", err.Error()}, args...)
	Default().Error("error", allArgs...)
}
