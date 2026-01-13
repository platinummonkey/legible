# Legible Menu Bar Application

macOS menu bar application for Legible reMarkable sync.

## Overview

The menu bar app provides a convenient GUI interface for managing the Legible sync daemon on macOS. It displays a status icon in the menu bar and allows quick access to common operations.

## Features

- **Status Icons**: Visual indication of sync state
  - Green: Idle (ready to sync)
  - Yellow: Actively syncing/processing
  - Red: Error state
- **Menu Actions**:
  - View current status
  - Start/stop sync
  - Open output directory in Finder
  - Preferences (coming soon)
  - Quit application

## Building

The menu bar app can only be built on macOS:

```bash
make build-menubar
```

The binary will be created at `dist/legible-menubar`.

## Usage

### Running from command line

```bash
./dist/legible-menubar
```

### Command line options

```bash
./dist/legible-menubar --help

Flags:
  --version              Show version information
  --config PATH          Path to configuration file
  --output DIR           Output directory for synced documents
  --daemon-addr URL      Daemon HTTP address (default: http://localhost:8080)
```

### Running with a daemon

The menu bar app requires the legible daemon to be running with the HTTP API enabled:

```bash
# Terminal 1: Start the daemon with HTTP API
legible daemon --health-addr localhost:8080

# Terminal 2 (or just double-click the app): Start menu bar
./dist/legible-menubar

# Or specify custom daemon address
./dist/legible-menubar --daemon-addr http://localhost:9090
```

### Configuration

The menu bar app uses the same configuration file as the CLI tool:

- `~/.legible.yaml` (default location)
- Custom path via `--config` flag

Example configuration:

```yaml
output-dir: ~/Documents/reMarkable
log-level: info
labels:
  - work
  - personal
```

## Current Status

Current implementation includes:

- ✅ Menu bar icon display
- ✅ Menu structure with status display
- ✅ Status icons (green/yellow/red) based on daemon state
- ✅ Open output directory action
- ✅ **Daemon communication via HTTP API**
- ✅ **Real-time status updates (polls every 3 seconds)**
- ✅ **Trigger sync action** (calls daemon API)
- ✅ **Cancel sync action** (calls daemon API)
- ✅ **Offline detection** (shows red icon when daemon not running)
- ⏳ Preferences dialog
- ⏳ Auto-start on login

## Architecture

```
cmd/legible-menubar/
  main.go           - Entry point, CLI parsing, config loading

internal/menubar/
  app.go            - Menu bar application logic
  icons.go          - Status icon data (placeholder PNGs)
```

## Development Notes

### Build Tags

All menubar code uses the `darwin` build tag to ensure it only compiles on macOS:

```go
//go:build darwin
// +build darwin
```

### Dependencies

- `fyne.io/systray` - Cross-platform system tray library
- Requires cgo (automatically enabled in Makefile)

### Next Steps

See the following issues for planned work:

- **remarkable-sync-u9j**: Design status API for daemon mode
- **remarkable-sync-3v6**: Connect menu bar app to daemon status API
- **remarkable-sync-fsd**: Design and implement proper status indicator icons
- **remarkable-sync-sch**: Add menu bar app build target and distribution
- **remarkable-sync-1dy**: Comprehensive testing

## Integration with Daemon

The menu bar app communicates with the `legible daemon` process via HTTP API:

- **Status polling**: Polls `/status` endpoint every 3 seconds
- **Icon updates**: Automatically changes based on sync state
  - 🟢 Green: Daemon idle, last sync successful
  - 🟡 Yellow: Sync in progress
  - 🔴 Red: Error or daemon offline
- **Control commands**:
  - "Trigger Sync" → `POST /api/sync/trigger`
  - "Cancel Sync" → `POST /api/sync/cancel`
- **Status display**: Shows real-time information
  - Last sync results (docs processed, success/fail counts)
  - Current sync progress (X/Y documents)
  - Current document being processed
  - Error messages

See `docs/daemon-api.md` for complete API documentation.

## .app Bundle (Future)

For distribution, the binary will be packaged as a proper macOS application bundle:

```
Legible.app/
  Contents/
    Info.plist
    MacOS/
      legible-menubar
    Resources/
      icons/
```

This will enable:
- Double-click to launch
- Application icon
- Proper macOS integration
- Code signing for distribution
