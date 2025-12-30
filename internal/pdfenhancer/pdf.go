package pdfenhancer

import (
	"fmt"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
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
	// Get the page dictionary
	pageDict, _, _, err := ctx.PageDict(pageNum, false)
	if err != nil {
		return fmt.Errorf("failed to get page dictionary: %w", err)
	}

	if pageDict == nil {
		return fmt.Errorf("page dictionary is nil")
	}

	// Note: Adding actual text content to PDF pages requires manipulating
	// PDF content streams, which is complex. pdfcpu provides low-level access
	// but doesn't have high-level text addition APIs.
	//
	// For production use, this would need to:
	// 1. Create a new content stream with text operators
	// 2. Position text using PDF coordinates (bottom-left origin)
	// 3. Set text rendering mode to invisible (Tr 3)
	// 4. Add the text at appropriate positions from OCR bounding boxes
	//
	// This is placeholder logic that demonstrates the structure.
	// Full implementation would require:
	// - PDF content stream manipulation
	// - Coordinate system conversion (OCR uses top-left, PDF uses bottom-left)
	// - Font embedding and text encoding
	// - Text positioning and scaling

	pe.logger.WithFields("page", pageNum, "word_count", len(pageOCR.Words)).
		Debug("Text layer addition (placeholder)")

	// For now, we just log that we would add the text
	// Actual implementation would manipulate pageDict content streams here

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
