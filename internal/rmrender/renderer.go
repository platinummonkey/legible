package rmrender

import (
	"bytes"
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/signintech/gopdf"
)

// Renderer handles rendering of parsed .rm documents to PDF
type Renderer struct {
	options *RenderOptions
}

// NewRenderer creates a new renderer with default options
func NewRenderer() *Renderer {
	return &Renderer{
		options: DefaultRenderOptions(),
	}
}

// NewRendererWithOptions creates a renderer with custom options
func NewRendererWithOptions(opts *RenderOptions) *Renderer {
	return &Renderer{
		options: opts,
	}
}

// RenderToPDF renders a parsed Document to PDF format
//
// This creates a single-page PDF with all strokes rendered as vector graphics.
// The PDF will use the reMarkable dimensions (1404x1872 pixels at 226 DPI).
func (r *Renderer) RenderToPDF(doc *Document) ([]byte, error) {
	if doc == nil {
		return nil, fmt.Errorf("document cannot be nil")
	}

	// Create new PDF with reMarkable dimensions
	// Convert pixels to points: pixels * 72 / DPI
	widthPt := float64(Width) * 72.0 / float64(DPI)
	heightPt := float64(Height) * 72.0 / float64(DPI)

	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{
		PageSize: gopdf.Rect{
			W: widthPt,
			H: heightPt,
		},
	})

	pdf.AddPage()

	// Render background
	red, green, blue := r.options.BackgroundColor.RGB()
	pdf.SetFillColor(red, green, blue)
	pdf.RectFromUpperLeftWithStyle(0, 0, widthPt, heightPt, "F")

	// Render each layer
	for layerIdx, layer := range doc.Layers {
		// Check if we should render this layer
		if r.options.RenderLayers != nil {
			found := false
			for _, idx := range r.options.RenderLayers {
				if idx == layerIdx {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Render strokes in this layer
		for _, stroke := range layer.Lines {
			if err := r.renderStrokeToPDF(&pdf, stroke); err != nil {
				// Log error but continue with other strokes
				continue
			}
		}
	}

	// Get PDF bytes
	var buf bytes.Buffer
	if err := pdf.Write(&buf); err != nil {
		return nil, fmt.Errorf("failed to write PDF: %w", err)
	}

	return buf.Bytes(), nil
}

// RenderPage renders a single page with the given strokes
//
// This is useful for multi-page documents where each .rm file represents one page.
func (r *Renderer) RenderPage(doc *Document, pageIndex int) ([]byte, error) {
	// For single-page documents, just render the whole document
	return r.RenderToPDF(doc)
}

// transformCoordinate converts reMarkable coordinates to PDF coordinates
//
// reMarkable: Origin at top-left, Y increases downward
// PDF: Origin at bottom-left, Y increases upward (but gopdf uses top-left)
func (r *Renderer) transformCoordinate(x, y float32) (float32, float32) {
	// gopdf uses top-left origin like reMarkable, so no Y inversion needed
	// But we need to convert from pixels to points
	pdfX := x
	pdfY := y
	return pdfX, pdfY
}

// transformCoordinateToPDF converts reMarkable pixel coordinates to PDF points
// gopdf uses top-left origin, same as reMarkable
func (r *Renderer) transformCoordinateToPDF(x, y float32) (float64, float64) {
	// Convert pixels to points: pixels * 72 / DPI
	pdfX := float64(x) * 72.0 / float64(DPI)
	pdfY := float64(y) * 72.0 / float64(DPI)
	return pdfX, pdfY
}

// calculateStrokeWidth calculates the actual stroke width based on pressure and brush size
func (r *Renderer) calculateStrokeWidth(point Point, baseBrushSize float32, brushType BrushType) float32 {
	if !r.options.EnablePressure {
		return baseBrushSize
	}

	// Base width from brush size
	baseWidth := baseBrushSize * 2.0 // Scale factor for visibility

	// Apply pressure
	pressure := point.Pressure
	if pressure < 0.1 {
		pressure = 0.1 // Minimum pressure
	}

	width := baseWidth * pressure

	// Brush-specific adjustments
	switch brushType {
	case BrushHighlighter:
		width *= 3.0 // Highlighters are wider
	case BrushMarker:
		width *= 2.0 // Markers are broader
	case BrushFineliner:
		width *= 0.7 // Fineliners are thinner
	case BrushSharpPencil:
		width *= 0.8 // Pencils are thin
	}

	return width
}

// renderStrokeToPDF renders a single stroke to the PDF
func (r *Renderer) renderStrokeToPDF(pdf *gopdf.GoPdf, stroke Line) error {
	if len(stroke.Points) < 2 {
		// Need at least 2 points to draw a line
		return nil
	}

	// Get color for this brush/color combination
	red, green, blue, _ := r.applyBrushStyle(stroke.BrushType, stroke.Color)
	// Note: gopdf doesn't support alpha, so we ignore it

	// Set stroke color
	pdf.SetStrokeColor(red, green, blue)

	// Draw lines connecting consecutive points
	for i := 0; i < len(stroke.Points)-1; i++ {
		p1 := stroke.Points[i]
		p2 := stroke.Points[i+1]

		// Transform coordinates to PDF space
		x1, y1 := r.transformCoordinateToPDF(p1.X, p1.Y)
		x2, y2 := r.transformCoordinateToPDF(p2.X, p2.Y)

		// Calculate stroke width based on pressure if enabled
		width := r.calculateStrokeWidth(p1, stroke.BrushSize, stroke.BrushType)

		// Set line width
		pdf.SetLineWidth(float64(width))

		// Draw line
		pdf.Line(x1, y1, x2, y2)
	}

	return nil
}

// renderEraser processes eraser strokes
//
// Eraser strokes need special handling as they remove underlying content
// rather than adding new content.
func (r *Renderer) renderEraser(pdf *gopdf.GoPdf, stroke Line) error {
	// TODO: Implement eraser logic
	// Options:
	// 1. Use white ink on white background
	// 2. Use clipping paths
	// 3. Pre-process document to remove intersecting strokes
	return fmt.Errorf("eraser rendering not yet implemented")
}

// applyBrushStyle applies brush-specific styling to a stroke
func (r *Renderer) applyBrushStyle(brushType BrushType, color Color) (red, green, blue uint8, alpha float32) {
	red, green, blue = color.RGB()
	alpha = 1.0

	switch brushType {
	case BrushHighlighter:
		alpha = 0.3 // Highlighters are semi-transparent
	case BrushMarker:
		alpha = 0.8 // Markers are slightly transparent
	}

	return red, green, blue, alpha
}

// renderBackground renders the page background
// This is now handled directly in RenderToPDF
func (r *Renderer) renderBackground(pdf *gopdf.GoPdf) error {
	// Background is rendered in RenderToPDF
	return nil
}

// RenderWithTemplate renders a document with a template background
//
// Templates include: Blank, Lined, Grid, Dots, etc.
func (r *Renderer) RenderWithTemplate(doc *Document, template string) ([]byte, error) {
	// TODO: Implement template rendering
	// Templates need to be:
	// 1. Generated programmatically (preferred)
	// 2. Or loaded from template images/PDFs
	return nil, fmt.Errorf("template rendering not yet implemented")
}

// Helper function to get reMarkable page dimensions in points
func getRemarkablePageDimensions() (width, height float64) {
	// reMarkable dimensions: 1404x1872 pixels at 226 DPI
	// Convert to points (PDF unit): pixels * 72 / DPI
	width = float64(Width) * 72.0 / float64(DPI)
	height = float64(Height) * 72.0 / float64(DPI)
	return width, height
}

// RenderLayers renders only specific layers from a document
func (r *Renderer) RenderLayers(doc *Document, layerIndices []int) ([]byte, error) {
	// Create a copy of render options with specified layers
	opts := *r.options
	opts.RenderLayers = layerIndices

	// Render with modified options
	tmpRenderer := NewRendererWithOptions(&opts)
	return tmpRenderer.RenderToPDF(doc)
}

// EstimateComplexity returns an estimate of rendering complexity
//
// Useful for progress reporting and performance optimization decisions.
func EstimateComplexity(doc *Document) int {
	if doc == nil {
		return 0
	}

	complexity := 0
	for _, layer := range doc.Layers {
		for _, stroke := range layer.Lines {
			complexity += len(stroke.Points)
		}
	}

	return complexity
}

// ExportMetadata extracts metadata about a document without rendering
func ExportMetadata(doc *Document) map[string]interface{} {
	if doc == nil {
		return nil
	}

	strokeCount := 0
	pointCount := 0
	brushTypes := make(map[BrushType]int)

	for _, layer := range doc.Layers {
		strokeCount += len(layer.Lines)
		for _, stroke := range layer.Lines {
			pointCount += len(stroke.Points)
			brushTypes[stroke.BrushType]++
		}
	}

	return map[string]interface{}{
		"version":      doc.Version.String(),
		"layer_count":  len(doc.Layers),
		"stroke_count": strokeCount,
		"point_count":  pointCount,
		"brush_types":  brushTypes,
		"complexity":   EstimateComplexity(doc),
	}
}

// Ensure we're importing pdfcpu at the top
var _ = api.ImportImagesFile // Verify pdfcpu import works
