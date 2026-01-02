// Package rmparse provides parsing for reMarkable v6 .rm files
package rmparse

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
)

// BlockType represents the type of block in the .rm file
type BlockType uint32

// Block type constants for different types of data in .rm files
const (
	BlockTypeLayerDef  BlockType = 0x1010100 // Layer definition
	BlockTypeLayerName BlockType = 0x2020100 // Layer names
	BlockTypeTextDef   BlockType = 0x7010100 // Text definition
	BlockTypeLayerInfo BlockType = 0x4010100 // Layer info
	BlockTypeLineDef   BlockType = 0x5020200 // Line/stroke definition
)

// Point represents a single point in a stroke
type Point struct {
	X         float32 // X coordinate
	Y         float32 // Y coordinate
	Speed     uint8   // Speed
	Width     uint8   // Width/thickness
	Direction uint8   // Direction/tilt
	Pressure  uint8   // Pressure
}

// Line represents a stroke/line with multiple points
type Line struct {
	PenType   uint32  // Pen type
	Color     uint32  // Color
	BrushSize float32 // Brush size
	Points    []Point // Array of points
}

// Layer represents a layer with its lines
type Layer struct {
	ID    uint32 // Layer ID
	Lines []Line // Lines in this layer
}

// RMFile represents a parsed .rm file
type RMFile struct {
	Version string  // File version
	Layers  []Layer // Layers in the file
}

// ParseRM parses a reMarkable v6 .rm file
func ParseRM(filename string) (*RMFile, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	if len(data) < 43 {
		return nil, fmt.Errorf("file too small")
	}

	// Verify header signature (32 bytes)
	expectedPrefix := "reMarkable .lines file, version="
	actualPrefix := string(data[:32])
	if actualPrefix != expectedPrefix {
		return nil, fmt.Errorf("invalid header signature: got %q, expected %q", actualPrefix, expectedPrefix)
	}

	// Extract version (1 byte)
	version := string(data[32:33])

	rmFile := &RMFile{
		Version: version,
		Layers:  make([]Layer, 0),
	}

	// Track current layer
	var currentLayerID uint32
	layerMap := make(map[uint32]*Layer)

	// Skip complex frontmatter by scanning for first block
	// Blocks start after the frontmatter and have specific flag values
	offset := 43 // Start after basic header
	offset = skipToFirstBlock(data, offset)

	// Parse blocks
	for offset+8 <= len(data) {
		// Read body length (4 bytes)
		bodyLen := binary.LittleEndian.Uint32(data[offset : offset+4])
		offset += 4

		// Read block type/flag (4 bytes)
		blockType := binary.LittleEndian.Uint32(data[offset : offset+4])
		offset += 4

		// Sanity check
		if offset+int(bodyLen) > len(data) {
			break
		}

		// Get block data
		blockData := data[offset : offset+int(bodyLen)]
		offset += int(bodyLen)

		// Process block based on type
		switch BlockType(blockType) {
		case BlockTypeLayerDef:
			// Layer definition - extract layer ID
			if len(blockData) >= 4 {
				layerID := binary.LittleEndian.Uint32(blockData[:4])
				currentLayerID = layerID
				if _, exists := layerMap[layerID]; !exists {
					layer := &Layer{
						ID:    layerID,
						Lines: make([]Line, 0),
					}
					layerMap[layerID] = layer
					rmFile.Layers = append(rmFile.Layers, *layer)
				}
			}

		case BlockTypeLineDef:
			// Line definition - parse stroke data
			line, err := parseLineDef(blockData)
			if err != nil {
				// Log warning but continue
				continue
			}
			// Add line to current layer
			if layer, exists := layerMap[currentLayerID]; exists {
				layer.Lines = append(layer.Lines, line)
				// Update in the slice as well
				for i := range rmFile.Layers {
					if rmFile.Layers[i].ID == currentLayerID {
						rmFile.Layers[i].Lines = append(rmFile.Layers[i].Lines, line)
						break
					}
				}
			}

		// Ignore other block types for now
		case BlockTypeLayerName, BlockTypeTextDef, BlockTypeLayerInfo:
			// Skip
		}
	}

	return rmFile, nil
}

// parseLineDef parses a line definition block according to v6 format
//
//nolint:gocyclo // Binary parsing requires extensive validation
func parseLineDef(data []byte) (Line, error) {
	line := Line{}
	offset := 0

	// Helper to check bounds
	check := func(need int) error {
		if offset+need > len(data) {
			return fmt.Errorf("unexpected end of data at offset %d (need %d more bytes)", offset, need)
		}
		return nil
	}

	// Skip initial header with layer_id, line_id, etc.
	// Structure: 0x1f + layer_id + 0x2f + line_id + 0x3f + last_line_id + 0x4f + id_field_0(2) + 0x54 + done_flag(4)

	// Scan for done_flag position by finding 0x54 magic byte followed by 4-byte done_flag
	foundDoneFlag := false
	for i := 0; i < len(data)-5; i++ {
		if data[i] == 0x54 {
			// Check if this is followed by a plausible done_flag (0 or non-zero)
			offset = i + 1
			foundDoneFlag = true
			break
		}
	}

	if !foundDoneFlag {
		return line, fmt.Errorf("could not find done_flag marker (0x54)")
	}

	// Read done_flag (4 bytes)
	if err := check(4); err != nil {
		return line, err
	}
	doneFlag := binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4

	if doneFlag != 0 {
		// Empty line, no points
		return line, nil
	}

	// Expect 0x6c magic
	if err := check(1); err != nil {
		return line, err
	}
	if data[offset] != 0x6c {
		return line, fmt.Errorf("expected magic 0x6c at offset %d, got 0x%02x", offset, data[offset])
	}
	offset++

	// Skip len_block_0 (4 bytes)
	if err := check(4); err != nil {
		return line, err
	}
	offset += 4

	// Expect 0x03, 0x14 magic
	if err := check(2); err != nil {
		return line, err
	}
	if data[offset] != 0x03 || data[offset+1] != 0x14 {
		return line, fmt.Errorf("expected magic [0x03, 0x14] at offset %d", offset)
	}
	offset += 2

	// Read pen_type (4 bytes)
	if err := check(4); err != nil {
		return line, err
	}
	line.PenType = binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4

	// Expect 0x24 magic
	if err := check(1); err != nil {
		return line, err
	}
	if data[offset] != 0x24 {
		return line, fmt.Errorf("expected magic 0x24 at offset %d, got 0x%02x", offset, data[offset])
	}
	offset++

	// Read color (4 bytes)
	if err := check(4); err != nil {
		return line, err
	}
	line.Color = binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4

	// Expect 0x38, 0x00, 0x00, 0x00, 0x00 magic
	if err := check(5); err != nil {
		return line, err
	}
	if data[offset] != 0x38 {
		return line, fmt.Errorf("expected magic 0x38 at offset %d, got 0x%02x", offset, data[offset])
	}
	offset += 5

	// Read brush_size (4 bytes, float32)
	if err := check(4); err != nil {
		return line, err
	}
	line.BrushSize = math.Float32frombits(binary.LittleEndian.Uint32(data[offset : offset+4]))
	offset += 4

	// Expect 0x44, 0x00, 0x00, 0x00, 0x00 magic
	if err := check(5); err != nil {
		return line, err
	}
	offset += 5

	// Expect 0x5c magic
	if err := check(1); err != nil {
		return line, err
	}
	if data[offset] != 0x5c {
		return line, fmt.Errorf("expected magic 0x5c at offset %d, got 0x%02x", offset, data[offset])
	}
	offset++

	// Read len_point_array (4 bytes)
	if err := check(4); err != nil {
		return line, err
	}
	lenPointArray := binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4

	// Calculate number of points (each point is 14 bytes)
	numPoints := lenPointArray / 14

	// Read points
	line.Points = make([]Point, 0, numPoints)
	for i := uint32(0); i < numPoints; i++ {
		if err := check(14); err != nil {
			break
		}
		point := Point{
			X:         math.Float32frombits(binary.LittleEndian.Uint32(data[offset : offset+4])),
			Y:         math.Float32frombits(binary.LittleEndian.Uint32(data[offset+4 : offset+8])),
			Speed:     data[offset+8],
			Width:     data[offset+10], // Note: offset 9 is padding
			Direction: data[offset+12], // Note: offset 11 is padding
			Pressure:  data[offset+13],
		}
		line.Points = append(line.Points, point)
		offset += 14
	}

	return line, nil
}

// skipToFirstBlock scans for the first known block type flag
func skipToFirstBlock(data []byte, start int) int {
	knownFlags := []uint32{
		uint32(BlockTypeLayerDef),
		uint32(BlockTypeLayerName),
		uint32(BlockTypeTextDef),
		uint32(BlockTypeLayerInfo),
		uint32(BlockTypeLineDef),
	}

	// Scan for first occurrence of a known block flag
	// A block starts with: len_body (4 bytes) + flag (4 bytes)
	for i := start; i+8 <= len(data); i++ {
		// Read potential flag at position i+4 (after len_body)
		if i+8 > len(data) {
			break
		}
		flag := binary.LittleEndian.Uint32(data[i+4 : i+8])

		// Check if this is a known flag
		for _, knownFlag := range knownFlags {
			if flag == knownFlag {
				return i
			}
		}
	}

	return start
}
