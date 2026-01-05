package ocr

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"github.com/platinummonkey/legible/internal/logger"
	ollamaTypes "github.com/platinummonkey/legible/internal/ollama"
	"google.golang.org/api/option"
)

// GoogleVisionClient implements VisionClient for Google's Gemini API
type GoogleVisionClient struct {
	client      *genai.Client
	logger      *logger.Logger
	temperature float64
	maxRetries  int
}

// NewGoogleVisionClient creates a new Google Gemini vision client
func NewGoogleVisionClient(ctx context.Context, apiKey string, temperature float64, maxRetries int, log *logger.Logger) (*GoogleVisionClient, error) {
	if log == nil {
		log = logger.Get()
	}

	opts := []option.ClientOption{
		option.WithAPIKey(apiKey),
	}

	client, err := genai.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &GoogleVisionClient{
		client:      client,
		logger:      log,
		temperature: temperature,
		maxRetries:  maxRetries,
	}, nil
}

// GenerateOCR performs OCR using Google's Gemini vision API
func (g *GoogleVisionClient) GenerateOCR(ctx context.Context, model string, imageData string) ([]ollamaTypes.OCRWord, error) {
	g.logger.WithFields("model", model, "provider", "google").Debug("Generating OCR with Google Gemini")

	// Construct the OCR prompt
	prompt := `Extract all handwritten text from this image of a reMarkable tablet note.
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
- Return {"words": []} if no text found`

	// Decode base64 image data
	imgBytes, err := base64.StdEncoding.DecodeString(imageData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 image: %w", err)
	}

	// Get the generative model
	genModel := g.client.GenerativeModel(model)
	genModel.SetTemperature(float32(g.temperature))
	genModel.ResponseMIMEType = "application/json"

	// Make the API call
	resp, err := genModel.GenerateContent(
		ctx,
		genai.Text(prompt),
		genai.ImageData("png", imgBytes),
	)

	if err != nil {
		return nil, fmt.Errorf("gemini API error: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no response from Gemini")
	}

	// Extract text content from response
	var content string
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			content = string(txt)
			break
		}
	}

	if content == "" {
		return nil, fmt.Errorf("no text content in Gemini response")
	}

	// Parse the JSON response
	var ocrResponse struct {
		Words []ollamaTypes.OCRWord `json:"words"`
	}

	if err := json.Unmarshal([]byte(content), &ocrResponse); err != nil {
		g.logger.WithFields("content", content).Debug("Failed to parse Gemini OCR response")
		return nil, fmt.Errorf("failed to parse OCR response: %w", err)
	}

	g.logger.WithFields("words", len(ocrResponse.Words)).Debug("Gemini OCR completed")
	return ocrResponse.Words, nil
}

// HealthCheck verifies that the Gemini API is accessible
func (g *GoogleVisionClient) HealthCheck(ctx context.Context, model string) error {
	// Make a minimal API call to verify credentials
	genModel := g.client.GenerativeModel(model)
	_, err := genModel.GenerateContent(ctx, genai.Text("test"))

	if err != nil {
		return fmt.Errorf("gemini health check failed: %w", err)
	}

	return nil
}

// Name returns the provider name
func (g *GoogleVisionClient) Name() string {
	return "google"
}

// SupportedModels returns a list of Google Gemini vision models
func (g *GoogleVisionClient) SupportedModels() []string {
	return []string{
		"gemini-1.5-pro",
		"gemini-1.5-pro-latest",
		"gemini-1.5-flash",
		"gemini-1.5-flash-latest",
		"gemini-pro-vision",
	}
}

// Close closes the Google client
func (g *GoogleVisionClient) Close() error {
	return g.client.Close()
}
