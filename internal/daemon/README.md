# Daemon Mode

Package daemon provides long-running periodic synchronization for reMarkable documents.

## Overview

The daemon mode enables hands-free continuous synchronization without manual intervention. It runs in the background, periodically triggering sync operations at configured intervals.

## Features

✅ **Periodic Synchronization**
- Ticker-based scheduling with configurable interval
- Runs initial sync immediately on startup
- Default interval: 5 minutes (configurable)

✅ **Graceful Shutdown**
- Handles SIGTERM and SIGINT signals
- Allows in-progress sync to complete before shutdown
- Clean resource cleanup (PID file, health check server)

✅ **Error Recovery**
- Continues running even if individual syncs fail
- Logs sync failures with details
- Automatic retry on next interval

✅ **Health Check Endpoint** (Optional)
- HTTP endpoint for monitoring daemon health
- `/health` - Returns 200 OK if daemon is running
- `/ready` - Returns 200 OK if daemon is ready
- Useful for container orchestration (Kubernetes, Docker Swarm)

✅ **PID File Management** (Optional)
- Writes process ID to configured file
- Useful for init scripts and process managers
- Automatic cleanup on shutdown

✅ **Lifecycle Logging**
- Structured logging of daemon events
- Startup, sync trigger, completion, shutdown
- Sync statistics and failure details

## Architecture

```
┌────────────────────────────────────────────────────────┐
│                    Daemon Process                       │
│                                                         │
│  ┌──────────────┐         ┌──────────────┐            │
│  │    Ticker    │────────▶│  Orchestrator│            │
│  │  (Interval)  │         │    (Sync)    │            │
│  └──────────────┘         └──────────────┘            │
│                                                         │
│  ┌──────────────┐         ┌──────────────┐            │
│  │   Signal     │         │    Health    │            │
│  │   Handler    │         │    Check     │            │
│  │(SIGTERM/INT) │         │    HTTP      │            │
│  └──────────────┘         └──────────────┘            │
│                                                         │
│  ┌──────────────┐                                      │
│  │   PID File   │                                      │
│  │  Management  │                                      │
│  └──────────────┘                                      │
└────────────────────────────────────────────────────────┘
```

## Usage

### Basic Usage

```go
package main

import (
	"context"
	"time"

	"github.com/platinummonkey/legible/internal/daemon"
	"github.com/platinummonkey/legible/internal/sync"
)

func main() {
	// Create sync orchestrator
	orchestrator, err := sync.New(&sync.Config{
		// ... sync configuration
	})
	if err != nil {
		panic(err)
	}

	// Create daemon
	d, err := daemon.New(&daemon.Config{
		Orchestrator: orchestrator,
		SyncInterval: 10 * time.Minute,
	})
	if err != nil {
		panic(err)
	}

	// Run daemon (blocks until shutdown signal)
	ctx := context.Background()
	if err := d.Run(ctx); err != nil {
		panic(err)
	}
}
```

### With Health Check

```go
d, err := daemon.New(&daemon.Config{
	Orchestrator:    orchestrator,
	SyncInterval:    5 * time.Minute,
	HealthCheckAddr: ":8080",  // Enable health check on port 8080
})
if err != nil {
	panic(err)
}

// Run daemon
if err := d.Run(context.Background()); err != nil {
	panic(err)
}

// Health check available at:
// http://localhost:8080/health
// http://localhost:8080/ready
```

### With PID File

```go
d, err := daemon.New(&daemon.Config{
	Orchestrator: orchestrator,
	SyncInterval: 5 * time.Minute,
	PIDFile:      "/var/run/legible.pid",
})
if err != nil {
	panic(err)
}

if err := d.Run(context.Background()); err != nil {
	panic(err)
}
```

### Complete Example

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/platinummonkey/legible/internal/config"
	"github.com/platinummonkey/legible/internal/converter"
	"github.com/platinummonkey/legible/internal/daemon"
	"github.com/platinummonkey/legible/internal/logger"
	"github.com/platinummonkey/legible/internal/pdfenhancer"
	"github.com/platinummonkey/legible/internal/rmclient"
	"github.com/platinummonkey/legible/internal/state"
	"github.com/platinummonkey/legible/internal/sync"
)

func main() {
	// Initialize logger
	log, err := logger.New(&logger.Config{
		Level:  "info",
		Format: "json",
	})
	if err != nil {
		panic(err)
	}

	// Load configuration
	cfg := &config.Config{
		OutputDir:  "/home/user/Legible",
		Labels:     []string{"work"},
		OCREnabled: false,
	}

	// Create components
	rmClient := rmclient.New(&rmclient.Config{Logger: log})
	stateStore := state.New(&state.Config{Logger: log})
	converter := converter.New(&converter.Config{Logger: log})
	pdfEnhancer := pdfenhancer.New(&pdfenhancer.Config{Logger: log})

	// Create orchestrator
	orch, err := sync.New(&sync.Config{
		Config:      cfg,
		Logger:      log,
		RMClient:    rmClient,
		StateStore:  stateStore,
		Converter:   converter,
		PDFEnhancer: pdfEnhancer,
	})
	if err != nil {
		panic(err)
	}

	// Create daemon
	d, err := daemon.New(&daemon.Config{
		Orchestrator:    orch,
		Logger:          log,
		SyncInterval:    10 * time.Minute,
		HealthCheckAddr: ":8080",
		PIDFile:         "/var/run/legible.pid",
	})
	if err != nil {
		panic(err)
	}

	// Run daemon
	log.Info("Starting legible daemon")
	if err := d.Run(context.Background()); err != nil && err != context.Canceled {
		log.WithFields("error", err).Fatal("Daemon error")
	}

	log.Info("Daemon shutdown complete")
}
```

## Lifecycle

### Startup

1. Write PID file (if configured)
2. Start health check HTTP server (if configured)
3. Setup signal handling for SIGTERM and SIGINT
4. Create ticker with configured interval
5. Run initial sync immediately
6. Enter main event loop

### Main Loop

The daemon responds to three events:

1. **Context Cancellation** - Shutdown on parent context cancellation
2. **Signal Reception** - Graceful shutdown on SIGTERM/SIGINT
3. **Ticker Tick** - Trigger periodic sync

### Sync Execution

For each sync:

1. Log sync start
2. Create timeout context (30 minutes default)
3. Call orchestrator.Sync()
4. Log results and duration
5. Log failures if any occurred
6. Continue to next interval even if sync failed

### Shutdown

1. Log shutdown signal received
2. Stop ticker
3. Cancel context
4. Shutdown health check server (5 second timeout)
5. Remove PID file
6. Return from Run()

## Signal Handling

The daemon handles these OS signals:

### SIGTERM (Termination Signal)
- Sent by `systemctl stop`, `kill` (default)
- Triggers graceful shutdown
- Allows current sync to complete

### SIGINT (Interrupt Signal)
- Sent by Ctrl+C in terminal
- Triggers graceful shutdown
- Allows current sync to complete

**Example:**
```bash
# Send SIGTERM to daemon
kill $(cat /var/run/legible.pid)

# Send SIGINT from terminal
# Press Ctrl+C
```

## Health Check Endpoints

When health check is enabled, these HTTP endpoints are available:

### GET /health

Returns daemon health status.

**Response:**
```
200 OK
OK
```

**Use Cases:**
- Monitoring systems (Prometheus, Datadog)
- Load balancers
- Service discovery

### GET /ready

Returns daemon readiness status.

**Response:**
```
200 OK
OK
```

**Use Cases:**
- Kubernetes readiness probes
- Container orchestration
- Rolling deployments

**Example:**
```bash
# Check health
curl http://localhost:8080/health

# Check readiness
curl http://localhost:8080/ready
```

## PID File

When PID file is configured, the daemon writes its process ID to the specified file.

**Format:**
```
12345
```

**Use Cases:**
- Init scripts (SysV, systemd)
- Process managers
- Monitoring scripts
- Graceful shutdown scripts

**Example:**
```bash
# Read PID
PID=$(cat /var/run/legible.pid)

# Send signal
kill -TERM $PID

# Check if running
if kill -0 $PID 2>/dev/null; then
    echo "Daemon is running"
else
    echo "Daemon is not running"
fi
```

## Logging

The daemon logs all lifecycle events at appropriate levels:

### INFO Level
```
INFO: Starting daemon interval=5m0s
INFO: Wrote PID file pid=12345 file=/var/run/legible.pid
INFO: Starting health check server addr=:8080
INFO: Running initial sync
INFO: Starting sync
INFO: Sync completed total=10 processed=3 successful=3 failed=0 duration=1m30s
INFO: Sync interval elapsed, triggering sync
INFO: Received shutdown signal signal=terminated
INFO: Stopping health check server
INFO: Health check server stopped
INFO: Removed PID file file=/var/run/legible.pid
```

### ERROR Level
```
ERROR: Sync failed error="API connection timeout" duration=5m0s
ERROR: Health check server failed error="bind: address already in use"
```

### WARN Level
```
WARN: Sync completed with failures count=2
WARN: Document sync failed document_id=abc-123 title="Meeting Notes" error="download failed"
WARN: Failed to remove PID file file=/var/run/legible.pid error="permission denied"
```

## Error Handling

The daemon implements robust error recovery:

### Sync Failures
- Individual sync failures don't stop the daemon
- Error is logged with details
- Next sync attempt happens at next interval
- No exponential backoff (fixed interval)

### Component Failures
- PID file write failure returns error, prevents daemon start
- Health check server failure returns error, prevents daemon start
- Orchestrator nil check prevents daemon creation

### Graceful Degradation
- PID file removal failure only logs warning
- Health check server shutdown failure only logs warning
- Both allow daemon to complete shutdown

## Testing

The package includes tests for:

- Daemon initialization
- Configuration validation (nil config, nil orchestrator)
- Interval defaults and custom values
- PID file write and remove
- Health check server start/stop
- Individual helper methods

**Integration tests** for full daemon operation require:
- Interface-based orchestrator for mocking
- Time-based testing infrastructure
- Signal handling test utilities

Run tests:
```bash
go test ./internal/daemon
go test -v ./internal/daemon
```

## Deployment

### Systemd Service

```ini
[Unit]
Description=Legible Sync Daemon
After=network.target

[Service]
Type=simple
User=legible
Group=legible
ExecStart=/usr/local/bin/legible daemon
Restart=on-failure
RestartSec=10s
PIDFile=/var/run/legible.pid

[Install]
WantedBy=multi-user.target
```

### Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o legible ./cmd/legible

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/legible /usr/local/bin/
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s \
  CMD wget --quiet --tries=1 --spider http://localhost:8080/health || exit 1
CMD ["legible", "daemon"]
```

### Docker Compose

```yaml
version: '3.8'

services:
  legible:
    build: .
    container_name: legible
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ./config:/etc/legible
      - ./output:/data/output
      - ./state:/data/state
    environment:
      - SYNC_INTERVAL=10m
      - OUTPUT_DIR=/data/output
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 3s
      retries: 3
```

## Future Enhancements

1. **Dynamic Interval Adjustment**
   - Increase interval on repeated failures
   - Decrease interval on high change frequency
   - Configurable backoff strategy

2. **Sync Scheduling**
   - Cron-like scheduling support
   - "Quiet hours" configuration
   - Time-based sync windows

3. **Metrics and Monitoring**
   - Prometheus metrics endpoint
   - Sync duration histograms
   - Success/failure rates
   - Document counts

4. **Advanced Health Checks**
   - Deep health checks (API connectivity, disk space)
   - Degraded state reporting
   - Component-level health status

5. **Hot Configuration Reload**
   - SIGHUP handler for config reload
   - Update interval without restart
   - Update filters without restart

6. **Multiple Sync Profiles**
   - Run multiple sync configurations
   - Different intervals per profile
   - Profile-specific filters

## Dependencies

- **sync**: Orchestrator for sync workflow
- **logger**: Structured logging
- **context**: Cancellation and timeouts
- **signal**: OS signal handling
- **http**: Health check HTTP server

## License

Part of legible project. See project LICENSE for details.
