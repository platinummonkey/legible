// Package logger provides structured logging functionality using zap.
package logger

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps zap.SugaredLogger to provide structured logging throughout the application
type Logger struct {
	*zap.SugaredLogger
	config *Config
}

// Config holds logger configuration options
type Config struct {
	// Level is the minimum log level to output (debug, info, warn, error)
	Level string

	// Format determines output format: "console" (human-readable) or "json" (machine-parseable)
	Format string

	// OutputPath is the file path for log output (empty = stdout only)
	OutputPath string

	// EnableCaller adds caller information to log entries
	EnableCaller bool

	// EnableStacktrace adds stack traces to error-level logs
	EnableStacktrace bool
}

var (
	// defaultLogger is the global logger instance
	defaultLogger *Logger
)

// New creates a new logger instance with the provided configuration
func New(cfg *Config) (*Logger, error) {
	if cfg == nil {
		cfg = &Config{
			Level:            "info",
			Format:           "console",
			EnableCaller:     false,
			EnableStacktrace: true,
		}
	}

	// Parse log level
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	// Configure encoder
	var encoderConfig zapcore.EncoderConfig
	if cfg.Format == "json" {
		encoderConfig = zap.NewProductionEncoderConfig()
	} else {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	// Create encoder
	var encoder zapcore.Encoder
	if cfg.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// Configure output
	var writeSyncs []zapcore.WriteSyncer
	writeSyncs = append(writeSyncs, zapcore.AddSync(os.Stdout))

	if cfg.OutputPath != "" {
		file, err := os.OpenFile(cfg.OutputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file %s: %w", cfg.OutputPath, err)
		}
		writeSyncs = append(writeSyncs, zapcore.AddSync(file))
	}

	// Combine outputs
	writer := zapcore.NewMultiWriteSyncer(writeSyncs...)

	// Create core
	core := zapcore.NewCore(encoder, writer, level)

	// Build logger options
	opts := []zap.Option{}
	if cfg.EnableCaller {
		opts = append(opts, zap.AddCaller())
	}
	if cfg.EnableStacktrace {
		opts = append(opts, zap.AddStacktrace(zapcore.ErrorLevel))
	}

	// Create zap logger
	zapLogger := zap.New(core, opts...)

	return &Logger{
		SugaredLogger: zapLogger.Sugar(),
		config:        cfg,
	}, nil
}

// Init initializes the global logger instance
func Init(cfg *Config) error {
	logger, err := New(cfg)
	if err != nil {
		return err
	}
	defaultLogger = logger
	return nil
}

// Get returns the global logger instance
func Get() *Logger {
	if defaultLogger == nil {
		// Create a default logger if none exists
		logger, _ := New(nil)
		defaultLogger = logger
	}
	return defaultLogger
}

// WithFields returns a logger with the specified fields attached for structured logging
func (l *Logger) WithFields(fields ...interface{}) *Logger {
	return &Logger{
		SugaredLogger: l.With(fields...),
		config:        l.config,
	}
}

// WithDocumentID returns a logger with document_id field attached
func (l *Logger) WithDocumentID(docID string) *Logger {
	return l.WithFields("document_id", docID)
}

// WithOperation returns a logger with operation field attached
func (l *Logger) WithOperation(operation string) *Logger {
	return l.WithFields("operation", operation)
}

// WithError returns a logger with error field attached
func (l *Logger) WithError(err error) *Logger {
	return l.WithFields("error", err)
}

// parseLevel converts a string log level to zapcore.Level
func parseLevel(level string) (zapcore.Level, error) {
	switch level {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info":
		return zapcore.InfoLevel, nil
	case "warn", "warning":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	default:
		return zapcore.InfoLevel, fmt.Errorf("invalid log level %q", level)
	}
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	return l.SugaredLogger.Sync()
}

// Package-level convenience functions that use the global logger

// Debug logs a debug message
func Debug(args ...interface{}) {
	Get().Debug(args...)
}

// Debugf logs a formatted debug message
func Debugf(template string, args ...interface{}) {
	Get().Debugf(template, args...)
}

// Info logs an info message
func Info(args ...interface{}) {
	Get().Info(args...)
}

// Infof logs a formatted info message
func Infof(template string, args ...interface{}) {
	Get().Infof(template, args...)
}

// Warn logs a warning message
func Warn(args ...interface{}) {
	Get().Warn(args...)
}

// Warnf logs a formatted warning message
func Warnf(template string, args ...interface{}) {
	Get().Warnf(template, args...)
}

// Error logs an error message
func Error(args ...interface{}) {
	Get().Error(args...)
}

// Errorf logs a formatted error message
func Errorf(template string, args ...interface{}) {
	Get().Errorf(template, args...)
}

// Fatal logs a fatal message and exits
func Fatal(args ...interface{}) {
	Get().Fatal(args...)
}

// Fatalf logs a formatted fatal message and exits
func Fatalf(template string, args ...interface{}) {
	Get().Fatalf(template, args...)
}

// WithFields returns a logger with the specified fields attached
func WithFields(fields ...interface{}) *Logger {
	return Get().WithFields(fields...)
}

// WithDocumentID returns a logger with document_id field attached
func WithDocumentID(docID string) *Logger {
	return Get().WithDocumentID(docID)
}

// WithOperation returns a logger with operation field attached
func WithOperation(operation string) *Logger {
	return Get().WithOperation(operation)
}

// WithError returns a logger with error field attached
func WithError(err error) *Logger {
	return Get().WithError(err)
}

// Sync flushes any buffered log entries
func Sync() error {
	return Get().Sync()
}
