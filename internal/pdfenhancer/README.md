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
- ⚠️  **Text layer addition (placeholder)**: The AddTextLayer method currently has placeholder logic

### Text Layer Addition - Future Work

Adding an invisible OCR text layer to PDFs requires low-level PDF content stream manipulation. The full implementation requires:

1. **PDF Content Stream Creation**
   ```
   Create new content stream with text operators:
   - BT/ET (Begin/End Text)
   - Tf (Set Font)
   - Tm (Text Matrix for positioning)
   - Tr 3 (Set text rendering mode to invisible)
   - Tj/TJ (Show text operators)
   ```

2. **Coordinate System Conversion**
   - OCR: Top-left origin (0,0), Y increases downward
   - PDF: Bottom-left origin (0,0), Y increases upward
   - Conversion: `PDF_Y = PageHeight - OCR_Y - TextHeight`

3. **Font Embedding and Text Encoding**
   - Embed standard font (e.g., Helvetica)
   - Handle text encoding (PDFDocEncoding or UTF-16BE)
   - Set appropriate font size for bounding box coverage

4. **Text Positioning and Scaling**
   - Position text at OCR bounding box coordinates
   - Scale text to match bounding box dimensions
   - Ensure text matches searchable content

**Reference Implementation:** The addTextToPage method (pdf.go:111-146) contains detailed comments outlining these requirements.

### Usage Example

```go
import "github.com/platinummonkey/remarkable-sync/internal/pdfenhancer"

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

The package includes comprehensive tests (83.2% coverage) covering:
- PDF validation (valid and invalid files)
- Page counting and extraction
- PDF information retrieval
- Optimization operations
- Merging and splitting
- Text layer addition structure

Test PDFs are generated programmatically using minimal valid PDF syntax to avoid external dependencies.

### Coordinate System Reference

Use `CompareCoordinateSystems(pageHeight)` to get detailed information about the coordinate system differences between PDF and OCR coordinate spaces.

## Future Enhancements

1. Complete text layer addition implementation with content stream manipulation
2. Support for different text rendering modes (visible, invisible, outline)
3. Advanced font handling (custom fonts, Unicode support)
4. Batch processing optimizations
5. Progress reporting for long operations
