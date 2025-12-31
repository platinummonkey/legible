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

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/platinummonkey/remarkable-sync/internal/logger"
	"github.com/platinummonkey/remarkable-sync/internal/ocr"
	"github.com/platinummonkey/remarkable-sync/internal/pdfenhancer"
	"github.com/platinummonkey/remarkable-sync/internal/rmparse"
	"github.com/signintech/gopdf"
)

// Converter handles conversion of .rmdoc files to PDF
type Converter struct {
	logger       *logger.Logger
	ocrEnabled   bool
	ocrLanguages []string
	ocrProc      *ocr.Processor
	pdfEnhancer  *pdfenhancer.PDFEnhancer
}

// Config holds configuration for the converter
type Config struct {
	Logger       *logger.Logger
	EnableOCR    bool     // Enable OCR text layer (default: true)
	OCRLanguages []string // Language codes for OCR via Ollama (default: ["eng"])
}

// New creates a new converter instance
func New(cfg *Config) *Converter {
	log := cfg.Logger
	if log == nil {
		log = logger.Get()
	}

	// Enable OCR by default
	enableOCR := cfg.EnableOCR
	if cfg == nil || (!cfg.EnableOCR && cfg.OCRLanguages == nil) {
		// If not explicitly configured, enable by default
		enableOCR = true
	}

	// Set default languages
	languages := cfg.OCRLanguages
	if len(languages) == 0 {
		languages = []string{"eng"}
	}

	// Create OCR processor if enabled
	var ocrProc *ocr.Processor
	var pdfEnhancer *pdfenhancer.PDFEnhancer
	if enableOCR {
		ocrProc = ocr.New(&ocr.Config{
			Logger: log,
			// Ollama handles language detection automatically via vision models
		})
		pdfEnhancer = pdfenhancer.New(&pdfenhancer.Config{
			Logger: log,
		})
	}

	return &Converter{
		logger:       log,
		ocrEnabled:   enableOCR,
		ocrLanguages: languages,
		ocrProc:      ocrProc,
		pdfEnhancer:  pdfEnhancer,
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
	FileType      string    `json:"fileType"`
	PageCount     int       `json:"pageCount"`
	Orientation   string    `json:"orientation"`
	FormatVersion int       `json:"formatVersion"`
	CPages        CPages    `json:"cPages"`
	Tags          []string  `json:"tags"`
	PageTags      []PageTag `json:"pageTags"`
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

// PageTag represents a tag associated with a specific page
type PageTag struct {
	Name      string `json:"name"`
	PageID    string `json:"pageId"`
	Timestamp int64  `json:"timestamp"`
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

	// Extract tags and add PDF metadata
	tags := c.extractTags(content)
	if len(tags) > 0 {
		if err := c.addPDFMetadata(outputPath, metadata, tags); err != nil {
			result.AddWarning(fmt.Sprintf("Failed to add PDF metadata: %v", err))
		} else {
			c.logger.WithFields("tags", tags).Info("Added PDF metadata with tags")
		}
	}

	// Add OCR text layer if enabled
	if c.ocrEnabled {
		if err := c.addOCRTextLayer(outputPath, content.PageCount, result); err != nil {
			result.AddWarning(fmt.Sprintf("Failed to add OCR text layer: %v", err))
			c.logger.WithFields("error", err).Warn("OCR processing failed, continuing without text layer")
		} else {
			c.logger.WithFields(
				"word_count", result.OCRWordCount,
				"confidence", result.OCRConfidence,
				"duration", result.OCRDuration,
			).Info("Successfully added OCR text layer")
		}
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

	c.logger.WithFields("rm_dir", rmDir).Debug("Found .rm files directory")

	// Create PDF with rendered pages
	if err := c.renderPagesToPDF(rmDir, content, outputPath); err != nil {
		return fmt.Errorf("failed to render pages: %w", err)
	}

	return nil
}

// renderPagesToPDF renders .rm files to PDF pages
func (c *Converter) renderPagesToPDF(rmDir string, content *ContentFile, outputPath string) error {
	// Initialize PDF
	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{
		PageSize: gopdf.Rect{W: rmparse.PDFWidth, H: rmparse.PDFHeight},
	})

	// Process each page in order
	for i, pageInfo := range content.CPages.Pages {
		c.logger.WithFields("page", i+1, "id", pageInfo.ID).Debug("Rendering page")

		// Add new page
		pdf.AddPage()

		// Find corresponding .rm file
		rmPath := filepath.Join(rmDir, pageInfo.ID+".rm")
		if _, err := os.Stat(rmPath); os.IsNotExist(err) {
			c.logger.WithFields("page", i+1, "path", rmPath).Warn("Page .rm file not found, skipping")
			continue
		}

		// Parse .rm file
		rmFile, err := rmparse.ParseRM(rmPath)
		if err != nil {
			c.logger.WithFields("page", i+1, "error", err).Warn("Failed to parse .rm file, skipping")
			continue
		}

		// Render to current page
		if err := rmparse.RenderToPage(&pdf, rmFile); err != nil {
			c.logger.WithFields("page", i+1, "error", err).Warn("Failed to render page, continuing")
			// Continue with blank page
		}

		c.logger.WithFields("page", i+1, "layers", len(rmFile.Layers)).Debug("Successfully rendered page")
	}

	// Write PDF to output file
	if err := pdf.WritePdf(outputPath); err != nil {
		return fmt.Errorf("failed to write PDF: %w", err)
	}

	return nil
}

// createPlaceholderPDF creates a valid PDF with the specified number of blank pages using pdfcpu
func (c *Converter) createPlaceholderPDF(outputPath string, pageCount int) error {
	// reMarkable tablet dimensions: 1404x1872 pixels at 226 DPI
	// This corresponds to approximately 157.6 x 210.3 mm (close to A5)
	// In PDF points (1/72 inch): 446.7 x 595.3 points
	// We'll use A5 dimensions: 420 x 595 points (width x height)

	// Create configuration
	conf := model.NewDefaultConfiguration()

	// Create a temporary input PDF with a single blank page first
	tmpFile, err := os.CreateTemp("", "blank-*.pdf")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()
	defer os.Remove(tmpPath)

	// Create a minimal valid PDF with one blank page
	// This uses proper PDF structure with correct byte offsets
	// Calculate byte offsets:
	// Header: %PDF-1.4\n = 9 bytes (offset 0-8)
	// Object 1 starts at byte 9
	// Object 2 starts after obj 1
	// Object 3 starts after obj 2
	minimalPDF := `%PDF-1.4
1 0 obj
<</Type/Catalog/Pages 2 0 R>>
endobj
2 0 obj
<</Type/Pages/Count 1/Kids[3 0 R]>>
endobj
3 0 obj
<</Type/Page/Parent 2 0 R/MediaBox[0 0 420 595]/Resources<<>>>>
endobj
xref
0 4
0000000000 65535 f
0000000009 00000 n
0000000058 00000 n
0000000113 00000 n
trailer
<</Size 4/Root 1 0 R>>
startxref
190
%%EOF`

	if err := os.WriteFile(tmpPath, []byte(minimalPDF), 0644); err != nil {
		return fmt.Errorf("failed to write temp PDF: %w", err)
	}

	// Now replicate this page N times using pdfcpu's MergeCreateFile
	if pageCount == 1 {
		// Just copy the single page
		return os.Rename(tmpPath, outputPath)
	}

	// For multiple pages, merge the same page multiple times
	inFiles := make([]string, pageCount)
	for i := 0; i < pageCount; i++ {
		inFiles[i] = tmpPath
	}

	// Use pdfcpu to merge pages
	if err := api.MergeCreateFile(inFiles, outputPath, false, conf); err != nil {
		return fmt.Errorf("failed to merge pages: %w", err)
	}

	return nil
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

// extractTags extracts all tags from the content file (document-level and page-level)
func (c *Converter) extractTags(content *ContentFile) []string {
	tagSet := make(map[string]bool)

	// Add document-level tags
	for _, tag := range content.Tags {
		if tag != "" {
			tagSet[tag] = true
		}
	}

	// Add page-level tags
	for _, pageTag := range content.PageTags {
		if pageTag.Name != "" {
			tagSet[pageTag.Name] = true
		}
	}

	// Convert map to slice
	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, tag)
	}

	return tags
}

// addOCRTextLayer performs OCR on the PDF and adds a searchable text layer
func (c *Converter) addOCRTextLayer(pdfPath string, pageCount int, result *ConversionResult) error {
	c.logger.WithFields("pdf", pdfPath, "pages", pageCount).Info("Starting OCR processing")

	ocrStartTime := time.Now()

	// Render PDF pages to images for OCR
	// Use 300 DPI for good OCR accuracy
	const ocrDPI = 300
	images, err := c.renderAllPagesToImages(pdfPath, ocrDPI)
	if err != nil {
		return fmt.Errorf("failed to render PDF pages: %w", err)
	}

	// Get PDF page dimensions for coordinate scaling
	pdfEnhancer := pdfenhancer.New(&pdfenhancer.Config{Logger: c.logger})
	pageInfo, err := pdfEnhancer.ExtractPageInfo(pdfPath, 1)
	if err != nil {
		return fmt.Errorf("failed to get page dimensions: %w", err)
	}

	// Create document OCR result
	docOCR := ocr.NewDocumentOCR("", strings.Join(c.ocrLanguages, "+"))

	// Process each page with OCR
	for i, img := range images {
		pageNum := i + 1
		c.logger.WithFields("page", pageNum, "total", pageCount).Debug("Processing page with OCR")

		// Convert image to bytes
		imageData, err := c.imageToBytes(img)
		if err != nil {
			c.logger.WithFields("page", pageNum, "error", err).Warn("Failed to convert image to bytes, skipping OCR for this page")
			continue
		}

		// Process with OCR
		pageOCR, err := c.ocrProc.ProcessImage(imageData, pageNum)
		if err != nil {
			c.logger.WithFields("page", pageNum, "error", err).Warn("Failed to process page with OCR, skipping")
			continue
		}

		// Scale OCR coordinates from image pixels to PDF points
		// Image was rendered at ocrDPI, so pixels -> PDF points conversion is:
		// pdfPoint = imagePixel * 72 / ocrDPI
		bounds := img.Bounds()
		imageWidth := bounds.Dx()
		imageHeight := bounds.Dy()
		scaleX := float64(pageInfo.Width) / float64(imageWidth)
		scaleY := float64(pageInfo.Height) / float64(imageHeight)

		// Scale all word bounding boxes
		for j := range pageOCR.Words {
			word := &pageOCR.Words[j]
			word.BoundingBox.X = int(float64(word.BoundingBox.X) * scaleX)
			word.BoundingBox.Y = int(float64(word.BoundingBox.Y) * scaleY)
			word.BoundingBox.Width = int(float64(word.BoundingBox.Width) * scaleX)
			word.BoundingBox.Height = int(float64(word.BoundingBox.Height) * scaleY)
		}

		// Update page dimensions to match PDF
		pageOCR.Width = pageInfo.Width
		pageOCR.Height = pageInfo.Height

		// Add to document results
		docOCR.AddPage(*pageOCR)
		c.logger.WithFields(
			"page", pageNum,
			"words", len(pageOCR.Words),
			"confidence", pageOCR.Confidence,
			"scale", fmt.Sprintf("%.3fx%.3f", scaleX, scaleY),
		).Debug("Completed OCR for page")
	}

	// Finalize document OCR statistics
	docOCR.Finalize()

	// Create temporary enhanced PDF
	tmpFile, err := os.CreateTemp(filepath.Dir(pdfPath), "enhanced-*.pdf")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// Add text layer to PDF
	if err := c.pdfEnhancer.AddTextLayer(pdfPath, tmpPath, docOCR); err != nil {
		return fmt.Errorf("failed to add text layer: %w", err)
	}

	// Replace original with enhanced PDF
	if err := os.Rename(tmpPath, pdfPath); err != nil {
		return fmt.Errorf("failed to replace PDF with enhanced version: %w", err)
	}

	ocrDuration := time.Since(ocrStartTime)

	// Update result with OCR statistics
	result.OCREnabled = true
	result.OCRWordCount = docOCR.TotalWords
	result.OCRConfidence = docOCR.AverageConfidence
	result.OCRDuration = ocrDuration

	return nil
}

// addPDFMetadata adds metadata to the PDF file using pdfcpu
func (c *Converter) addPDFMetadata(pdfPath string, metadata *DocumentMetadata, tags []string) error {
	// Prepare metadata properties
	properties := map[string]string{
		"Creator":  "remarkable-sync",
		"Producer": "remarkable-sync",
	}

	// Add title
	if metadata.VisibleName != "" {
		properties["Title"] = metadata.VisibleName
	}

	// Add subject (tags)
	if len(tags) > 0 {
		properties["Subject"] = strings.Join(tags, ", ")
	}

	// Add creation date
	if metadata.CreatedTime != "" {
		if createdTime := parseTimestamp(metadata.CreatedTime); !createdTime.IsZero() {
			properties["CreationDate"] = createdTime.Format("D:20060102150405")
		}
	}

	// Create temp file for output
	tmpFile, err := os.CreateTemp(filepath.Dir(pdfPath), "pdf-metadata-*.pdf")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()
	defer os.Remove(tmpPath)

	// Add properties
	conf := model.NewDefaultConfiguration()
	if err := api.AddPropertiesFile(pdfPath, tmpPath, properties, conf); err != nil {
		return fmt.Errorf("failed to add properties: %w", err)
	}

	// Replace original file
	if err := os.Rename(tmpPath, pdfPath); err != nil {
		return fmt.Errorf("failed to replace original file: %w", err)
	}

	return nil
}

