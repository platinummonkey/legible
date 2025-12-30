package pdfenhancer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/platinummonkey/remarkable-sync/internal/logger"
	"github.com/platinummonkey/remarkable-sync/internal/ocr"
)

// createTestPDF creates a simple test PDF file
func createTestPDF(t *testing.T, path string, pageCount int) {
	t.Helper()

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	// Create a minimal valid PDF with specified number of pages
	// This is a simple PDF structure that pdfcpu can read
	pdfContent := `%PDF-1.4
1 0 obj
<<
/Type /Catalog
/Pages 2 0 R
>>
endobj
2 0 obj
<<
/Type /Pages
/Kids [3 0 R]
/Count 1
>>
endobj
3 0 obj
<<
/Type /Page
/Parent 2 0 R
/MediaBox [0 0 612 792]
/Contents 4 0 R
/Resources <<
/Font <<
/F1 <<
/Type /Font
/Subtype /Type1
/BaseFont /Helvetica
>>
>>
>>
>>
endobj
4 0 obj
<<
/Length 44
>>
stream
BT
/F1 12 Tf
100 700 Td
(Test PDF) Tj
ET
endstream
endobj
xref
0 5
0000000000 65535 f
0000000009 00000 n
0000000058 00000 n
0000000115 00000 n
0000000317 00000 n
trailer
<<
/Size 5
/Root 1 0 R
>>
startxref
410
%%EOF`

	if err := os.WriteFile(path, []byte(pdfContent), 0644); err != nil {
		t.Fatalf("failed to write test PDF: %v", err)
	}
}

func TestNew(t *testing.T) {
	cfg := &Config{}
	enhancer := New(cfg)

	if enhancer == nil {
		t.Fatal("New() returned nil")
	}

	if enhancer.logger == nil {
		t.Error("logger should be initialized")
	}
}

func TestNew_CustomLogger(t *testing.T) {
	customLogger, err := logger.New(&logger.Config{
		Level:  "debug",
		Format: "console",
	})
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	cfg := &Config{
		Logger: customLogger,
	}
	enhancer := New(cfg)

	if enhancer.logger != customLogger {
		t.Error("custom logger should be used")
	}
}

func TestPDFEnhancer_ValidatePDF_FileNotFound(t *testing.T) {
	enhancer := New(&Config{})

	err := enhancer.ValidatePDF("/nonexistent/file.pdf")
	if err == nil {
		t.Error("ValidatePDF() should error for nonexistent file")
	}
}

func TestPDFEnhancer_ValidatePDF_InvalidPDF(t *testing.T) {
	tmpDir := t.TempDir()
	invalidPDF := filepath.Join(tmpDir, "invalid.pdf")

	// Create an invalid PDF (just random text)
	if err := os.WriteFile(invalidPDF, []byte("not a pdf"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	enhancer := New(&Config{})
	err := enhancer.ValidatePDF(invalidPDF)

	if err == nil {
		t.Error("ValidatePDF() should error for invalid PDF")
	}
}

func TestPDFEnhancer_GetPageCount(t *testing.T) {
	tmpDir := t.TempDir()
	pdfPath := filepath.Join(tmpDir, "test.pdf")

	createTestPDF(t, pdfPath, 1)

	enhancer := New(&Config{})
	pageCount, err := enhancer.GetPageCount(pdfPath)

	if err != nil {
		t.Fatalf("GetPageCount() error = %v", err)
	}

	if pageCount != 1 {
		t.Errorf("expected page count 1, got %d", pageCount)
	}
}

func TestPDFEnhancer_GetPageCount_InvalidFile(t *testing.T) {
	enhancer := New(&Config{})

	_, err := enhancer.GetPageCount("/nonexistent/file.pdf")
	if err == nil {
		t.Error("GetPageCount() should error for nonexistent file")
	}
}

func TestPDFEnhancer_GetPDFInfo(t *testing.T) {
	tmpDir := t.TempDir()
	pdfPath := filepath.Join(tmpDir, "test.pdf")

	createTestPDF(t, pdfPath, 1)

	enhancer := New(&Config{})
	info, err := enhancer.GetPDFInfo(pdfPath)

	if err != nil {
		t.Fatalf("GetPDFInfo() error = %v", err)
	}

	if info.PageCount != 1 {
		t.Errorf("expected page count 1, got %d", info.PageCount)
	}

	if info.FileSize <= 0 {
		t.Errorf("expected positive file size, got %d", info.FileSize)
	}

	if info.PDFVersion == "" {
		t.Error("PDF version should not be empty")
	}
}

func TestPDFEnhancer_ExtractPageInfo(t *testing.T) {
	tmpDir := t.TempDir()
	pdfPath := filepath.Join(tmpDir, "test.pdf")

	createTestPDF(t, pdfPath, 1)

	enhancer := New(&Config{})
	pageInfo, err := enhancer.ExtractPageInfo(pdfPath, 1)

	if err != nil {
		t.Fatalf("ExtractPageInfo() error = %v", err)
	}

	if pageInfo.PageNumber != 1 {
		t.Errorf("expected page number 1, got %d", pageInfo.PageNumber)
	}

	if pageInfo.Width <= 0 {
		t.Errorf("expected positive width, got %d", pageInfo.Width)
	}

	if pageInfo.Height <= 0 {
		t.Errorf("expected positive height, got %d", pageInfo.Height)
	}
}

func TestPDFEnhancer_ExtractPageInfo_InvalidPage(t *testing.T) {
	tmpDir := t.TempDir()
	pdfPath := filepath.Join(tmpDir, "test.pdf")

	createTestPDF(t, pdfPath, 1)

	enhancer := New(&Config{})

	// Page 0 should be invalid
	_, err := enhancer.ExtractPageInfo(pdfPath, 0)
	if err == nil {
		t.Error("ExtractPageInfo() should error for page 0")
	}

	// Page beyond count should be invalid
	_, err = enhancer.ExtractPageInfo(pdfPath, 100)
	if err == nil {
		t.Error("ExtractPageInfo() should error for page beyond count")
	}
}

func TestPDFEnhancer_OptimizePDF(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "input.pdf")
	outputPath := filepath.Join(tmpDir, "output.pdf")

	createTestPDF(t, inputPath, 1)

	enhancer := New(&Config{})
	err := enhancer.OptimizePDF(inputPath, outputPath)

	if err != nil {
		t.Fatalf("OptimizePDF() error = %v", err)
	}

	// Verify output file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("output PDF should exist after optimization")
	}
}

func TestPDFEnhancer_AddTextLayer_NilOCR(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "input.pdf")
	outputPath := filepath.Join(tmpDir, "output.pdf")

	createTestPDF(t, inputPath, 1)

	enhancer := New(&Config{})
	err := enhancer.AddTextLayer(inputPath, outputPath, nil)

	if err == nil {
		t.Error("AddTextLayer() should error with nil OCR results")
	}
}

func TestPDFEnhancer_AddTextLayer_PageCountMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "input.pdf")
	outputPath := filepath.Join(tmpDir, "output.pdf")

	createTestPDF(t, inputPath, 1)

	// Create OCR results with wrong page count
	ocrResults := ocr.NewDocumentOCR("test-doc", "eng")
	page1 := ocr.NewPageOCR(1, 1404, 1872, "eng")
	page2 := ocr.NewPageOCR(2, 1404, 1872, "eng")
	ocrResults.AddPage(*page1)
	ocrResults.AddPage(*page2)

	enhancer := New(&Config{})
	err := enhancer.AddTextLayer(inputPath, outputPath, ocrResults)

	if err == nil {
		t.Error("AddTextLayer() should error when page counts don't match")
	}
}

func TestPDFEnhancer_AddTextLayer_ValidInput(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "input.pdf")
	outputPath := filepath.Join(tmpDir, "output.pdf")

	createTestPDF(t, inputPath, 1)

	// Create matching OCR results
	ocrResults := ocr.NewDocumentOCR("test-doc", "eng")
	page := ocr.NewPageOCR(1, 1404, 1872, "eng")
	page.AddWord(ocr.NewWord("test", ocr.NewRectangle(10, 20, 50, 15), 95.0))
	page.BuildText()
	page.CalculateConfidence()
	ocrResults.AddPage(*page)
	ocrResults.Finalize()

	enhancer := New(&Config{})
	err := enhancer.AddTextLayer(inputPath, outputPath, ocrResults)

	if err != nil {
		t.Fatalf("AddTextLayer() error = %v", err)
	}

	// Verify output file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("output PDF should exist after adding text layer")
	}
}

func TestPDFEnhancer_MergePDFs_NoInput(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "merged.pdf")

	enhancer := New(&Config{})
	err := enhancer.MergePDFs([]string{}, outputPath)

	if err == nil {
		t.Error("MergePDFs() should error with no input files")
	}
}

func TestPDFEnhancer_MergePDFs_SingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	input1 := filepath.Join(tmpDir, "input1.pdf")
	outputPath := filepath.Join(tmpDir, "merged.pdf")

	createTestPDF(t, input1, 1)

	enhancer := New(&Config{})
	err := enhancer.MergePDFs([]string{input1}, outputPath)

	if err != nil {
		t.Fatalf("MergePDFs() error = %v", err)
	}

	// Verify output file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("merged PDF should exist")
	}
}

func TestPDFEnhancer_SplitPDF(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "input.pdf")
	outputDir := filepath.Join(tmpDir, "split")

	createTestPDF(t, inputPath, 1)

	enhancer := New(&Config{})
	err := enhancer.SplitPDF(inputPath, outputDir)

	if err != nil {
		t.Fatalf("SplitPDF() error = %v", err)
	}

	// Verify output directory and files exist
	// pdfcpu names files as <inputBasename>_page_N.pdf
	page1Path := filepath.Join(outputDir, "input_page_1.pdf")
	if _, err := os.Stat(page1Path); os.IsNotExist(err) {
		t.Errorf("split page should exist at %s", page1Path)
	}
}

func TestPDFEnhancer_CompareCoordinateSystems(t *testing.T) {
	enhancer := New(&Config{})

	result := enhancer.CompareCoordinateSystems(1872)

	if result == "" {
		t.Error("CompareCoordinateSystems() should return non-empty string")
	}

	if !contains(result, "PDF") || !contains(result, "OCR") {
		t.Error("result should mention both PDF and OCR coordinate systems")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}
