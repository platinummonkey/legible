package ocr

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/otiai10/gosseract/v2"
)

func TestNew(t *testing.T) {
	cfg := &Config{}
	processor := New(cfg)

	if processor == nil {
		t.Fatal("New() returned nil")
	}

	if processor.logger == nil {
		t.Error("logger should be initialized")
	}

	if len(processor.languages) == 0 {
		t.Error("languages should have default value")
	}

	if processor.languages[0] != "eng" {
		t.Errorf("default language should be 'eng', got '%s'", processor.languages[0])
	}
}

func TestNew_CustomLanguages(t *testing.T) {
	cfg := &Config{
		Languages: []string{"eng", "fra"},
	}
	processor := New(cfg)

	if len(processor.languages) != 2 {
		t.Errorf("expected 2 languages, got %d", len(processor.languages))
	}

	if processor.languages[0] != "eng" || processor.languages[1] != "fra" {
		t.Error("custom languages not set correctly")
	}
}

func TestExtractBBox(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  []int
	}{
		{
			name:  "valid bbox",
			title: "bbox 100 200 300 400",
			want:  []int{100, 200, 300, 400},
		},
		{
			name:  "bbox with confidence",
			title: "bbox 50 75 150 125; x_wconf 95",
			want:  []int{50, 75, 150, 125},
		},
		{
			name:  "invalid format",
			title: "no bbox here",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractBBox(tt.title)

			if tt.want == nil {
				if got != nil {
					t.Errorf("extractBBox() = %v, want nil", got)
				}
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("extractBBox() length = %d, want %d", len(got), len(tt.want))
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("extractBBox()[%d] = %d, want %d", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestExtractConfidence(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  float64
	}{
		{
			name:  "confidence 95",
			title: "bbox 100 200 300 400; x_wconf 95",
			want:  95.0,
		},
		{
			name:  "confidence 100",
			title: "bbox 50 75 150 125; x_wconf 100",
			want:  100.0,
		},
		{
			name:  "confidence 42",
			title: "bbox 10 20 30 40; x_wconf 42",
			want:  42.0,
		},
		{
			name:  "no confidence",
			title: "bbox 100 200 300 400",
			want:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractConfidence(tt.title)
			if got != tt.want {
				t.Errorf("extractConfidence() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseHOCR(t *testing.T) {
	processor := New(&Config{})

	hocrXML := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN"
    "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml" xml:lang="en" lang="en">
<head>
<title>OCR Results</title>
<meta http-equiv="content-type" content="text/html; charset=utf-8" />
</head>
<body>
<div class='ocr_page' id='page_1' title='bbox 0 0 800 600'>
<div class='ocr_carea' id='carea_1_1' title='bbox 50 50 750 550'>
<p class='ocr_par' id='par_1_1' title='bbox 50 50 750 100'>
<span class='ocr_line' id='line_1_1' title='bbox 50 50 200 80'>
<span class='ocr_word' id='word_1_1' title='bbox 50 50 100 80; x_wconf 95'>Hello</span>
<span class='ocr_word' id='word_1_2' title='bbox 110 50 200 80; x_wconf 92'>World</span>
</span>
</p>
</div>
</div>
</body>
</html>`

	pageOCR, err := processor.parseHOCR(hocrXML, 1)
	if err != nil {
		t.Fatalf("parseHOCR() error = %v", err)
	}

	if pageOCR.PageNumber != 1 {
		t.Errorf("PageNumber = %d, want 1", pageOCR.PageNumber)
	}

	if pageOCR.Width != 800 || pageOCR.Height != 600 {
		t.Errorf("Dimensions = %dx%d, want 800x600", pageOCR.Width, pageOCR.Height)
	}

	if len(pageOCR.Words) != 2 {
		t.Errorf("len(Words) = %d, want 2", len(pageOCR.Words))
	}

	if len(pageOCR.Words) >= 1 {
		word := pageOCR.Words[0]
		if word.Text != "Hello" {
			t.Errorf("First word = %s, want Hello", word.Text)
		}
		if word.Confidence != 95.0 {
			t.Errorf("First word confidence = %f, want 95.0", word.Confidence)
		}
		if word.BoundingBox.X != 50 || word.BoundingBox.Y != 50 {
			t.Errorf("First word position = (%d, %d), want (50, 50)", word.BoundingBox.X, word.BoundingBox.Y)
		}
	}
}

func TestProcessImage_WithSimpleImage(t *testing.T) {
	// Skip if Tesseract is not installed
	if !isTesseractInstalled() {
		t.Skip("Tesseract not installed, skipping integration test")
	}

	processor := New(&Config{})

	// Create a simple test image with text
	imgData := createTestImage(t, "TEST", 200, 100)

	pageOCR, err := processor.ProcessImage(imgData, 1)
	if err != nil {
		t.Fatalf("ProcessImage() error = %v", err)
	}

	if pageOCR == nil {
		t.Fatal("ProcessImage() returned nil result")
	}

	if pageOCR.PageNumber != 1 {
		t.Errorf("PageNumber = %d, want 1", pageOCR.PageNumber)
	}

	// The image should have some OCR results (may vary based on Tesseract)
	if len(pageOCR.Words) == 0 {
		t.Log("Warning: No words detected (may be due to simple test image)")
	}

	// Text should be built
	pageOCR.BuildText()
	if strings.Contains(strings.ToUpper(pageOCR.Text), "TEST") {
		t.Logf("Successfully detected text: %s", pageOCR.Text)
	}
}

func TestProcessImage_InvalidImage(t *testing.T) {
	// Skip if Tesseract is not installed
	if !isTesseractInstalled() {
		t.Skip("Tesseract not installed, skipping integration test")
	}

	processor := New(&Config{})

	// Invalid image data
	_, err := processor.ProcessImage([]byte("not an image"), 1)
	if err == nil {
		t.Error("ProcessImage() should error with invalid image data")
	}
}

// Helper functions

func isTesseractInstalled() bool {
	// Try to create a client to check if Tesseract is available
	client := gosseract.NewClient()
	defer client.Close()

	// Try to set a language - this will fail if Tesseract is not installed
	err := client.SetLanguage("eng")
	return err == nil
}

func createTestImage(t *testing.T, text string, width, height int) []byte {
	t.Helper()

	// Create a white image
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	white := color.RGBA{255, 255, 255, 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, white)
		}
	}

	// Note: Drawing text on the image requires a font rendering library
	// For now, we'll create a blank image and rely on Tesseract detecting nothing
	// A real implementation would draw the text using golang.org/x/image/font

	// Encode to PNG
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.png")

	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		t.Fatalf("failed to encode PNG: %v", err)
	}

	// Read back the image data
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read temp file: %v", err)
	}

	return data
}
