package rmrender

import (
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
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

	// TODO: Implement PDF rendering
	// Steps:
	// 1. Create new PDF page with correct dimensions
	// 2. For each layer (if enabled in options):
	//    a. For each stroke:
	//       - Transform coordinates from reMarkable to PDF coordinate system
	//       - Render stroke based on brush type
	//       - Apply color and width
	//       - Handle pressure sensitivity if enabled
	// 3. Return PDF as bytes

	return nil, fmt.Errorf("PDF rendering not yet implemented - foundation in place")
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
// PDF: Origin at bottom-left, Y increases upward
func (r *Renderer) transformCoordinate(x, y float32) (float32, float32) {
	// PDF coordinate system has origin at bottom-left
	// reMarkable has origin at top-left
	pdfX := x
	pdfY := float32(Height) - y
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

// renderStroke renders a single stroke to the PDF page
func (r *Renderer) renderStroke(page *model.Page, stroke Line) error {
	if len(stroke.Points) < 2 {
		// Need at least 2 points to draw a line
		return nil
	}

	// TODO: Implement stroke rendering
	// For each pair of points:
	// 1. Transform coordinates
	// 2. Calculate stroke width based on pressure
	// 3. Draw line segment with appropriate style
	// 4. Handle brush-specific rendering (e.g., transparency for highlighter)

	return fmt.Errorf("stroke rendering not yet implemented")
}

// renderEraser processes eraser strokes
//
// Eraser strokes need special handling as they remove underlying content
// rather than adding new content.
func (r *Renderer) renderEraser(page *model.Page, stroke Line) error {
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
func (r *Renderer) renderBackground(page *model.Page) error {
	// Fill page with background color
	red, green, blue := r.options.BackgroundColor.RGB()
	_ = red
	_ = green
	_ = blue

	// TODO: Implement background rendering
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

// Helper function to create a new PDF page with reMarkable dimensions
func createRemarkablePage() (*model.Page, error) {
	// reMarkable dimensions: 1404x1872 pixels at 226 DPI
	// Convert to points (PDF unit): pixels * 72 / DPI
	widthPt := float32(Width) * 72.0 / float32(DPI)
	heightPt := float32(Height) * 72.0 / float32(DPI)

	_ = widthPt
	_ = heightPt

	// TODO: Create PDF page with pdfcpu
	return nil, fmt.Errorf("page creation not yet implemented")
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
