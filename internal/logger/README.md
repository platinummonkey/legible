# Logger Package

Structured logging infrastructure for legible using uber-go/zap.

## Features

- **Multiple Log Levels**: debug, info, warn, error
- **Flexible Output**: Console (human-readable) or JSON (machine-parseable)
- **File Output**: Optional file logging for daemon mode
- **Structured Logging**: Context-aware fields (document_id, operation, duration, error)
- **High Performance**: Built on zap for zero-allocation logging in hot paths
- **Global Logger**: Package-level convenience functions

## Basic Usage

### Initialize Logger

```go
import "github.com/platinummonkey/legible/internal/logger"

// Initialize with config
cfg := &logger.Config{
    Level:  "info",
    Format: "console",
}
if err := logger.Init(cfg); err != nil {
    log.Fatal(err)
}
defer logger.Sync()
```

### Simple Logging

```go
import "github.com/platinummonkey/legible/internal/logger"

// Package-level functions use the global logger
logger.Info("Starting sync operation")
logger.Infof("Processing %d documents", count)
logger.Warn("No documents found")
logger.Error("Failed to connect")
```

### Structured Logging

```go
// Add contextual fields
logger.WithFields(
    "document_id", "abc-123",
    "operation", "sync",
    "duration_ms", 250,
).Info("Document synced successfully")

// Convenience methods
logger.WithDocumentID("abc-123").Info("Processing document")
logger.WithOperation("ocr").Info("Starting OCR")
logger.WithError(err).Error("Operation failed")
```

### Instance Logger

```go
// Create a custom logger instance
cfg := &logger.Config{
    Level:      "debug",
    Format:     "json",
    OutputPath: "/var/log/legible.log",
}
log, err := logger.New(cfg)
if err != nil {
    return err
}
defer log.Sync()

log.Debug("Debug information")
log.Info("Information message")
```

## Configuration

```go
type Config struct {
    // Level is the minimum log level (debug, info, warn, error)
    Level string

    // Format is "console" (human-readable) or "json" (machine-parseable)
    Format string

    // OutputPath is the file path for log output (empty = stdout only)
    OutputPath string

    // EnableCaller adds caller information to log entries
    EnableCaller bool

    // EnableStacktrace adds stack traces to error-level logs
    EnableStacktrace bool
}
```

## Examples

### Development Mode (Console)

```go
cfg := &logger.Config{
    Level:  "debug",
    Format: "console",
}
logger.Init(cfg)

logger.Debug("Detailed debug information")
// Output: 2025-12-30T09:59:17.940-0600  DEBUG  Detailed debug information
```

### Production Mode (JSON)

```go
cfg := &logger.Config{
    Level:  "info",
    Format: "json",
    OutputPath: "/var/log/app.log",
}
logger.Init(cfg)

logger.WithFields("user_id", 123).Info("User logged in")
// Output: {"level":"info","ts":1767110357.929,"msg":"User logged in","user_id":123}
```

### Context-Aware Logging

```go
// Create a logger with document context
docLogger := logger.WithDocumentID("doc-abc-123")

// All subsequent logs include the document_id
docLogger.Info("Starting download")
docLogger.WithOperation("convert").Info("Converting to PDF")
docLogger.WithFields("size_bytes", 1024).Info("Download complete")

// Output (JSON format):
// {"level":"info","ts":...,"msg":"Starting download","document_id":"doc-abc-123"}
// {"level":"info","ts":...,"msg":"Converting to PDF","document_id":"doc-abc-123","operation":"convert"}
// {"level":"info","ts":...,"msg":"Download complete","document_id":"doc-abc-123","size_bytes":1024}
```

### Error Logging

```go
if err := processDocument(doc); err != nil {
    logger.WithError(err).
        WithDocumentID(doc.ID).
        Error("Failed to process document")
}

// Output (JSON):
// {"level":"error","ts":...,"msg":"Failed to process document","document_id":"doc-123","error":"connection timeout"}
```

## Integration with Config Package

```go
import (
    "github.com/platinummonkey/legible/internal/config"
    "github.com/platinummonkey/legible/internal/logger"
)

// Load application config
cfg, err := config.Load("")
if err != nil {
    log.Fatal(err)
}

// Initialize logger from config
logCfg := &logger.Config{
    Level:  cfg.LogLevel,
    Format: "console",
}
if cfg.DaemonMode {
    logCfg.Format = "json"
    logCfg.OutputPath = "/var/log/legible.log"
}

if err := logger.Init(logCfg); err != nil {
    log.Fatal(err)
}
defer logger.Sync()
```

## Testing

Run tests:
```bash
make test
```

The logger package has 92.4% test coverage with comprehensive tests for:
- Log level filtering
- Console and JSON output formats
- File output
- Structured fields
- Global and instance loggers
- Error handling
