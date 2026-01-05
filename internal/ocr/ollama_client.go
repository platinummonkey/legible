package ocr

import (
	"context"

	"github.com/platinummonkey/legible/internal/logger"
	"github.com/platinummonkey/legible/internal/ollama"
)

// OllamaVisionClient is an adapter that implements VisionClient for Ollama
type OllamaVisionClient struct {
	client *ollama.Client
	logger *logger.Logger
}

// NewOllamaVisionClient creates a new Ollama vision client
func NewOllamaVisionClient(endpoint string, maxRetries int, log *logger.Logger) *OllamaVisionClient {
	if log == nil {
		log = logger.Get()
	}

	clientOpts := []ollama.ClientOption{
		ollama.WithLogger(log),
	}

	if endpoint != "" {
		clientOpts = append(clientOpts, ollama.WithEndpoint(endpoint))
	}

	if maxRetries > 0 {
		clientOpts = append(clientOpts, ollama.WithMaxRetries(maxRetries))
	}

	return &OllamaVisionClient{
		client: ollama.NewClient(clientOpts...),
		logger: log,
	}
}

// GenerateOCR performs OCR on a base64-encoded image and returns structured word data
func (o *OllamaVisionClient) GenerateOCR(ctx context.Context, model string, imageData string) ([]ollama.OCRWord, error) {
	return o.client.GenerateOCR(ctx, model, imageData)
}

// HealthCheck verifies that Ollama is accessible and the model is available
func (o *OllamaVisionClient) HealthCheck(ctx context.Context, model string) error {
	// Check if Ollama is running
	if err := o.client.HealthCheck(ctx); err != nil {
		return err
	}

	// Check if model is available, pull if needed
	models, err := o.client.ListModels(ctx)
	if err != nil {
		return err
	}

	modelFound := false
	for _, m := range models.Models {
		if m.Name == model || m.Name == model+":latest" {
			modelFound = true
			break
		}
	}

	if !modelFound {
		o.logger.WithFields("model", model).Info("Model not found, pulling...")
		if err := o.client.PullModel(ctx, model); err != nil {
			return err
		}
	}

	return nil
}

// Name returns the provider name
func (o *OllamaVisionClient) Name() string {
	return "ollama"
}

// SupportedModels returns a list of commonly used Ollama vision models
func (o *OllamaVisionClient) SupportedModels() []string {
	return []string{
		"llava",
		"llava:7b",
		"llava:13b",
		"llava:34b",
		"bakllava",
		"llava-llama3",
		"llava-phi3",
		"moondream",
	}
}
