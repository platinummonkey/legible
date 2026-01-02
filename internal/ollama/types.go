package ollama

import "time"

// GenerateRequest represents a request to the Ollama generate API
type GenerateRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	Images  []string               `json:"images,omitempty"` // base64 encoded
	Stream  bool                   `json:"stream"`
	Format  string                 `json:"format,omitempty"` // "json" for structured output
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
	BBox       []int   `json:"bbox"` // [x, y, width, height]
	Confidence float64 `json:"confidence,omitempty"`
}

// PromptConfig represents the YAML configuration for OCR prompts
type PromptConfig struct {
	Model  string `yaml:"model"`
	System string `yaml:"system"`
	Prompt string `yaml:"prompt"`
}

// StructuredOCRResponse represents the structured response from advanced OCR prompt
type StructuredOCRResponse struct {
	Lines []OCRLine `json:"lines"`
}

// OCRLine represents a line of text, table, or diagram from the OCR
type OCRLine struct {
	BBox          []int          `json:"bbox"` // [x1, y1, x2, y2]
	Type          string         `json:"type"` // "text", "table", "diagram"
	Content       string         `json:"content,omitempty"`
	Headers       []string       `json:"headers,omitempty"`        // for tables
	Rows          [][]string     `json:"rows,omitempty"`           // for tables
	DiagramBlocks []DiagramBlock `json:"diagram_blocks,omitempty"` // for diagrams
}

// DiagramBlock represents a block within a diagram
type DiagramBlock struct {
	Type string `json:"type"` // "block", "arrow", etc.
	Text string `json:"text"`
	BBox []int  `json:"bbox"` // [x1, y1, x2, y2]
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
