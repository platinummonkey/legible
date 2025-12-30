# OCR Package

Package ocr provides Optical Character Recognition (OCR) functionality using Tesseract OCR engine via the gosseract Go wrapper.

## Overview

The OCR package processes images to extract text with positional information (bounding boxes). It's designed to make PDF documents searchable by extracting text from rendered pages or handwritten notes.

## Features

✅ **Tesseract Integration**
- Uses [gosseract v2](https://github.com/otiai10/gosseract) for Tesseract OCR
- Support for 100+ languages
- Configurable language detection
- HOCR output for structured results with positions

✅ **Structured Output**
- Word-level text extraction with bounding boxes
- Confidence scores for each word
- Page-level text and confidence aggregation
- Document-level statistics

✅ **Error Handling**
- Graceful handling of OCR failures
- Detailed logging of processing steps
- Configurable confidence thresholds

## System Requirements

### Tesseract OCR Installation

**gosseract requires Tesseract OCR and Leptonica to be installed on the system.**

#### macOS (Homebrew)
```bash
brew install tesseract
brew install leptonica
```

#### Ubuntu/Debian
```bash
sudo apt-get install tesseract-ocr
sudo apt-get install libtesseract-dev
sudo apt-get install libleptonica-dev
```

#### Windows
1. Download Tesseract installer from [GitHub releases](https://github.com/UB-Mannheim/tesseract/wiki)
2. Install to default location (C:\Program Files\Tesseract-OCR)
3. Add to PATH environment variable

### Language Data

Install additional language data if needed:

**macOS:**
```bash
brew install tesseract-lang  # All languages
```

**Ubuntu/Debian:**
```bash
sudo apt-get install tesseract-ocr-eng  # English
sudo apt-get install tesseract-ocr-fra  # French
sudo apt-get install tesseract-ocr-spa  # Spanish
# etc...
```

**Check available languages:**
```bash
tesseract --list-langs
```

## Usage

### Basic OCR Processing

```go
import "github.com/platinummonkey/remarkable-sync/internal/ocr"

// Create processor
processor := ocr.New(&ocr.Config{
    Languages: []string{"eng"},  // English
})

// Process image
imageData, _ := os.ReadFile("page.png")
pageOCR, err := processor.ProcessImage(imageData, 1)
if err != nil {
    log.Fatal(err)
}

// Access results
fmt.Printf("Page text: %s\n", pageOCR.Text)
fmt.Printf("Confidence: %.2f%%\n", pageOCR.Confidence)
fmt.Printf("Words found: %d\n", len(pageOCR.Words))

// Iterate over words with positions
for _, word := range pageOCR.Words {
    fmt.Printf("Word: %s at (%d, %d) confidence: %.2f\n",
        word.Text,
        word.BoundingBox.X,
        word.BoundingBox.Y,
        word.Confidence)
}
```

### Multi-Language OCR

```go
processor := ocr.New(&ocr.Config{
    Languages: []string{"eng", "fra", "deu"},  // English, French, German
})

pageOCR, err := processor.ProcessImage(imageData, 1)
```

### Document Processing

```go
doc := ocr.NewDocumentOCR("doc-123", "eng")

for pageNum, imageData := range pageImages {
    pageOCR, err := processor.ProcessImage(imageData, pageNum+1)
    if err != nil {
        log.Printf("Failed to process page %d: %v", pageNum+1, err)
        continue
    }

    doc.AddPage(*pageOCR)
}

doc.Finalize()

fmt.Printf("Total pages: %d\n", doc.TotalPages)
fmt.Printf("Total words: %d\n", doc.TotalWords)
fmt.Printf("Average confidence: %.2f%%\n", doc.AverageConfidence)
```

## Implementation Details

### HOCR Format

The processor uses Tesseract's HOCR output format, which provides structured HTML with:
- Page boundaries (`ocr_page`)
- Content areas (`ocr_carea`)
- Paragraphs (`ocr_par`)
- Lines (`ocr_line`)
- Words (`ocr_word`) with bounding boxes and confidence scores

**HOCR Example:**
```html
<span class='ocr_word' title='bbox 100 200 150 220; x_wconf 95'>Hello</span>
```

Where:
- `bbox x0 y0 x1 y1`: Bounding box coordinates (top-left to bottom-right)
- `x_wconf`: Word confidence (0-100)

### Coordinate System

**OCR Coordinate System:**
- Origin: Top-left corner (0, 0)
- X-axis: Increases rightward
- Y-axis: Increases downward

**Bounding Box Format:**
- X: Left edge position (pixels from left)
- Y: Top edge position (pixels from top)
- Width: Box width in pixels
- Height: Box height in pixels

### Processing Pipeline

1. **Image Input** - Accept image data as bytes
2. **Tesseract Init** - Create and configure Tesseract client
3. **OCR Execution** - Run Tesseract with HOCR output
4. **HOCR Parsing** - Parse XML to extract words and positions
5. **Structure Building** - Create PageOCR with words, text, confidence
6. **Result Return** - Return structured OCR results

## Testing

### Running Tests

**Note:** Tests require Tesseract to be installed on the system.

```bash
# Run all tests
go test ./internal/ocr

# Run with verbose output
go test -v ./internal/ocr

# Run specific test
go test -v ./internal/ocr -run TestParseHOCR
```

### Test Coverage

The package includes tests for:
- ✅ Processor initialization
- ✅ HOCR parsing with structured XML
- ✅ Bounding box extraction
- ✅ Confidence score extraction
- ✅ Custom language configuration
- ⚠️  Integration tests (require Tesseract installation)

### CI/CD Considerations

For CI/CD pipelines, ensure Tesseract is installed:

**GitHub Actions:**
```yaml
- name: Install Tesseract
  run: |
    sudo apt-get update
    sudo apt-get install -y tesseract-ocr libtesseract-dev libleptonica-dev
```

**Docker:**
```dockerfile
RUN apt-get update && apt-get install -y \
    tesseract-ocr \
    libtesseract-dev \
    libleptonica-dev
```

## Performance Considerations

### Optimization Tips

1. **Image Preprocessing**
   - Convert to grayscale
   - Adjust contrast/brightness
   - Remove noise
   - Deskew/straighten pages

2. **Page Segmentation Mode**
   - Use appropriate PSM for document type
   - Single column: `PSM_SINGLE_COLUMN`
   - Single block: `PSM_SINGLE_BLOCK`

3. **Language Selection**
   - Use only required languages
   - More languages = slower processing
   - Consider language detection first

4. **Parallel Processing**
   - Process pages in parallel
   - Use worker pools for batch jobs
   - Be mindful of memory usage

### Typical Performance

- **Simple page (200 words)**: ~500ms - 1s
- **Complex page (500 words)**: ~1s - 2s
- **Handwritten notes**: ~2s - 5s (higher variance)

Performance depends on:
- Image size and resolution
- Text density
- Number of languages
- Hardware (CPU, RAM)

## Future Enhancements

### Planned Features

1. **PDF-to-Image Conversion**
   - Integrate PDF rendering
   - Extract pages as images for OCR
   - Handle multi-page PDFs

2. **Preprocessing Pipeline**
   - Image enhancement filters
   - Automatic deskewing
   - Noise removal

3. **Advanced Configuration**
   - Page segmentation mode selection
   - Whitelist/blacklist characters
   - Custom Tesseract variables

4. **Caching & Optimization**
   - Cache OCR results
   - Skip unchanged pages
   - Batch processing optimizations

5. **Quality Metrics**
   - Confidence-based filtering
   - Word-level quality scores
   - Automatic quality assessment

## References

**Libraries:**
- [gosseract](https://github.com/otiai10/gosseract) - Go wrapper for Tesseract
- [Tesseract OCR](https://github.com/tesseract-ocr/tesseract) - OCR engine
- [HOCR Specification](https://en.wikipedia.org/wiki/HOCR) - HTML-based OCR format

**Documentation:**
- [Tesseract Documentation](https://tesseract-ocr.github.io/)
- [gosseract API Reference](https://pkg.go.dev/github.com/otiai10/gosseract/v2)
- [Tesseract Training](https://github.com/tesseract-ocr/tessdoc/blob/main/tess4/TrainingTesseract-4.00.md)

## License

Part of remarkable-sync project. See project LICENSE for details.
