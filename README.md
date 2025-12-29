# reMarkable Sync

Sync documents from your reMarkable tablet and add OCR text layers to make handwritten notes searchable.

## Features

- 📥 Download documents from reMarkable cloud
- 🏷️ Optional label-based filtering
- 🔍 OCR processing with Tesseract
- 📄 Add hidden searchable text layer to PDFs
- 🔄 Incremental sync support

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
go install github.com/platinummonkey/remarkable-sync@latest
```

Or build from source:

```bash
git clone https://github.com/platinummonkey/remarkable-sync.git
cd remarkable-sync
go build -o remarkable-sync
```

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

### Full command options

```bash
remarkable-sync sync [flags]

Flags:
  -h, --help              Help for sync
  -l, --labels strings    Filter by labels (comma-separated)
  -o, --output string     Output directory (default: "./remarkable-docs")
      --no-ocr           Skip OCR processing
      --force            Force re-sync all documents
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

### Running tests

```bash
go test ./...
```

### Building

```bash
go build -o remarkable-sync
```

## Project Structure

See [AGENTS.md](./AGENTS.md) for detailed architecture and design documentation.

## Troubleshooting

**"Tesseract not found"**
- Ensure Tesseract is installed and in your PATH
- Verify installation: `tesseract --version`

**Authentication fails**
- Ensure your reMarkable has cloud sync enabled
- Try re-authenticating: `remarkable-sync auth --reset`

**OCR quality is poor**
- Check Tesseract language data is installed
- Consider training Tesseract for your handwriting style

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
