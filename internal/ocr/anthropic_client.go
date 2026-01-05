package ocr

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/platinummonkey/legible/internal/logger"
	ollamaTypes "github.com/platinummonkey/legible/internal/ollama"
)

// AnthropicVisionClient implements VisionClient for Anthropic's Claude API
type AnthropicVisionClient struct {
	client      anthropic.Client
	logger      *logger.Logger
	temperature float64
	maxRetries  int
}

// NewAnthropicVisionClient creates a new Anthropic Claude vision client
func NewAnthropicVisionClient(apiKey string, temperature float64, maxRetries int, log *logger.Logger) *AnthropicVisionClient {
	if log == nil {
		log = logger.Get()
	}

	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
	}

	if maxRetries > 0 {
		opts = append(opts, option.WithMaxRetries(maxRetries))
	}

	client := anthropic.NewClient(opts...)

	return &AnthropicVisionClient{
		client:      client,
		logger:      log,
		temperature: temperature,
		maxRetries:  maxRetries,
	}
}

// GenerateOCR performs OCR using Anthropic's Claude vision API
func (a *AnthropicVisionClient) GenerateOCR(ctx context.Context, model string, imageData string) ([]ollamaTypes.OCRWord, error) {
	a.logger.WithFields("model", model, "provider", "anthropic").Debug("Generating OCR with Anthropic Claude")

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

	// Make the API call
	resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: 4096,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewTextBlock(prompt),
				anthropic.NewImageBlockBase64("image/png", imageData),
			),
		},
		Temperature: anthropic.Float(a.temperature),
	})

	if err != nil {
		return nil, fmt.Errorf("anthropic API error: %w", err)
	}

	if len(resp.Content) == 0 {
		return nil, fmt.Errorf("no response from Anthropic")
	}

	// Extract text content from response
	var content string
	for _, block := range resp.Content {
		if block.Type == "text" {
			content = block.Text
			break
		}
	}

	if content == "" {
		return nil, fmt.Errorf("no text content in Anthropic response")
	}

	// Parse the JSON response
	var ocrResponse struct {
		Words []ollamaTypes.OCRWord `json:"words"`
	}

	if err := json.Unmarshal([]byte(content), &ocrResponse); err != nil {
		a.logger.WithFields("content", content).Debug("Failed to parse Anthropic OCR response")
		return nil, fmt.Errorf("failed to parse OCR response: %w", err)
	}

	a.logger.WithFields("words", len(ocrResponse.Words)).Debug("Anthropic OCR completed")
	return ocrResponse.Words, nil
}

// HealthCheck verifies that the Anthropic API is accessible
func (a *AnthropicVisionClient) HealthCheck(ctx context.Context, model string) error {
	// Make a minimal API call to verify credentials
	_, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: 10,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock("test")),
		},
	})

	if err != nil {
		return fmt.Errorf("anthropic health check failed: %w", err)
	}

	return nil
}

// Name returns the provider name
func (a *AnthropicVisionClient) Name() string {
	return "anthropic"
}

// SupportedModels returns a list of Anthropic Claude models with vision capabilities
func (a *AnthropicVisionClient) SupportedModels() []string {
	return []string{
		"claude-3-5-sonnet-20241022",
		"claude-3-5-sonnet-20240620",
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
	}
}
