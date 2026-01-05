// Package ocr provides optical character recognition capabilities for document processing.
package ocr

import (
	"context"

	"github.com/platinummonkey/legible/internal/ollama"
)

// VisionClient is an interface for vision-capable LLM providers that can perform OCR
type VisionClient interface {
	// GenerateOCR performs OCR on a base64-encoded image and returns structured word data
	GenerateOCR(ctx context.Context, model string, imageData string) ([]ollama.OCRWord, error)

	// HealthCheck verifies that the provider is accessible and the model is available
	HealthCheck(ctx context.Context, model string) error

	// Name returns the name of the provider (e.g., "ollama", "openai", "anthropic", "google")
	Name() string

	// SupportedModels returns a list of supported model names for this provider
	SupportedModels() []string
}

// ProviderType represents the type of LLM provider
type ProviderType string

const (
	// ProviderOllama represents a local Ollama instance
	ProviderOllama ProviderType = "ollama"

	// ProviderOpenAI represents OpenAI's GPT-4 Vision API
	ProviderOpenAI ProviderType = "openai"

	// ProviderAnthropic represents Anthropic's Claude API with vision
	ProviderAnthropic ProviderType = "anthropic"

	// ProviderGoogle represents Google's Gemini API
	ProviderGoogle ProviderType = "google"
)

// VisionClientConfig holds common configuration for all vision clients
type VisionClientConfig struct {
	// Provider is the LLM provider type (ollama, openai, anthropic, google)
	Provider ProviderType

	// Model is the specific model to use (e.g., "llava", "gpt-4-vision-preview", "claude-3-5-sonnet-20241022", "gemini-1.5-pro")
	Model string

	// Endpoint is the API endpoint (required for Ollama, optional for cloud providers)
	Endpoint string

	// APIKey is the API key for cloud providers (read from env vars)
	APIKey string

	// MaxRetries is the maximum number of retry attempts
	MaxRetries int

	// Temperature controls randomness (0.0 = deterministic, recommended for OCR)
	Temperature float64
}
