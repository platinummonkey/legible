# PDF Enhancer

Package pdfenhancer provides utilities for reading, validating, and enhancing PDF files using the pdfcpu library.

## Library Choice: pdfcpu

This package uses [pdfcpu](https://github.com/pdfcpu/pdfcpu) v0.11.1 as the PDF manipulation library.

### Rationale

**Why pdfcpu?**

1. **Pure Go**: No CGO dependencies, making it cross-platform compatible and easy to build
2. **Open Source**: Apache 2.0 license, well-maintained with active development
3. **Comprehensive API**: Provides low-level access to PDF internals needed for text layer addition
4. **PDF Compliance**: Supports PDF versions 1.0 through 2.0
5. **Feature-Rich**: Includes validation, optimization, merging, splitting, and more
6. **Production Ready**: Actively used in production environments

**Alternatives Considered:**

- **gopdf**: Limited to PDF creation, doesn't support reading/modification
- **gofpdf**: Primarily for PDF generation, not manipulation of existing files
- **unidoc/unipdf**: Commercial license required for production use
- **CGO-based libraries (poppler, mupdf)**: Cross-platform compilation challenges

### Current Implementation

The current implementation provides:

- ✅ PDF validation and reading
- ✅ Page count extraction
- ✅ PDF optimization
- ✅ PDF merging and splitting
- ✅ Page dimension extraction
- ✅ **Text layer addition**: Fully implemented with invisible OCR text overlay

### Text Layer Addition - Implementation Details

The package adds an invisible OCR text layer to PDFs using low-level PDF content stream manipulation. This makes PDFs searchable while preserving their original visual appearance.

**Key Features:**

1. **PDF Content Stream Creation**
   - Creates new content streams with proper PDF text operators
   - Uses BT/ET (Begin/End Text) to define text objects
   - Sets font with Tf operator (Helvetica 10pt)
   - Sets text rendering mode to invisible with `Tr 3` (no fill, no stroke)
   - Positions text using Tm operator (text matrix)
   - Renders text with Tj operator (show text string)

2. **Coordinate System Conversion**
   - Automatically converts OCR coordinates (top-left origin) to PDF coordinates (bottom-left origin)
   - OCR: (0,0) is top-left, Y increases downward
   - PDF: (0,0) is bottom-left, Y increases upward
   - Conversion formula: `PDF_Y = PageHeight - OCR_Y - OCR_Height`

3. **Text Encoding and Escaping**
   - Properly escapes special characters in PDF strings
   - Handles parentheses, backslashes, newlines, tabs, carriage returns
   - Uses standard Helvetica font (no embedding needed)
   - Compatible with PDF string encoding requirements

4. **Content Stream Integration**
   - Appends new content streams to existing page contents
   - Handles both single content stream and content array cases
   - Preserves existing page content and appearance
   - Uses proper PDF indirect reference management

**Implementation Methods:**
- `AddTextLayer()`: Main entry point, processes all pages (pdf.go:72-108)
- `addTextToPage()`: Adds text to a single page (pdf.go:113-152)
- `createTextContentStream()`: Generates PDF content stream with text operators (pdf.go:154-201)
- `escapePDFString()`: Escapes special characters for PDF strings (pdf.go:203-213)
- `appendContentStream()`: Adds content stream to page dictionary (pdf.go:215-258)

### Usage Example

```go
import "github.com/platinummonkey/legible/internal/pdfenhancer"

// Create enhancer
enhancer := pdfenhancer.New(&pdfenhancer.Config{})

// Validate PDF
if err := enhancer.ValidatePDF("input.pdf"); err != nil {
    log.Fatal(err)
}

// Get page count
pageCount, err := enhancer.GetPageCount("input.pdf")

// Optimize PDF
err = enhancer.OptimizePDF("input.pdf", "output.pdf")

// Add text layer (when OCR data is available)
ocrResults := ocr.NewDocumentOCR("doc-id", "eng")
// ... populate OCR results ...
err = enhancer.AddTextLayer("input.pdf", "output.pdf", ocrResults)
```

### Testing

The package includes comprehensive tests with high coverage:

**Test Coverage:**
- PDF validation (valid and invalid files)
- Page counting and extraction
- PDF information retrieval
- Optimization operations
- Merging and splitting
- Text layer addition with OCR data
- Content stream generation and text positioning
- Coordinate system conversion
- Special character escaping
- Empty OCR handling
- Multiple words positioning

**Test Approach:**
- Test PDFs are generated programmatically using minimal valid PDF syntax
- Mock OCR data used to test text layer addition without Tesseract dependency
- Edge cases tested: empty text, special characters, multiple words, no OCR data
- Integration tests verify generated PDFs are valid and can be read

### Coordinate System Reference

Use `CompareCoordinateSystems(pageHeight)` to get detailed information about the coordinate system differences between PDF and OCR coordinate spaces.

## Future Enhancements

1. **Advanced Text Rendering**
   - Support for different text rendering modes (visible, invisible, outline)
   - Text sizing to exactly match bounding box dimensions
   - Rotated text support for angled words

2. **Font Handling**
   - Custom font embedding for better Unicode support
   - Font subsetting for reduced file size
   - Multi-language font support

3. **Performance Optimizations**
   - Batch processing for multiple PDFs
   - Parallel page processing
   - Streaming for large files
   - Memory-efficient content stream building

4. **Quality Improvements**
   - Confidence-based text filtering (only add high-confidence words)
   - Text layer validation and verification
   - OCR accuracy metrics in output

5. **Monitoring and Progress**
   - Progress callbacks for long operations
   - Detailed logging of text addition statistics
   - Performance profiling and metrics
