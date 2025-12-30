package rmrender

import (
	"os"
	"testing"
)

func TestRenderToPDF(t *testing.T) {
	// Parse the example file
	doc, err := ParseFile("../../example/b68e57f6-4fc9-4a71-b300-e0fa100ef8d7/aefd8acc-a17d-4e24-a76c-66a3ee15b4ba.rm")
	if err != nil {
		t.Fatalf("Failed to parse example file: %v", err)
	}

	// Create renderer with default options
	renderer := NewRenderer()

	// Render to PDF
	pdfData, err := renderer.RenderToPDF(doc)
	if err != nil {
		t.Fatalf("Failed to render PDF: %v", err)
	}

	// Verify we got PDF data
	if len(pdfData) == 0 {
		t.Fatal("Rendered PDF is empty")
	}

	// Check PDF magic bytes
	if len(pdfData) < 4 {
		t.Fatal("PDF data too short")
	}

	// PDF files start with "%PDF"
	if string(pdfData[:4]) != "%PDF" {
		t.Errorf("PDF magic bytes incorrect, got: %s", string(pdfData[:4]))
	}

	t.Logf("Successfully rendered PDF: %d bytes", len(pdfData))

	// Optionally save for visual inspection
	if os.Getenv("SAVE_TEST_PDF") != "" {
		outputPath := "/tmp/test_render.pdf"
		if err := os.WriteFile(outputPath, pdfData, 0644); err != nil {
			t.Logf("Failed to save test PDF: %v", err)
		} else {
			t.Logf("Saved test PDF to: %s", outputPath)
		}
	}
}

func TestRenderWithOptions(t *testing.T) {
	// Parse the example file
	doc, err := ParseFile("../../example/b68e57f6-4fc9-4a71-b300-e0fa100ef8d7/aefd8acc-a17d-4e24-a76c-66a3ee15b4ba.rm")
	if err != nil {
		t.Fatalf("Failed to parse example file: %v", err)
	}

	// Create renderer with custom options
	opts := &RenderOptions{
		BackgroundColor:    ColorWhite,
		EnablePressure:     true,
		EnableAntialiasing: true,
		StrokeQuality:      7,
	}
	renderer := NewRendererWithOptions(opts)

	// Render to PDF
	pdfData, err := renderer.RenderToPDF(doc)
	if err != nil {
		t.Fatalf("Failed to render PDF with options: %v", err)
	}

	// Verify we got PDF data
	if len(pdfData) == 0 {
		t.Fatal("Rendered PDF is empty")
	}

	t.Logf("Successfully rendered PDF with options: %d bytes", len(pdfData))
}

func TestRenderPage(t *testing.T) {
	// Parse the example file
	doc, err := ParseFile("../../example/b68e57f6-4fc9-4a71-b300-e0fa100ef8d7/aefd8acc-a17d-4e24-a76c-66a3ee15b4ba.rm")
	if err != nil {
		t.Fatalf("Failed to parse example file: %v", err)
	}

	// Create renderer
	renderer := NewRenderer()

	// Render single page
	pdfData, err := renderer.RenderPage(doc, 0)
	if err != nil {
		t.Fatalf("Failed to render page: %v", err)
	}

	// Verify we got PDF data
	if len(pdfData) == 0 {
		t.Fatal("Rendered PDF is empty")
	}

	t.Logf("Successfully rendered page: %d bytes", len(pdfData))
}

func TestRenderEmptyDocument(t *testing.T) {
	// Create empty document
	doc := &Document{
		Version: Version6,
		Layers:  []Layer{},
	}

	// Create renderer
	renderer := NewRenderer()

	// Render to PDF
	pdfData, err := renderer.RenderToPDF(doc)
	if err != nil {
		t.Fatalf("Failed to render empty PDF: %v", err)
	}

	// Verify we got PDF data
	if len(pdfData) == 0 {
		t.Fatal("Rendered PDF is empty")
	}

	t.Logf("Successfully rendered empty PDF: %d bytes", len(pdfData))
}

func TestRenderNilDocument(t *testing.T) {
	// Create renderer
	renderer := NewRenderer()

	// Render nil document - should return error
	_, err := renderer.RenderToPDF(nil)
	if err == nil {
		t.Fatal("Expected error for nil document, got nil")
	}

	t.Logf("Correctly returned error for nil document: %v", err)
}
