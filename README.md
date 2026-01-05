# Legible

Sync documents from your reMarkable tablet and add OCR text layers to make handwritten notes searchable.

## Features

- 📥 Download documents from reMarkable cloud
- 📄 Convert .rmdoc files to standard PDF format
- 🏷️ Optional label-based filtering
- 🔍 Multiple OCR provider support:
  - Ollama (local, free) - Privacy-focused offline processing
  - OpenAI GPT-4 Vision (cloud) - High accuracy recognition
  - Anthropic Claude (cloud) - Excellent handwriting understanding
  - Google Gemini (cloud) - Fast processing with free tier
- 🤖 Superior handwriting recognition using AI vision models
- 📝 Add hidden searchable text layer to PDFs
- 🔄 Incremental sync with state tracking
- ⚙️ Daemon mode for continuous background sync
- 🏥 Health check endpoints for monitoring
- 📋 Comprehensive logging and error handling

## Prerequisites

- Go 1.21 or higher
- OCR Provider (optional, for searchable PDFs):
  - **Ollama** (local, free) - Recommended for privacy and offline use
  - **OpenAI GPT-4 Vision** (cloud, paid) - High accuracy, requires API key
  - **Anthropic Claude** (cloud, paid) - Excellent handwriting recognition, requires API key
  - **Google Gemini** (cloud, paid/free tier) - Fast processing, requires API key
- reMarkable tablet with cloud sync enabled

### Setting up OCR Providers

#### Option 1: Ollama (Local, Recommended)

Ollama runs locally on your machine and is completely free. Best for privacy and offline use.

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

#### Option 2: OpenAI GPT-4 Vision (Cloud)

Best for high accuracy and when you already have OpenAI API access.

```bash
# Set your API key
export OPENAI_API_KEY=sk-...

# Configure in ~/.legible.yaml
llm:
  provider: openai
  model: gpt-4o  # or gpt-4o-mini, gpt-4-turbo
  temperature: 0.0
  max-retries: 3
```

**Pricing:** Pay per API call based on OpenAI's pricing. See [OpenAI Pricing](https://openai.com/api/pricing/).

#### Option 3: Anthropic Claude (Cloud)

Excellent handwriting recognition and understanding of context.

```bash
# Set your API key
export ANTHROPIC_API_KEY=sk-ant-...

# Configure in ~/.legible.yaml
llm:
  provider: anthropic
  model: claude-3-5-sonnet-20241022  # or claude-3-opus-20240229
  temperature: 0.0
  max-retries: 3
```

**Pricing:** Pay per API call based on Anthropic's pricing. See [Anthropic Pricing](https://www.anthropic.com/pricing).

#### Option 4: Google Gemini (Cloud)

Fast processing with free tier available for testing.

```bash
# Set your API key
export GOOGLE_API_KEY=...

# Configure in ~/.legible.yaml
llm:
  provider: google
  model: gemini-1.5-pro  # or gemini-1.5-flash
  temperature: 0.0
  max-retries: 3
```

**Pricing:** Free tier available with rate limits. See [Google AI Pricing](https://ai.google.dev/pricing).

## Installation

### Using Docker (Recommended)

Pull the official container image from GitHub Container Registry:

```bash
# Pull latest version
docker pull ghcr.io/platinummonkey/legible:latest

# Or specific version
docker pull ghcr.io/platinummonkey/legible:v0.1.0
```

Multi-platform images available:
- `linux/amd64` (Intel/AMD 64-bit)
- `linux/arm64` (ARM 64-bit, including Apple Silicon)

**Building with a specific OCR model:**

The default image includes the `mistral-small3.1` model (~7-8GB). To build with a different model:

```bash
# Build with llava for faster, lighter processing
docker build --build-arg OCR_MODEL=llava -t legible:llava .

# Build without pre-downloading models (smallest image, download on first run)
docker build --build-arg OCR_MODEL=none -t legible:minimal .

# Build with llava:13b for higher accuracy
docker build --build-arg OCR_MODEL=llava:13b -t legible:llava13b .
```

**Using host-mounted Ollama models:**

If you already have Ollama models downloaded on your host, you can mount them:

```bash
# First, download the model on your host
ollama pull mistral-small3.1

# Then mount your host's Ollama models directory when running the container
docker run --rm \
  -v $HOME/.rmapi:/home/legible/.rmapi \
  -v $PWD/output:/output \
  -v $HOME/.ollama/models:/home/legible/.ollama/models:ro \
  -e OCR_MODEL=mistral-small3.1 \
  ghcr.io/platinummonkey/legible:latest sync --output /output
```

This approach:
- Avoids downloading models inside the container
- Allows sharing models across multiple containers
- Reduces image size and startup time

### Using Homebrew (macOS/Linux)

```bash
brew install platinummonkey/tap/legible
```

This installs the latest release binary for your platform.

### Using Go Install

```bash
go install github.com/platinummonkey/legible/cmd/legible@latest
```

### Build from Source

```bash
git clone https://github.com/platinummonkey/legible.git
cd legible
make build-local
```

The binary will be available at `dist/legible_*/legible`.

## Configuration

First time setup requires authenticating with your reMarkable account:

```bash
legible auth
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
legible sync
```

**Sync documents with specific labels:**

```bash
legible sync --labels "work,personal"
```

**Specify output directory:**

```bash
legible sync --output ./my-remarkable-docs
```

**Run in daemon mode:**

For continuous background synchronization:

```bash
legible daemon --interval 10m --health-addr :8080
```

### Using Docker

**First-time authentication:**

```bash
docker run --rm -it \
  -v $HOME/.rmapi:/home/legible/.rmapi \
  ghcr.io/platinummonkey/legible:latest auth
```

**Sync documents:**

```bash
docker run --rm \
  -v $HOME/.rmapi:/home/legible/.rmapi \
  -v $PWD/output:/output \
  ghcr.io/platinummonkey/legible:latest sync --output /output
```

**Sync with labels:**

```bash
docker run --rm \
  -v $HOME/.rmapi:/home/legible/.rmapi \
  -v $PWD/output:/output \
  ghcr.io/platinummonkey/legible:latest sync \
    --output /output \
    --labels "work,personal"
```

**Run in daemon mode:**

```bash
docker run -d \
  --name legible \
  -v $HOME/.rmapi:/home/legible/.rmapi \
  -v $PWD/output:/output \
  ghcr.io/platinummonkey/legible:latest daemon \
    --interval 30m \
    --output /output
```

**Docker Compose:**

Create a `docker-compose.yml`:

```yaml
version: '3.8'

services:
  legible:
    image: ghcr.io/platinummonkey/legible:latest
    container_name: legible
    restart: unless-stopped
    volumes:
      - ./credentials:/home/legible/.rmapi
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
legible sync [flags]

Flags:
  --output string      Output directory (default: ~/Legible)
  --labels strings     Filter by labels (comma-separated)
  --no-ocr            Skip OCR processing
  --force             Force re-sync all documents
  --log-level string  Log level: debug, info, warn, error (default: info)
  --config string     Config file (default: ~/.legible.yaml)
```

**Daemon command:**
```bash
legible daemon [flags]

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
legible auth      # Authenticate with reMarkable API
legible version   # Display version information
legible help      # Display help
```

## Common Use Cases

### Quick Start: First Time Sync

```bash
# 1. Authenticate with reMarkable cloud
legible auth

# 2. Sync all documents (with OCR)
legible sync

# 3. Check your documents
ls ~/legible/
```

### Selective Sync by Label

Organize your reMarkable documents with labels, then sync only what you need:

```bash
# Sync only work documents
legible sync --labels work

# Sync multiple label categories
legible sync --labels "work,personal,important"
```

### Background Daemon Mode

Run continuous sync in the background:

```bash
# Start daemon with 15-minute interval
legible daemon --interval 15m --log-level info

# With health check endpoint (for monitoring)
legible daemon --interval 10m --health-addr :8080

# Check health status
curl http://localhost:8080/health
```

### Development Workflow

For developers who want searchable notes without OCR overhead:

```bash
# Quick sync without OCR (faster)
legible sync --no-ocr

# Debug sync issues
legible sync --log-level debug --force
```

### Custom Output Organization

```bash
# Sync to specific directory
legible sync --output ~/Dropbox/ReMarkable

# Use config file for consistent setup
legible --config ~/.legible-work.yaml sync
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

## Configuration

Legible uses a flexible configuration system with multiple sources that follow this precedence:

**CLI flags > Environment variables > Config file > Built-in defaults**

### Configuration File

Create a `~/.legible.yaml` file for persistent settings. Here's a complete example:

```yaml
# Core Settings
output-dir: ~/Documents/remarkable    # Where to save synced PDFs
labels:                               # Filter by reMarkable labels (empty = all)
  - work
  - important
ocr-enabled: true                     # Add searchable text layer to PDFs
ocr-languages: eng                    # OCR language(s): eng, fra, deu, etc.

# LLM Configuration for OCR
llm:
  provider: ollama                    # ollama, openai, anthropic, google
  model: llava                        # Provider-specific model name
  endpoint: http://localhost:11434    # Only for Ollama
  temperature: 0.0                    # 0.0 = deterministic (recommended for OCR)
  max-retries: 3                      # API retry attempts

# Sync and State
sync-interval: 10m                    # Daemon sync frequency (0 = run once)
state-file: ~/.legible-state.json     # Tracks synced documents
daemon-mode: false                    # Enable continuous sync

# Logging
log-level: info                       # debug, info, warn, error
```

See [examples/config.yaml](examples/config.yaml) for detailed documentation of all options.

### Configuration Options Reference

#### Core Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `output-dir` | string | `~/legible` | Output directory for synced PDF files |
| `labels` | list | `[]` | Filter documents by reMarkable labels (empty = sync all) |
| `ocr-enabled` | bool | `true` | Enable OCR text layer generation |
| `ocr-languages` | string | `eng` | OCR language codes (e.g., `eng+fra`) |

#### LLM Configuration

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `llm.provider` | string | `ollama` | LLM provider: `ollama`, `openai`, `anthropic`, `google` |
| `llm.model` | string | `llava` | Model name (provider-specific, see below) |
| `llm.endpoint` | string | `http://localhost:11434` | API endpoint (Ollama only) |
| `llm.temperature` | float | `0.0` | Generation temperature (0.0-2.0, lower = more deterministic) |
| `llm.max-retries` | int | `3` | Maximum API retry attempts |

**Supported Models by Provider:**

**Ollama** (local, free):
- `llava` - Fast, good for most handwriting (~4GB) [DEFAULT]
- `mistral-small3.1` - Better multilingual and complex handwriting (~7-8GB)
- `llava:13b` - Higher accuracy, slower (~7GB)
- `llava:34b` - Best accuracy, very slow (~20GB)

**OpenAI** (cloud, paid):
- `gpt-4o` - Latest, high accuracy
- `gpt-4o-mini` - Faster, cost-effective
- `gpt-4-turbo` - Previous generation

**Anthropic** (cloud, paid):
- `claude-3-5-sonnet-20241022` - Latest Sonnet, excellent handwriting
- `claude-3-opus-20240229` - Highest accuracy, slower
- `claude-3-haiku-20240307` - Fastest, cost-effective

**Google** (cloud, free tier available):
- `gemini-1.5-pro` - High accuracy
- `gemini-1.5-flash` - Faster, free tier

#### Sync and State Management

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `sync-interval` | duration | `5m` | Sync interval for daemon mode (e.g., `5m`, `1h`) |
| `state-file` | string | `~/.legible-state.json` | Path to sync state file |
| `daemon-mode` | bool | `false` | Enable continuous sync operation |

#### Logging and Advanced

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `log-level` | string | `info` | Logging level: `debug`, `info`, `warn`, `error` |
| `api-token` | string | `""` | reMarkable API token path (auto-detected if empty) |
| `tesseract-path` | string | `""` | Tesseract executable path (legacy, not used) |

### Environment Variables

All configuration options can be set via environment variables using the `LEGIBLE_` prefix with uppercase names:

```bash
# Core settings
export LEGIBLE_OUTPUT_DIR=~/Documents/remarkable
export LEGIBLE_LABELS=work,personal
export LEGIBLE_OCR_ENABLED=true
export LEGIBLE_OCR_LANGUAGES=eng

# LLM configuration
export LEGIBLE_LLM_PROVIDER=openai
export LEGIBLE_LLM_MODEL=gpt-4o-mini
export LEGIBLE_LLM_ENDPOINT=http://localhost:11434
export LEGIBLE_LLM_TEMPERATURE=0.0
export LEGIBLE_LLM_MAX_RETRIES=3

# API keys (required for cloud providers)
export OPENAI_API_KEY=sk-...
export ANTHROPIC_API_KEY=sk-ant-...
export GOOGLE_API_KEY=...

# Sync and logging
export LEGIBLE_SYNC_INTERVAL=10m
export LEGIBLE_STATE_FILE=~/.legible/state.json
export LEGIBLE_LOG_LEVEL=debug
```

**Note:** API keys for cloud providers are always read from their standard environment variables (`OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `GOOGLE_API_KEY`) and cannot be set in the config file for security reasons.

### Configuration Examples

**Minimal setup (all defaults):**
```yaml
output-dir: ~/legible
ocr-enabled: true
```

**Work documents with cloud OCR:**
```yaml
output-dir: ~/work-documents
labels: [work, meetings]
llm:
  provider: openai
  model: gpt-4o-mini
```

**High-accuracy handwriting recognition:**
```yaml
output-dir: ~/documents
ocr-enabled: true
llm:
  provider: anthropic
  model: claude-3-5-sonnet-20241022
```

**Fast sync without OCR:**
```yaml
output-dir: ~/remarkable-backup
ocr-enabled: false
sync-interval: 5m
```

**Background daemon with monitoring:**
```yaml
output-dir: ~/legible
daemon-mode: true
sync-interval: 10m
log-level: info
llm:
  provider: ollama
  model: mistral-small3.1
```

**Multi-language OCR:**
```yaml
output-dir: ~/legible
ocr-enabled: true
ocr-languages: eng+fra+deu
llm:
  provider: google
  model: gemini-1.5-flash
```

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
- Built binaries are in `dist/legible_<os>_<arch>/` directory
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
  rm -rf ~/.legible
  legible auth
  ```

**"Device not registered" error**
- Remar notable may have unlinked the device
- Re-authenticate with `legible auth`
- Contact reMarkable support if issues persist

### Sync Issues

**Documents not syncing**
- Check you have cloud sync enabled on your reMarkable
- Verify documents have been uploaded to cloud
- Force re-sync: `legible sync --force`
- Check logs: `legible sync --log-level debug`

**Only some documents sync**
- Check if you're using label filters: `--labels`
- Verify document labels in reMarkable app
- Try syncing without filters: `legible sync`

**Sync is very slow**
- OCR processing is CPU-intensive
- Try without OCR first: `legible sync --no-ocr`
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
- Check logs: `legible daemon --log-level debug`
- Verify port isn't in use: `lsof -i :8080` (if using --health-addr)
- Check file permissions for PID file location
- Ensure output directory is writable

**Daemon stops unexpectedly**
- Check system logs: `journalctl -u legible` (if using systemd)
- Verify sufficient disk space
- Check for OOM (out of memory) errors in system logs

**Health check endpoint not responding**
- Verify port is correct: `curl http://localhost:8080/health`
- Check firewall settings
- Ensure daemon is running: `ps aux | grep legible`

### Other Issues

**Large output files**
- PDFs with OCR text layers are larger than originals
- Disable OCR if size is a concern: `--no-ocr`
- OCR'd PDFs are searchable but larger (~2-3x original size)

**Permission denied errors**
- Check output directory permissions
- Verify state file location is writable
- Run with appropriate user permissions

For more help, see [FAQ.md](FAQ.md) or [open an issue](https://github.com/platinummonkey/legible/issues).

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
curl -LO https://github.com/platinummonkey/legible/releases/download/v1.0.0/legible_1.0.0_linux_amd64.sbom.spdx.json
```

**Extracting Container SBOMs:**

Container images include embedded SBOMs:

```bash
# Pull image
docker pull ghcr.io/platinummonkey/legible:latest

# Extract SBOM
docker buildx imagetools inspect ghcr.io/platinummonkey/legible:latest --format "{{ json .SBOM }}"
```

**Vulnerability Scanning:**

Use the SBOM to scan for known vulnerabilities:

```bash
# Install grype (vulnerability scanner)
brew install anchore/grype/grype

# Scan using SBOM
grype sbom:./legible_1.0.0_linux_amd64.sbom.spdx.json

# Or scan the container image directly
grype ghcr.io/platinummonkey/legible:latest
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
