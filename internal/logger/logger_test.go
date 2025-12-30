package logger

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew_DefaultConfig(t *testing.T) {
	logger, err := New(nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if logger == nil {
		t.Fatal("expected non-nil logger")
	}

	if logger.config.Level != "info" {
		t.Errorf("expected default level = info, got %s", logger.config.Level)
	}

	if logger.config.Format != "console" {
		t.Errorf("expected default format = console, got %s", logger.config.Format)
	}
}

func TestNew_ConsoleFormat(t *testing.T) {
	cfg := &Config{
		Level:  "debug",
		Format: "console",
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if logger == nil {
		t.Fatal("expected non-nil logger")
	}

	// Test that we can log without errors
	logger.Debug("test debug message")
	logger.Info("test info message")
	logger.Warn("test warn message")
	logger.Error("test error message")
}

func TestNew_JSONFormat(t *testing.T) {
	cfg := &Config{
		Level:  "info",
		Format: "json",
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if logger == nil {
		t.Fatal("expected non-nil logger")
	}

	// Test that we can log without errors
	logger.Info("test info message")
	logger.Warn("test warn message")
}

func TestNew_FileOutput(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	cfg := &Config{
		Level:      "info",
		Format:     "json",
		OutputPath: logFile,
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	testMsg := "test log message"
	logger.Info(testMsg)

	// Sync to ensure message is written
	if err := logger.Sync(); err != nil {
		t.Logf("Sync() returned error (expected on stdout): %v", err)
	}

	// Read log file
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), testMsg) {
		t.Errorf("log file should contain %q, got: %s", testMsg, string(content))
	}

	// Verify JSON format
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var logEntry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
			t.Errorf("log line is not valid JSON: %v\nLine: %s", err, line)
		}
	}
}

func TestNew_InvalidLogLevel(t *testing.T) {
	cfg := &Config{
		Level:  "invalid",
		Format: "console",
	}

	_, err := New(cfg)
	if err == nil {
		t.Error("expected error for invalid log level")
	}

	if !strings.Contains(err.Error(), "invalid log level") {
		t.Errorf("expected error about invalid log level, got: %v", err)
	}
}

func TestNew_InvalidLogFile(t *testing.T) {
	cfg := &Config{
		Level:      "info",
		Format:     "console",
		OutputPath: "/nonexistent/directory/test.log",
	}

	_, err := New(cfg)
	if err == nil {
		t.Error("expected error for invalid log file path")
	}

	if !strings.Contains(err.Error(), "failed to open log file") {
		t.Errorf("expected error about log file, got: %v", err)
	}
}

func TestInit_GlobalLogger(t *testing.T) {
	// Reset global logger
	defaultLogger = nil

	cfg := &Config{
		Level:  "debug",
		Format: "console",
	}

	err := Init(cfg)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	logger := Get()
	if logger == nil {
		t.Fatal("expected non-nil global logger")
	}

	if logger.config.Level != "debug" {
		t.Errorf("expected level = debug, got %s", logger.config.Level)
	}
}

func TestGet_CreatesDefaultLogger(t *testing.T) {
	// Reset global logger
	defaultLogger = nil

	logger := Get()
	if logger == nil {
		t.Fatal("Get() should create default logger if none exists")
	}

	if logger.config.Level != "info" {
		t.Errorf("expected default level = info, got %s", logger.config.Level)
	}
}

func TestWithFields(t *testing.T) {
	logger, err := New(&Config{
		Level:  "info",
		Format: "json",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	fieldLogger := logger.WithFields("key1", "value1", "key2", 42)
	if fieldLogger == nil {
		t.Fatal("WithFields() returned nil")
	}

	// Just test that logging doesn't panic
	fieldLogger.Info("test message with fields")
}

func TestWithDocumentID(t *testing.T) {
	logger, err := New(&Config{
		Level:  "info",
		Format: "json",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	docLogger := logger.WithDocumentID("doc-123")
	if docLogger == nil {
		t.Fatal("WithDocumentID() returned nil")
	}

	// Just test that logging doesn't panic
	docLogger.Info("test message with document ID")
}

func TestWithOperation(t *testing.T) {
	logger, err := New(&Config{
		Level:  "info",
		Format: "json",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	opLogger := logger.WithOperation("sync")
	if opLogger == nil {
		t.Fatal("WithOperation() returned nil")
	}

	// Just test that logging doesn't panic
	opLogger.Info("test message with operation")
}

func TestWithError(t *testing.T) {
	logger, err := New(&Config{
		Level:  "info",
		Format: "json",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	testErr := bytes.ErrTooLarge
	errLogger := logger.WithError(testErr)
	if errLogger == nil {
		t.Fatal("WithError() returned nil")
	}

	// Just test that logging doesn't panic
	errLogger.Error("test message with error")
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name    string
		level   string
		wantErr bool
	}{
		{"debug", "debug", false},
		{"info", "info", false},
		{"warn", "warn", false},
		{"warning", "warning", false},
		{"error", "error", false},
		{"invalid", "invalid", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseLevel(tt.level)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseLevel(%q) error = %v, wantErr %v", tt.level, err, tt.wantErr)
			}
		})
	}
}

func TestPackageLevelFunctions(t *testing.T) {
	// Reset and initialize global logger
	defaultLogger = nil
	err := Init(&Config{
		Level:  "debug",
		Format: "console",
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Test that package-level functions don't panic
	Debug("debug message")
	Debugf("debug %s", "formatted")
	Info("info message")
	Infof("info %s", "formatted")
	Warn("warn message")
	Warnf("warn %s", "formatted")
	Error("error message")
	Errorf("error %s", "formatted")

	// Test package-level With* functions
	WithFields("key", "value").Info("message with fields")
	WithDocumentID("doc-123").Info("message with document ID")
	WithOperation("test").Info("message with operation")
	WithError(bytes.ErrTooLarge).Error("message with error")
}

func TestLogLevels(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name          string
		logLevel      string
		logMessage    func(*Logger)
		shouldAppear  bool
	}{
		{
			name:     "debug level logs debug",
			logLevel: "debug",
			logMessage: func(l *Logger) {
				l.Debug("debug message")
			},
			shouldAppear: true,
		},
		{
			name:     "info level skips debug",
			logLevel: "info",
			logMessage: func(l *Logger) {
				l.Debug("debug message")
			},
			shouldAppear: false,
		},
		{
			name:     "info level logs info",
			logLevel: "info",
			logMessage: func(l *Logger) {
				l.Info("info message")
			},
			shouldAppear: true,
		},
		{
			name:     "warn level skips info",
			logLevel: "warn",
			logMessage: func(l *Logger) {
				l.Info("info message")
			},
			shouldAppear: false,
		},
		{
			name:     "warn level logs warn",
			logLevel: "warn",
			logMessage: func(l *Logger) {
				l.Warn("warn message")
			},
			shouldAppear: true,
		},
		{
			name:     "error level skips warn",
			logLevel: "error",
			logMessage: func(l *Logger) {
				l.Warn("warn message")
			},
			shouldAppear: false,
		},
		{
			name:     "error level logs error",
			logLevel: "error",
			logMessage: func(l *Logger) {
				l.Error("error message")
			},
			shouldAppear: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create unique log file for each test
			testLogFile := filepath.Join(tmpDir, tt.name+".log")

			cfg := &Config{
				Level:      tt.logLevel,
				Format:     "json",
				OutputPath: testLogFile,
			}

			logger, err := New(cfg)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			tt.logMessage(logger)
			logger.Sync()

			content, err := os.ReadFile(testLogFile)
			if err != nil {
				t.Fatalf("failed to read log file: %v", err)
			}

			hasContent := len(strings.TrimSpace(string(content))) > 0
			if hasContent != tt.shouldAppear {
				t.Errorf("log message appearance = %v, want %v\nContent: %s",
					hasContent, tt.shouldAppear, string(content))
			}
		})
	}
}
