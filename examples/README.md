# reMarkable Sync Examples

This directory contains example configuration files and scripts for various use cases.

## Configuration Files

### config.yaml

Complete configuration file template with all available options documented.

**Usage:**

```bash
# Copy to default location
cp examples/config.yaml ~/.remarkable-sync.yaml

# Edit to customize
vi ~/.remarkable-sync.yaml

# Use with remarkable-sync
remarkable-sync sync
```

**Or specify config file explicitly:**

```bash
remarkable-sync --config /path/to/config.yaml sync
```

## System Integration

### systemd (Linux)

The `systemd/remarkable-sync.service` file is a systemd service unit for running remarkable-sync as a system daemon.

**Installation:**

```bash
# Copy service file
sudo cp examples/systemd/remarkable-sync.service /etc/systemd/system/remarkable-sync@.service

# Reload systemd
sudo systemctl daemon-reload

# Enable service for your user
sudo systemctl enable remarkable-sync@$USER

# Start service
sudo systemctl start remarkable-sync@$USER

# Check status
sudo systemctl status remarkable-sync@$USER

# View logs
sudo journalctl -u remarkable-sync@$USER -f
```

**Configuration:**

Edit the service file before installation to customize:
- ExecStart path (location of remarkable-sync binary)
- User and Group
- WorkingDirectory
- Environment variables
- Resource limits

**Health Monitoring:**

The example service enables the health check endpoint on port 8080:

```bash
# Check if daemon is healthy
curl http://localhost:8080/health

# Expected response: {"status":"ok","last_sync":"2024-01-15T10:30:00Z"}
```

**Stopping the Service:**

```bash
# Stop the service
sudo systemctl stop remarkable-sync@$USER

# Disable from starting on boot
sudo systemctl disable remarkable-sync@$USER
```

### launchd (macOS)

Create a launchd plist file for macOS:

**File:** `~/Library/LaunchAgents/com.remarkable-sync.plist`

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.remarkable-sync</string>

    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/remarkable-sync</string>
        <string>daemon</string>
        <string>--interval</string>
        <string>10m</string>
        <string>--health-addr</string>
        <string>:8080</string>
    </array>

    <key>RunAtLoad</key>
    <true/>

    <key>KeepAlive</key>
    <dict>
        <key>SuccessfulExit</key>
        <false/>
    </dict>

    <key>StandardOutPath</key>
    <string>/tmp/remarkable-sync.log</string>

    <key>StandardErrorPath</key>
    <string>/tmp/remarkable-sync-error.log</string>
</dict>
</plist>
```

**Usage:**

```bash
# Load the service
launchctl load ~/Library/LaunchAgents/com.remarkable-sync.plist

# Check status
launchctl list | grep remarkable-sync

# Unload the service
launchctl unload ~/Library/LaunchAgents/com.remarkable-sync.plist
```

## Usage Examples

### Basic Sync

```bash
# First time: authenticate
remarkable-sync auth

# Sync all documents with OCR
remarkable-sync sync

# Sync without OCR (faster)
remarkable-sync sync --no-ocr
```

### Label-based Sync

```bash
# Sync only work documents
remarkable-sync sync --labels work

# Sync multiple categories
remarkable-sync sync --labels "work,personal,meetings"

# Use config file for consistent labels
remarkable-sync --config ~/.remarkable-work.yaml sync
```

### Daemon Mode

```bash
# Run daemon with default 5-minute interval
remarkable-sync daemon

# Custom interval
remarkable-sync daemon --interval 15m

# With health monitoring
remarkable-sync daemon --interval 10m --health-addr :8080

# With PID file
remarkable-sync daemon --interval 10m --pid-file ~/.remarkable-sync.pid
```

### Custom Output Locations

```bash
# Sync to Dropbox
remarkable-sync sync --output ~/Dropbox/ReMarkable

# Sync to external drive
remarkable-sync sync --output /Volumes/Backup/ReMarkable

# Sync to NAS
remarkable-sync sync --output /mnt/nas/documents/remarkable
```

### Debugging

```bash
# Enable debug logging
remarkable-sync sync --log-level debug

# Force re-sync all documents (ignore state)
remarkable-sync sync --force

# Test authentication
remarkable-sync auth
```

## Configuration Examples

### Minimal Setup

**File:** `~/.remarkable-sync.yaml`

```yaml
output-dir: ~/remarkable-sync
ocr-enabled: true
log-level: info
```

### Work Documents Only

**File:** `~/.remarkable-work.yaml`

```yaml
output-dir: ~/work-documents
labels:
  - work
  - meetings
  - projects
ocr-enabled: true
ocr-languages: eng
sync-interval: 15m
log-level: info
```

### Fast Backup (No OCR)

**File:** `~/.remarkable-backup.yaml`

```yaml
output-dir: ~/remarkable-backup
ocr-enabled: false
sync-interval: 5m
log-level: warn
```

### Multi-language OCR

**File:** `~/.remarkable-multilang.yaml`

```yaml
output-dir: ~/Documents/remarkable
ocr-enabled: true
ocr-languages: eng+spa+fra+deu
sync-interval: 20m
log-level: info
```

### Production Daemon

**File:** `/etc/remarkable-sync.yaml`

```yaml
output-dir: /var/lib/remarkable-sync/documents
ocr-enabled: true
ocr-languages: eng
sync-interval: 10m
state-file: /var/lib/remarkable-sync/state.json
log-level: info
daemon-mode: true
health-addr: ":8080"
pid-file: /var/run/remarkable-sync.pid
```

## Automation Scripts

### Sync and Upload to Cloud

```bash
#!/bin/bash
# sync-and-upload.sh - Sync documents and upload to cloud storage

# Sync from reMarkable
remarkable-sync sync --output ~/remarkable-temp

# Upload to cloud (example: rclone)
rclone sync ~/remarkable-temp remote:remarkable

# Clean up
rm -rf ~/remarkable-temp
```

### Scheduled Sync with Cron

```bash
# Add to crontab (crontab -e)

# Sync every hour
0 * * * * /usr/local/bin/remarkable-sync sync --output ~/Documents/remarkable

# Sync at 8 AM and 8 PM
0 8,20 * * * /usr/local/bin/remarkable-sync sync

# Sync every 30 minutes during work hours
*/30 9-17 * * 1-5 /usr/local/bin/remarkable-sync sync --labels work
```

### Backup Script

```bash
#!/bin/bash
# backup-remarkable.sh - Daily backup of reMarkable documents

DATE=$(date +%Y-%m-%d)
BACKUP_DIR=~/backups/remarkable/$DATE

# Sync documents
remarkable-sync sync --output "$BACKUP_DIR"

# Create archive
tar -czf "$BACKUP_DIR.tar.gz" "$BACKUP_DIR"
rm -rf "$BACKUP_DIR"

# Keep only last 30 days
find ~/backups/remarkable -name "*.tar.gz" -mtime +30 -delete
```

## Testing

Before deploying in production, test your configuration:

```bash
# Test authentication
remarkable-sync auth

# Test sync with your config
remarkable-sync --config your-config.yaml sync --no-ocr

# Test daemon mode (run for a few minutes then stop)
remarkable-sync --config your-config.yaml daemon --interval 1m

# Check health endpoint if enabled
curl http://localhost:8080/health
```

## Monitoring

### Health Checks

```bash
# Simple health check
curl -f http://localhost:8080/health || echo "Daemon unhealthy"

# Monitor with uptime monitoring tool
# Add HTTP check for http://localhost:8080/health
```

### Logging

```bash
# View systemd logs
journalctl -u remarkable-sync@$USER -f

# View recent errors
journalctl -u remarkable-sync@$USER -p err -n 50

# View logs for specific time period
journalctl -u remarkable-sync@$USER --since "1 hour ago"
```

### Metrics

Monitor these metrics for production deployments:
- Sync success/failure rate
- Sync duration
- Number of documents processed
- Disk space usage
- Memory usage (OCR can be memory-intensive)
- Health endpoint response time

## Troubleshooting

If examples don't work:

1. **Check paths**: Ensure binary and config paths are correct
2. **Check permissions**: Service user needs read/write access to output directory
3. **Check logs**: Look for errors in systemd journal or launchd logs
4. **Test manually**: Run remarkable-sync manually to verify it works
5. **Verify authentication**: Ensure auth token exists and is valid

For more help, see:
- [FAQ.md](../FAQ.md)
- [README.md](../README.md)
- [CONTRIBUTING.md](../CONTRIBUTING.md)
