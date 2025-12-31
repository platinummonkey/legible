package converter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	cfg := &Config{}
	converter := New(cfg)

	if converter == nil {
		t.Fatal("New() returned nil")
	}

	if converter.logger == nil {
		t.Error("logger should be initialized")
	}
}

func TestConvertRmdoc_FileNotFound(t *testing.T) {
	converter := New(&Config{})

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.pdf")

	_, err := converter.ConvertRmdoc("/nonexistent/file.rmdoc", outputPath)
	if err == nil {
		t.Error("ConvertRmdoc() should error for nonexistent file")
	}
}

func TestConvertRmdoc_WithTestFile(t *testing.T) {
	converter := New(&Config{})

	// Use the example Test.rmdoc file from the repository
	rmdocPath := "../../example/Test.rmdoc"
	if _, err := os.Stat(rmdocPath); os.IsNotExist(err) {
		t.Skipf("Test file not found: %s", rmdocPath)
	}

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.pdf")

	result, err := converter.ConvertRmdoc(rmdocPath, outputPath)
	if err != nil {
		t.Fatalf("ConvertRmdoc() error = %v", err)
	}

	if result == nil {
		t.Fatal("ConvertRmdoc() returned nil result")
	}

	if !result.Success {
		t.Error("ConvertRmdoc() result.Success should be true")
	}

	if result.PageCount != 2 {
		t.Errorf("Expected 2 pages, got %d", result.PageCount)
	}

	if result.OutputPath != outputPath {
		t.Errorf("Expected outputPath '%s', got '%s'", outputPath, result.OutputPath)
	}

	// Verify output PDF was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("output PDF should exist")
	}

	// Check file size is reasonable (at least 100 bytes)
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("failed to stat output PDF: %v", err)
	}

	if info.Size() < 100 {
		t.Errorf("output PDF size too small: %d bytes", info.Size())
	}
}

func TestExtractRmdoc(t *testing.T) {
	converter := New(&Config{})

	rmdocPath := "../../example/Test.rmdoc"
	if _, err := os.Stat(rmdocPath); os.IsNotExist(err) {
		t.Skipf("Test file not found: %s", rmdocPath)
	}

	tmpDir := t.TempDir()

	err := converter.extractRmdoc(rmdocPath, tmpDir)
	if err != nil {
		t.Fatalf("extractRmdoc() error = %v", err)
	}

	// Check that metadata file exists
	metadataFiles, err := filepath.Glob(filepath.Join(tmpDir, "*.metadata"))
	if err != nil || len(metadataFiles) == 0 {
		t.Error("metadata file should exist after extraction")
	}

	// Check that content file exists
	contentFiles, err := filepath.Glob(filepath.Join(tmpDir, "*.content"))
	if err != nil || len(contentFiles) == 0 {
		t.Error("content file should exist after extraction")
	}

	// Check that .rm directory exists
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("failed to read temp directory: %v", err)
	}

	foundDir := false
	for _, entry := range entries {
		if entry.IsDir() {
			foundDir = true
			break
		}
	}

	if !foundDir {
		t.Error(".rm files directory should exist after extraction")
	}
}

func TestReadMetadata(t *testing.T) {
	converter := New(&Config{})

	rmdocPath := "../../example/Test.rmdoc"
	if _, err := os.Stat(rmdocPath); os.IsNotExist(err) {
		t.Skipf("Test file not found: %s", rmdocPath)
	}

	tmpDir := t.TempDir()
	if err := converter.extractRmdoc(rmdocPath, tmpDir); err != nil {
		t.Fatalf("failed to extract: %v", err)
	}

	metadata, err := converter.readMetadata(tmpDir)
	if err != nil {
		t.Fatalf("readMetadata() error = %v", err)
	}

	if metadata.VisibleName != "Test" {
		t.Errorf("expected VisibleName 'Test', got '%s'", metadata.VisibleName)
	}

	if metadata.Type != "DocumentType" {
		t.Errorf("expected Type 'DocumentType', got '%s'", metadata.Type)
	}

	if metadata.CreatedTime == "" {
		t.Error("CreatedTime should not be empty")
	}
}

func TestReadContent(t *testing.T) {
	converter := New(&Config{})

	rmdocPath := "../../example/Test.rmdoc"
	if _, err := os.Stat(rmdocPath); os.IsNotExist(err) {
		t.Skipf("Test file not found: %s", rmdocPath)
	}

	tmpDir := t.TempDir()
	if err := converter.extractRmdoc(rmdocPath, tmpDir); err != nil {
		t.Fatalf("failed to extract: %v", err)
	}

	content, err := converter.readContent(tmpDir)
	if err != nil {
		t.Fatalf("readContent() error = %v", err)
	}

	if content.FileType != "notebook" {
		t.Errorf("expected FileType 'notebook', got '%s'", content.FileType)
	}

	if content.PageCount != 2 {
		t.Errorf("expected PageCount 2, got %d", content.PageCount)
	}

	if content.Orientation != "portrait" {
		t.Errorf("expected Orientation 'portrait', got '%s'", content.Orientation)
	}

	if len(content.CPages.Pages) != 2 {
		t.Errorf("expected 2 pages, got %d", len(content.CPages.Pages))
	}
}

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantZero bool
	}{
		{
			name:     "valid timestamp",
			input:    "1767048603250",
			wantZero: false,
		},
		{
			name:     "empty timestamp",
			input:    "",
			wantZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTimestamp(tt.input)

			if tt.wantZero && !result.IsZero() {
				t.Error("expected zero time for empty timestamp")
			}

			if !tt.wantZero && result.IsZero() {
				t.Error("expected non-zero time for valid timestamp")
			}
		})
	}
}

func TestCreatePlaceholderPDF(t *testing.T) {
	converter := New(&Config{})

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test.pdf")

	err := converter.createPlaceholderPDF(outputPath, 3)
	if err != nil {
		t.Fatalf("createPlaceholderPDF() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("output PDF should exist")
	}

	// Read and verify it starts with PDF header
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output PDF: %v", err)
	}

	if len(data) < 8 {
		t.Error("PDF file too short")
	}

	// Check for valid PDF header (should start with %PDF-)
	if !strings.HasPrefix(string(data), "%PDF-") {
		t.Error("PDF should start with %PDF- header")
	}
}
