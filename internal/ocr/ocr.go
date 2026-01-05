// Package ocr provides optical character recognition capabilities for document processing.
package ocr

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"strings"
	"time"

	"github.com/platinummonkey/legible/internal/logger"
	"github.com/platinummonkey/legible/internal/ollama"
)

const (
	// DefaultModel is the default Ollama model for OCR
	DefaultModel = "llava"

	// DefaultTemperature for OCR (0.0 for deterministic output)
	DefaultTemperature = 0.0
)

// OCR prompt template for extracting text with bounding boxes
const ocrPromptTemplate = `You are analyzing a handwritten note from a reMarkable tablet.

Extract ALL visible handwritten text from this image.
Return ONLY valid JSON with no markdown formatting, no code blocks, no explanation.

Format:
{
  "words": [
    {"text": "word", "bbox": [x, y, width, height], "confidence": 0.95}
  ]
}

Rules:
- Include ALL text, even if partially visible
- bbox coordinates are pixels from top-left (0,0)
- confidence is 0.0-1.0, use 0.8 if uncertain
- Return {"words": []} if no text found
`

// Processor handles OCR processing using a vision client
type Processor struct {
	logger         *logger.Logger
	visionClient   VisionClient
	model          string
	promptTemplate string
	imageDimCache  map[int]image.Point // cache image dimensions by page number
}

// Config holds configuration for the OCR processor
type Config struct {
	Logger       *logger.Logger
	VisionClient VisionClient // Pre-configured vision client (optional, for advanced usage)
	// Legacy Ollama-specific fields (deprecated, use VisionConfig instead)
	OllamaEndpoint string  // default: "http://localhost:11434"
	Model          string  // default: "llava"
	Temperature    float64 // default: 0.0 for deterministic output
	MaxRetries     int     // default: 3
	// New unified configuration
	VisionConfig *VisionClientConfig // Vision client configuration (preferred)
}

// New creates a new OCR processor with a vision client
func New(cfg *Config) (*Processor, error) {
	log := cfg.Logger
	if log == nil {
		log = logger.Get()
	}

	// Determine the vision client to use
	var visionClient VisionClient
	var model string

	// Priority 1: Use pre-configured vision client if provided
	if cfg.VisionClient != nil {
		visionClient = cfg.VisionClient
		model = cfg.Model
		if model == "" {
			model = DefaultModel
		}
		log.WithFields("provider", visionClient.Name()).Info("Using pre-configured vision client")
	} else if cfg.VisionConfig != nil {
		// Priority 2: Use VisionConfig to create client
		ctx := context.Background()
		client, err := NewVisionClient(ctx, cfg.VisionConfig, log)
		if err != nil {
			return nil, fmt.Errorf("failed to create vision client: %w", err)
		}
		visionClient = client
		model = cfg.VisionConfig.Model
		log.WithFields("provider", cfg.VisionConfig.Provider, "model", model).Info("Created vision client from config")
	} else {
		// Priority 3: Fall back to legacy Ollama configuration for backward compatibility
		model = cfg.Model
		if model == "" {
			model = DefaultModel
		}
		endpoint := cfg.OllamaEndpoint
		if endpoint == "" {
			endpoint = "http://localhost:11434"
		}
		maxRetries := cfg.MaxRetries
		if maxRetries == 0 {
			maxRetries = 3
		}
		visionClient = NewOllamaVisionClient(endpoint, maxRetries, log)
		log.WithFields("endpoint", endpoint, "model", model).Info("Using legacy Ollama configuration")
	}

	return &Processor{
		logger:         log,
		visionClient:   visionClient,
		model:          model,
		promptTemplate: ocrPromptTemplate,
		imageDimCache:  make(map[int]image.Point),
	}, nil
}

// ProcessImage performs OCR on an image and returns structured results
func (p *Processor) ProcessImage(imageData []byte, pageNumber int) (*PageOCR, error) {
	p.logger.WithFields("page", pageNumber, "image_size", len(imageData), "provider", p.visionClient.Name()).Debug("Processing image with OCR")

	startTime := time.Now()

	// Decode image to get dimensions
	img, format, err := image.Decode(strings.NewReader(string(imageData)))
	if err != nil {
		// If we can't decode, try without dimensions (less accurate OCR)
		p.logger.WithFields("page", pageNumber, "error", err).Warn("Failed to decode image for dimensions")
	}

	var width, height int
	if img != nil {
		bounds := img.Bounds()
		width = bounds.Dx()
		height = bounds.Dy()
		p.imageDimCache[pageNumber] = image.Point{X: width, Y: height}
		p.logger.WithFields("page", pageNumber, "width", width, "height", height, "format", format).Debug("Image dimensions")
	} else if cached, ok := p.imageDimCache[pageNumber]; ok {
		// Use cached dimensions if available
		width = cached.X
		height = cached.Y
	}

	// Encode image to base64
	base64Image := ollama.EncodeBytesToBase64(imageData)

	// Call vision client OCR API
	ctx := context.Background()
	words, err := p.visionClient.GenerateOCR(ctx, p.model, base64Image)
	if err != nil {
		return nil, fmt.Errorf("failed to generate OCR with %s: %w", p.visionClient.Name(), err)
	}

	// Convert response to PageOCR
	pageOCR := NewPageOCR(pageNumber, width, height, p.model)
	for _, oWord := range words {
		if len(oWord.BBox) < 4 {
			p.logger.WithFields("word", oWord.Text, "bbox", oWord.BBox).Warn("Invalid bounding box, skipping word")
			continue
		}

		// Convert confidence from 0.0-1.0 to 0-100 scale
		confidence := oWord.Confidence * 100.0
		if confidence == 0 {
			confidence = 80.0 // default confidence if not provided
		}

		word := NewWord(
			oWord.Text,
			NewRectangle(oWord.BBox[0], oWord.BBox[1], oWord.BBox[2], oWord.BBox[3]),
			confidence,
		)
		pageOCR.AddWord(word)
	}

	// Build full text and calculate confidence
	pageOCR.BuildText()
	pageOCR.CalculateConfidence()

	duration := time.Since(startTime)
	p.logger.WithFields(
		"page", pageNumber,
		"words", len(pageOCR.Words),
		"confidence", pageOCR.Confidence,
		"duration", duration,
		"provider", p.visionClient.Name(),
	).Info("OCR processing completed")

	return pageOCR, nil
}

// ProcessImageWithCustomPrompt allows using a custom prompt template
func (p *Processor) ProcessImageWithCustomPrompt(imageData []byte, pageNumber int, customPrompt string) (*PageOCR, error) {
	originalPrompt := p.promptTemplate
	p.promptTemplate = customPrompt
	defer func() {
		p.promptTemplate = originalPrompt
	}()

	return p.ProcessImage(imageData, pageNumber)
}

// HealthCheck verifies that the vision client is accessible and the model is available
func (p *Processor) HealthCheck() error {
	ctx := context.Background()

	// Use the vision client's health check
	if err := p.visionClient.HealthCheck(ctx, p.model); err != nil {
		return fmt.Errorf("%s health check failed: %w", p.visionClient.Name(), err)
	}

	p.logger.WithFields("provider", p.visionClient.Name(), "model", p.model).Info("Health check passed")
	return nil
}

// Model returns the configured model name
func (p *Processor) Model() string {
	return p.model
}

// ollamaOCRResponse represents the JSON response from Ollama
type ollamaOCRResponse struct {
	Words []struct {
		Text       string  `json:"text"`
		BBox       []int   `json:"bbox"`
		Confidence float64 `json:"confidence"`
	} `json:"words"`
}

// parseOllamaResponse parses the Ollama JSON response into OCR words
func parseOllamaResponse(jsonResponse string) ([]ollama.OCRWord, error) {
	var response ollamaOCRResponse
	if err := json.Unmarshal([]byte(jsonResponse), &response); err != nil {
		return nil, fmt.Errorf("failed to parse Ollama OCR response: %w", err)
	}

	words := make([]ollama.OCRWord, len(response.Words))
	for i, w := range response.Words {
		words[i] = ollama.OCRWord{
			Text:       w.Text,
			BBox:       w.BBox,
			Confidence: w.Confidence,
		}
	}

	return words, nil
}
