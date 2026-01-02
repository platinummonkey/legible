# legible CLI

Command-line interface for synchronizing and enhancing reMarkable documents.

## Installation

```bash
go install github.com/platinummonkey/legible/cmd/legible@latest
```

Or build from source:

```bash
git clone https://github.com/platinummonkey/legible.git
cd legible
go build -o legible ./cmd/legible
```

## Quick Start

```bash
# 1. Authenticate with reMarkable API (one-time)
legible auth

# 2. Sync documents
legible sync --output ~/Documents/ReMarkable

# 3. Or run in daemon mode for continuous sync
legible daemon --interval 10m
```

## Commands

### `auth` - Authenticate with reMarkable API

Authenticate with the reMarkable cloud API using a one-time code.

**Usage:**
```bash
legible auth
```

**What it does:**
1. Opens your browser to get a one-time code from reMarkable
2. You enter the code in the terminal
3. Credentials are saved for future use

**Example:**
```bash
$ legible auth
INFO: Starting reMarkable authentication
INFO: Opening browser for authentication...
INFO: Please follow the instructions in your browser
Enter one-time code: abc123def
INFO: Authentication successful!
INFO: Credentials saved for future use
```

### `sync` - One-time synchronization

Perform a one-time synchronization of documents from reMarkable cloud.

**Usage:**
```bash
legible sync [flags]
```

**Flags:**
```
--output string       Output directory for PDFs (default: ~/Legible)
--labels strings      Filter documents by labels (comma-separated)
--no-ocr              Disable OCR processing
--force               Force re-sync all documents (ignore state)
--log-level string    Log level: debug, info, warn, error (default: info)
--config string       Config file (default: ~/.legible.yaml)
```

**Examples:**
```bash
# Sync all documents to default directory
legible sync

# Sync only documents with "work" label
legible sync --labels work

# Sync multiple labels
legible sync --labels work,personal

# Sync to specific directory without OCR
legible sync --output ~/Documents/ReMarkable --no-ocr

# Force re-sync everything
legible sync --force

# Debug logging
legible sync --log-level debug
```

**Output:**
```
Using config file: /home/user/.legible.yaml
INFO: Starting sync output_dir=/home/user/ReMarkable
INFO: Listing documents from reMarkable API
INFO: Retrieved documents from API count=25
INFO: Filtered documents by labels original=25 filtered=15
INFO: Identified documents to sync count=5
INFO: Processing document document=1 total=5 id=abc-123 title="Meeting Notes"
...
INFO: Sync workflow completed total=15 processed=5 successful=5 failed=0 duration=1m30s

=== Sync Complete ===
Total documents: 15
Processed: 5
Successful: 5
Failed: 0
Duration: 1m30s
```

### `daemon` - Run in daemon mode

Run legible as a long-running daemon process with periodic sync.

**Usage:**
```bash
legible daemon [flags]
```

**Flags:**
```
--interval duration      Sync interval (default: 5m)
--health-addr string     Health check HTTP address (e.g., :8080)
--pid-file string        PID file path
--output string          Output directory for PDFs
--labels strings         Filter documents by labels
--no-ocr                 Disable OCR processing
--log-level string       Log level (default: info)
--config string          Config file
```

**Examples:**
```bash
# Run daemon with default 5 minute interval
legible daemon

# Run with custom interval
legible daemon --interval 10m

# Run with health check endpoint on port 8080
legible daemon --health-addr :8080

# Run with PID file
legible daemon --pid-file /var/run/legible.pid

# Full example with all options
legible daemon \
  --interval 10m \
  --health-addr :8080 \
  --pid-file /var/run/legible.pid \
  --output ~/Documents/ReMarkable \
  --labels work,personal \
  --log-level info
```

**Output (JSON logging):**
```json
{"level":"info","time":"2025-12-30T10:00:00Z","message":"Starting daemon","interval":"5m0s"}
{"level":"info","time":"2025-12-30T10:00:00Z","message":"Wrote PID file","pid":12345,"file":"/var/run/legible.pid"}
{"level":"info","time":"2025-12-30T10:00:00Z","message":"Starting health check server","addr":":8080"}
{"level":"info","time":"2025-12-30T10:00:00Z","message":"Running initial sync"}
{"level":"info","time":"2025-12-30T10:00:00Z","message":"Starting sync"}
{"level":"info","time":"2025-12-30T10:01:30Z","message":"Sync completed","total":10,"processed":3,"successful":3,"failed":0,"duration":"1m30s"}
```

**Shutdown:**
```bash
# Send SIGTERM to gracefully shutdown
kill $(cat /var/run/legible.pid)

# Or press Ctrl+C in terminal
```

**Health Check Endpoints:**

When `--health-addr` is specified, these HTTP endpoints are available:

- `GET /health` - Returns 200 OK if daemon is running
- `GET /ready` - Returns 200 OK if daemon is ready

```bash
# Check health
curl http://localhost:8080/health
# OK

# Check readiness
curl http://localhost:8080/ready
# OK
```

### `version` - Display version information

Display version, build date, and Git commit information.

**Usage:**
```bash
legible version
```

**Output:**
```
legible version 0.1.0
  Git commit: a1b2c3d
  Built: 2025-12-30T10:00:00Z
  Go version: go1.21.5
  OS/Arch: linux/amd64
```

## Global Flags

These flags can be used with any command:

```
--config string       Config file (default: ~/.legible.yaml)
--output string       Output directory for PDFs
--labels strings      Filter documents by labels (comma-separated)
--log-level string    Log level: debug, info, warn, error (default: info)
--no-ocr              Disable OCR processing
```

## Configuration File

legible can be configured using a YAML file. By default, it looks for `~/.legible.yaml`.

**Example configuration:**
```yaml
# Output directory for synced PDFs
output_dir: ~/Documents/ReMarkable

# Filter documents by these labels (empty = sync all)
labels:
  - work
  - personal

# Enable OCR processing
ocr_enabled: true

# OCR languages (ISO 639-2 codes)
ocr_languages:
  - eng

# Log level: debug, info, warn, error
log_level: info

# Daemon settings
daemon:
  interval: 10m
  health_addr: :8080
  pid_file: /var/run/legible.pid
```

**Specify custom config file:**
```bash
legible sync --config /etc/legible.yaml
```

## Environment Variables

Configuration can also be set via environment variables with the `RMSYNC_` prefix:

```bash
export RMSYNC_OUTPUT_DIR=~/Documents/ReMarkable
export RMSYNC_LABELS=work,personal
export RMSYNC_LOG_LEVEL=debug
export RMSYNC_OCR_ENABLED=true

legible sync
```

## Systemd Service

Run as a systemd service for automatic startup:

**`/etc/systemd/system/legible.service`:**
```ini
[Unit]
Description=reMarkable Sync Daemon
After=network.target

[Service]
Type=simple
User=remarkable
Group=remarkable
ExecStart=/usr/local/bin/legible daemon
Restart=on-failure
RestartSec=10s
Environment="RMSYNC_OUTPUT_DIR=/home/legible/Documents/ReMarkable"
Environment="RMSYNC_LABELS=work"

[Install]
WantedBy=multi-user.target
```

**Enable and start:**
```bash
sudo systemctl enable legible
sudo systemctl start legible
sudo systemctl status legible
```

## Docker

Run in a Docker container:

**`Dockerfile`:**
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o legible ./cmd/legible

FROM alpine:latest
RUN apk --no-cache add ca-certificates tesseract-ocr
COPY --from=builder /app/legible /usr/local/bin/
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s \
  CMD wget --quiet --tries=1 --spider http://localhost:8080/health || exit 1
CMD ["legible", "daemon", "--health-addr", ":8080"]
```

**Build and run:**
```bash
docker build -t legible .
docker run -d \
  --name legible \
  -p 8080:8080 \
  -v ~/.legible.yaml:/root/.legible.yaml:ro \
  -v ~/Documents/ReMarkable:/data/output \
  -e RMSYNC_OUTPUT_DIR=/data/output \
  legible
```

**Check health:**
```bash
curl http://localhost:8080/health
```

## Build from Source

**Development build:**
```bash
go build -o legible ./cmd/legible
```

**Production build with version info:**
```bash
VERSION=0.1.0
GIT_COMMIT=$(git rev-parse --short HEAD)
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

go build \
  -ldflags="-X main.Version=${VERSION} -X main.GitCommit=${GIT_COMMIT} -X main.BuildDate=${BUILD_DATE}" \
  -o legible \
  ./cmd/legible
```

## Troubleshooting

### Authentication fails

```bash
# Clear credentials and re-authenticate
rm ~/.rmapi
legible auth
```

### OCR not working

Ensure Tesseract OCR is installed:

```bash
# macOS
brew install tesseract

# Ubuntu/Debian
sudo apt-get install tesseract-ocr libtesseract-dev

# Verify installation
tesseract --version
```

### Daemon won't start

Check logs for errors:
```bash
legible daemon --log-level debug
```

Common issues:
- Port already in use (health check address)
- Permission denied (PID file or output directory)
- Invalid configuration file

### Force fresh sync

Clear state file to force re-sync all documents:
```bash
rm ~/Documents/ReMarkable/.legible-state.json
legible sync
```

Or use the `--force` flag:
```bash
legible sync --force
```

## Exit Codes

- `0` - Success
- `1` - Error (check error message for details)

## Support

- GitHub Issues: https://github.com/platinummonkey/legible/issues
- Documentation: https://github.com/platinummonkey/legible

## License

Part of legible project. See project LICENSE for details.
