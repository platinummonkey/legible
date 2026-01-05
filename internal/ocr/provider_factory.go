package ocr

import (
	"context"
	"fmt"

	"github.com/platinummonkey/legible/internal/logger"
)

// NewVisionClient creates a vision client based on the provider configuration
func NewVisionClient(ctx context.Context, cfg *VisionClientConfig, log *logger.Logger) (VisionClient, error) {
	if log == nil {
		log = logger.Get()
	}

	switch cfg.Provider {
	case ProviderOllama:
		return NewOllamaVisionClient(cfg.Endpoint, cfg.MaxRetries, log), nil

	case ProviderOpenAI:
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("OpenAI API key is required (set OPENAI_API_KEY environment variable)")
		}
		return NewOpenAIVisionClient(cfg.APIKey, cfg.Temperature, cfg.MaxRetries, log), nil

	case ProviderAnthropic:
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("anthropic API key is required (set ANTHROPIC_API_KEY environment variable)")
		}
		return NewAnthropicVisionClient(cfg.APIKey, cfg.Temperature, cfg.MaxRetries, log), nil

	case ProviderGoogle:
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("google API key is required (set GOOGLE_API_KEY or GOOGLE_APPLICATION_CREDENTIALS environment variable)")
		}
		client, err := NewGoogleVisionClient(ctx, cfg.APIKey, cfg.Temperature, cfg.MaxRetries, log)
		if err != nil {
			return nil, fmt.Errorf("failed to create Google vision client: %w", err)
		}
		return client, nil

	default:
		return nil, fmt.Errorf("unsupported provider: %s (supported: ollama, openai, anthropic, google)", cfg.Provider)
	}
}

// ValidateProviderConfig validates that the provider configuration is complete and correct
func ValidateProviderConfig(cfg *VisionClientConfig) error {
	if cfg == nil {
		return fmt.Errorf("vision client config is nil")
	}

	// Validate provider
	validProviders := map[ProviderType]bool{
		ProviderOllama:    true,
		ProviderOpenAI:    true,
		ProviderAnthropic: true,
		ProviderGoogle:    true,
	}

	if !validProviders[cfg.Provider] {
		return fmt.Errorf("invalid provider: %s", cfg.Provider)
	}

	// Validate model
	if cfg.Model == "" {
		return fmt.Errorf("model is required")
	}

	// Provider-specific validation
	switch cfg.Provider {
	case ProviderOllama:
		if cfg.Endpoint == "" {
			return fmt.Errorf("endpoint is required for Ollama provider")
		}

	case ProviderOpenAI, ProviderAnthropic, ProviderGoogle:
		if cfg.APIKey == "" {
			return fmt.Errorf("API key is required for %s provider", cfg.Provider)
		}
	}

	// Validate temperature
	if cfg.Temperature < 0.0 || cfg.Temperature > 2.0 {
		return fmt.Errorf("temperature must be between 0.0 and 2.0, got %f", cfg.Temperature)
	}

	// Validate max retries
	if cfg.MaxRetries < 0 {
		return fmt.Errorf("max retries must be non-negative, got %d", cfg.MaxRetries)
	}

	return nil
}

// GetDefaultModelForProvider returns a recommended default model for the given provider
func GetDefaultModelForProvider(provider ProviderType) string {
	switch provider {
	case ProviderOllama:
		return "llava"
	case ProviderOpenAI:
		return "gpt-4o"
	case ProviderAnthropic:
		return "claude-3-5-sonnet-20241022"
	case ProviderGoogle:
		return "gemini-1.5-pro"
	default:
		return ""
	}
}
