package rmrender

import "time"

// Version represents the .rm file format version
type Version int

const (
	// VersionUnknown represents an unknown or unsupported version
	VersionUnknown Version = 0

	// Version3 is the original .rm format
	Version3 Version = 3

	// Version5 is an intermediate format version
	Version5 Version = 5

	// Version6 is the current format (as of Dec 2025)
	Version6 Version = 6
)

// String returns the version as a string
func (v Version) String() string {
	switch v {
	case Version3:
		return "v3"
	case Version5:
		return "v5"
	case Version6:
		return "v6"
	default:
		return "unknown"
	}
}

// Dimensions defines the reMarkable screen dimensions
const (
	// Width in pixels (1404)
	Width = 1404

	// Height in pixels (1872)
	Height = 1872

	// DPI of the reMarkable screen
	DPI = 226
)

// Document represents a complete .rm file with all its layers and strokes
type Document struct {
	Version Version
	Layers  []Layer
}

// Layer represents a drawing layer in the document
// The reMarkable supports multiple layers that can be toggled on/off
type Layer struct {
	// Lines contains all the strokes in this layer
	Lines []Line
}

// Line represents a single stroke (pen stroke, eraser stroke, etc.)
type Line struct {
	// BrushType specifies what tool was used
	BrushType BrushType

	// Color of the stroke
	Color Color

	// BrushSize is the thickness/size setting (0-2 usually)
	BrushSize float32

	// Points contains all the points that make up this stroke
	Points []Point
}

// Point represents a single point in a stroke with position and pressure
type Point struct {
	// X coordinate (0 to Width)
	X float32

	// Y coordinate (0 to Height)
	Y float32

	// Pressure (0.0 to 1.0) - affects stroke width
	Pressure float32

	// Speed of the stroke at this point (for texture simulation)
	Speed float32

	// Direction of stroke (for brush effects)
	Direction float32

	// Width at this point (may be calculated from pressure and brush size)
	Width float32

	// Tilt of the pen (for advanced brush effects)
	TiltX float32
	TiltY float32
}

// BrushType represents the type of brush/tool used
type BrushType int

const (
	// BrushBallpoint is a standard pen
	BrushBallpoint BrushType = 2

	// BrushMarker is a broad marker
	BrushMarker BrushType = 3

	// BrushFineliner is a thin, precise pen
	BrushFineliner BrushType = 4

	// BrushSharpPencil is a mechanical pencil
	BrushSharpPencil BrushType = 7

	// BrushTiltPencil is a pencil with tilt support
	BrushTiltPencil BrushType = 1

	// BrushBrush is a paintbrush/calligraphy brush
	BrushBrush BrushType = 0

	// BrushHighlighter is a semi-transparent highlighter
	BrushHighlighter BrushType = 5

	// BrushEraser removes strokes
	BrushEraser BrushType = 6

	// BrushEraseSection removes entire sections
	BrushEraseSection BrushType = 8

	// BrushCalligraphy is a calligraphy pen
	BrushCalligraphy BrushType = 21
)

// String returns the brush type name
func (b BrushType) String() string {
	switch b {
	case BrushBallpoint:
		return "Ballpoint"
	case BrushMarker:
		return "Marker"
	case BrushFineliner:
		return "Fineliner"
	case BrushSharpPencil:
		return "Sharp Pencil"
	case BrushTiltPencil:
		return "Tilt Pencil"
	case BrushBrush:
		return "Brush"
	case BrushHighlighter:
		return "Highlighter"
	case BrushEraser:
		return "Eraser"
	case BrushEraseSection:
		return "Erase Section"
	case BrushCalligraphy:
		return "Calligraphy"
	default:
		return "Unknown"
	}
}

// Color represents stroke colors
type Color int

const (
	// ColorBlack is standard black ink
	ColorBlack Color = 0

	// ColorGray is gray ink
	ColorGray Color = 1

	// ColorWhite is white ink (eraser on black background)
	ColorWhite Color = 2

	// ColorYellow is yellow highlighter
	ColorYellow Color = 3

	// ColorGreen is green highlighter
	ColorGreen Color = 4

	// ColorPink is pink highlighter
	ColorPink Color = 5

	// ColorBlue is blue highlighter
	ColorBlue Color = 6

	// ColorRed is red ink
	ColorRed Color = 7

	// ColorGrayOverlay is gray overlay
	ColorGrayOverlay Color = 8
)

// RGB returns the RGB values for this color (0-255)
func (c Color) RGB() (r, g, b uint8) {
	switch c {
	case ColorBlack:
		return 0, 0, 0
	case ColorGray:
		return 125, 125, 125
	case ColorWhite:
		return 255, 255, 255
	case ColorYellow:
		return 255, 242, 0
	case ColorGreen:
		return 0, 255, 0
	case ColorPink:
		return 255, 0, 255
	case ColorBlue:
		return 0, 0, 255
	case ColorRed:
		return 255, 0, 0
	case ColorGrayOverlay:
		return 125, 125, 125
	default:
		return 0, 0, 0
	}
}

// String returns the color name
func (c Color) String() string {
	switch c {
	case ColorBlack:
		return "Black"
	case ColorGray:
		return "Gray"
	case ColorWhite:
		return "White"
	case ColorYellow:
		return "Yellow"
	case ColorGreen:
		return "Green"
	case ColorPink:
		return "Pink"
	case ColorBlue:
		return "Blue"
	case ColorRed:
		return "Red"
	case ColorGrayOverlay:
		return "Gray Overlay"
	default:
		return "Unknown"
	}
}

// ParseResult contains metadata about the parsing operation
type ParseResult struct {
	Version     Version
	LayerCount  int
	StrokeCount int
	PointCount  int
	ParsedAt    time.Time
	ParseError  error
}

// RenderOptions configures the rendering process
type RenderOptions struct {
	// RenderLayers controls which layers to render (nil = all)
	RenderLayers []int

	// BackgroundColor for the PDF page
	BackgroundColor Color

	// EnablePressure enables pressure-sensitive stroke width
	EnablePressure bool

	// EnableAntialiasing enables anti-aliasing for smoother strokes
	EnableAntialiasing bool

	// StrokeQuality controls rendering quality (higher = more points, smoother)
	StrokeQuality int // 1-10, default 5
}

// DefaultRenderOptions returns sensible default rendering options
func DefaultRenderOptions() *RenderOptions {
	return &RenderOptions{
		RenderLayers:       nil, // Render all layers
		BackgroundColor:    ColorWhite,
		EnablePressure:     true,
		EnableAntialiasing: true,
		StrokeQuality:      5,
	}
}
