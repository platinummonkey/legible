package ocr

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"strings"
	"time"

	"github.com/platinummonkey/remarkable-sync/internal/logger"
	"github.com/platinummonkey/remarkable-sync/internal/ollama"
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

// Processor handles OCR processing using Ollama
type Processor struct {
	logger         *logger.Logger
	ollamaClient   *ollama.Client
	model          string
	promptTemplate string
	imageDimCache  map[int]image.Point // cache image dimensions by page number
}

// Config holds configuration for the OCR processor
type Config struct {
	Logger         *logger.Logger
	OllamaEndpoint string  // default: "http://localhost:11434"
	Model          string  // default: "llava"
	Temperature    float64 // default: 0.0 for deterministic output
	MaxRetries     int     // default: 3
}

// New creates a new OCR processor using Ollama
func New(cfg *Config) *Processor {
	log := cfg.Logger
	if log == nil {
		log = logger.Get()
	}

	model := cfg.Model
	if model == "" {
		model = DefaultModel
	}

	// Build Ollama client options
	clientOpts := []ollama.ClientOption{}
	if cfg.OllamaEndpoint != "" {
		clientOpts = append(clientOpts, ollama.WithEndpoint(cfg.OllamaEndpoint))
	}
	if cfg.MaxRetries > 0 {
		clientOpts = append(clientOpts, ollama.WithMaxRetries(cfg.MaxRetries))
	}
	clientOpts = append(clientOpts, ollama.WithLogger(log))

	return &Processor{
		logger:         log,
		ollamaClient:   ollama.NewClient(clientOpts...),
		model:          model,
		promptTemplate: ocrPromptTemplate,
		imageDimCache:  make(map[int]image.Point),
	}
}

// ProcessImage performs OCR on an image and returns structured results
func (p *Processor) ProcessImage(imageData []byte, pageNumber int) (*PageOCR, error) {
	p.logger.WithFields("page", pageNumber, "image_size", len(imageData)).Debug("Processing image with Ollama OCR")

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

	// Call Ollama OCR API
	ctx := context.Background()
	words, err := p.ollamaClient.GenerateOCR(ctx, p.model, base64Image)
	if err != nil {
		return nil, fmt.Errorf("failed to generate OCR with Ollama: %w", err)
	}

	// Convert Ollama response to PageOCR
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
	).Info("Ollama OCR processing completed")

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

// HealthCheck verifies that Ollama is accessible and the model is available
func (p *Processor) HealthCheck() error {
	ctx := context.Background()

	// Check if Ollama is running
	if err := p.ollamaClient.HealthCheck(ctx); err != nil {
		return fmt.Errorf("ollama health check failed: %w", err)
	}

	// Check if model is available
	models, err := p.ollamaClient.ListModels(ctx)
	if err != nil {
		return fmt.Errorf("failed to list models: %w", err)
	}

	// Look for the configured model
	modelFound := false
	for _, m := range models.Models {
		if strings.Contains(m.Name, p.model) {
			modelFound = true
			p.logger.WithFields("model", m.Name, "size", m.Size).Debug("Found OCR model")
			break
		}
	}

	if !modelFound {
		p.logger.WithFields("model", p.model).Warn("Model not found, attempting to pull")
		if err := p.ollamaClient.PullModel(ctx, p.model); err != nil {
			return fmt.Errorf("model %s not found and pull failed: %w", p.model, err)
		}
	}

	return nil
}

// Model returns the configured model name
func (p *Processor) Model() string {
	return p.model
}

// ollamaOCRResponse represents the JSON response from Ollama
type ollamaOCRResponse struct {
	Words []struct {
		Text       string    `json:"text"`
		BBox       []int     `json:"bbox"`
		Confidence float64   `json:"confidence"`
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
