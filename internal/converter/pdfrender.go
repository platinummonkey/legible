package converter

import (
	"fmt"
	"image"
	"image/png"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/unidoc/unipdf/v3/common"
	unipdf "github.com/unidoc/unipdf/v3/model"
	"github.com/unidoc/unipdf/v3/render"
)

// init sets up unidoc licensing (metered mode for free usage)
func init() {
	// Use metered mode for free usage with rate limits
	// For production, set a license key via: common.SetLicenseKey()
	common.SetLogger(common.NewConsoleLogger(common.LogLevelError))
}

// renderPDFPageToImage renders a PDF page to an image at the specified DPI
func (c *Converter) renderPDFPageToImage(pdfPath string, pageNum int, dpi int) (image.Image, error) {
	c.logger.WithFields("pdf", pdfPath, "page", pageNum, "dpi", dpi).Debug("Rendering PDF page to image")

	// Open PDF file
	f, err := os.Open(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF: %w", err)
	}
	defer f.Close()

	// Parse PDF
	pdfReader, err := unipdf.NewPdfReaderLazy(f)
	if err != nil {
		return nil, fmt.Errorf("failed to create PDF reader: %w", err)
	}

	// Get page count
	numPages, err := pdfReader.GetNumPages()
	if err != nil {
		return nil, fmt.Errorf("failed to get page count: %w", err)
	}

	if pageNum < 1 || pageNum > numPages {
		return nil, fmt.Errorf("invalid page number %d (PDF has %d pages)", pageNum, numPages)
	}

	// Get the specific page
	page, err := pdfReader.GetPage(pageNum)
	if err != nil {
		return nil, fmt.Errorf("failed to get page %d: %w", pageNum, err)
	}

	// Create renderer with specified DPI
	device := render.NewImageDevice()

	// Calculate dimensions based on page size and DPI
	// PDF points are 1/72 inch, so we convert to pixels at target DPI
	mediaBox, err := page.GetMediaBox()
	if err != nil {
		return nil, fmt.Errorf("failed to get media box: %w", err)
	}

	pageWidth := mediaBox.Urx - mediaBox.Llx

	// Convert PDF points to pixels at target DPI
	// pixels = points * DPI / 72
	pixelWidth := int(float64(pageWidth) * float64(dpi) / 72.0)

	// Set output width - height will be calculated automatically to maintain aspect ratio
	device.OutputWidth = pixelWidth

	// Render the page
	img, err := device.Render(page)
	if err != nil {
		return nil, fmt.Errorf("failed to render page: %w", err)
	}

	// Get actual image dimensions
	bounds := img.Bounds()
	c.logger.WithFields("width", bounds.Dx(), "height", bounds.Dy()).Debug("Successfully rendered page to image")
	return img, nil
}

// renderAllPagesToImages renders all pages of a PDF to images
func (c *Converter) renderAllPagesToImages(pdfPath string, dpi int) ([]image.Image, error) {
	c.logger.WithFields("pdf", pdfPath, "dpi", dpi).Debug("Rendering all PDF pages to images")

	// Get page count using pdfcpu (lightweight check)
	ctx, err := api.ReadContextFile(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF: %w", err)
	}

	pageCount := ctx.PageCount
	images := make([]image.Image, pageCount)

	// Render each page
	for i := 1; i <= pageCount; i++ {
		img, err := c.renderPDFPageToImage(pdfPath, i, dpi)
		if err != nil {
			return nil, fmt.Errorf("failed to render page %d: %w", i, err)
		}
		images[i-1] = img
		c.logger.WithFields("page", i, "total", pageCount).Debug("Rendered page")
	}

	c.logger.WithFields("page_count", pageCount).Info("Successfully rendered all pages")
	return images, nil
}

// imageToBytes converts an image to PNG bytes for OCR processing
func (c *Converter) imageToBytes(img image.Image) ([]byte, error) {
	// Create temporary file
	tmpFile, err := os.CreateTemp("", "page-*.png")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Encode image as PNG
	if err := png.Encode(tmpFile, img); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}
	tmpFile.Close()

	// Read back as bytes
	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	return data, nil
}

// validatePDFWithPdfcpu validates that a PDF is readable
func (c *Converter) validatePDFWithPdfcpu(pdfPath string) error {
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed

	if err := api.ValidateFile(pdfPath, conf); err != nil {
		return fmt.Errorf("PDF validation failed: %w", err)
	}

	return nil
}
