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

```bash
go install github.com/platinummonkey/remarkable-sync/cmd/remarkable-sync@latest
```

Or build from source:

```bash
git clone https://github.com/platinummonkey/remarkable-sync.git
cd remarkable-sync
make build
```

The binary will be available at `bin/remarkable-sync`.

## Configuration

First time setup requires authenticating with your reMarkable account:

```bash
remarkable-sync auth
```

This will prompt you for a one-time code from https://my.remarkable.com/device/connect/desktop

## Usage

### Sync all documents

```bash
remarkable-sync sync
```

### Sync documents with specific labels

```bash
remarkable-sync sync --labels "work,personal"
```

### Specify output directory

```bash
remarkable-sync sync --output ./my-remarkable-docs
```

### Run in daemon mode

For continuous background synchronization:

```bash
remarkable-sync daemon --interval 10m --health-addr :8080
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

## How It Works

1. **Authentication**: Connects to reMarkable cloud using your credentials
2. **Download**: Fetches `.rmdoc` files and renders them as PDFs
3. **OCR**: Processes each page with Tesseract to extract text
4. **Enhancement**: Adds invisible text layer to PDF at correct positions
5. **Save**: Outputs searchable PDF files to the specified directory

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
output_dir: ~/Documents/remarkable
labels:
  - work
  - important
ocr_enabled: true
languages:
  - eng
  - spa  # Add additional language support
```

## Development

### Building

```bash
make build          # Build binary to bin/remarkable-sync
make build-all      # Build for multiple platforms
make install        # Install to $GOPATH/bin
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

**"Tesseract not found" or build errors**
- Ensure Tesseract is installed and in your PATH
- Verify installation: `tesseract --version`
- Install development libraries:
  - macOS: `brew install tesseract leptonica`
  - Ubuntu: `sudo apt-get install tesseract-ocr libtesseract-dev libleptonica-dev`

**Authentication fails**
- Ensure your reMarkable has cloud sync enabled
- Clear credentials and re-authenticate: `rm ~/.rmapi && remarkable-sync auth`

**OCR quality is poor**
- Check Tesseract language data is installed
- Use `--no-ocr` flag to skip OCR if not needed
- Install additional language data: `brew install tesseract-lang` (macOS)

**Daemon won't start**
- Check logs: `remarkable-sync daemon --log-level debug`
- Verify port isn't in use (if using --health-addr)
- Check file permissions for PID file location

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Submit a pull request

## License

MIT License - See LICENSE file for details

## Acknowledgments

- [rmapi](https://github.com/ddvk/rmapi) - reMarkable cloud API client
- [Tesseract OCR](https://github.com/tesseract-ocr/tesseract) - OCR engine

## Related Projects

- [rmapi](https://github.com/ddvk/rmapi) - Command-line tool for reMarkable cloud
- [remarkable-fs](https://github.com/nick8325/remarkable-fs) - FUSE filesystem for reMarkable
