package ollama

import "time"

// GenerateRequest represents a request to the Ollama generate API
type GenerateRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	Images  []string               `json:"images,omitempty"`  // base64 encoded
	Stream  bool                   `json:"stream"`
	Format  string                 `json:"format,omitempty"`  // "json" for structured output
	Options map[string]interface{} `json:"options,omitempty"`
}

// GenerateResponse represents a response from the Ollama generate API
type GenerateResponse struct {
	Model     string    `json:"model"`
	Response  string    `json:"response"`
	Done      bool      `json:"done"`
	Context   []int     `json:"context,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// OCRWord represents extracted text with bounding box information
type OCRWord struct {
	Text       string  `json:"text"`
	BBox       []int   `json:"bbox"`                // [x, y, width, height]
	Confidence float64 `json:"confidence,omitempty"`
}

// Model represents an Ollama model
type Model struct {
	Name       string    `json:"name"`
	ModifiedAt time.Time `json:"modified_at"`
	Size       int64     `json:"size"`
	Digest     string    `json:"digest"`
	Details    struct {
		Format            string   `json:"format"`
		Family            string   `json:"family"`
		Families          []string `json:"families"`
		ParameterSize     string   `json:"parameter_size"`
		QuantizationLevel string   `json:"quantization_level"`
	} `json:"details"`
}

// ListModelsResponse represents a response from the list models API
type ListModelsResponse struct {
	Models []Model `json:"models"`
}

// PullRequest represents a request to pull/download a model
type PullRequest struct {
	Name   string `json:"name"`
	Stream bool   `json:"stream"`
}

// PullResponse represents a response from the pull API
type PullResponse struct {
	Status    string `json:"status"`
	Digest    string `json:"digest,omitempty"`
	Total     int64  `json:"total,omitempty"`
	Completed int64  `json:"completed,omitempty"`
}

// ErrorResponse represents an error response from Ollama
type ErrorResponse struct {
	Error string `json:"error"`
}
