# macOS Menu Bar Application

The Legible menu bar application provides a convenient interface for managing document synchronization from your reMarkable tablet directly from the macOS menu bar.

## Features

- **Visual Status Indicators**: Green (idle), yellow (syncing), red (error)
- **Auto-Launch Daemon**: Automatically starts and manages the daemon process
- **Real-time Status**: Shows sync progress and last sync results
- **Quick Actions**: Trigger sync, cancel sync, open output folder
- **Background Operation**: Runs in the menu bar, doesn't clutter your Dock

## Installation

### From Release (Recommended)

1. Download the latest `legible-menubar_VERSION_darwin_ARCH.zip` from the [releases page](https://github.com/platinummonkey/legible/releases)
2. Extract the zip file
3. Copy `Legible.app` to `/Applications/`
4. Copy the `legible` daemon binary to `/usr/local/bin/` or another location in your PATH
5. Launch `Legible.app` from Applications

**Note**: You may need to allow the app in System Preferences > Security & Privacy if macOS blocks it on first launch.

### From Source

```bash
# Clone the repository
git clone https://github.com/platinummonkey/legible.git
cd legible

# Build the app bundle and daemon
make build-menubar-app

# Install
cp dist/Legible.app /Applications/
cp dist/legible /usr/local/bin/

# Launch
open /Applications/Legible.app
```

### Via Homebrew (Coming Soon)

```bash
brew install platinummonkey/tap/legible-menubar
```

## Configuration

The menu bar app uses the same configuration file as the CLI tool:

```bash
# Create or edit config
vi ~/.legible.yaml
```

### Configuration Options

```yaml
# reMarkable API configuration
remarkable:
  device_token: ""  # Will be set after authentication

# Sync settings
sync:
  output_dir: "~/remarkable"
  labels: []  # Optional: filter by labels

# OCR settings (optional)
ocr:
  enabled: true
  provider: "ollama"
  ollama:
    base_url: "http://localhost:11434"
    model: "llava"

# Daemon settings
daemon:
  interval: 30m  # How often to sync automatically
  health_addr: "localhost:8080"  # HTTP API address
```

## First-Time Setup

1. **Authenticate with reMarkable**:
   ```bash
   legible auth
   ```
   This only needs to be done once. The CLI and menu bar app share credentials.

2. **Configure Output Directory** (optional):
   Edit `~/.legible.yaml` to set your preferred output directory.

3. **Install Ollama** (optional, for OCR):
   ```bash
   brew install ollama
   ollama pull llava
   ```

4. **Launch the App**:
   Open `Legible.app` from Applications.

## Usage

### Menu Bar Interface

The menu bar icon shows the current sync status:

- **Green circle**: Idle - no sync in progress
- **Yellow circle**: Syncing - sync in progress
- **Red circle**: Error - last sync failed or daemon offline

### Menu Options

- **Status**: Shows current status and last sync results
- **Trigger Sync**: Manually start a sync
- **Cancel Sync**: Cancel the running sync
- **Open Output Folder**: Opens the output directory in Finder
- **Preferences**: Configure settings (coming soon)
- **Quit**: Exit the application and stop the daemon

### Command-Line Options

```bash
# Start with custom configuration
/Applications/Legible.app/Contents/MacOS/legible-menubar --config ~/custom-config.yaml

# Use a different output directory
/Applications/Legible.app/Contents/MacOS/legible-menubar --output ~/Documents/reMarkable

# Connect to daemon on different port
/Applications/Legible.app/Contents/MacOS/legible-menubar --daemon-addr http://localhost:8081

# Disable auto-launch (connect to existing daemon)
/Applications/Legible.app/Contents/MacOS/legible-menubar --no-auto-launch
```

## How It Works

### Architecture

The menu bar application consists of two components:

1. **Menu Bar App** (`Legible.app`): The UI that sits in your menu bar
2. **Daemon** (`legible`): The background service that performs syncs

### Daemon Management

The menu bar app automatically:

- Launches the daemon when the app starts
- Monitors daemon health every 5 seconds
- Restarts the daemon if it crashes (up to 5 attempts)
- Stops the daemon when you quit the app

You can disable auto-launch with the `--no-auto-launch` flag if you prefer to manage the daemon manually.

### Status API

The daemon exposes an HTTP API for status monitoring:

- **GET /status**: Current sync status and progress
- **POST /api/sync/trigger**: Trigger a manual sync
- **POST /api/sync/cancel**: Cancel running sync
- **GET /health**: Health check endpoint

See [Daemon API Documentation](daemon-api.md) for details.

## Troubleshooting

### Menu Bar App Won't Launch

**Problem**: Double-clicking Legible.app does nothing or shows an error.

**Solutions**:
1. Check System Preferences > Security & Privacy
2. Right-click the app and select "Open" to bypass Gatekeeper
3. Remove quarantine attribute:
   ```bash
   xattr -dr com.apple.quarantine /Applications/Legible.app
   ```

### Red Status Icon - "Daemon offline"

**Problem**: The icon is red and status shows "Daemon offline".

**Solutions**:
1. Check if daemon is in PATH:
   ```bash
   which legible
   ```
2. Check daemon logs:
   ```bash
   tail -f ~/Library/Logs/legible-daemon.log
   ```
3. Try restarting the app
4. Manually start daemon:
   ```bash
   legible daemon --health-addr localhost:8080
   ```

### Sync Not Working

**Problem**: Trigger sync does nothing or fails.

**Solutions**:
1. Check authentication:
   ```bash
   legible auth --status
   ```
2. Check daemon status:
   ```bash
   curl http://localhost:8080/status
   ```
3. Check output directory exists and is writable
4. View daemon logs for errors

### High CPU Usage

**Problem**: Menu bar app or daemon using lots of CPU.

**Solutions**:
1. Check if OCR is enabled and Ollama is running
2. Reduce sync interval in config
3. Disable OCR temporarily:
   ```yaml
   ocr:
     enabled: false
   ```

### Menu Bar App Not Updating Status

**Problem**: Status stuck on "Checking daemon..." or outdated.

**Solutions**:
1. Check daemon is responsive:
   ```bash
   curl http://localhost:8080/health
   ```
2. Restart the menu bar app
3. Check for network issues blocking localhost

## Building from Source

### Prerequisites

- macOS 11.0 or later
- Go 1.25 or later
- Xcode Command Line Tools

### Build Steps

```bash
# Clone repository
git clone https://github.com/platinummonkey/legible.git
cd legible

# Build everything
make build-menubar-app

# Output:
# - dist/Legible.app (app bundle)
# - dist/Legible.zip (for distribution)
# - dist/legible (daemon binary)
```

### Development Build

```bash
# Build just the menu bar binary
make build-menubar

# Run directly
./dist/legible-menubar
```

### Creating a Release

```bash
# Tag a new version
git tag -a v1.3.0 -m "Release v1.3.0"
git push origin v1.3.0

# GitHub Actions will automatically:
# 1. Build for all platforms
# 2. Create macOS .app bundle
# 3. Sign binaries with cosign
# 4. Create GitHub release
# 5. Update Homebrew tap
```

## Uninstallation

```bash
# Remove app bundle
rm -rf /Applications/Legible.app

# Remove daemon binary
rm /usr/local/bin/legible

# Remove configuration (optional)
rm ~/.legible.yaml

# Remove synced documents (optional)
rm -rf ~/remarkable
```

## FAQ

### Can I use the CLI and menu bar app at the same time?

Yes! They share the same configuration and credentials. You can use the CLI for scripting and the menu bar app for convenience.

### Does the menu bar app work without the daemon?

No, the menu bar app requires the daemon to perform syncs. However, it will automatically launch and manage the daemon for you.

### Can I sync multiple reMarkable tablets?

Not currently. The app is designed for a single tablet per user account.

### Does this work on older macOS versions?

The minimum supported version is macOS 11.0 (Big Sur). For older versions, use the CLI tool instead.

### Can I run multiple instances of the menu bar app?

No, only one instance should run at a time. Multiple instances will conflict over the daemon management.

## Related Documentation

- [Daemon API Documentation](daemon-api.md)
- [Menu Bar Testing Report](menubar-testing-report.md)
- [Menu Bar Icon Design](../assets/menubar-icons/README.md)
- [Main README](../README.md)

## Contributing

See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines on contributing to the menu bar application.

## License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details.
