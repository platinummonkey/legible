package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/platinummonkey/legible/internal/converter"
	"github.com/platinummonkey/legible/internal/logger"
)

// TestConverterPipeline tests the full conversion pipeline from .rmdoc to PDF
func TestConverterPipeline(t *testing.T) {
	tmpDir := t.TempDir()

	log, err := logger.New(&logger.Config{
		Level:  "info",
		Format: "console",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	conv, err := converter.New(&converter.Config{
		Logger: log,
	})
	if err != nil {
		t.Fatalf("Failed to create converter: %v", err)
	}

	// Use the test .rmdoc file from testdata
	testRmdoc := "../../testdata/rmdoc/Test.rmdoc"
	if _, err := os.Stat(testRmdoc); os.IsNotExist(err) {
		t.Skip("Test.rmdoc not found in testdata, skipping converter integration test")
	}

	outputPDF := filepath.Join(tmpDir, "test-output.pdf")

	// Convert .rmdoc to PDF
	result, err := conv.ConvertRmdoc(testRmdoc, outputPDF)
	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	// Verify conversion result
	if !result.Success {
		t.Error("Conversion should be successful")
	}

	if result.PageCount != 2 {
		t.Errorf("Expected 2 pages, got %d", result.PageCount)
	}

	if result.OutputPath != outputPDF {
		t.Errorf("Expected output path %s, got %s", outputPDF, result.OutputPath)
	}

	// Verify output file exists
	if _, err := os.Stat(outputPDF); os.IsNotExist(err) {
		t.Error("Output PDF should exist after conversion")
	}

	// Verify output file has content
	info, err := os.Stat(outputPDF)
	if err != nil {
		t.Fatalf("Failed to stat output PDF: %v", err)
	}

	if info.Size() == 0 {
		t.Error("Output PDF should not be empty")
	}

	// Verify warnings if any
	if len(result.Warnings) > 0 {
		t.Logf("Conversion warnings: %v", result.Warnings)
	}
}

// TestConverterWithInvalidInput tests converter error handling
func TestConverterWithInvalidInput(t *testing.T) {
	tmpDir := t.TempDir()

	log, err := logger.New(&logger.Config{
		Level:  "info",
		Format: "console",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	conv, err := converter.New(&converter.Config{
		Logger: log,
	})
	if err != nil {
		t.Fatalf("Failed to create converter: %v", err)
	}

	tests := []struct {
		name        string
		inputPath   string
		outputPath  string
		expectError bool
	}{
		{
			name:        "nonexistent file",
			inputPath:   "/nonexistent/file.rmdoc",
			outputPath:  filepath.Join(tmpDir, "output1.pdf"),
			expectError: true,
		},
		{
			name:        "invalid file extension",
			inputPath:   filepath.Join(tmpDir, "test.txt"),
			outputPath:  filepath.Join(tmpDir, "output2.pdf"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file for invalid extension test
			if tt.name == "invalid file extension" {
				if err := os.WriteFile(tt.inputPath, []byte("not a rmdoc"), 0644); err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
			}

			result, err := conv.ConvertRmdoc(tt.inputPath, tt.outputPath)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error for invalid input")
				}
				if result != nil && result.Success {
					t.Error("Conversion should not succeed with invalid input")
				}
			}
		})
	}
}

// TestConverterMultipleDocuments tests converting multiple documents in sequence
func TestConverterMultipleDocuments(t *testing.T) {
	testRmdoc := "../../testdata/rmdoc/Test.rmdoc"
	if _, err := os.Stat(testRmdoc); os.IsNotExist(err) {
		t.Skip("Test.rmdoc not found in testdata, skipping test")
	}

	tmpDir := t.TempDir()

	log, err := logger.New(&logger.Config{
		Level:  "info",
		Format: "console",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	conv, err := converter.New(&converter.Config{
		Logger: log,
	})
	if err != nil {
		t.Fatalf("Failed to create converter: %v", err)
	}

	// Convert same document multiple times (simulating batch processing)
	numDocs := 3
	var results []*converter.ConversionResult

	for i := 0; i < numDocs; i++ {
		outputPDF := filepath.Join(tmpDir, filepath.Base(testRmdoc)+"-"+string(rune('a'+i))+".pdf")
		result, err := conv.ConvertRmdoc(testRmdoc, outputPDF)
		if err != nil {
			t.Fatalf("Conversion %d failed: %v", i, err)
		}
		results = append(results, result)
	}

	// Verify all conversions succeeded
	for i, result := range results {
		if !result.Success {
			t.Errorf("Conversion %d should succeed", i)
		}

		if _, err := os.Stat(result.OutputPath); os.IsNotExist(err) {
			t.Errorf("Output PDF %d should exist", i)
		}
	}

	// Verify all output files are different (different paths)
	seenPaths := make(map[string]bool)
	for i, result := range results {
		if seenPaths[result.OutputPath] {
			t.Errorf("Duplicate output path for conversion %d", i)
		}
		seenPaths[result.OutputPath] = true
	}
}

// TestConverterOutputDirectory tests output directory creation
func TestConverterOutputDirectory(t *testing.T) {
	testRmdoc := "../../testdata/rmdoc/Test.rmdoc"
	if _, err := os.Stat(testRmdoc); os.IsNotExist(err) {
		t.Skip("Test.rmdoc not found in testdata, skipping test")
	}

	tmpDir := t.TempDir()

	log, err := logger.New(&logger.Config{
		Level:  "info",
		Format: "console",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	conv, err := converter.New(&converter.Config{
		Logger: log,
	})
	if err != nil {
		t.Fatalf("Failed to create converter: %v", err)
	}

	// Output path with nested directories that don't exist
	outputPDF := filepath.Join(tmpDir, "deeply", "nested", "path", "output.pdf")

	result, err := conv.ConvertRmdoc(testRmdoc, outputPDF)
	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	if !result.Success {
		t.Error("Conversion should succeed even with nested output path")
	}

	// Verify nested directories were created
	if _, err := os.Stat(filepath.Dir(outputPDF)); os.IsNotExist(err) {
		t.Error("Nested output directory should be created")
	}

	// Verify output file exists
	if _, err := os.Stat(outputPDF); os.IsNotExist(err) {
		t.Error("Output PDF should exist in nested directory")
	}
}

// TestConverterMetadataExtraction tests metadata extraction from .rmdoc
func TestConverterMetadataExtraction(t *testing.T) {
	testRmdoc := "../../testdata/rmdoc/Test.rmdoc"
	if _, err := os.Stat(testRmdoc); os.IsNotExist(err) {
		t.Skip("Test.rmdoc not found in testdata, skipping test")
	}

	tmpDir := t.TempDir()

	log, err := logger.New(&logger.Config{
		Level:  "info",
		Format: "console",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	conv, err := converter.New(&converter.Config{
		Logger: log,
	})
	if err != nil {
		t.Fatalf("Failed to create converter: %v", err)
	}

	outputPDF := filepath.Join(tmpDir, "test-metadata.pdf")

	result, err := conv.ConvertRmdoc(testRmdoc, outputPDF)
	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	// Verify conversion succeeded and has valid page count
	if !result.Success {
		t.Error("Conversion should succeed")
	}

	if result.PageCount <= 0 {
		t.Error("Result should have valid page count")
	}

	t.Logf("Conversion result - Pages: %d, Duration: %v", result.PageCount, result.Duration)
}
