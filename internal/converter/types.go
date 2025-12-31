package converter

import "time"

// ConversionOptions holds configuration for PDF conversion
type ConversionOptions struct {
	// InputPath is the path to the input .rmdoc file
	InputPath string

	// OutputPath is the path where the PDF should be written
	OutputPath string

	// IncludeAnnotations determines if annotations should be rendered in the PDF
	IncludeAnnotations bool

	// Quality is the rendering quality (1-100, higher is better but slower)
	Quality int

	// PaperSize is the paper size for the output PDF (e.g., "A4", "Letter")
	PaperSize PaperSize

	// Orientation is the page orientation
	Orientation Orientation

	// DPI is the dots per inch for rendering (default: 300)
	DPI int

	// RenderLayers determines which layers to render
	RenderLayers []int

	// BackgroundColor is the background color for pages (default: white)
	BackgroundColor string

	// Metadata contains PDF metadata to embed
	Metadata PDFMetadata

	// Compression enables PDF compression to reduce file size
	Compression bool

	// CopyTags enables copying reMarkable labels/tags to PDF metadata (default: true)
	CopyTags bool

	// TagPrefix is an optional prefix to add to tags (e.g., "rm:" -> "rm:work")
	TagPrefix string

	// IncludePageTags determines if page-level tags should be included (default: true)
	IncludePageTags bool

	// EnableOCR determines if OCR processing should be performed to add searchable text layer
	EnableOCR bool

	// OCRLanguages is the list of language codes to use for OCR via Ollama (default: ["eng"])
	OCRLanguages []string
}

// ConversionResult represents the result of a PDF conversion
type ConversionResult struct {
	// OutputPath is the path to the generated PDF
	OutputPath string

	// PageCount is the number of pages in the output PDF
	PageCount int

	// FileSize is the size of the output PDF in bytes
	FileSize int64

	// Duration is the time taken for conversion
	Duration time.Duration

	// Success indicates if conversion completed successfully
	Success bool

	// Error contains any error message if Success is false
	Error string

	// Warnings contains any warnings encountered during conversion
	Warnings []string

	// OCREnabled indicates if OCR processing was performed
	OCREnabled bool

	// OCRWordCount is the total number of words recognized via OCR
	OCRWordCount int

	// OCRConfidence is the average OCR confidence score (0-100)
	OCRConfidence float64

	// OCRDuration is the time taken for OCR processing
	OCRDuration time.Duration
}

// PDFMetadata represents metadata to embed in the PDF
type PDFMetadata struct {
	// Title is the PDF title
	Title string

	// Author is the PDF author
	Author string

	// Subject is the PDF subject
	Subject string

	// Keywords are the PDF keywords
	Keywords []string

	// Creator is the application that created the PDF
	Creator string

	// Producer is the library that produced the PDF
	Producer string

	// CreationDate is when the PDF was created
	CreationDate time.Time

	// ModificationDate is when the PDF was last modified
	ModificationDate time.Time
}

// PaperSize represents standard paper sizes
type PaperSize string

const (
	// PaperSizeA4 is ISO A4 (210mm × 297mm)
	PaperSizeA4 PaperSize = "A4"

	// PaperSizeA5 is ISO A5 (148mm × 210mm)
	PaperSizeA5 PaperSize = "A5"

	// PaperSizeLetter is US Letter (8.5" × 11")
	PaperSizeLetter PaperSize = "Letter"

	// PaperSizeLegal is US Legal (8.5" × 14")
	PaperSizeLegal PaperSize = "Legal"

	// PaperSizeRemarkable is reMarkable tablet size (1404 × 1872 pixels)
	PaperSizeRemarkable PaperSize = "Remarkable"
)

// Orientation represents page orientation
type Orientation string

const (
	// OrientationPortrait is portrait orientation
	OrientationPortrait Orientation = "Portrait"

	// OrientationLandscape is landscape orientation
	OrientationLandscape Orientation = "Landscape"
)

// Default values
const (
	DefaultQuality     = 85
	DefaultDPI         = 300
	DefaultPaperSize   = PaperSizeRemarkable
	DefaultOrientation = OrientationPortrait
)

// NewConversionOptions creates ConversionOptions with default values
func NewConversionOptions(inputPath, outputPath string) *ConversionOptions {
	return &ConversionOptions{
		InputPath:          inputPath,
		OutputPath:         outputPath,
		IncludeAnnotations: true,
		Quality:            DefaultQuality,
		PaperSize:          DefaultPaperSize,
		Orientation:        DefaultOrientation,
		DPI:                DefaultDPI,
		BackgroundColor:    "#FFFFFF",
		Compression:        true,
		CopyTags:           true,
		TagPrefix:          "",
		IncludePageTags:    true,
		EnableOCR:          true, // Enable OCR by default for searchable text layer
		OCRLanguages:       []string{"eng"},
		Metadata: PDFMetadata{
			Creator:  "remarkable-sync",
			Producer: "remarkable-sync",
		},
	}
}

// NewConversionResult creates a new ConversionResult
func NewConversionResult() *ConversionResult {
	return &ConversionResult{
		Success:  false,
		Warnings: []string{},
	}
}

// AddWarning adds a warning message to the conversion result
func (cr *ConversionResult) AddWarning(warning string) {
	cr.Warnings = append(cr.Warnings, warning)
}

// SetError sets the error and marks the conversion as failed
func (cr *ConversionResult) SetError(err error) {
	cr.Success = false
	cr.Error = err.Error()
}

// SetSuccess marks the conversion as successful
func (cr *ConversionResult) SetSuccess(outputPath string, pageCount int, fileSize int64, duration time.Duration) {
	cr.Success = true
	cr.OutputPath = outputPath
	cr.PageCount = pageCount
	cr.FileSize = fileSize
	cr.Duration = duration
}

// Page represents a single page from a .rmdoc file
type Page struct {
	// UUID is the unique identifier for this page
	UUID string

	// Number is the page number (1-indexed)
	Number int

	// Template is the page template name
	Template string

	// Layers contains the drawing layers
	Layers []Layer

	// Width is the page width in pixels
	Width int

	// Height is the page height in pixels
	Height int
}

// Layer represents a drawing layer on a page
type Layer struct {
	// Name is the layer name
	Name string

	// Strokes contains the pen/brush strokes
	Strokes []Stroke
}

// Stroke represents a pen or brush stroke
type Stroke struct {
	// Tool is the tool type (pen, pencil, highlighter, eraser, etc.)
	Tool ToolType

	// Color is the stroke color (RGB hex)
	Color string

	// Width is the stroke width
	Width float64

	// Points contains the stroke points
	Points []Point
}

// Point represents a point in a stroke
type Point struct {
	// X is the x-coordinate
	X float64

	// Y is the y-coordinate
	Y float64

	// Pressure is the pen pressure (0.0 - 1.0)
	Pressure float64

	// Tilt is the pen tilt angle
	Tilt float64
}

// ToolType represents the drawing tool type
type ToolType string

const (
	// ToolTypePen is a pen tool
	ToolTypePen ToolType = "pen"

	// ToolTypePencil is a pencil tool
	ToolTypePencil ToolType = "pencil"

	// ToolTypeHighlighter is a highlighter tool
	ToolTypeHighlighter ToolType = "highlighter"

	// ToolTypeEraser is an eraser tool
	ToolTypeEraser ToolType = "eraser"

	// ToolTypeMarker is a marker tool
	ToolTypeMarker ToolType = "marker"
)

// Dimensions returns the dimensions for a paper size
func (ps PaperSize) Dimensions() (width, height int) {
	switch ps {
	case PaperSizeA4:
		return 2480, 3508 // 210mm × 297mm at 300 DPI
	case PaperSizeA5:
		return 1748, 2480 // 148mm × 210mm at 300 DPI
	case PaperSizeLetter:
		return 2550, 3300 // 8.5" × 11" at 300 DPI
	case PaperSizeLegal:
		return 2550, 4200 // 8.5" × 14" at 300 DPI
	case PaperSizeRemarkable:
		return 1404, 1872 // reMarkable tablet size
	default:
		return 1404, 1872 // Default to reMarkable size
	}
}
