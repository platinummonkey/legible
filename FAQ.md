# Frequently Asked Questions (FAQ)

## General Questions

### What is reMarkable Sync?

reMarkable Sync is a tool that synchronizes documents from your reMarkable tablet to your computer and optionally adds OCR (Optical Character Recognition) to make handwritten notes searchable.

### Do I need a reMarkable subscription?

You need reMarkable's free cloud sync feature enabled on your tablet. A reMarkable Connect subscription is not required, only the standard cloud sync that comes with every device.

### Is this an official reMarkable product?

No, this is an independent open-source project. It uses the reMarkable cloud API but is not affiliated with or endorsed by reMarkable AS.

### Does this work with reMarkable 1 and reMarkable 2?

Yes, it works with both reMarkable 1 and reMarkable 2 tablets, as long as cloud sync is enabled.

## Installation and Setup

### Do I need to install anything on my reMarkable tablet?

No! This tool runs entirely on your computer and syncs through the reMarkable cloud. No tablet modifications required.

### Can I use this without Tesseract?

Yes! OCR is optional. You can sync and convert documents without OCR by using the `--no-ocr` flag. However, without OCR, your handwritten notes won't be searchable.

### How do I get the authentication code?

1. Run `remarkable-sync auth`
2. Visit https://my.remarkable.com/device/browser/connect
3. Enter the code displayed on the website
4. The token will be saved automatically

### Where are my authentication credentials stored?

By default, credentials are stored in `~/.remarkable-sync/token.json`. This file contains your device token and should be kept secure.

## Usage Questions

### How often does the daemon sync?

By default, every 5 minutes. You can change this with `--interval`:
```bash
remarkable-sync daemon --interval 15m  # Every 15 minutes
remarkable-sync daemon --interval 1h   # Every hour
```

### Can I sync only specific documents?

Yes! Use labels in the reMarkable app to organize your documents, then sync specific labels:
```bash
remarkable-sync sync --labels "work,personal"
```

### Does syncing modify my reMarkable documents?

No, syncing is read-only. It downloads copies of your documents but never modifies or uploads anything back to your reMarkable.

### What format are the synced documents?

Documents are converted to PDF format. OCR-enabled PDFs have an invisible text layer that makes them searchable while preserving the original appearance.

### Can I sync to Dropbox/Google Drive/etc?

Yes! Just specify your cloud storage folder as the output directory:
```bash
remarkable-sync sync --output ~/Dropbox/ReMarkable
```

### How much disk space do I need?

Typically 2-3x the size of your reMarkable documents. OCR adds a text layer which increases file size. A 1MB handwritten notebook might become a 2-3MB searchable PDF.

## Technical Questions

### How does OCR work?

1. The PDF pages are rendered as images
2. Tesseract OCR analyzes the images to extract text
3. The extracted text with positional information is added as an invisible layer to the PDF
4. The result is a searchable PDF that looks identical to the original

### Why is syncing slow?

OCR is CPU-intensive and processes each page individually. Factors affecting speed:
- Document size (more pages = longer processing)
- System resources (CPU speed, available RAM)
- OCR complexity (handwriting is slower than typed text)

Typical speeds: 2-5 seconds per page on a modern laptop.

### Can I speed up syncing?

Yes:
- Skip OCR: `--no-ocr` (much faster, but not searchable)
- Use incremental sync: Only new/modified documents are processed
- Reduce sync frequency in daemon mode
- Run on a faster machine with more CPU cores

### What languages does OCR support?

Tesseract supports 100+ languages. Common languages:
- English (eng) - installed by default
- Spanish (spa)
- French (fra)
- German (deu)
- Chinese Simplified (chi_sim)
- Japanese (jpn)

Install additional languages:
```bash
# macOS
brew install tesseract-lang

# Ubuntu
sudo apt-get install tesseract-ocr-spa tesseract-ocr-fra

# Configure
remarkable-sync sync --config ~/.remarkable-sync.yaml
# In config: ocr-languages: eng+spa+fra
```

### Does this work offline?

No, reMarkable Sync requires internet access to sync with the reMarkable cloud. However, once documents are synced, you can view them offline.

## Privacy and Security

### Is my data secure?

- Your authentication token is stored locally with restricted permissions (0600)
- Communication with reMarkable cloud uses HTTPS
- No data is sent to third parties
- Documents are stored locally on your computer

### What data does this tool collect?

None. This tool does not collect, transmit, or store any analytics or telemetry. All data stays on your computer.

### Can I audit the code?

Yes! This is open-source software. You can review the code on GitHub: https://github.com/platinummonkey/remarkable-sync

## Troubleshooting

### "Authentication failed" error

**Solution:**
1. Ensure cloud sync is enabled on your reMarkable (Settings → Storage → Connect)
2. Get a fresh authentication code
3. Clear old credentials: `rm -rf ~/.remarkable-sync`
4. Re-authenticate: `remarkable-sync auth`

### OCR produces garbage text

**Possible causes:**
- Language mismatch: Ensure Tesseract has the correct language pack
- Poor handwriting: OCR works best with clear, legible writing
- Scanned images: Original PDFs with scanned images may not OCR well

**Solutions:**
- Specify correct language: `ocr-languages: eng+spa`
- Try without OCR if quality is unacceptable
- Use reMarkable's built-in text recognition for better handwriting recognition

### Daemon keeps stopping

**Common causes:**
1. **Out of memory**: OCR is memory-intensive
   - Solution: Reduce sync frequency or skip OCR
2. **Permission errors**: Can't write to output directory
   - Solution: Check directory permissions
3. **Disk full**: No space for new documents
   - Solution: Free up disk space

**Check logs:**
```bash
remarkable-sync daemon --log-level debug
```

### Syncing takes forever on first run

This is normal! The first sync processes all your documents. Subsequent syncs are much faster as only new/changed documents are processed.

**Estimates:**
- 10 documents: 30-60 seconds
- 50 documents: 3-5 minutes
- 100+ documents: 10-20 minutes

Use `--no-ocr` for a much faster first sync, then enable OCR later.

### "document not found" error

**Causes:**
- Document was deleted from reMarkable cloud
- Sync state is out of date
- Document hasn't synced to cloud yet

**Solutions:**
- Force re-sync: `remarkable-sync sync --force`
- Wait for cloud sync to complete on your tablet
- Check document still exists in reMarkable app

## Advanced Usage

### Can I run multiple instances?

Yes, but use different output directories and state files:
```bash
# Instance 1: Work documents
remarkable-sync daemon --output ~/work-remarkable --labels work

# Instance 2: Personal documents
remarkable-sync daemon --output ~/personal-remarkable --labels personal
```

### How do I run as a system service?

See [examples/systemd/remarkable-sync.service](examples/systemd/remarkable-sync.service) for a systemd service file template.

### Can I customize the output filenames?

Currently, filenames match your reMarkable document names. Customization would require forking the project.

### Is there an API or library I can use?

The internal packages in `internal/` are available for use, but they are not considered a stable public API and may change between versions.

### Can I contribute features?

Yes! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on contributing.

## Comparison with Other Tools

### How is this different from rmapi?

[rmapi](https://github.com/ddvk/rmapi) is a lower-level tool for accessing the reMarkable cloud. reMarkable Sync builds on rmapi to provide:
- Automatic PDF conversion
- OCR text layer addition
- Daemon mode for continuous sync
- State tracking for incremental updates

### Why not use the reMarkable desktop app?

The official reMarkable desktop app is great, but doesn't offer:
- OCR for searchable text
- Automated background sync
- Bulk export
- Label-based filtering
- Command-line interface for automation

## Still Have Questions?

- **Check the documentation**: [README.md](README.md) has detailed usage information
- **Search existing issues**: https://github.com/platinummonkey/remarkable-sync/issues
- **Open a new issue**: If you found a bug or have a feature request
- **Start a discussion**: For general questions or ideas

## Useful Links

- [GitHub Repository](https://github.com/platinummonkey/remarkable-sync)
- [reMarkable Official Site](https://remarkable.com)
- [Tesseract OCR Documentation](https://github.com/tesseract-ocr/tesseract)
- [rmapi Project](https://github.com/ddvk/rmapi)
