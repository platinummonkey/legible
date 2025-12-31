# reMarkable Sync

Sync documents from your reMarkable tablet and add OCR text layers to make handwritten notes searchable.

## Features

- 📥 Download documents from reMarkable cloud
- 📄 Convert .rmdoc files to standard PDF format
- 🏷️ Optional label-based filtering
- 🔍 OCR processing with Ollama vision models (optional)
- 🤖 Superior handwriting recognition using AI models
- 📝 Add hidden searchable text layer to PDFs
- 🔄 Incremental sync with state tracking
- ⚙️ Daemon mode for continuous background sync
- 🏥 Health check endpoints for monitoring
- 📋 Comprehensive logging and error handling

## Prerequisites

- Go 1.21 or higher
- Ollama (for OCR functionality, optional)
- reMarkable tablet with cloud sync enabled

### Installing Ollama (Optional)

Ollama is required if you want searchable PDFs with OCR text layers. You can skip this if you only need PDF conversion without OCR.

**System Requirements:**
- Disk space: 10-12GB (for Ollama + mistral-small3.1 model)
- RAM: 6GB+ recommended for OCR processing (8GB+ for mistral-small3.1)
- CPU: Modern CPU with decent single-thread performance

**macOS/Linux:**
```bash
# Install Ollama
curl -fsSL https://ollama.ai/install.sh | sh

# Download the recommended vision model (required for OCR)
# Option 1: Mistral Small 3.1 (recommended - better multilingual and complex handwriting)
ollama pull mistral-small3.1

# Option 2: LLaVA (faster, lighter, good for basic handwriting)
ollama pull llava
```

**Windows:**
Download from [ollama.ai](https://ollama.ai/download)

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

**Building with a specific OCR model:**

The default image includes the `mistral-small3.1` model (~7-8GB). To build with a different model:

```bash
# Build with llava for faster, lighter processing
docker build --build-arg OCR_MODEL=llava -t remarkable-sync:llava .

# Build without pre-downloading models (smallest image, download on first run)
docker build --build-arg OCR_MODEL=none -t remarkable-sync:minimal .

# Build with llava:13b for higher accuracy
docker build --build-arg OCR_MODEL=llava:13b -t remarkable-sync:llava13b .
```

**Using host-mounted Ollama models:**

If you already have Ollama models downloaded on your host, you can mount them:

```bash
# First, download the model on your host
ollama pull mistral-small3.1

# Then mount your host's Ollama models directory when running the container
docker run --rm \
  -v $HOME/.rmapi:/home/remarkable/.rmapi \
  -v $PWD/output:/output \
  -v $HOME/.ollama/models:/home/remarkable/.ollama/models:ro \
  -e OCR_MODEL=mistral-small3.1 \
  ghcr.io/platinummonkey/remarkable-sync:latest sync --output /output
```

This approach:
- Avoids downloading models inside the container
- Allows sharing models across multiple containers
- Reduces image size and startup time

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

This will guide you through the authentication process:
1. Visit https://my.remarkable.com/device/desktop/connect
2. Copy the one-time code displayed
3. Enter the code when prompted
4. The device token will be saved automatically

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
3. **OCR**: Processes each page with Ollama vision models to extract handwritten text (optional)
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

# Ollama configuration for OCR
ollama:
  endpoint: http://localhost:11434  # Ollama API endpoint (default)
  model: mistral-small3.1            # Vision model for OCR (default)
  # Recommended models:
  # - mistral-small3.1: Better multilingual and complex handwriting (~7-8GB) [DEFAULT]
  # - llava: Faster, good for most handwriting (~4GB)
  # - llava:13b: Higher accuracy, slower (~7GB)
  temperature: 0.0                   # Lower = more deterministic (default: 0.0)
  timeout: 30s                       # Request timeout
  max-retries: 3                     # Retry attempts for failed requests

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

The build system is simple and doesn't require any system dependencies since OCR now uses Ollama's HTTP API instead of native libraries.

**Prerequisites:**
- Go 1.21+
- (Optional) Ollama for OCR testing

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
- CGO is disabled (`CGO_ENABLED=0`), enabling simple cross-compilation
- Built binaries are in `dist/remarkable-sync_<os>_<arch>/` directory
- No system dependencies required - builds work on any platform

**Installing goreleaser:**
```bash
# macOS:
brew install goreleaser

# Linux:
# See https://goreleaser.com/install/
```

### Running tests

```bash
make test           # Run all tests (uses mock Ollama server)
make test-coverage  # Run tests with coverage report
```

**Note:** OCR tests use a mock HTTP server and don't require Ollama to be installed or running.

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

**Build errors**
```bash
# Ensure Go modules are downloaded
go mod download
go mod tidy

# Try building again
make build
```

**"CGO errors" or "undefined symbols"**
- This project doesn't use CGO - if you see CGO errors, ensure you're using the latest version
- Clean build cache: `go clean -cache && make clean && make build`

### Authentication Issues

**Authentication fails or "token not found"**
- Ensure your reMarkable has cloud sync enabled in Settings → Storage
- Get a new one-time code from https://my.remarkable.com/device/desktop/connect
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
- Try a different Ollama vision model for better accuracy:
  - `mistral-small3.1` - Better for multilingual and complex handwriting
  - `llava:13b` - Larger LLaVA model with improved accuracy
- Ensure Ollama is running: `ollama list`
- Check that the model is downloaded: `ollama pull mistral-small3.1`
- Verify Ollama endpoint is accessible: `curl http://localhost:11434/api/tags`

**OCR is slow**
- OCR takes 2-5 seconds per page (this is normal with vision models)
- The default mistral-small3.1 model is larger and more accurate but slower
- Use a smaller/faster model: `llava` (fastest) for basic handwriting
- Consider external Ollama on a more powerful machine
- Increase CPU allocation if running in Docker

**OCR crashes or fails**
- Verify Ollama is running: `ollama list`
- Check system resources (4GB+ RAM recommended)
- Check Ollama logs for errors
- Try a different model: `ollama pull mistral`
- Skip OCR for testing: `--no-ocr`

**"Connection refused" or "Ollama not found"**
- Start Ollama service: `ollama serve` (or background: `ollama serve &`)
- Check Ollama is listening: `curl http://localhost:11434/api/tags`
- Verify endpoint in config matches Ollama address
- For Docker: ensure Ollama container is running and accessible

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
- [Ollama](https://ollama.ai/) - Local AI model runtime for OCR

## Related Projects

- [rmapi](https://github.com/ddvk/rmapi) - Command-line tool for reMarkable cloud
- [remarkable-fs](https://github.com/nick8325/remarkable-fs) - FUSE filesystem for reMarkable
