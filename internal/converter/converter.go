// Package converter handles conversion of reMarkable .rmdoc files to PDF format.
package converter

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/platinummonkey/remarkable-sync/internal/logger"
)

// Converter handles conversion of .rmdoc files to PDF
type Converter struct {
	logger *logger.Logger
}

// Config holds configuration for the converter
type Config struct {
	Logger *logger.Logger
}

// New creates a new converter instance
func New(cfg *Config) *Converter {
	log := cfg.Logger
	if log == nil {
		log = logger.Get()
	}

	return &Converter{
		logger: log,
	}
}

// DocumentMetadata represents the .metadata JSON file from a .rmdoc
type DocumentMetadata struct {
	CreatedTime    string `json:"createdTime"`
	LastModified   string `json:"lastModified"`
	LastOpened     string `json:"lastOpened"`
	LastOpenedPage int    `json:"lastOpenedPage"`
	Parent         string `json:"parent"`
	Type           string `json:"type"`
	VisibleName    string `json:"visibleName"`
}

// ContentFile represents the .content JSON file from a .rmdoc
type ContentFile struct {
	FileType      string `json:"fileType"`
	PageCount     int    `json:"pageCount"`
	Orientation   string `json:"orientation"`
	FormatVersion int    `json:"formatVersion"`
	CPages        CPages `json:"cPages"`
}

// CPages represents the pages section of content file
type CPages struct {
	Pages []PageInfo `json:"pages"`
}

// PageInfo represents a single page's metadata
type PageInfo struct {
	ID       string `json:"id"`
	Modified string `json:"modifed"` // Note: typo in reMarkable format
	Template struct {
		Value string `json:"value"`
	} `json:"template"`
}

// ConvertRmdoc converts a .rmdoc file to PDF
func (c *Converter) ConvertRmdoc(rmdocPath, outputPath string) (*ConversionResult, error) {
	c.logger.WithFields("input", rmdocPath, "output", outputPath).Info("Converting .rmdoc to PDF")

	startTime := time.Now()
	result := NewConversionResult()

	// Extract .rmdoc (ZIP file) to temporary directory
	tmpDir, err := os.MkdirTemp("", "rmdoc-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	if err := c.extractRmdoc(rmdocPath, tmpDir); err != nil {
		return nil, fmt.Errorf("failed to extract .rmdoc: %w", err)
	}

	// Read metadata
	metadata, err := c.readMetadata(tmpDir)
	if err != nil {
		result.AddWarning(fmt.Sprintf("Failed to read metadata: %v", err))
		metadata = &DocumentMetadata{VisibleName: "Untitled"}
	}

	// Read content
	content, err := c.readContent(tmpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	c.logger.WithFields(
		"title", metadata.VisibleName,
		"pages", content.PageCount,
		"format", content.FormatVersion,
	).Debug("Extracted document metadata")

	// Convert pages to PDF
	if err := c.convertPages(tmpDir, content, outputPath); err != nil {
		return nil, fmt.Errorf("failed to convert pages: %w", err)
	}

	// Get output file size
	fileInfo, err := os.Stat(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat output file: %w", err)
	}

	duration := time.Since(startTime)
	result.SetSuccess(outputPath, content.PageCount, fileInfo.Size(), duration)
	c.logger.WithFields("output", outputPath, "pages", content.PageCount, "duration", duration).Info("Successfully converted .rmdoc to PDF")

	return result, nil
}

// extractRmdoc extracts a .rmdoc ZIP file to the specified directory
func (c *Converter) extractRmdoc(rmdocPath, destDir string) error {
	r, err := zip.OpenReader(rmdocPath)
	if err != nil {
		return fmt.Errorf("failed to open ZIP: %w", err)
	}
	defer func() { _ = r.Close() }()

	for _, f := range r.File {
		fpath := filepath.Join(destDir, f.Name)

		// Security: prevent path traversal
		if !strings.HasPrefix(fpath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path in archive: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fpath, os.ModePerm); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			continue
		}

		// Create parent directory
		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}

		// Extract file
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}

		rc, err := f.Open()
		if err != nil {
			_ = outFile.Close()
			return fmt.Errorf("failed to open file in archive: %w", err)
		}

		_, err = io.Copy(outFile, rc)
		_ = outFile.Close()
		_ = rc.Close()

		if err != nil {
			return fmt.Errorf("failed to extract file: %w", err)
		}
	}

	return nil
}

// readMetadata reads the .metadata JSON file
func (c *Converter) readMetadata(extractDir string) (*DocumentMetadata, error) {
	// Find .metadata file
	files, err := filepath.Glob(filepath.Join(extractDir, "*.metadata"))
	if err != nil || len(files) == 0 {
		return nil, fmt.Errorf("metadata file not found")
	}

	data, err := os.ReadFile(files[0])
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}

	var metadata DocumentMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata JSON: %w", err)
	}

	return &metadata, nil
}

// readContent reads the .content JSON file
func (c *Converter) readContent(extractDir string) (*ContentFile, error) {
	// Find .content file
	files, err := filepath.Glob(filepath.Join(extractDir, "*.content"))
	if err != nil || len(files) == 0 {
		return nil, fmt.Errorf("content file not found")
	}

	data, err := os.ReadFile(files[0])
	if err != nil {
		return nil, fmt.Errorf("failed to read content file: %w", err)
	}

	var content ContentFile
	if err := json.Unmarshal(data, &content); err != nil {
		return nil, fmt.Errorf("failed to parse content JSON: %w", err)
	}

	return &content, nil
}

// convertPages converts the .rm files to PDF pages
func (c *Converter) convertPages(extractDir string, content *ContentFile, outputPath string) error {
	c.logger.WithFields("pages", content.PageCount).Debug("Converting pages to PDF")

	// Find the directory containing .rm files
	var rmDir string
	entries, err := os.ReadDir(extractDir)
	if err != nil {
		return fmt.Errorf("failed to read extract directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() && !strings.HasSuffix(entry.Name(), ".metadata") && !strings.HasSuffix(entry.Name(), ".content") {
			rmDir = filepath.Join(extractDir, entry.Name())
			break
		}
	}

	if rmDir == "" {
		return fmt.Errorf(".rm files directory not found")
	}

	// For now, use a simple approach: create a basic PDF with blank pages
	// TODO: Implement actual .rm file rendering using go-remarkable2pdf or custom renderer
	if err := c.createPlaceholderPDF(outputPath, content.PageCount); err != nil {
		return fmt.Errorf("failed to create PDF: %w", err)
	}

	c.logger.WithFields("rm_dir", rmDir).Debug("Found .rm files directory")
	c.logger.Warn("Note: Currently creating placeholder PDF - full .rm rendering to be implemented")

	return nil
}

// createPlaceholderPDF creates a basic PDF with the specified number of blank pages
func (c *Converter) createPlaceholderPDF(outputPath string, pageCount int) error {
	// Create minimal PDF structure
	// reMarkable dimensions: 1404x1872 pixels at 226 DPI = 157.6 x 210.3 mm (close to A5)
	pdfContent := fmt.Sprintf(`%%PDF-1.4
1 0 obj
<<
/Type /Catalog
/Pages 2 0 R
>>
endobj
2 0 obj
<<
/Type /Pages
/Kids [%s]
/Count %d
>>
endobj`, buildPageRefs(pageCount), pageCount)

	// Add page objects
	for i := 0; i < pageCount; i++ {
		pageNum := i + 3
		pdfContent += fmt.Sprintf(`
%d 0 obj
<<
/Type /Page
/Parent 2 0 R
/MediaBox [0 0 420 595]
/Contents %d 0 R
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
endobj`, pageNum, pageNum+pageCount)
	}

	// Add content streams
	for i := 0; i < pageCount; i++ {
		contentNum := i + 3 + pageCount
		text := fmt.Sprintf("Page %d (Placeholder)", i+1)
		stream := fmt.Sprintf("BT\n/F1 12 Tf\n50 550 Td\n(%s) Tj\nET", text)
		pdfContent += fmt.Sprintf(`
%d 0 obj
<<
/Length %d
>>
stream
%s
endstream
endobj`, contentNum, len(stream), stream)
	}

	// Add xref and trailer
	xrefOffset := len(pdfContent)
	objCount := 3 + (pageCount * 2)

	pdfContent += fmt.Sprintf(`
xref
0 %d
0000000000 65535 f
0000000009 00000 n
0000000058 00000 n
`, objCount)

	// Add xref entries for pages and contents
	// This is simplified - actual offsets would need to be calculated
	for i := 0; i < pageCount*2; i++ {
		pdfContent += "0000000000 00000 n\n"
	}

	pdfContent += fmt.Sprintf(`trailer
<<
/Size %d
/Root 1 0 R
>>
startxref
%d
%%%%EOF`, objCount, xrefOffset)

	return os.WriteFile(outputPath, []byte(pdfContent), 0644)
}

// buildPageRefs builds the page reference array for the PDF
func buildPageRefs(pageCount int) string {
	refs := make([]string, pageCount)
	for i := 0; i < pageCount; i++ {
		refs[i] = fmt.Sprintf("%d 0 R", i+3)
	}
	return strings.Join(refs, " ")
}

// parseTimestamp converts reMarkable timestamp (milliseconds since epoch) to time.Time
func parseTimestamp(ts string) time.Time {
	if ts == "" {
		return time.Time{}
	}

	// reMarkable timestamps are in milliseconds
	var ms int64
	if _, err := fmt.Sscanf(ts, "%d", &ms); err != nil {
		return time.Time{}
	}
	return time.Unix(ms/1000, (ms%1000)*1000000)
}
