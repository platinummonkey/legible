# Test Data

This directory contains test fixtures and sample data for testing remarkable-sync components.

## Directory Structure

```
testdata/
├── rmdoc/           # Sample .rmdoc files and extracted content
├── pdfs/            # Sample PDF files for testing
├── images/          # Sample images for OCR testing
├── api-mocks/       # Mock API responses (JSON)
├── golden/          # Expected outputs for comparison testing
└── README.md        # This file
```

## Contents

### rmdoc/

Contains sample reMarkable document files for testing the converter package.

**Files:**
- `Test.rmdoc` - Small 2-page document (29KB)
- `b68e57f6-4fc9-4a71-b300-e0fa100ef8d7.metadata` - Extracted metadata JSON
- `b68e57f6-4fc9-4a71-b300-e0fa100ef8d7.content` - Extracted content JSON

**Usage:**
- `internal/converter/converter_test.go` - Testing .rmdoc extraction and parsing
- Integration tests for full pipeline

**Document Details:**
- Document ID: `b68e57f6-4fc9-4a71-b300-e0fa100ef8d7`
- Title: "Test"
- Pages: 2
- Format: V6 (formatVersion=2)
- Contains: Handwritten content on blank pages

### pdfs/

Contains sample PDF files for testing PDF enhancement and manipulation.

**Expected Files:**
- Sample PDFs with known content for OCR testing
- PDFs with various page counts and dimensions
- PDFs with and without existing text layers

**Usage:**
- `internal/pdfenhancer/pdf_test.go` - Testing text layer addition
- `internal/ocr/ocr_test.go` - Testing OCR on rendered PDFs

**Note:** Test PDFs are generated programmatically in tests to avoid binary bloat.

### images/

Contains sample images for OCR testing.

**Expected Files:**
- `simple-text.png` - Clear printed text (generated in tests)
- `handwriting.png` - Handwritten sample (generated in tests)

**Usage:**
- `internal/ocr/ocr_test.go` - Testing Tesseract OCR
- Validation of text extraction and positioning

**Note:** Images are generated programmatically in tests using Go's `image` package.
See `images/README.md` for generation details.

### api-mocks/

Contains mock API responses for testing without live reMarkable API.

**Files:**
- `list-documents.json` - Sample document list response

**Usage:**
- `internal/rmclient/*_test.go` - Testing API client without network
- Integration tests for sync workflow

**Format:**
Mock responses match the actual reMarkable API response format.

### golden/

Contains expected outputs for comparison testing.

**Structure:**
- `converter/` - Expected converter outputs
- `ocr/` - Expected OCR results
- `sync/` - Expected sync state and results

**Usage:**
All test packages use golden files for regression testing.

See `golden/README.md` for detailed documentation on golden file management.

## Test Data Guidelines

### Size Constraints

Keep test data small to avoid repository bloat:

- **.rmdoc files**: < 100KB each
- **Images**: < 50KB each
- **PDFs**: < 200KB each
- **JSON**: < 10KB each

### Realism

Test data should be realistic but minimal:

- Use actual .rmdoc format (from real devices)
- Use valid API response structures
- Include edge cases (empty, large, malformed)

### Organization

- One directory per data type
- Descriptive filenames
- README in each subdirectory
- Document file origins and purposes

### Binary Files

Minimize binary files in git:

- Prefer programmatic generation in tests
- Use small, compressed files when needed
- Document why binary file is required

### Updating Test Data

When adding or modifying test data:

1. **Document**: Add description to relevant README
2. **Justify**: Explain why the data is needed
3. **Minimize**: Use smallest possible file
4. **Review**: Verify file is necessary and appropriate

## Usage in Tests

### Loading Test Files

```go
import "path/filepath"

func loadTestData(filename string) ([]byte, error) {
    path := filepath.Join("testdata", "rmdoc", filename)
    return os.ReadFile(path)
}
```

### Using Golden Files

```go
func TestWithGolden(t *testing.T) {
    result := processData(input)

    goldenPath := filepath.Join("testdata", "golden", "expected-output.json")
    golden, _ := os.ReadFile(goldenPath)

    if !bytes.Equal(result, golden) {
        t.Errorf("output differs from golden file")
    }
}
```

### Generating Test Data

```go
func createTestImage(t *testing.T) *image.RGBA {
    t.Helper()
    img := image.NewRGBA(image.Rect(0, 0, 400, 100))
    // Fill with white background
    white := color.RGBA{255, 255, 255, 255}
    for y := 0; y < 100; y++ {
        for x := 0; x < 400; x++ {
            img.Set(x, y, white)
        }
    }
    return img
}
```

## Test Data Sources

### Real reMarkable Documents

The `Test.rmdoc` file was created on a real reMarkable 2 tablet:

- Created: 2023-12-01
- Device: reMarkable 2
- Software: v3.x
- Format: V6 (.rm format)
- Content: Sample handwritten notes

### Mock API Data

Mock API responses are based on actual reMarkable cloud API responses, but with:

- Sanitized identifiers
- Minimal data sets
- Representative structures

## Maintenance

### Regular Reviews

Periodically review test data for:

- **Obsolescence**: Remove unused files
- **Bloat**: Reduce file sizes where possible
- **Relevance**: Ensure data matches current formats
- **Documentation**: Keep READMEs up to date

### Format Updates

When reMarkable updates file formats:

1. Add new format examples
2. Keep old format for backward compatibility tests
3. Document format version in README
4. Update tests to handle both versions

## Contributing

When adding test data:

- Follow the guidelines above
- Update relevant READMEs
- Keep pull requests focused
- Explain why new data is needed

## Related Documentation

- [internal/converter/README.md](../internal/converter/README.md) - .rmdoc format details
- [internal/ocr/README.md](../internal/ocr/README.md) - OCR testing requirements
- [internal/pdfenhancer/README.md](../internal/pdfenhancer/README.md) - PDF testing requirements

## License

Test data files are part of the remarkable-sync project. See project LICENSE for details.

Actual .rmdoc files created on reMarkable devices are owned by their creators and included here solely for testing purposes.
