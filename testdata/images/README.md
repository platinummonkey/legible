# Test Images

This directory contains sample images for OCR testing.

## Files

### simple-text.png (to be generated)
A simple image with clear printed text for basic OCR testing.
- Size: 400x100 pixels
- Content: "Hello World"
- Background: White
- Text: Black, clear font

### handwriting.png (to be generated)
An image with handwritten text for handwriting OCR testing.
- Size: 600x200 pixels
- Content: Sample handwritten notes
- Background: White
- Text: Black ink simulation

## Generating Test Images

Test images can be generated programmatically in tests using Go's `image` package:

```go
import (
    "image"
    "image/color"
    "image/png"
    "os"
)

func createTestImage(width, height int, text string) *image.RGBA {
    img := image.NewRGBA(image.Rect(0, 0, width, height))
    white := color.RGBA{255, 255, 255, 255}
    for y := 0; y < height; y++ {
        for x := 0; x < width; x++ {
            img.Set(x, y, white)
        }
    }
    // Note: Drawing text requires font rendering library
    // For tests, use golang.org/x/image/font or freetype-go
    return img
}
```

## Usage in Tests

Images in this directory are used by:
- `internal/ocr/ocr_test.go` - OCR processing tests
- `internal/pdfenhancer/pdf_test.go` - Text layer addition tests

These images should remain small (<100KB each) to avoid repository bloat.
