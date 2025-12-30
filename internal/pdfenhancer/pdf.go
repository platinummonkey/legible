package pdfenhancer

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/platinummonkey/remarkable-sync/internal/logger"
	"github.com/platinummonkey/remarkable-sync/internal/ocr"
)

// PDFEnhancer provides utilities for reading and enhancing PDF files
type PDFEnhancer struct {
	logger *logger.Logger
}

// Config holds configuration for the PDF enhancer
type Config struct {
	Logger *logger.Logger
}

// New creates a new PDF enhancer instance
func New(cfg *Config) *PDFEnhancer {
	log := cfg.Logger
	if log == nil {
		log = logger.Get()
	}

	return &PDFEnhancer{
		logger: log,
	}
}

// GetPageCount returns the number of pages in a PDF file
func (pe *PDFEnhancer) GetPageCount(pdfPath string) (int, error) {
	pe.logger.WithFields("pdf_path", pdfPath).Debug("Getting page count")

	// Read PDF context
	ctx, err := api.ReadContextFile(pdfPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read PDF: %w", err)
	}

	pageCount := ctx.PageCount
	pe.logger.WithFields("page_count", pageCount).Debug("Retrieved page count")

	return pageCount, nil
}

// ValidatePDF checks if a file is a valid PDF
func (pe *PDFEnhancer) ValidatePDF(pdfPath string) error {
	pe.logger.WithFields("pdf_path", pdfPath).Debug("Validating PDF")

	// Check if file exists
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		return fmt.Errorf("PDF file does not exist: %s", pdfPath)
	}

	// Try to read the PDF
	_, err := api.ReadContextFile(pdfPath)
	if err != nil {
		return fmt.Errorf("invalid PDF file: %w", err)
	}

	pe.logger.Debug("PDF validation successful")
	return nil
}

// AddTextLayer adds an invisible OCR text layer to a PDF
// This makes the PDF searchable while preserving the original appearance
func (pe *PDFEnhancer) AddTextLayer(inputPath, outputPath string, ocrResults *ocr.DocumentOCR) error {
	pe.logger.WithFields("input", inputPath, "output", outputPath).Info("Adding text layer to PDF")

	if ocrResults == nil {
		return fmt.Errorf("OCR results cannot be nil")
	}

	// Read the input PDF
	ctx, err := api.ReadContextFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input PDF: %w", err)
	}

	// Verify page count matches
	if len(ocrResults.Pages) != ctx.PageCount {
		return fmt.Errorf("OCR page count (%d) does not match PDF page count (%d)",
			len(ocrResults.Pages), ctx.PageCount)
	}

	// Add text layer to each page
	for i, pageOCR := range ocrResults.Pages {
		pageNum := i + 1
		pe.logger.WithFields("page", pageNum).Debug("Adding text layer to page")

		if err := pe.addTextToPage(ctx, pageNum, &pageOCR); err != nil {
			return fmt.Errorf("failed to add text to page %d: %w", pageNum, err)
		}
	}

	// Write the enhanced PDF
	if err := api.WriteContextFile(ctx, outputPath); err != nil {
		return fmt.Errorf("failed to write output PDF: %w", err)
	}

	pe.logger.WithFields("output", outputPath).Info("Successfully added text layer to PDF")
	return nil
}

// addTextToPage adds OCR text to a specific page
func (pe *PDFEnhancer) addTextToPage(ctx *model.Context, pageNum int, pageOCR *ocr.PageOCR) error {
	// Get the page dictionary and inherited attributes
	pageDict, _, inheritedAttrs, err := ctx.PageDict(pageNum, false)
	if err != nil {
		return fmt.Errorf("failed to get page dictionary: %w", err)
	}

	if pageDict == nil {
		return fmt.Errorf("page dictionary is nil")
	}

	// Get page dimensions for coordinate conversion
	if inheritedAttrs == nil || inheritedAttrs.MediaBox == nil {
		return fmt.Errorf("page has no media box")
	}
	pageHeight := inheritedAttrs.MediaBox.Height()

	// Skip if no words to add
	if len(pageOCR.Words) == 0 {
		pe.logger.WithFields("page", pageNum).Debug("No OCR words to add, skipping")
		return nil
	}

	// Create content stream with invisible text
	contentStream, err := pe.createTextContentStream(pageOCR, pageHeight)
	if err != nil {
		return fmt.Errorf("failed to create content stream: %w", err)
	}

	// Add content stream to page
	if err := pe.appendContentStream(ctx, pageDict, contentStream); err != nil {
		return fmt.Errorf("failed to append content stream: %w", err)
	}

	pe.logger.WithFields("page", pageNum, "word_count", len(pageOCR.Words)).
		Debug("Successfully added text layer")

	return nil
}

// createTextContentStream generates a PDF content stream with invisible text
func (pe *PDFEnhancer) createTextContentStream(pageOCR *ocr.PageOCR, pageHeight float64) ([]byte, error) {
	var buf bytes.Buffer

	// Start with graphics state save
	buf.WriteString("q\n")

	// Begin text object
	buf.WriteString("BT\n")

	// Set font (Helvetica, 10pt) - standard PDF font, no embedding needed
	buf.WriteString("/Helvetica 10 Tf\n")

	// Set text rendering mode to invisible (Tr 3 = no fill, no stroke)
	buf.WriteString("3 Tr\n")

	// Add each word with its position
	for _, word := range pageOCR.Words {
		// Skip empty words
		if strings.TrimSpace(word.Text) == "" {
			continue
		}

		// Convert OCR coordinates (top-left origin) to PDF coordinates (bottom-left origin)
		// OCR: (0,0) is top-left, Y increases downward
		// PDF: (0,0) is bottom-left, Y increases upward
		pdfX := float64(word.BoundingBox.X)
		pdfY := pageHeight - float64(word.BoundingBox.Y) - float64(word.BoundingBox.Height)

		// Escape text for PDF string
		escapedText := pe.escapePDFString(word.Text)

		// Position text using Tm (text matrix) operator
		// [a b c d e f] Tm - we use simple translation: [1 0 0 1 x y]
		buf.WriteString(fmt.Sprintf("1 0 0 1 %.2f %.2f Tm\n", pdfX, pdfY))

		// Show text using Tj operator
		buf.WriteString(fmt.Sprintf("(%s) Tj\n", escapedText))
	}

	// End text object
	buf.WriteString("ET\n")

	// Restore graphics state
	buf.WriteString("Q\n")

	return buf.Bytes(), nil
}

// escapePDFString escapes special characters in a PDF string literal
func (pe *PDFEnhancer) escapePDFString(s string) string {
	// Escape backslash, parentheses, and other special characters
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

// appendContentStream adds a content stream to an existing page
func (pe *PDFEnhancer) appendContentStream(ctx *model.Context, pageDict types.Dict, contentData []byte) error {
	// Create a new stream dictionary for our content
	streamDict := types.NewDict()
	streamDict.Insert("Length", types.Integer(len(contentData)))

	// Create stream object
	sd := &types.StreamDict{
		Dict:    streamDict,
		Content: contentData,
	}

	// Add stream to context and get indirect reference
	indRef, err := ctx.IndRefForNewObject(*sd)
	if err != nil {
		return fmt.Errorf("failed to create indirect reference: %w", err)
	}

	// Get existing Contents entry
	contentsEntry := pageDict.Entry("Contents")
	if contentsEntry == nil {
		// No existing contents, set our stream as the only content
		pageDict.Update("Contents", *indRef)
		return nil
	}

	// Handle existing contents
	switch contents := contentsEntry.(type) {
	case types.IndirectRef:
		// Single content stream - convert to array and append
		arr := types.Array{contents, *indRef}
		pageDict.Update("Contents", arr)

	case types.Array:
		// Multiple content streams - append to array
		contents = append(contents, *indRef)
		pageDict.Update("Contents", contents)

	default:
		return fmt.Errorf("unexpected Contents type: %T", contents)
	}

	return nil
}

// OptimizePDF optimizes a PDF file by compressing and removing unnecessary data
func (pe *PDFEnhancer) OptimizePDF(inputPath, outputPath string) error {
	pe.logger.WithFields("input", inputPath, "output", outputPath).Info("Optimizing PDF")

	// Use pdfcpu's optimize functionality
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed

	if err := api.OptimizeFile(inputPath, outputPath, conf); err != nil {
		return fmt.Errorf("failed to optimize PDF: %w", err)
	}

	pe.logger.Info("PDF optimization successful")
	return nil
}

// ExtractPageInfo extracts basic information about a PDF page
func (pe *PDFEnhancer) ExtractPageInfo(pdfPath string, pageNum int) (*PageInfo, error) {
	pe.logger.WithFields("pdf_path", pdfPath, "page", pageNum).Debug("Extracting page info")

	ctx, err := api.ReadContextFile(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF: %w", err)
	}

	if pageNum < 1 || pageNum > ctx.PageCount {
		return nil, fmt.Errorf("invalid page number %d (PDF has %d pages)", pageNum, ctx.PageCount)
	}

	// Get page dictionary and inherited attributes
	_, _, inheritedAttrs, err := ctx.PageDict(pageNum, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get page dictionary: %w", err)
	}

	// Get media box (page dimensions) from inherited attributes
	if inheritedAttrs == nil || inheritedAttrs.MediaBox == nil {
		return nil, fmt.Errorf("page has no media box")
	}
	mediaBox := inheritedAttrs.MediaBox

	info := &PageInfo{
		PageNumber: pageNum,
		Width:      int(mediaBox.Width()),
		Height:     int(mediaBox.Height()),
	}

	pe.logger.WithFields("width", info.Width, "height", info.Height).Debug("Extracted page info")
	return info, nil
}

// PageInfo contains basic information about a PDF page
type PageInfo struct {
	PageNumber int
	Width      int
	Height     int
}

// MergePDFs merges multiple PDF files into a single output file
func (pe *PDFEnhancer) MergePDFs(inputPaths []string, outputPath string) error {
	if len(inputPaths) == 0 {
		return fmt.Errorf("no input files provided")
	}

	pe.logger.WithFields("input_count", len(inputPaths), "output", outputPath).Info("Merging PDFs")

	conf := model.NewDefaultConfiguration()
	if err := api.MergeCreateFile(inputPaths, outputPath, false, conf); err != nil {
		return fmt.Errorf("failed to merge PDFs: %w", err)
	}

	pe.logger.Info("PDF merge successful")
	return nil
}

// SplitPDF splits a PDF into individual pages
func (pe *PDFEnhancer) SplitPDF(inputPath, outputDir string) error {
	pe.logger.WithFields("input", inputPath, "output_dir", outputDir).Info("Splitting PDF")

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Get page count
	pageCount, err := pe.GetPageCount(inputPath)
	if err != nil {
		return fmt.Errorf("failed to get page count: %w", err)
	}

	// Split into individual pages
	// ExtractPagesFile will create files named <inputBasename>_page_N.pdf in outputDir
	conf := model.NewDefaultConfiguration()

	// Build page list (all pages)
	pages := make([]string, pageCount)
	for i := 1; i <= pageCount; i++ {
		pages[i-1] = fmt.Sprintf("%d", i)
	}

	if err := api.ExtractPagesFile(inputPath, outputDir, pages, conf); err != nil {
		return fmt.Errorf("failed to extract pages: %w", err)
	}

	pe.logger.WithFields("page_count", pageCount).Info("PDF split successful")
	return nil
}

// GetPDFInfo returns basic information about a PDF file
func (pe *PDFEnhancer) GetPDFInfo(pdfPath string) (*PDFInfo, error) {
	pe.logger.WithFields("pdf_path", pdfPath).Debug("Getting PDF info")

	ctx, err := api.ReadContextFile(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF: %w", err)
	}

	// Get PDF version
	var versionStr string
	if ctx.HeaderVersion != nil {
		versionStr = ctx.HeaderVersion.String()
	} else {
		versionStr = "unknown"
	}

	info := &PDFInfo{
		PageCount:   ctx.PageCount,
		PDFVersion:  versionStr,
		FileSize:    0, // Would need to stat the file
		Encrypted:   ctx.Encrypt != nil,
		Linearized:  false, // pdfcpu doesn't expose this directly
	}

	// Get file size
	if stat, err := os.Stat(pdfPath); err == nil {
		info.FileSize = stat.Size()
	}

	pe.logger.WithFields("pages", info.PageCount, "version", info.PDFVersion).Debug("Retrieved PDF info")
	return info, nil
}

// PDFInfo contains information about a PDF file
type PDFInfo struct {
	PageCount  int
	PDFVersion string
	FileSize   int64
	Encrypted  bool
	Linearized bool
}

// CompareCoordinateSystems returns information about coordinate system differences
// between OCR (top-left origin) and PDF (bottom-left origin)
func (pe *PDFEnhancer) CompareCoordinateSystems(pageHeight int) string {
	return fmt.Sprintf(`PDF and OCR coordinate systems differ:
- PDF: Bottom-left origin (0,0), Y increases upward
- OCR: Top-left origin (0,0), Y increases downward
- Conversion: PDF_Y = PageHeight - OCR_Y - TextHeight
- Page height: %d pixels`, pageHeight)
}
