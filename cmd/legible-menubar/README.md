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
  --version            Show version information
  --config PATH        Path to configuration file
  --output DIR         Output directory for synced documents
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

**This is a work in progress.** Current implementation includes:

- ✅ Menu bar icon display
- ✅ Basic menu structure
- ✅ Placeholder status icons (green/yellow/red)
- ✅ Open output directory action
- ⏳ Daemon communication (coming in remarkable-sync-u9j)
- ⏳ Real-time status updates
- ⏳ Start/stop sync actions
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

The menu bar app will communicate with the `legible daemon` process via:

- HTTP API (preferred) or Unix socket
- Poll for status every 2-5 seconds
- Send control commands (start/stop sync)

See `remarkable-sync-u9j` for daemon status API design.

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
