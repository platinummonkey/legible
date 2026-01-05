package converter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func TestNew(t *testing.T) {
	cfg := &Config{}
	converter, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	if converter == nil {
		t.Fatal("New() returned nil")
	}

	if converter.logger == nil {
		t.Error("logger should be initialized")
	}
}

func TestConvertRmdoc_FileNotFound(t *testing.T) {
	converter, err := New(&Config{})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.pdf")

	_, err = converter.ConvertRmdoc("/nonexistent/file.rmdoc", outputPath)
	if err == nil {
		t.Error("ConvertRmdoc() should error for nonexistent file")
	}
}

func TestConvertRmdoc_WithTestFile(t *testing.T) {
	converter, err := New(&Config{})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

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
	converter, err := New(&Config{})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	rmdocPath := "../../example/Test.rmdoc"
	if _, statErr := os.Stat(rmdocPath); os.IsNotExist(statErr) {
		t.Skipf("Test file not found: %s", rmdocPath)
	}

	tmpDir := t.TempDir()

	err = converter.extractRmdoc(rmdocPath, tmpDir)
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
	converter, err := New(&Config{})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}


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
	converter, err := New(&Config{})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}


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
	converter, err := New(&Config{})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}


	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test.pdf")

	err = converter.createPlaceholderPDF(outputPath, 3)
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

func TestExtractTags(t *testing.T) {
	converter, err := New(&Config{})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}


	tests := []struct {
		name     string
		content  *ContentFile
		expected []string
	}{
		{
			name: "document-level tags only",
			content: &ContentFile{
				Tags:     []string{"work", "important"},
				PageTags: []PageTag{},
			},
			expected: []string{"work", "important"},
		},
		{
			name: "page-level tags only",
			content: &ContentFile{
				Tags: []string{},
				PageTags: []PageTag{
					{Name: "test", PageID: "page1"},
					{Name: "draft", PageID: "page2"},
				},
			},
			expected: []string{"test", "draft"},
		},
		{
			name: "both document and page tags",
			content: &ContentFile{
				Tags: []string{"work"},
				PageTags: []PageTag{
					{Name: "test", PageID: "page1"},
				},
			},
			expected: []string{"work", "test"},
		},
		{
			name: "duplicate tags",
			content: &ContentFile{
				Tags: []string{"test"},
				PageTags: []PageTag{
					{Name: "test", PageID: "page1"},
				},
			},
			expected: []string{"test"},
		},
		{
			name: "empty tags",
			content: &ContentFile{
				Tags:     []string{},
				PageTags: []PageTag{},
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.extractTags(tt.content)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d tags, got %d", len(tt.expected), len(result))
			}

			// Convert to map for easier comparison (order doesn't matter)
			resultMap := make(map[string]bool)
			for _, tag := range result {
				resultMap[tag] = true
			}

			for _, expectedTag := range tt.expected {
				if !resultMap[expectedTag] {
					t.Errorf("expected tag '%s' not found in result", expectedTag)
				}
			}
		})
	}
}

func TestExtractTags_RealDocument(t *testing.T) {
	converter, err := New(&Config{})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}


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

	tags := converter.extractTags(content)

	// The Test.rmdoc has a "test" tag on page 2
	if len(tags) == 0 {
		t.Error("expected at least one tag from Test.rmdoc")
	}

	foundTestTag := false
	for _, tag := range tags {
		if tag == "test" {
			foundTestTag = true
			break
		}
	}

	if !foundTestTag {
		t.Error("expected 'test' tag from Test.rmdoc, got:", tags)
	}
}

func TestReadContent_WithTags(t *testing.T) {
	converter, err := New(&Config{})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}


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

	// Verify PageTags field is populated
	if len(content.PageTags) == 0 {
		t.Error("expected PageTags to be populated from Test.rmdoc")
	}

	// The Test.rmdoc has a "test" page tag
	foundTestTag := false
	for _, pageTag := range content.PageTags {
		if pageTag.Name == "test" {
			foundTestTag = true
			if pageTag.PageID == "" {
				t.Error("PageTag should have a PageID")
			}
			break
		}
	}

	if !foundTestTag {
		t.Error("expected 'test' page tag from Test.rmdoc")
	}
}

func TestConvertRmdoc_PDFMetadata(t *testing.T) {
	converter, err := New(&Config{})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}


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

	if !result.Success {
		t.Fatal("ConvertRmdoc() should succeed")
	}

	// Verify PDF metadata using pdfcpu PDFInfo
	pdfFile, err := os.Open(outputPath)
	if err != nil {
		t.Fatalf("failed to open PDF: %v", err)
	}
	defer func() { _ = pdfFile.Close() }()

	pdfInfo, err := api.PDFInfo(pdfFile, outputPath, nil, false, model.NewDefaultConfiguration())
	if err != nil {
		t.Fatalf("failed to read PDF info: %v", err)
	}

	// Debug: print PDF info
	t.Logf("PDF Info - Title: %s, Subject: %s, Creator: %s", pdfInfo.Title, pdfInfo.Subject, pdfInfo.Creator)

	// Check for title
	if pdfInfo.Title != "Test" {
		t.Errorf("expected Title 'Test', got '%s'", pdfInfo.Title)
	}

	// Check for subject (tags are stored in Subject field since gopdf doesn't have Keywords)
	if !strings.Contains(pdfInfo.Subject, "test") {
		t.Errorf("expected Subject to contain 'test', got '%s'", pdfInfo.Subject)
	}

	// Check for creator
	if pdfInfo.Creator != "legible" {
		t.Errorf("expected Creator 'legible', got '%s'", pdfInfo.Creator)
	}
}
