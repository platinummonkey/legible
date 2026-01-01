package ollama

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"time"

	"github.com/platinummonkey/remarkable-sync/internal/logger"
)

const (
	// DefaultEndpoint is the default Ollama API endpoint
	DefaultEndpoint = "http://localhost:11434"

	// DefaultTimeout is the default HTTP client timeout
	DefaultTimeout = 5 * time.Minute

	// DefaultMaxRetries is the default number of retries
	DefaultMaxRetries = 3

	// DefaultRetryDelay is the initial delay between retries
	DefaultRetryDelay = 1 * time.Second
)

// OCRPrompt is the prompt template for OCR with bounding boxes
const OCRPrompt = `Extract all handwritten text from this image of a reMarkable tablet note.
Return ONLY a JSON array with no additional text or explanation.
Each object must have:
- "text": the extracted text
- "bbox": bounding box as [x, y, width, height] in pixels from top-left origin

Example format:
[
  {"text": "Hello", "bbox": [120, 45, 85, 32]},
  {"text": "World", "bbox": [210, 45, 78, 32]}
]

If no text is found, return an empty array: []`

// Client is an HTTP client for the Ollama API
type Client struct {
	endpoint   string
	httpClient *http.Client
	logger     *logger.Logger
	maxRetries int
	retryDelay time.Duration
}

// ClientOption is a function that configures a Client
type ClientOption func(*Client)

// WithEndpoint sets the Ollama API endpoint
func WithEndpoint(endpoint string) ClientOption {
	return func(c *Client) {
		c.endpoint = endpoint
	}
}

// WithTimeout sets the HTTP client timeout
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithLogger sets the logger
func WithLogger(log *logger.Logger) ClientOption {
	return func(c *Client) {
		c.logger = log
	}
}

// WithMaxRetries sets the maximum number of retries
func WithMaxRetries(maxRetries int) ClientOption {
	return func(c *Client) {
		c.maxRetries = maxRetries
	}
}

// WithRetryDelay sets the initial retry delay
func WithRetryDelay(delay time.Duration) ClientOption {
	return func(c *Client) {
		c.retryDelay = delay
	}
}

// NewClient creates a new Ollama client
func NewClient(opts ...ClientOption) *Client {
	// Create default logger
	defaultLogger, err := logger.New(&logger.Config{
		Level:  "info",
		Format: "console",
	})
	if err != nil {
		// Fallback to global logger if default creation fails
		defaultLogger = logger.Get()
	}

	client := &Client{
		endpoint: DefaultEndpoint,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		logger:     defaultLogger,
		maxRetries: DefaultMaxRetries,
		retryDelay: DefaultRetryDelay,
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// doRequest performs an HTTP request with retry logic
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}, response interface{}) error {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			delay := c.retryDelay * time.Duration(1<<uint(attempt-1)) // exponential backoff
			c.logger.Debugf("Retrying request (attempt %d/%d) after %v", attempt, c.maxRetries, delay)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		var reqBody io.Reader
		if body != nil {
			jsonData, err := json.Marshal(body)
			if err != nil {
				return fmt.Errorf("failed to marshal request body: %w", err)
			}
			reqBody = bytes.NewReader(jsonData)
		}

		req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, reqBody)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to execute request: %w", err)
			c.logger.Debugf("Request failed: %v", lastErr)
			continue
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			c.logger.Debugf("Failed to read response: %v", lastErr)
			continue
		}

		// Check for HTTP errors
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			var errResp ErrorResponse
			var errMsg string
			if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.Error != "" {
				errMsg = fmt.Sprintf("ollama API error (status %d): %s", resp.StatusCode, errResp.Error)
			} else {
				errMsg = fmt.Sprintf("ollama API error (status %d): %s", resp.StatusCode, string(respBody))
			}

			// For 5xx server errors, retry. For 4xx client errors, return immediately
			if resp.StatusCode >= 500 {
				lastErr = errors.New(errMsg)
				c.logger.Debugf("Server error: %v", lastErr)
				continue
			}
			return errors.New(errMsg)
		}

		// Parse response
		if response != nil {
			if err := json.Unmarshal(respBody, response); err != nil {
				return fmt.Errorf("failed to unmarshal response: %w", err)
			}
		}

		return nil
	}

	return fmt.Errorf("request failed after %d attempts: %w", c.maxRetries+1, lastErr)
}

// Generate sends a text generation request to Ollama
func (c *Client) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	var resp GenerateResponse
	if err := c.doRequest(ctx, http.MethodPost, "/api/generate", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GenerateWithVision sends a vision model inference request with image input
func (c *Client) GenerateWithVision(ctx context.Context, model string, prompt string, images []string) (*GenerateResponse, error) {
	req := &GenerateRequest{
		Model:  model,
		Prompt: prompt,
		Images: images,
		Stream: false,
		Format: "json",
	}
	return c.Generate(ctx, req)
}

// GenerateOCR performs OCR on an image using a vision model
func (c *Client) GenerateOCR(ctx context.Context, model string, imageData string) ([]OCRWord, error) {
	resp, err := c.GenerateWithVision(ctx, model, OCRPrompt, []string{imageData})
	if err != nil {
		return nil, fmt.Errorf("failed to generate OCR: %w", err)
	}

	// Try parsing as array first (expected format)
	var words []OCRWord
	if err := json.Unmarshal([]byte(resp.Response), &words); err == nil {
		return words, nil
	}

	// If that fails, try parsing as object with "words" field
	// Some vision models wrap the array in an object despite the prompt
	var wrappedResponse struct {
		Words []OCRWord `json:"words"`
	}
	if err := json.Unmarshal([]byte(resp.Response), &wrappedResponse); err != nil {
		// Log the actual response for debugging
		c.logger.WithFields("response", resp.Response).Debug("Failed to parse OCR response in any format")
		return nil, fmt.Errorf("failed to parse OCR response as array or object: %w", err)
	}

	return wrappedResponse.Words, nil
}

// ListModels lists available models
func (c *Client) ListModels(ctx context.Context) (*ListModelsResponse, error) {
	var resp ListModelsResponse
	if err := c.doRequest(ctx, http.MethodGet, "/api/tags", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// PullModel downloads a model if it's not already available
func (c *Client) PullModel(ctx context.Context, modelName string) error {
	req := &PullRequest{
		Name:   modelName,
		Stream: false,
	}
	var resp PullResponse
	if err := c.doRequest(ctx, http.MethodPost, "/api/pull", req, &resp); err != nil {
		return err
	}
	c.logger.Infof("Model pull status: %s", resp.Status)
	return nil
}

// HealthCheck verifies that Ollama is running and accessible
func (c *Client) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ollama is not accessible: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama health check failed with status: %d", resp.StatusCode)
	}

	return nil
}

// EncodeImageToBase64 encodes an image to base64 string
func EncodeImageToBase64(img image.Image, format string) (string, error) {
	var buf bytes.Buffer

	switch format {
	case "png", "PNG":
		if err := png.Encode(&buf, img); err != nil {
			return "", fmt.Errorf("failed to encode PNG: %w", err)
		}
	case "jpeg", "jpg", "JPEG", "JPG":
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
			return "", fmt.Errorf("failed to encode JPEG: %w", err)
		}
	default:
		return "", fmt.Errorf("unsupported image format: %s", format)
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// EncodeBytesToBase64 encodes raw bytes to base64 string
func EncodeBytesToBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
