package converter

import (
	"errors"
	"testing"
	"time"
)

func TestNewConversionOptions(t *testing.T) {
	opts := NewConversionOptions("/input/test.rmdoc", "/output/test.pdf")

	if opts.InputPath != "/input/test.rmdoc" {
		t.Errorf("expected InputPath /input/test.rmdoc, got %s", opts.InputPath)
	}

	if opts.OutputPath != "/output/test.pdf" {
		t.Errorf("expected OutputPath /output/test.pdf, got %s", opts.OutputPath)
	}

	if !opts.IncludeAnnotations {
		t.Error("expected IncludeAnnotations to be true by default")
	}

	if opts.Quality != DefaultQuality {
		t.Errorf("expected Quality %d, got %d", DefaultQuality, opts.Quality)
	}

	if opts.DPI != DefaultDPI {
		t.Errorf("expected DPI %d, got %d", DefaultDPI, opts.DPI)
	}

	if opts.PaperSize != DefaultPaperSize {
		t.Errorf("expected PaperSize %s, got %s", DefaultPaperSize, opts.PaperSize)
	}

	if opts.Orientation != DefaultOrientation {
		t.Errorf("expected Orientation %s, got %s", DefaultOrientation, opts.Orientation)
	}

	if !opts.Compression {
		t.Error("expected Compression to be true by default")
	}
}

func TestNewConversionResult(t *testing.T) {
	result := NewConversionResult()

	if result.Success {
		t.Error("expected Success to be false by default")
	}

	if result.Warnings == nil {
		t.Error("Warnings should be initialized")
	}

	if len(result.Warnings) != 0 {
		t.Error("Warnings should be empty initially")
	}
}

func TestConversionResult_AddWarning(t *testing.T) {
	result := NewConversionResult()

	result.AddWarning("warning 1")
	result.AddWarning("warning 2")

	if len(result.Warnings) != 2 {
		t.Errorf("expected 2 warnings, got %d", len(result.Warnings))
	}

	if result.Warnings[0] != "warning 1" {
		t.Errorf("expected first warning 'warning 1', got %s", result.Warnings[0])
	}
}

func TestConversionResult_SetError(t *testing.T) {
	result := NewConversionResult()
	result.Success = true // Start as successful

	err := errors.New("conversion failed")
	result.SetError(err)

	if result.Success {
		t.Error("expected Success to be false after error")
	}

	if result.Error != "conversion failed" {
		t.Errorf("expected error 'conversion failed', got %s", result.Error)
	}
}

func TestConversionResult_SetSuccess(t *testing.T) {
	result := NewConversionResult()
	duration := 5 * time.Second

	result.SetSuccess("/output/test.pdf", 10, 1024000, duration)

	if !result.Success {
		t.Error("expected Success to be true")
	}

	if result.OutputPath != "/output/test.pdf" {
		t.Errorf("expected OutputPath /output/test.pdf, got %s", result.OutputPath)
	}

	if result.PageCount != 10 {
		t.Errorf("expected PageCount 10, got %d", result.PageCount)
	}

	if result.FileSize != 1024000 {
		t.Errorf("expected FileSize 1024000, got %d", result.FileSize)
	}

	if result.Duration != duration {
		t.Errorf("expected Duration %v, got %v", duration, result.Duration)
	}
}

func TestPaperSize_Dimensions(t *testing.T) {
	tests := []struct {
		name       string
		paperSize  PaperSize
		wantWidth  int
		wantHeight int
	}{
		{
			name:       "A4",
			paperSize:  PaperSizeA4,
			wantWidth:  2480,
			wantHeight: 3508,
		},
		{
			name:       "A5",
			paperSize:  PaperSizeA5,
			wantWidth:  1748,
			wantHeight: 2480,
		},
		{
			name:       "Letter",
			paperSize:  PaperSizeLetter,
			wantWidth:  2550,
			wantHeight: 3300,
		},
		{
			name:       "Legal",
			paperSize:  PaperSizeLegal,
			wantWidth:  2550,
			wantHeight: 4200,
		},
		{
			name:       "Remarkable",
			paperSize:  PaperSizeRemarkable,
			wantWidth:  1404,
			wantHeight: 1872,
		},
		{
			name:       "Unknown defaults to Remarkable",
			paperSize:  PaperSize("Unknown"),
			wantWidth:  1404,
			wantHeight: 1872,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			width, height := tt.paperSize.Dimensions()
			if width != tt.wantWidth {
				t.Errorf("expected width %d, got %d", tt.wantWidth, width)
			}
			if height != tt.wantHeight {
				t.Errorf("expected height %d, got %d", tt.wantHeight, height)
			}
		})
	}
}

func TestPDFMetadata(t *testing.T) {
	now := time.Now()
	metadata := PDFMetadata{
		Title:            "Test Document",
		Author:           "Test Author",
		Subject:          "Testing",
		Keywords:         []string{"test", "pdf"},
		Creator:          "remarkable-sync",
		Producer:         "remarkable-sync",
		CreationDate:     now,
		ModificationDate: now,
	}

	if metadata.Title != "Test Document" {
		t.Errorf("expected title 'Test Document', got %s", metadata.Title)
	}

	if len(metadata.Keywords) != 2 {
		t.Errorf("expected 2 keywords, got %d", len(metadata.Keywords))
	}
}

func TestToolTypeConstants(t *testing.T) {
	tools := []ToolType{
		ToolTypePen,
		ToolTypePencil,
		ToolTypeHighlighter,
		ToolTypeEraser,
		ToolTypeMarker,
	}

	for _, tool := range tools {
		if tool == "" {
			t.Errorf("tool type should not be empty")
		}
	}
}

func TestOrientationConstants(t *testing.T) {
	if OrientationPortrait == "" {
		t.Error("OrientationPortrait should not be empty")
	}

	if OrientationLandscape == "" {
		t.Error("OrientationLandscape should not be empty")
	}

	if OrientationPortrait == OrientationLandscape {
		t.Error("OrientationPortrait and OrientationLandscape should be different")
	}
}
