package rmrender

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// Parser handles parsing of .rm binary files
type Parser struct {
	version Version
}

// NewParser creates a new .rm file parser
func NewParser() *Parser {
	return &Parser{}
}

// Parse parses a .rm file from bytes and returns a Document
func (p *Parser) Parse(data []byte) (*Document, error) {
	reader := bytes.NewReader(data)

	// Parse header to determine version
	version, err := p.parseHeader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse header: %w", err)
	}

	p.version = version

	// Parse based on version
	switch version {
	case Version3:
		return p.parseV3(reader)
	case Version5:
		return p.parseV5(reader)
	case Version6:
		return p.parseV6(reader)
	default:
		return nil, fmt.Errorf("unsupported .rm file version: %d", version)
	}
}

// parseHeader reads the .rm file header and extracts the version
//
// Header format: "reMarkable .lines file, version=N          " (43 bytes)
// where N is the version number (3, 5, or 6)
func (p *Parser) parseHeader(reader io.Reader) (Version, error) {
	header := make([]byte, 43)
	if _, err := io.ReadFull(reader, header); err != nil {
		return VersionUnknown, fmt.Errorf("failed to read header: %w", err)
	}

	// Verify header starts with expected prefix
	expectedPrefix := "reMarkable .lines file, version="
	if string(header[:len(expectedPrefix)]) != expectedPrefix {
		return VersionUnknown, fmt.Errorf("invalid .rm file header")
	}

	// Extract version number (single digit after "version=")
	versionByte := header[len(expectedPrefix)]
	switch versionByte {
	case '3':
		return Version3, nil
	case '5':
		return Version5, nil
	case '6':
		return Version6, nil
	default:
		return VersionUnknown, fmt.Errorf("unknown version: %c", versionByte)
	}
}

// parseV3 parses a version 3 .rm file
//
// Version 3 format is documented and can use ddvk/rmapi/encoding/rm
// as a reference implementation.
func (p *Parser) parseV3(reader io.Reader) (*Document, error) {
	// TODO: Implement v3 parsing
	// Can reference ddvk/rmapi/encoding/rm for implementation
	return nil, fmt.Errorf("version 3 parsing not yet implemented")
}

// parseV5 parses a version 5 .rm file
func (p *Parser) parseV5(reader io.Reader) (*Document, error) {
	// TODO: Implement v5 parsing
	// Format is similar to v6 but with some differences
	return nil, fmt.Errorf("version 5 parsing not yet implemented")
}

// parseV6 parses a version 6 .rm file
//
// Version 6 is the current format used by reMarkable tablets (as of Dec 2025).
// Format uses a tag-based system with magic delimiter bytes.
//
// This is a simplified parser that scans for recognizable stroke patterns.
// For a complete implementation, see:
// - https://github.com/YakBarber/remarkable_file_format (Kaitai Struct spec)
// - https://github.com/ricklupton/rmscene (Python implementation)
func (p *Parser) parseV6(reader io.Reader) (*Document, error) {
	doc := &Document{
		Version: Version6,
		Layers:  []Layer{},
	}

	// Read all remaining data into memory for scanning
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read file data: %w", err)
	}

	// For now, create a single layer and scan for stroke patterns
	layer := Layer{Lines: []Line{}}

	// Scan through the data looking for stroke patterns
	// Strokes typically have: coordinates (floats), pressure data, etc.
	// This is a simplified approach - a full parser would use the tag structure

	// Look for sequences that resemble coordinate data:
	// - Floats in range 0-1500 for X (reMarkable width is 1404)
	// - Floats in range 0-2000 for Y (reMarkable height is 1872)

	i := 0
	for i < len(data)-14 { // Need at least 14 bytes for a point
		// Try to read potential point data
		if i+13 >= len(data) {
			break
		}

		pointReader := bytes.NewReader(data[i : i+14])
		x, _ := readFloat32(pointReader)
		y, _ := readFloat32(pointReader)

		// Check if this looks like a valid coordinate
		if x >= 0 && x <= 1500 && y >= 0 && y <= 2000 {
			// This might be a point - try to read the rest
			speed, _ := readUint8(pointReader)
			width, _ := readUint8(pointReader)
			direction, _ := readUint8(pointReader)
			pressure, _ := readUint8(pointReader)

			// Collect points that form a stroke
			// For simplicity, we'll group consecutive valid points
			points := []Point{
				{
					X:         x,
					Y:         y,
					Speed:     float32(speed) / 255.0,
					Width:     float32(width),
					Direction: float32(direction) / 255.0 * 360.0,
					Pressure:  float32(pressure) / 255.0,
				},
			}

			// Try to read more points
			j := i + 14
			for j < len(data)-14 && len(points) < 1000 {
				pointReader := bytes.NewReader(data[j : j+14])
				nextX, _ := readFloat32(pointReader)
				nextY, _ := readFloat32(pointReader)

				if nextX >= 0 && nextX <= 1500 && nextY >= 0 && nextY <= 2000 {
					speed, _ := readUint8(pointReader)
					width, _ := readUint8(pointReader)
					direction, _ := readUint8(pointReader)
					pressure, _ := readUint8(pointReader)

					points = append(points, Point{
						X:         nextX,
						Y:         nextY,
						Speed:     float32(speed) / 255.0,
						Width:     float32(width),
						Direction: float32(direction) / 255.0 * 360.0,
						Pressure:  float32(pressure) / 255.0,
					})
					j += 14
				} else {
					break
				}
			}

			// If we have at least 2 points, create a stroke
			if len(points) >= 2 {
				line := Line{
					BrushType: BrushBallpoint, // Default brush
					Color:     ColorBlack,      // Default color
					BrushSize: 2.0,             // Default size
					Points:    points,
				}
				layer.Lines = append(layer.Lines, line)
				i = j
				continue
			}
		}

		i++
	}

	// Add layer to document if it has strokes
	if len(layer.Lines) > 0 {
		doc.Layers = append(doc.Layers, layer)
	}

	return doc, nil
}

// Helper functions for binary parsing

// readUint32 reads a 32-bit unsigned integer in little-endian format
func readUint32(reader io.Reader) (uint32, error) {
	var value uint32
	err := binary.Read(reader, binary.LittleEndian, &value)
	return value, err
}

// readUint64 reads a 64-bit unsigned integer in little-endian format
func readUint64(reader io.Reader) (uint64, error) {
	var value uint64
	err := binary.Read(reader, binary.LittleEndian, &value)
	return value, err
}

// readFloat32 reads a 32-bit float in little-endian format
func readFloat32(reader io.Reader) (float32, error) {
	var value float32
	err := binary.Read(reader, binary.LittleEndian, &value)
	return value, err
}

// readInt32 reads a 32-bit signed integer in little-endian format
func readInt32(reader io.Reader) (int32, error) {
	var value int32
	err := binary.Read(reader, binary.LittleEndian, &value)
	return value, err
}

// v6Block represents a single block in the v6 format
type v6Block struct {
	LenBody uint32 // Size of the body in bytes
	Flag    uint32 // Block type identifier
	Body    []byte // Block payload
}

// skipV6Frontmatter skips the variable-length frontmatter section
// The v6 format actually uses a tag-based system with magic delimiter bytes
// For now, we'll read all remaining data and parse it as a whole
func (p *Parser) skipV6Frontmatter(reader io.Reader) error {
	// The frontmatter is complex and variable-length
	// Instead of skipping, we'll just read a byte to check if there's data
	var test [1]byte
	_, err := reader.Read(test[:])
	if err != nil {
		return err
	}

	// Seek back one byte since we just did a test read
	if seeker, ok := reader.(io.Seeker); ok {
		_, err = seeker.Seek(-1, io.SeekCurrent)
		return err
	}

	return nil
}

// readV6Block reads a single block from the v6 format
func (p *Parser) readV6Block(reader io.Reader) (*v6Block, error) {
	block := &v6Block{}

	// Read block header: len_body (u4) + flag (u4)
	lenBody, err := readUint32(reader)
	if err != nil {
		return nil, err
	}

	flag, err := readUint32(reader)
	if err != nil {
		return nil, err
	}

	block.LenBody = lenBody
	block.Flag = flag

	// Read block body
	if lenBody > 0 {
		block.Body = make([]byte, lenBody)
		if _, err := io.ReadFull(reader, block.Body); err != nil {
			return nil, fmt.Errorf("failed to read block body: %w", err)
		}
	}

	return block, nil
}

// extractLayerID extracts the layer ID from a layer definition block
func (p *Parser) extractLayerID(body []byte) string {
	// Layer definition blocks contain magic delimiters and the layer ID
	// Format: magic bytes + layer_id (null-terminated string) + metadata
	// We'll scan for printable characters between magic delimiters

	if len(body) < 10 {
		return ""
	}

	// Look for null-terminated string
	for i := 0; i < len(body)-1; i++ {
		if body[i] >= 0x20 && body[i] <= 0x7E { // Printable ASCII
			// Found start of potential ID
			end := i
			for end < len(body) && body[end] != 0x00 && body[end] >= 0x20 && body[end] <= 0x7E {
				end++
			}
			if end > i {
				return string(body[i:end])
			}
		}
	}

	return ""
}

// parseV6Line parses a line/stroke block
// Returns: Line, layerID, error
func (p *Parser) parseV6Line(body []byte) (Line, string, error) {
	line := Line{}
	reader := bytes.NewReader(body)

	// Parse line header
	// Format (approximate, needs adjustment based on actual data):
	// - Magic delimiters
	// - layer_id (null-terminated string)
	// - line_id
	// - last_line_id
	// - done_flag (u4) - should be 0 for actual stroke data
	// - pen_type (u4)
	// - color (u4)
	// - brush_size (f4)
	// - len_point_array (u4)

	// Skip magic bytes and extract layer ID
	layerID := ""
	for i := 0; i < len(body)-20; i++ {
		if body[i] >= 0x20 && body[i] <= 0x7E {
			end := i
			for end < len(body) && body[end] != 0x00 && body[end] >= 0x20 && body[end] <= 0x7E {
				end++
			}
			if end > i {
				layerID = string(body[i:end])
				// Seek past this string
				reader.Seek(int64(end+1), io.SeekStart)
				break
			}
		}
	}

	// Look for line header pattern
	// We need to find: pen_type (u4), color (u4), brush_size (f4), len_point_array (u4)
	// This is tricky without exact offsets, so we'll use heuristics

	// For now, use simplified parsing
	// Skip to a reasonable offset where header data starts
	if _, err := reader.Seek(20, io.SeekCurrent); err != nil {
		return line, layerID, err
	}

	// Try to read pen type, color, brush size
	penType, _ := readUint32(reader)
	color, _ := readUint32(reader)
	brushSize, _ := readFloat32(reader)
	lenPointArray, _ := readUint32(reader)

	line.BrushType = BrushType(penType)
	line.Color = Color(color)
	line.BrushSize = brushSize

	// Parse points - each point is 14 bytes
	// x (f4) + y (f4) + speed (u1) + width (u1) + direction (u1) + pressure (u1) + padding (2 bytes)
	numPoints := lenPointArray / 14
	if numPoints > 0 && numPoints < 10000 { // Sanity check
		line.Points = make([]Point, 0, numPoints)

		for i := uint32(0); i < numPoints; i++ {
			point := Point{}

			x, err := readFloat32(reader)
			if err != nil {
				break
			}
			y, err := readFloat32(reader)
			if err != nil {
				break
			}

			speed, _ := readUint8(reader)
			width, _ := readUint8(reader)
			direction, _ := readUint8(reader)
			pressure, _ := readUint8(reader)

			// Skip padding (2 bytes)
			reader.Seek(2, io.SeekCurrent)

			point.X = x
			point.Y = y
			point.Speed = float32(speed) / 255.0
			point.Width = float32(width)
			point.Direction = float32(direction) / 255.0 * 360.0 // Convert to degrees
			point.Pressure = float32(pressure) / 255.0

			line.Points = append(line.Points, point)
		}
	}

	return line, layerID, nil
}

// readUint8 reads an 8-bit unsigned integer
func readUint8(reader io.Reader) (uint8, error) {
	var value uint8
	err := binary.Read(reader, binary.LittleEndian, &value)
	return value, err
}

// ParseFile is a convenience function to parse a .rm file from disk
func ParseFile(filename string) (*Document, error) {
	data, err := readFileContents(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	parser := NewParser()
	return parser.Parse(data)
}

// readFileContents reads entire file into memory
func readFileContents(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}
