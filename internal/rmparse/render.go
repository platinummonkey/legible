// Package rmparse provides rendering of parsed .rm files to PDF
package rmparse

import (
	"fmt"

	"github.com/signintech/gopdf"
)

// reMarkable tablet dimensions and scaling
const (
	// reMarkable 2 tablet screen: 1404 x 1872 pixels at 226 DPI
	RMWidth  = 1404.0
	RMHeight = 1872.0

	// PDF page size (A5 portrait in points: 1pt = 1/72 inch)
	PDFWidth  = 420.0 // ~148mm
	PDFHeight = 595.0 // ~210mm

	// Scale factor from reMarkable coordinates to PDF points
	ScaleX = PDFWidth / RMWidth
	ScaleY = PDFHeight / RMHeight
)

// PenColor maps pen color values to RGB
type PenColor struct {
	R, G, B uint8
}

var colorMap = map[uint32]PenColor{
	0: {R: 0, G: 0, B: 0},       // Black
	1: {R: 128, G: 128, B: 128}, // Gray
	2: {R: 255, G: 255, B: 255}, // White
	3: {R: 255, G: 0, B: 0},     // Red
	4: {R: 0, G: 255, B: 0},     // Green
	5: {R: 0, G: 0, B: 255},     // Blue
}

// RenderToPDF renders an RMFile to a PDF file
func RenderToPDF(rmFile *RMFile, outputPath string) error {
	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{
		PageSize: gopdf.Rect{W: PDFWidth, H: PDFHeight},
	})

	pdf.AddPage()

	// Render each layer
	for _, layer := range rmFile.Layers {
		for _, line := range layer.Lines {
			if err := renderLine(&pdf, line); err != nil {
				return fmt.Errorf("failed to render line: %w", err)
			}
		}
	}

	// Write to file
	if err := pdf.WritePdf(outputPath); err != nil {
		return fmt.Errorf("failed to write PDF: %w", err)
	}

	return nil
}

// RenderToPage renders an RMFile to an existing PDF page
func RenderToPage(pdf *gopdf.GoPdf, rmFile *RMFile) error {
	// Render each layer
	for _, layer := range rmFile.Layers {
		for _, line := range layer.Lines {
			if err := renderLine(pdf, line); err != nil {
				return fmt.Errorf("failed to render line: %w", err)
			}
		}
	}
	return nil
}

// renderLine renders a single line (stroke) to the PDF
func renderLine(pdf *gopdf.GoPdf, line Line) error {
	if len(line.Points) < 2 {
		return nil // Need at least 2 points to draw a line
	}

	// Set stroke color
	color := colorMap[line.Color]
	pdf.SetStrokeColor(color.R, color.G, color.B)

	// Calculate line width based on brush size
	// Typical brush sizes range from 1-10, scale appropriately
	lineWidth := float64(line.BrushSize) * 0.5
	if lineWidth < 0.5 {
		lineWidth = 0.5
	}
	pdf.SetLineWidth(lineWidth)

	// Set line cap and join for smoother strokes
	pdf.SetLineType("round")

	// Draw the stroke as a series of line segments
	firstPoint := line.Points[0]
	x1, y1 := transformPoint(firstPoint.X, firstPoint.Y)

	for i := 1; i < len(line.Points); i++ {
		point := line.Points[i]
		x2, y2 := transformPoint(point.X, point.Y)

		// Draw line segment
		pdf.Line(x1, y1, x2, y2)

		// Move to next segment
		x1, y1 = x2, y2
	}

	return nil
}

// transformPoint converts reMarkable coordinates to PDF coordinates
func transformPoint(x, y float32) (float64, float64) {
	// reMarkable coordinates (from rmv6 spec):
	//   X: origin at CENTER of page, ranges approximately -702 to +702 (1404 pixels / 2)
	//   Y: origin at TOP of page, ranges 0 to 1872
	//
	// PDF coordinates:
	//   Origin at top-left corner
	//   X: ranges 0 to PDFWidth (420 pts)
	//   Y: ranges 0 to PDFHeight (595 pts)

	// Convert X from centered (±702) to left-aligned (0 to 1404)
	rmX := float64(x) + (RMWidth / 2)

	// Y is already top-aligned, just use as-is
	rmY := float64(y)

	// Scale to PDF dimensions
	pdfX := rmX * ScaleX
	pdfY := rmY * ScaleY

	return pdfX, pdfY
}
