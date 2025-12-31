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

const (
	BlockTypeLayerDef  BlockType = 0x1010100 // Layer definition
	BlockTypeLayerName BlockType = 0x2020100 // Layer names
	BlockTypeTextDef   BlockType = 0x7010100 // Text definition
	BlockTypeLayerInfo BlockType = 0x4010100 // Layer info
	BlockTypeLineDef   BlockType = 0x5020200 // Line/stroke definition
)

// Point represents a single point in a stroke
type Point struct {
	X        float32 // X coordinate
	Y        float32 // Y coordinate
	Speed    uint8   // Speed
	Width    uint8   // Width/thickness
	Direction uint8  // Direction/tilt
	Pressure uint8  // Pressure
}

// Line represents a stroke/line with multiple points
type Line struct {
	PenType   uint32   // Pen type
	Color     uint32   // Color
	BrushSize float32  // Brush size
	Points    []Point  // Array of points
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

// parseLineDef parses a line definition block
func parseLineDef(data []byte) (Line, error) {
	if len(data) < 16 {
		return Line{}, fmt.Errorf("line block too small")
	}

	line := Line{}

	// Read pen type (4 bytes)
	line.PenType = binary.LittleEndian.Uint32(data[0:4])

	// Read color (4 bytes)
	line.Color = binary.LittleEndian.Uint32(data[4:8])

	// Skip 4 bytes (padding)

	// Read brush size (4 bytes, float32)
	line.BrushSize = math.Float32frombits(binary.LittleEndian.Uint32(data[12:16]))

	// Read number of points (4 bytes)
	offset := 16
	if len(data) < offset+4 {
		return line, nil
	}
	numPoints := binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4

	// Read points (14 bytes each)
	line.Points = make([]Point, 0, numPoints)
	for i := uint32(0); i < numPoints; i++ {
		if len(data) < offset+14 {
			break
		}
		point := Point{
			X:         math.Float32frombits(binary.LittleEndian.Uint32(data[offset : offset+4])),
			Y:         math.Float32frombits(binary.LittleEndian.Uint32(data[offset+4 : offset+8])),
			Speed:     data[offset+8],
			Width:     data[offset+9],
			Direction:  data[offset+10],
			Pressure:  data[offset+11],
			// bytes 12-13 are padding
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
