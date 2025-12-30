# Converter Package

Package converter handles conversion of reMarkable .rmdoc files to PDF format.

## Overview

The converter package processes .rmdoc files (ZIP archives) from reMarkable tablets and converts them to standard PDF documents. It handles extraction, metadata parsing, and PDF generation.

## reMarkable File Format

### .rmdoc Structure

A .rmdoc file is a ZIP archive containing:

```
document-id.rmdoc (ZIP)
├── document-id.metadata     # Document metadata (JSON)
├── document-id.content      # Page and layer information (JSON)
└── document-id/             # Directory with page data
    ├── page-uuid-1.rm       # Page 1 rendering data (binary)
    ├── page-uuid-2.rm       # Page 2 rendering data (binary)
    └── ...
```

### Metadata Format

The `.metadata` JSON file contains:
- `visibleName`: Document title
- `createdTime`: Creation timestamp (milliseconds since epoch)
- `lastModified`: Last modification timestamp
- `lastOpened`: Last opened timestamp
- `parent`: Parent folder ID
- `type`: Document type (usually "DocumentType")

### Content Format

The `.content` JSON file contains:
- `fileType`: "notebook" or "pdf"
- `pageCount`: Number of pages
- `orientation`: "portrait" or "landscape"
- `formatVersion`: File format version (2 = v6 format)
- `cPages.pages[]`: Array of page information
  - `id`: Page UUID (matches .rm filename)
  - `template`: Page template name (e.g., "Blank", "Lined", "Grid")

### .rm File Format

`.rm` files contain binary vector rendering data in reMarkable's proprietary format. The v6 format (formatVersion=2) includes:
- Layers of drawing strokes
- Brush type, color, and size information
- Point coordinates with pressure and tilt data

**Format References:**
- [reMarkable file format spec](https://github.com/YakBarber/remarkable_file_format)
- [rmscene - v6 .rm file reader](https://github.com/ricklupton/rmscene)
- [Axel Huebl's format analysis](https://plasma.ninja/blog/devices/remarkable/binary/format/2017/12/26/reMarkable-lines-file-format.html)

## Implementation

### Current Status: Phase 1 (Extraction & Placeholder)

The converter currently implements:

✅ **ZIP Extraction**
- Extracts .rmdoc files to temporary directory
- Security: path traversal prevention
- Preserves file structure and permissions

✅ **Metadata Parsing**
- Reads and parses `.metadata` JSON
- Reads and parses `.content` JSON
- Extracts document title, timestamps, page count, orientation

✅ **Placeholder PDF Generation**
- Creates valid PDF with correct page count
- Uses reMarkable dimensions (1404x1872 pixels)
- Includes page number labels
- Valid PDF structure for testing

⚠️  **Stroke Rendering - Not Yet Implemented**
- TODO: Parse .rm binary files
- TODO: Render strokes to PDF using vector graphics
- TODO: Support layers, brush types, colors

### Design Decisions

**Why Placeholder PDF First?**
1. Validates extraction and metadata parsing logic
2. Enables end-to-end testing of the sync workflow
3. Allows other components to be built and tested
4. Provides immediate value (document list with correct metadata)

**Rendering Strategy (Future)**

Several approaches for full rendering:

1. **Use existing library**: `poundifdef/go-remarkable2pdf`
   - ✅ Simple integration
   - ❌ May not support v6 format
   - ❌ AGPL-3.0 license (source distribution required)

2. **Use rmapi encoding/rm + custom renderer**: `deionizedoatmeal/rmapi/encoding/rm`
   - ✅ Supports v6 format
   - ✅ Well-typed Go structs
   - ❌ Need to implement rendering logic
   - ✅ More control over output

3. **Hybrid approach**
   - Use `encoding/rm` for parsing
   - Use `pdfcpu` for PDF vector graphics
   - Custom stroke-to-PDF rendering

## Usage

```go
import "github.com/platinummonkey/remarkable-sync/internal/converter"

// Create converter
conv := converter.New(&converter.Config{})

// Convert .rmdoc to PDF
result, err := conv.ConvertRmdoc("input.rmdoc", "output.pdf")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Converted %d pages in %v\n", result.PageCount, result.Duration)
```

## Testing

The package includes comprehensive tests (82.2% coverage) using the real `example/Test.rmdoc` file:

- ZIP extraction and security
- Metadata and content parsing
- Placeholder PDF generation
- Timestamp parsing
- Error handling

Run tests:
```bash
go test -v ./internal/converter
```

## Dependencies

- **Standard library**: `archive/zip`, `encoding/json`
- **pdfcpu**: PDF manipulation (future rendering)
- **go-remarkable2pdf**: Reference for rendering (future, optional)

## Future Enhancements

### Phase 2: Full Stroke Rendering

1. **Parse .rm files**
   - Integrate `deionizedoatmeal/rmapi/encoding/rm`
   - Extract layers, lines, and points

2. **Render to PDF**
   - Convert stroke coordinates to PDF space
   - Render brush strokes as vector graphics
   - Support multiple layers
   - Implement brush types (pen, pencil, highlighter, eraser)

3. **Template Support**
   - Render page templates (lined, grid, dots)
   - Template library integration

4. **Advanced Features**
   - Layer visibility control
   - Color customization
   - Stroke thickness scaling
   - Pressure-sensitive rendering

### Phase 3: Performance & Quality

1. **Optimize rendering**
   - Batch processing
   - Parallel page conversion
   - Memory-efficient streaming

2. **Quality improvements**
   - Anti-aliasing
   - Brush stroke smoothing
   - High-DPI output

3. **Format support**
   - Annotated PDFs (overlay on existing PDF)
   - ePUB export
   - SVG export

## References

**Go Libraries:**
- [rorycl/rm2pdf](https://github.com/rorycl/rm2pdf) - Convert reMarkable notebooks to PDF
- [poundifdef/go-remarkable2pdf](https://github.com/poundifdef/go-remarkable2pdf) - Parse and render remarkable files
- [deionizedoatmeal/rmapi/encoding/rm](https://pkg.go.dev/github.com/deionizedoatmeal/rmapi/encoding/rm) - v3/v6 format parser
- [lobre/rm](https://github.com/lobre/rm) - reMarkable ZIP file parser

**Format Documentation:**
- [reMarkable file format specification](https://github.com/YakBarber/remarkable_file_format)
- [rmscene - v6 format reader](https://github.com/ricklupton/rmscene)
- [Axel Huebl's format analysis](https://plasma.ninja/blog/devices/remarkable/binary/format/2017/12/26/reMarkable-lines-file-format.html)

## License

Part of remarkable-sync project. See project LICENSE for details.
