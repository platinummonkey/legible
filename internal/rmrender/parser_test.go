package rmrender

import (
	"bytes"
	"testing"
)

func TestParseV6_ExampleFile(t *testing.T) {
	// Test with the first example .rm file
	doc, err := ParseFile("../../example/b68e57f6-4fc9-4a71-b300-e0fa100ef8d7/aefd8acc-a17d-4e24-a76c-66a3ee15b4ba.rm")
	if err != nil {
		t.Fatalf("Failed to parse example file: %v", err)
	}

	// Verify document was parsed
	if doc == nil {
		t.Fatal("Parsed document is nil")
	}

	// Verify version
	if doc.Version != Version6 {
		t.Errorf("Expected version 6, got %s", doc.Version)
	}

	// Log document info
	t.Logf("Parsed document with %d layers", len(doc.Layers))

	totalStrokes := 0
	totalPoints := 0
	for i, layer := range doc.Layers {
		strokeCount := len(layer.Lines)
		totalStrokes += strokeCount
		t.Logf("Layer %d: %d strokes", i, strokeCount)

		for j, stroke := range layer.Lines {
			pointCount := len(stroke.Points)
			totalPoints += pointCount
			if pointCount > 0 {
				t.Logf("  Stroke %d: brush=%s, color=%s, size=%.2f, points=%d",
					j, stroke.BrushType, stroke.Color, stroke.BrushSize, pointCount)
			}
		}
	}

	t.Logf("Total: %d strokes, %d points", totalStrokes, totalPoints)

	// Verify we got some data
	if totalStrokes == 0 {
		t.Error("Expected to parse some strokes, got 0")
	}
	if totalPoints == 0 {
		t.Error("Expected to parse some points, got 0")
	}
}

func TestParseV6_SecondExampleFile(t *testing.T) {
	// Test with the second example .rm file
	doc, err := ParseFile("../../example/b68e57f6-4fc9-4a71-b300-e0fa100ef8d7/7ac5c320-e3e5-4c6c-8adc-204662ee929a.rm")
	if err != nil {
		t.Fatalf("Failed to parse example file: %v", err)
	}

	// Verify document was parsed
	if doc == nil {
		t.Fatal("Parsed document is nil")
	}

	// Verify version
	if doc.Version != Version6 {
		t.Errorf("Expected version 6, got %s", doc.Version)
	}

	// Log document info
	t.Logf("Parsed document with %d layers", len(doc.Layers))

	totalStrokes := 0
	totalPoints := 0
	for i, layer := range doc.Layers {
		strokeCount := len(layer.Lines)
		totalStrokes += strokeCount
		t.Logf("Layer %d: %d strokes", i, strokeCount)

		for j, stroke := range layer.Lines {
			pointCount := len(stroke.Points)
			totalPoints += pointCount
			if j < 3 && pointCount > 0 { // Only log first 3 strokes
				t.Logf("  Stroke %d: brush=%s, color=%s, size=%.2f, points=%d",
					j, stroke.BrushType, stroke.Color, stroke.BrushSize, pointCount)
			}
		}
	}

	t.Logf("Total: %d strokes, %d points", totalStrokes, totalPoints)
}

func TestParseHeader(t *testing.T) {
	tests := []struct {
		name        string
		header      []byte
		wantVersion Version
		wantErr     bool
	}{
		{
			name:        "Version 6",
			header:      []byte("reMarkable .lines file, version=6          "),
			wantVersion: Version6,
			wantErr:     false,
		},
		{
			name:        "Version 5",
			header:      []byte("reMarkable .lines file, version=5          "),
			wantVersion: Version5,
			wantErr:     false,
		},
		{
			name:        "Version 3",
			header:      []byte("reMarkable .lines file, version=3          "),
			wantVersion: Version3,
			wantErr:     false,
		},
		{
			name:        "Invalid header",
			header:      []byte("Not a reMarkable file                      "),
			wantVersion: VersionUnknown,
			wantErr:     true,
		},
		{
			name:        "Unknown version",
			header:      []byte("reMarkable .lines file, version=9          "),
			wantVersion: VersionUnknown,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			version, err := p.parseHeader(bytes.NewReader(tt.header))

			if (err != nil) != tt.wantErr {
				t.Errorf("parseHeader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if version != tt.wantVersion {
				t.Errorf("parseHeader() version = %v, want %v", version, tt.wantVersion)
			}
		})
	}
}

func TestEstimateComplexity(t *testing.T) {
	tests := []struct {
		name string
		doc  *Document
		want int
	}{
		{
			name: "nil document",
			doc:  nil,
			want: 0,
		},
		{
			name: "empty document",
			doc:  &Document{Layers: []Layer{}},
			want: 0,
		},
		{
			name: "document with strokes",
			doc: &Document{
				Layers: []Layer{
					{
						Lines: []Line{
							{Points: make([]Point, 10)},
							{Points: make([]Point, 20)},
						},
					},
					{
						Lines: []Line{
							{Points: make([]Point, 15)},
						},
					},
				},
			},
			want: 45, // 10 + 20 + 15
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateComplexity(tt.doc)
			if got != tt.want {
				t.Errorf("EstimateComplexity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExportMetadata(t *testing.T) {
	doc := &Document{
		Version: Version6,
		Layers: []Layer{
			{
				Lines: []Line{
					{
						BrushType: BrushBallpoint,
						Points:    make([]Point, 10),
					},
					{
						BrushType: BrushHighlighter,
						Points:    make([]Point, 20),
					},
				},
			},
		},
	}

	metadata := ExportMetadata(doc)

	if metadata == nil {
		t.Fatal("ExportMetadata() returned nil")
	}

	if metadata["version"] != "v6" {
		t.Errorf("Expected version v6, got %v", metadata["version"])
	}

	if metadata["layer_count"] != 1 {
		t.Errorf("Expected 1 layer, got %v", metadata["layer_count"])
	}

	if metadata["stroke_count"] != 2 {
		t.Errorf("Expected 2 strokes, got %v", metadata["stroke_count"])
	}

	if metadata["point_count"] != 30 {
		t.Errorf("Expected 30 points, got %v", metadata["point_count"])
	}
}
