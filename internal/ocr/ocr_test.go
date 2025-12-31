package ocr

import (
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/platinummonkey/remarkable-sync/internal/logger"
)

func TestNew(t *testing.T) {
	cfg := &Config{}
	processor := New(cfg)

	if processor == nil {
		t.Fatal("New() returned nil")
	}

	if processor.logger == nil {
		t.Error("logger should be initialized")
	}

	if processor.ollamaClient == nil {
		t.Error("ollama client should be initialized")
	}

	if processor.model == "" {
		t.Error("model should have default value")
	}

	if processor.model != DefaultModel {
		t.Errorf("default model should be '%s', got '%s'", DefaultModel, processor.model)
	}
}

func TestNew_CustomConfig(t *testing.T) {
	log, _ := logger.New(&logger.Config{Level: "debug", Format: "console"})
	cfg := &Config{
		Logger:      log,
		Model:       "llava",
		Temperature: 0.1,
		MaxRetries:  5,
	}
	processor := New(cfg)

	if processor.model != "llava" {
		t.Errorf("expected model 'llava', got '%s'", processor.model)
	}

	if processor.logger == nil {
		t.Error("logger should be initialized")
	}
}

func TestNew_CustomEndpoint(t *testing.T) {
	cfg := &Config{
		OllamaEndpoint: "http://custom:8080",
		Model:          "mistral",
	}
	processor := New(cfg)

	if processor.model != "mistral" {
		t.Errorf("expected model 'mistral', got '%s'", processor.model)
	}
}

func TestProcessImage_Success(t *testing.T) {
	// Create mock Ollama server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}

		// Mock OCR response
		response := map[string]interface{}{
			"model":    "llava",
			"response": `[{"text":"Hello","bbox":[50,50,100,30],"confidence":0.95},{"text":"World","bbox":[160,50,100,30],"confidence":0.92}]`,
			"done":     true,
			"created_at": time.Now().Format(time.RFC3339),
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	processor := New(&Config{
		OllamaEndpoint: server.URL,
		Model:          "llava",
	})

	// Create a simple test image
	imgData := createTestImage(t, 200, 100)

	pageOCR, err := processor.ProcessImage(imgData, 1)
	if err != nil {
		t.Fatalf("ProcessImage() error = %v", err)
	}

	if pageOCR == nil {
		t.Fatal("ProcessImage() returned nil result")
	}

	if pageOCR.PageNumber != 1 {
		t.Errorf("PageNumber = %d, want 1", pageOCR.PageNumber)
	}

	if len(pageOCR.Words) != 2 {
		t.Errorf("len(Words) = %d, want 2", len(pageOCR.Words))
	}

	if len(pageOCR.Words) >= 1 {
		word := pageOCR.Words[0]
		if word.Text != "Hello" {
			t.Errorf("First word = %s, want Hello", word.Text)
		}
		if word.Confidence != 95.0 {
			t.Errorf("First word confidence = %f, want 95.0", word.Confidence)
		}
		if word.BoundingBox.X != 50 || word.BoundingBox.Y != 50 {
			t.Errorf("First word position = (%d, %d), want (50, 50)", word.BoundingBox.X, word.BoundingBox.Y)
		}
		if word.BoundingBox.Width != 100 || word.BoundingBox.Height != 30 {
			t.Errorf("First word size = (%d, %d), want (100, 30)", word.BoundingBox.Width, word.BoundingBox.Height)
		}
	}

	// Check that text was built
	if pageOCR.Text == "" {
		t.Error("Text should be built from words")
	}

	if !strings.Contains(pageOCR.Text, "Hello") || !strings.Contains(pageOCR.Text, "World") {
		t.Errorf("Text should contain 'Hello World', got: %s", pageOCR.Text)
	}

	// Check confidence
	if pageOCR.Confidence == 0 {
		t.Error("Confidence should be calculated")
	}
}

func TestProcessImage_EmptyResult(t *testing.T) {
	// Create mock Ollama server that returns empty results
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"model":      "llava",
			"response":   `[]`,
			"done":       true,
			"created_at": time.Now().Format(time.RFC3339),
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	processor := New(&Config{
		OllamaEndpoint: server.URL,
		Model:          "llava",
	})

	imgData := createTestImage(t, 200, 100)

	pageOCR, err := processor.ProcessImage(imgData, 1)
	if err != nil {
		t.Fatalf("ProcessImage() error = %v", err)
	}

	if len(pageOCR.Words) != 0 {
		t.Errorf("expected 0 words, got %d", len(pageOCR.Words))
	}

	if pageOCR.Text != "" {
		t.Errorf("expected empty text, got: %s", pageOCR.Text)
	}
}

func TestProcessImage_OllamaError(t *testing.T) {
	// Create mock Ollama server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "model not found",
		})
	}))
	defer server.Close()

	processor := New(&Config{
		OllamaEndpoint: server.URL,
		Model:          "llava",
		MaxRetries:     0, // disable retries for faster test
	})

	imgData := createTestImage(t, 200, 100)

	_, err := processor.ProcessImage(imgData, 1)
	if err == nil {
		t.Error("ProcessImage() should error when Ollama returns error")
	}

	if !strings.Contains(err.Error(), "failed to generate OCR") {
		t.Errorf("error message should mention OCR failure, got: %v", err)
	}
}

func TestProcessImage_InvalidBBox(t *testing.T) {
	// Create mock Ollama server that returns invalid bounding box
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"model":      "llava",
			"response":   `[{"text":"Invalid","bbox":[50,50],"confidence":0.95}]`, // only 2 coords
			"done":       true,
			"created_at": time.Now().Format(time.RFC3339),
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	processor := New(&Config{
		OllamaEndpoint: server.URL,
		Model:          "llava",
	})

	imgData := createTestImage(t, 200, 100)

	pageOCR, err := processor.ProcessImage(imgData, 1)
	if err != nil {
		t.Fatalf("ProcessImage() error = %v", err)
	}

	// Should skip words with invalid bounding boxes
	if len(pageOCR.Words) != 0 {
		t.Errorf("expected 0 words (invalid bbox should be skipped), got %d", len(pageOCR.Words))
	}
}

func TestProcessImage_DefaultConfidence(t *testing.T) {
	// Create mock Ollama server that returns words without confidence
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"model":      "llava",
			"response":   `[{"text":"Test","bbox":[50,50,100,30]}]`, // no confidence
			"done":       true,
			"created_at": time.Now().Format(time.RFC3339),
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	processor := New(&Config{
		OllamaEndpoint: server.URL,
		Model:          "llava",
	})

	imgData := createTestImage(t, 200, 100)

	pageOCR, err := processor.ProcessImage(imgData, 1)
	if err != nil {
		t.Fatalf("ProcessImage() error = %v", err)
	}

	if len(pageOCR.Words) != 1 {
		t.Fatalf("expected 1 word, got %d", len(pageOCR.Words))
	}

	// Should use default confidence of 80.0
	if pageOCR.Words[0].Confidence != 80.0 {
		t.Errorf("expected default confidence 80.0, got %f", pageOCR.Words[0].Confidence)
	}
}

func TestProcessImageWithCustomPrompt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"model":      "llava",
			"response":   `[{"text":"Custom","bbox":[10,10,50,20],"confidence":0.9}]`,
			"done":       true,
			"created_at": time.Now().Format(time.RFC3339),
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	processor := New(&Config{
		OllamaEndpoint: server.URL,
		Model:          "llava",
	})

	imgData := createTestImage(t, 200, 100)
	customPrompt := "Extract text from this custom image."

	pageOCR, err := processor.ProcessImageWithCustomPrompt(imgData, 1, customPrompt)
	if err != nil {
		t.Fatalf("ProcessImageWithCustomPrompt() error = %v", err)
	}

	if len(pageOCR.Words) != 1 {
		t.Fatalf("expected 1 word, got %d", len(pageOCR.Words))
	}

	if pageOCR.Words[0].Text != "Custom" {
		t.Errorf("expected word 'Custom', got '%s'", pageOCR.Words[0].Text)
	}

	// Verify prompt template was restored
	if processor.promptTemplate != ocrPromptTemplate {
		t.Error("prompt template should be restored after custom prompt use")
	}
}

func TestHealthCheck_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "" || r.URL.Path == "/" {
			// Health check endpoint
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path == "/api/tags" {
			// List models endpoint
			response := map[string]interface{}{
				"models": []map[string]interface{}{
					{
						"name":        "llava:latest",
						"modified_at": time.Now().Format(time.RFC3339),
						"size":        1234567890,
						"digest":      "abc123",
					},
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return
		}
	}))
	defer server.Close()

	processor := New(&Config{
		OllamaEndpoint: server.URL,
		Model:          "llava",
	})

	err := processor.HealthCheck()
	if err != nil {
		t.Errorf("HealthCheck() error = %v", err)
	}
}

func TestHealthCheck_OllamaDown(t *testing.T) {
	// Use invalid endpoint
	processor := New(&Config{
		OllamaEndpoint: "http://localhost:99999",
		Model:          "llava",
		MaxRetries:     0,
	})

	err := processor.HealthCheck()
	if err == nil {
		t.Error("HealthCheck() should error when Ollama is down")
	}
}

func TestHealthCheck_ModelNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "" || r.URL.Path == "/" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path == "/api/tags" {
			// Return empty model list
			response := map[string]interface{}{
				"models": []map[string]interface{}{},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return
		}
		if r.URL.Path == "/api/pull" {
			// Simulate pull failure
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "failed to pull model",
			})
			return
		}
	}))
	defer server.Close()

	processor := New(&Config{
		OllamaEndpoint: server.URL,
		Model:          "nonexistent",
		MaxRetries:     0,
	})

	err := processor.HealthCheck()
	if err == nil {
		t.Error("HealthCheck() should error when model cannot be pulled")
	}

	if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "pull failed") {
		t.Errorf("error should mention model not found or pull failure, got: %v", err)
	}
}

func TestModel(t *testing.T) {
	processor := New(&Config{
		Model: "test-model",
	})

	if processor.Model() != "test-model" {
		t.Errorf("Model() = %s, want test-model", processor.Model())
	}
}

func TestParseOllamaResponse(t *testing.T) {
	tests := []struct {
		name        string
		jsonResp    string
		wantWords   int
		wantErr     bool
		checkFirst  bool
		firstText   string
		firstBBox   []int
		firstConf   float64
	}{
		{
			name:       "valid response",
			jsonResp:   `{"words":[{"text":"Hello","bbox":[10,20,50,30],"confidence":0.95}]}`,
			wantWords:  1,
			wantErr:    false,
			checkFirst: true,
			firstText:  "Hello",
			firstBBox:  []int{10, 20, 50, 30},
			firstConf:  0.95,
		},
		{
			name:      "empty response",
			jsonResp:  `{"words":[]}`,
			wantWords: 0,
			wantErr:   false,
		},
		{
			name:     "invalid JSON",
			jsonResp: `not json`,
			wantErr:  true,
		},
		{
			name:       "multiple words",
			jsonResp:   `{"words":[{"text":"One","bbox":[1,2,3,4],"confidence":0.9},{"text":"Two","bbox":[5,6,7,8],"confidence":0.8}]}`,
			wantWords:  2,
			wantErr:    false,
			checkFirst: true,
			firstText:  "One",
			firstBBox:  []int{1, 2, 3, 4},
			firstConf:  0.9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			words, err := parseOllamaResponse(tt.jsonResp)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseOllamaResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if len(words) != tt.wantWords {
				t.Errorf("got %d words, want %d", len(words), tt.wantWords)
				return
			}

			if tt.checkFirst && len(words) > 0 {
				word := words[0]
				if word.Text != tt.firstText {
					t.Errorf("first word text = %s, want %s", word.Text, tt.firstText)
				}
				if len(word.BBox) != len(tt.firstBBox) {
					t.Errorf("first word bbox length = %d, want %d", len(word.BBox), len(tt.firstBBox))
				} else {
					for i, v := range tt.firstBBox {
						if word.BBox[i] != v {
							t.Errorf("first word bbox[%d] = %d, want %d", i, word.BBox[i], v)
						}
					}
				}
				if word.Confidence != tt.firstConf {
					t.Errorf("first word confidence = %f, want %f", word.Confidence, tt.firstConf)
				}
			}
		})
	}
}

// Helper functions

func createTestImage(t *testing.T, width, height int) []byte {
	t.Helper()

	// Create a white image
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	white := color.RGBA{255, 255, 255, 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, white)
		}
	}

	// Encode to PNG
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.png")

	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		t.Fatalf("failed to encode PNG: %v", err)
	}

	// Read back the image data
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read temp file: %v", err)
	}

	return data
}

// Benchmark for ProcessImage
func BenchmarkProcessImage(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"model":      "llava",
			"response":   `[{"text":"Benchmark","bbox":[50,50,100,30],"confidence":0.95}]`,
			"done":       true,
			"created_at": time.Now().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	processor := New(&Config{
		OllamaEndpoint: server.URL,
		Model:          "llava",
	})

	imgData := createBenchmarkImage(200, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := processor.ProcessImage(imgData, 1)
		if err != nil {
			b.Fatalf("ProcessImage() error = %v", err)
		}
	}
}

func createBenchmarkImage(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	white := color.RGBA{255, 255, 255, 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, white)
		}
	}

	// Use in-memory buffer for benchmark
	var buf []byte
	_ = png.Encode(&writerAdapter{buf: &buf}, img)
	return buf
}

type writerAdapter struct {
	buf *[]byte
}

func (w *writerAdapter) Write(p []byte) (n int, err error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}
