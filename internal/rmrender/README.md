# rmrender

Package rmrender provides rendering of reMarkable .rm binary files to PDF format.

## Overview

This package handles parsing and rendering of reMarkable .lines files (`.rm` format) which contain stroke data from the reMarkable tablet. The goal is to convert these binary stroke files into vector graphics in PDF format.

## reMarkable .rm File Format

The .rm format is a binary format that stores drawing data. As of 2025, there are multiple versions:

- **Version 3**: Original format, documented and supported by ddvk/rmapi/encoding/rm
- **Version 6**: Current format used by reMarkable tablet (as of Dec 2025)

### Version 6 Format Structure

Our test files use version 6 format:

```
Header: "reMarkable .lines file, version=6          " (43 bytes)
+ Binary data containing:
  - Layers
  - Lines/Strokes
  - Points with coordinates, pressure, tilt
  - Brush types and colors
```

### Coordinate System

- reMarkable dimensions: **1404 x 1872 pixels** at 226 DPI
- Physical size: ~157.6 x 210.3 mm (close to A5)
- Origin: Top-left corner
- PDF coordinates need transformation

## Implementation Status

### ✅ Completed
- Package structure created
- Documentation established

### 🚧 In Progress
- Version 6 format parser
- Stroke rendering engine

### ⏳ Planned
- Layer support
- Template rendering
- All brush types (pen, pencil, marker, highlighter, eraser)
- Pressure sensitivity
- Color support

## Architecture

```
.rm file → Parser → Stroke Data → Renderer → PDF (pdfcpu)
```

### Components

1. **Parser** (`parser.go`)
   - Reads .rm binary format
   - Extracts layers, strokes, points
   - Handles version detection

2. **Renderer** (`renderer.go`)
   - Converts strokes to PDF vector graphics
   - Handles coordinate transformation
   - Applies brush styles

3. **Types** (`types.go`)
   - Data structures for strokes, layers, points
   - Brush definitions
   - Color constants

## Brush Types

The reMarkable supports various brush types that need different rendering:

- **Ballpoint**: Standard pen, solid lines
- **Marker**: Broad strokes, semi-transparent
- **Fineliner**: Thin, precise lines
- **Pencil**: Textured, pressure-sensitive
- **Mechanical Pencil**: Sharp, consistent
- **Brush**: Calligraphy-style, pressure-sensitive
- **Highlighter**: Wide, semi-transparent
- **Eraser**: Removes underlying strokes
- **Erase Section**: Removes entire stroke segments

## Usage Example

```go
import "github.com/platinummonkey/legible/internal/rmrender"

// Parse .rm file
data, err := os.ReadFile("page.rm")
if err != nil {
    log.Fatal(err)
}

parser := rmrender.NewParser()
document, err := parser.Parse(data)
if err != nil {
    log.Fatal(err)
}

// Render to PDF
renderer := rmrender.NewRenderer()
pdfData, err := renderer.RenderToPDF(document)
if err != nil {
    log.Fatal(err)
}

os.WriteFile("output.pdf", pdfData, 0644)
```

## References

### Format Specifications
- [reMarkable file format](https://github.com/YakBarber/remarkable_file_format) - Community documentation
- [Axel Huebl's blog post](https://plasma.ninja/blog/devices/remarkable/binary/format/2017/12/26/reMarkable-lines-file-format.html) - Original format analysis
- [rmscene](https://github.com/ricklupton/rmscene) - Python library for v6 format

### Related Projects
- [ddvk/rmapi](https://github.com/ddvk/rmapi) - Active fork of rmapi Go client with v3 encoder
- [go-remarkable2pdf](https://github.com/poundifdef/go-remarkable2pdf) - Alternative Go renderer
- [lines-are-beautiful](https://github.com/ax3l/lines-are-beautiful) - C++ implementation

## Development Notes

### Version 6 Parsing Challenge

The current `ddvk/rmapi/encoding/rm` package only supports version 3 format. Our test files use version 6. Options:

1. **Implement v6 parser from scratch** - Most control, significant work
2. **Adapt existing v3 parser** - May not support all v6 features
3. **Use external library** - Dependency management issues (deionizedoatmeal/rmapi has module path conflicts)
4. **Hybrid approach** - Use reference implementations for guidance, implement clean Go version

**Recommendation**: Implement v6 parser based on format specification and existing Python/C++ implementations as reference.

### Testing Strategy

Test files in `example/`:
- `Test.rmdoc` - Contains 2 pages with handwriting
- `.rm` files are version 6 format
- Use these for validation and visual comparison

## TODO

- [ ] Implement version 6 binary format parser
- [ ] Create stroke data structures
- [ ] Implement coordinate transformation
- [ ] Basic line rendering to PDF
- [ ] Support pressure-sensitive stroke width
- [ ] Implement brush types
- [ ] Layer support
- [ ] Color support
- [ ] Template rendering
- [ ] Performance optimization
- [ ] Comprehensive testing
- [ ] Documentation and examples

## Contributing

When implementing features:
1. Start with basic stroke rendering
2. Add brush types incrementally
3. Test with example files
4. Compare output with reMarkable app exports
5. Document any format discoveries
