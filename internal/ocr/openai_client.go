package ocr

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/platinummonkey/legible/internal/logger"
	ollamaTypes "github.com/platinummonkey/legible/internal/ollama"
)

// OpenAIVisionClient implements VisionClient for OpenAI's GPT-4 Vision API
type OpenAIVisionClient struct {
	client      openai.Client
	logger      *logger.Logger
	temperature float64
	maxRetries  int
}

// NewOpenAIVisionClient creates a new OpenAI vision client
func NewOpenAIVisionClient(apiKey string, temperature float64, maxRetries int, log *logger.Logger) *OpenAIVisionClient {
	if log == nil {
		log = logger.Get()
	}

	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
	}

	if maxRetries > 0 {
		opts = append(opts, option.WithMaxRetries(maxRetries))
	}

	client := openai.NewClient(opts...)

	return &OpenAIVisionClient{
		client:      client,
		logger:      log,
		temperature: temperature,
		maxRetries:  maxRetries,
	}
}

// GenerateOCR performs OCR using OpenAI's vision API
func (o *OpenAIVisionClient) GenerateOCR(ctx context.Context, model string, imageData string) ([]ollamaTypes.OCRWord, error) {
	o.logger.WithFields("model", model, "provider", "openai").Debug("Generating OCR with OpenAI")

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
	resp, err := o.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{
				openai.TextContentPart(prompt),
				openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
					URL: fmt.Sprintf("data:image/png;base64,%s", imageData),
				}),
			}),
		},
		Temperature: openai.Float(o.temperature),
	})

	if err != nil {
		return nil, fmt.Errorf("openai API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	// Parse the JSON response
	content := resp.Choices[0].Message.Content
	var ocrResponse struct {
		Words []ollamaTypes.OCRWord `json:"words"`
	}

	if err := json.Unmarshal([]byte(content), &ocrResponse); err != nil {
		o.logger.WithFields("content", content).Debug("Failed to parse OpenAI OCR response")
		return nil, fmt.Errorf("failed to parse OCR response: %w", err)
	}

	o.logger.WithFields("words", len(ocrResponse.Words)).Debug("OpenAI OCR completed")
	return ocrResponse.Words, nil
}

// HealthCheck verifies that the OpenAI API is accessible
func (o *OpenAIVisionClient) HealthCheck(ctx context.Context, model string) error {
	// For OpenAI, we can just check if we can list models
	// or make a simple API call to verify credentials
	_, err := o.client.Models.Get(ctx, model)
	if err != nil {
		return fmt.Errorf("openai health check failed: %w", err)
	}
	return nil
}

// Name returns the provider name
func (o *OpenAIVisionClient) Name() string {
	return "openai"
}

// SupportedModels returns a list of OpenAI vision models
func (o *OpenAIVisionClient) SupportedModels() []string {
	return []string{
		"gpt-4o",
		"gpt-4o-mini",
		"gpt-4-turbo",
		"gpt-4-vision-preview",
	}
}
