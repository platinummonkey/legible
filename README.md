# reMarkable Sync

Sync documents from your reMarkable tablet and add OCR text layers to make handwritten notes searchable.

## Features

- 📥 Download documents from reMarkable cloud
- 📄 Convert .rmdoc files to standard PDF format
- 🏷️ Optional label-based filtering
- 🔍 OCR processing with Tesseract (optional)
- 📝 Add hidden searchable text layer to PDFs
- 🔄 Incremental sync with state tracking
- ⚙️ Daemon mode for continuous background sync
- 🏥 Health check endpoints for monitoring
- 📋 Comprehensive logging and error handling

## Prerequisites

- Go 1.21 or higher
- Tesseract OCR installed on your system
- reMarkable tablet with cloud sync enabled

### Installing Tesseract

**macOS:**
```bash
brew install tesseract
```

**Ubuntu/Debian:**
```bash
sudo apt-get install tesseract-ocr
```

**Windows:**
Download from [GitHub releases](https://github.com/UB-Mannheim/tesseract/wiki)

## Installation

### Using Docker (Recommended)

Pull the official container image from GitHub Container Registry:

```bash
# Pull latest version
docker pull ghcr.io/platinummonkey/remarkable-sync:latest

# Or specific version
docker pull ghcr.io/platinummonkey/remarkable-sync:v0.1.0
```

Multi-platform images available:
- `linux/amd64` (Intel/AMD 64-bit)
- `linux/arm64` (ARM 64-bit, including Apple Silicon)

### Using Go Install

```bash
go install github.com/platinummonkey/remarkable-sync/cmd/remarkable-sync@latest
```

### Build from Source

```bash
git clone https://github.com/platinummonkey/remarkable-sync.git
cd remarkable-sync
make build-local
```

The binary will be available at `dist/remarkable-sync_*/remarkable-sync`.

## Configuration

First time setup requires authenticating with your reMarkable account:

```bash
remarkable-sync auth
```

This will prompt you for a one-time code from https://my.remarkable.com/device/connect/desktop

## Usage

### Native Binary

**Sync all documents:**

```bash
remarkable-sync sync
```

**Sync documents with specific labels:**

```bash
remarkable-sync sync --labels "work,personal"
```

**Specify output directory:**

```bash
remarkable-sync sync --output ./my-remarkable-docs
```

**Run in daemon mode:**

For continuous background synchronization:

```bash
remarkable-sync daemon --interval 10m --health-addr :8080
```

### Using Docker

**First-time authentication:**

```bash
docker run --rm -it \
  -v $HOME/.rmapi:/home/remarkable/.rmapi \
  ghcr.io/platinummonkey/remarkable-sync:latest auth
```

**Sync documents:**

```bash
docker run --rm \
  -v $HOME/.rmapi:/home/remarkable/.rmapi \
  -v $PWD/output:/output \
  ghcr.io/platinummonkey/remarkable-sync:latest sync --output /output
```

**Sync with labels:**

```bash
docker run --rm \
  -v $HOME/.rmapi:/home/remarkable/.rmapi \
  -v $PWD/output:/output \
  ghcr.io/platinummonkey/remarkable-sync:latest sync \
    --output /output \
    --labels "work,personal"
```

**Run in daemon mode:**

```bash
docker run -d \
  --name remarkable-sync \
  -v $HOME/.rmapi:/home/remarkable/.rmapi \
  -v $PWD/output:/output \
  ghcr.io/platinummonkey/remarkable-sync:latest daemon \
    --interval 30m \
    --output /output
```

**Docker Compose:**

Create a `docker-compose.yml`:

```yaml
version: '3.8'

services:
  remarkable-sync:
    image: ghcr.io/platinummonkey/remarkable-sync:latest
    container_name: remarkable-sync
    restart: unless-stopped
    volumes:
      - ./credentials:/home/remarkable/.rmapi
      - ./output:/output
    command: daemon --interval 1h --output /output
```

Run with:
```bash
docker-compose up -d
```

### Full command options

**Sync command:**
```bash
remarkable-sync sync [flags]

Flags:
  --output string      Output directory (default: ~/ReMarkable)
  --labels strings     Filter by labels (comma-separated)
  --no-ocr            Skip OCR processing
  --force             Force re-sync all documents
  --log-level string  Log level: debug, info, warn, error (default: info)
  --config string     Config file (default: ~/.remarkable-sync.yaml)
```

**Daemon command:**
```bash
remarkable-sync daemon [flags]

Flags:
  --interval duration   Sync interval (default: 5m)
  --health-addr string  Health check HTTP address (e.g., :8080)
  --pid-file string     PID file path
  --output string       Output directory
  --labels strings      Filter by labels
  --no-ocr             Skip OCR processing
```

**Other commands:**
```bash
remarkable-sync auth      # Authenticate with reMarkable API
remarkable-sync version   # Display version information
remarkable-sync help      # Display help
```

## Common Use Cases

### Quick Start: First Time Sync

```bash
# 1. Authenticate with reMarkable cloud
remarkable-sync auth

# 2. Sync all documents (with OCR)
remarkable-sync sync

# 3. Check your documents
ls ~/remarkable-sync/
```

### Selective Sync by Label

Organize your reMarkable documents with labels, then sync only what you need:

```bash
# Sync only work documents
remarkable-sync sync --labels work

# Sync multiple label categories
remarkable-sync sync --labels "work,personal,important"
```

### Background Daemon Mode

Run continuous sync in the background:

```bash
# Start daemon with 15-minute interval
remarkable-sync daemon --interval 15m --log-level info

# With health check endpoint (for monitoring)
remarkable-sync daemon --interval 10m --health-addr :8080

# Check health status
curl http://localhost:8080/health
```

### Development Workflow

For developers who want searchable notes without OCR overhead:

```bash
# Quick sync without OCR (faster)
remarkable-sync sync --no-ocr

# Debug sync issues
remarkable-sync sync --log-level debug --force
```

### Custom Output Organization

```bash
# Sync to specific directory
remarkable-sync sync --output ~/Dropbox/ReMarkable

# Use config file for consistent setup
remarkable-sync --config ~/.remarkable-work.yaml sync
```

## How It Works

1. **Authentication**: Connects to reMarkable cloud using your credentials
2. **Download**: Fetches `.rmdoc` files and renders them as PDFs
3. **OCR**: Processes each page with Tesseract to extract text (optional)
4. **Enhancement**: Adds invisible text layer to PDF at correct positions
5. **Save**: Outputs searchable PDF files to the specified directory
6. **State**: Tracks synced documents to enable incremental updates

## Output Structure

```
remarkable-docs/
├── My Notebook/
│   ├── notes.pdf          # Original rendered PDF
│   └── notes_ocr.pdf      # Enhanced PDF with text layer
├── Work Documents/
│   └── meeting_notes_ocr.pdf
└── .sync-state.json       # Tracks sync state
```

## Configuration File

Create a `~/.remarkable-sync.yaml` for default settings:

```yaml
# Output directory for synced PDFs (default: ~/remarkable-sync)
output-dir: ~/Documents/remarkable

# Filter documents by labels (empty = sync all)
labels:
  - work
  - important

# Enable/disable OCR processing (default: true)
ocr-enabled: true

# OCR languages (default: eng)
ocr-languages: eng+spa

# Logging level: debug, info, warn, error (default: info)
log-level: info

# Sync interval for daemon mode (default: 5m)
sync-interval: 10m

# State file location (default: ~/.remarkable-sync-state.json)
state-file: ~/.remarkable-sync/state.json
```

**Configuration precedence:** CLI flags > Environment variables > Config file > Defaults

**Environment variables:**

```bash
export REMARKABLE_SYNC_OUTPUT_DIR=~/Documents/remarkable
export REMARKABLE_SYNC_LABELS=work,personal
export REMARKABLE_SYNC_OCR_ENABLED=false
export REMARKABLE_SYNC_LOG_LEVEL=debug
```

See [examples/config.yaml](examples/config.yaml) for a complete configuration template.

## Development

### Building

The build system uses `pkg-config` to automatically detect Tesseract and Leptonica library paths. This ensures builds work across different platforms and library versions without hardcoded paths.

**Prerequisites:**
- `pkg-config` (usually pre-installed on macOS/Linux)
- Tesseract and Leptonica development libraries

```bash
# Install build dependencies (if not already installed)
# macOS:
brew install pkg-config tesseract leptonica

# Ubuntu/Debian:
sudo apt-get install pkg-config libtesseract-dev libleptonica-dev

# Verify pkg-config can find libraries:
pkg-config --modversion tesseract lept
```

**Build commands:**
```bash
make build-local    # Build for current platform (recommended, uses goreleaser)
make build-all      # Build for all platforms (uses goreleaser)
make build          # Simple Go build (single platform, no goreleaser)
make install        # Install to $GOPATH/bin
```

**Build System:**
- The project uses [goreleaser](https://goreleaser.com) for consistent, reproducible builds
- `make build-local` and `make build-all` use goreleaser for production-quality builds
- The Makefile automatically sets `CGO_CFLAGS`, `CGO_CXXFLAGS`, and `CGO_LDFLAGS` using `pkg-config`
- Built binaries are in `dist/remarkable-sync_<os>_<arch>/` directory
- If pkg-config cannot find the libraries, you'll see a warning and the build may fail

**Installing goreleaser:**
```bash
# macOS:
brew install goreleaser

# Linux:
# See https://goreleaser.com/install/
```

### Running tests

```bash
make test           # Run all tests
make test-coverage  # Run tests with coverage report
```

### Code quality

```bash
make lint           # Run linter (golangci-lint)
make fmt            # Format code
make vet            # Run go vet
make tidy           # Tidy go modules
```

### Development workflow

```bash
make dev            # Run in development mode
make run            # Build and run
make clean          # Clean build artifacts
```

### Available Make targets

Run `make help` to see all available targets.

## Project Structure

See [AGENTS.md](./AGENTS.md) for detailed architecture and design documentation.

## Troubleshooting

### Build and Installation Issues

**"Tesseract not found" or build errors**
- Ensure Tesseract is installed and in your PATH
- Verify installation: `tesseract --version`
- Verify pkg-config can find libraries: `pkg-config --modversion tesseract lept`
- Install development libraries:
  - macOS: `brew install pkg-config tesseract leptonica`
  - Ubuntu: `sudo apt-get install pkg-config tesseract-ocr libtesseract-dev libleptonica-dev`
  - Windows: Download from [UB Mannheim](https://github.com/UB-Mannheim/tesseract/wiki)

If pkg-config cannot find the libraries, you may need to set `PKG_CONFIG_PATH`:
```bash
# macOS (Homebrew):
export PKG_CONFIG_PATH="/opt/homebrew/lib/pkgconfig:$PKG_CONFIG_PATH"

# Linux (custom install):
export PKG_CONFIG_PATH="/usr/local/lib/pkgconfig:$PKG_CONFIG_PATH"
```

**"go: module not found" errors**
```bash
# Ensure Go modules are downloaded
go mod download
go mod tidy

# Try building again
make build
```

### Authentication Issues

**Authentication fails or "token not found"**
- Ensure your reMarkable has cloud sync enabled in Settings → Storage
- Get a new one-time code from https://my.remarkable.com/device/connect/desktop
- Clear old credentials and re-authenticate:
  ```bash
  rm -rf ~/.remarkable-sync
  remarkable-sync auth
  ```

**"Device not registered" error**
- Remar notable may have unlinked the device
- Re-authenticate with `remarkable-sync auth`
- Contact reMarkable support if issues persist

### Sync Issues

**Documents not syncing**
- Check you have cloud sync enabled on your reMarkable
- Verify documents have been uploaded to cloud
- Force re-sync: `remarkable-sync sync --force`
- Check logs: `remarkable-sync sync --log-level debug`

**Only some documents sync**
- Check if you're using label filters: `--labels`
- Verify document labels in reMarkable app
- Try syncing without filters: `remarkable-sync sync`

**Sync is very slow**
- OCR processing is CPU-intensive
- Try without OCR first: `remarkable-sync sync --no-ocr`
- Reduce sync frequency in daemon mode
- Check available disk space

### OCR Issues

**OCR quality is poor**
- Check Tesseract language data is installed
- Install additional languages:
  - macOS: `brew install tesseract-lang`
  - Ubuntu: `sudo apt-get install tesseract-ocr-eng tesseract-ocr-spa`
- Specify languages in config: `ocr-languages: eng+spa+fra`

**OCR crashes or fails**
- Verify Tesseract installation: `tesseract --version`
- Check system resources (OCR is memory-intensive)
- Skip OCR for testing: `--no-ocr`

### Daemon Issues

**Daemon won't start**
- Check logs: `remarkable-sync daemon --log-level debug`
- Verify port isn't in use: `lsof -i :8080` (if using --health-addr)
- Check file permissions for PID file location
- Ensure output directory is writable

**Daemon stops unexpectedly**
- Check system logs: `journalctl -u remarkable-sync` (if using systemd)
- Verify sufficient disk space
- Check for OOM (out of memory) errors in system logs

**Health check endpoint not responding**
- Verify port is correct: `curl http://localhost:8080/health`
- Check firewall settings
- Ensure daemon is running: `ps aux | grep remarkable-sync`

### Other Issues

**Large output files**
- PDFs with OCR text layers are larger than originals
- Disable OCR if size is a concern: `--no-ocr`
- OCR'd PDFs are searchable but larger (~2-3x original size)

**Permission denied errors**
- Check output directory permissions
- Verify state file location is writable
- Run with appropriate user permissions

For more help, see [FAQ.md](FAQ.md) or [open an issue](https://github.com/platinummonkey/remarkable-sync/issues).

## Security

### Software Bill of Materials (SBOM)

Each release includes Software Bill of Materials (SBOMs) in SPDX format for transparency and security auditing:

- **Binary SBOMs**: Available for each platform-specific binary
- **Container SBOMs**: Attached to Docker images as attestations
- **Source SBOM**: Complete dependency tree for the project

**Downloading SBOMs:**

SBOMs are published with each release on GitHub:

```bash
# Download SBOM for a specific release
curl -LO https://github.com/platinummonkey/remarkable-sync/releases/download/v1.0.0/remarkable-sync_1.0.0_linux_amd64.sbom.spdx.json
```

**Extracting Container SBOMs:**

Container images include embedded SBOMs:

```bash
# Pull image
docker pull ghcr.io/platinummonkey/remarkable-sync:latest

# Extract SBOM
docker buildx imagetools inspect ghcr.io/platinummonkey/remarkable-sync:latest --format "{{ json .SBOM }}"
```

**Vulnerability Scanning:**

Use the SBOM to scan for known vulnerabilities:

```bash
# Install grype (vulnerability scanner)
brew install anchore/grype/grype

# Scan using SBOM
grype sbom:./remarkable-sync_1.0.0_linux_amd64.sbom.spdx.json

# Or scan the container image directly
grype ghcr.io/platinummonkey/remarkable-sync:latest
```

**SBOM Contents:**

The SBOM includes:
- All Go dependencies with versions
- Transitive dependencies
- Package licenses
- File checksums
- Build information

This enables users to:
- Track known vulnerabilities in dependencies
- Meet compliance requirements (Executive Order 14028, NTIA guidelines)
- Audit the software supply chain
- Verify software integrity

For more information on SBOMs, see:
- [NTIA SBOM Minimum Elements](https://www.ntia.gov/sbom)
- [SPDX Specification](https://spdx.dev/)

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines on:

- Setting up your development environment
- Running tests and quality checks
- Code style and conventions
- Submitting pull requests
- Reporting bugs and requesting features

## License

MIT License - See LICENSE file for details

## Acknowledgments

- [rmapi](https://github.com/ddvk/rmapi) - reMarkable cloud API client
- [Tesseract OCR](https://github.com/tesseract-ocr/tesseract) - OCR engine

## Related Projects

- [rmapi](https://github.com/ddvk/rmapi) - Command-line tool for reMarkable cloud
- [remarkable-fs](https://github.com/nick8325/remarkable-fs) - FUSE filesystem for reMarkable
