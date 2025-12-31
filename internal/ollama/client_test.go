package ollama

import (
	"context"
	"encoding/json"
	"image"
	"image/color"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/platinummonkey/remarkable-sync/internal/logger"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name     string
		opts     []ClientOption
		wantURL  string
		wantLogs bool
	}{
		{
			name:    "default client",
			opts:    nil,
			wantURL: DefaultEndpoint,
		},
		{
			name:    "custom endpoint",
			opts:    []ClientOption{WithEndpoint("http://custom:8080")},
			wantURL: "http://custom:8080",
		},
		{
			name: "with logger",
			opts: []ClientOption{WithLogger(func() *logger.Logger {
				l, _ := logger.New(&logger.Config{Level: "debug", Format: "console"})
				return l
			}())},
			wantURL: DefaultEndpoint,
		},
		{
			name:    "with timeout",
			opts:    []ClientOption{WithTimeout(10 * time.Second)},
			wantURL: DefaultEndpoint,
		},
		{
			name:    "with retries",
			opts:    []ClientOption{WithMaxRetries(5), WithRetryDelay(2 * time.Second)},
			wantURL: DefaultEndpoint,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.opts...)
			if client == nil {
				t.Fatal("expected client to be created")
			}
			if client.endpoint != tt.wantURL {
				t.Errorf("endpoint = %v, want %v", client.endpoint, tt.wantURL)
			}
		})
	}
}

func TestClient_Generate(t *testing.T) {
	tests := []struct {
		name       string
		request    *GenerateRequest
		mockStatus int
		mockBody   string
		wantErr    bool
		checkResp  func(*testing.T, *GenerateResponse)
	}{
		{
			name: "successful generation",
			request: &GenerateRequest{
				Model:  "llama2",
				Prompt: "Hello",
				Stream: false,
			},
			mockStatus: http.StatusOK,
			mockBody: `{
				"model": "llama2",
				"response": "Hi there!",
				"done": true,
				"created_at": "2025-12-31T12:00:00Z"
			}`,
			wantErr: false,
			checkResp: func(t *testing.T, resp *GenerateResponse) {
				if resp.Model != "llama2" {
					t.Errorf("model = %v, want llama2", resp.Model)
				}
				if resp.Response != "Hi there!" {
					t.Errorf("response = %v, want 'Hi there!'", resp.Response)
				}
				if !resp.Done {
					t.Error("expected done to be true")
				}
			},
		},
		{
			name: "server error",
			request: &GenerateRequest{
				Model:  "llama2",
				Prompt: "Hello",
			},
			mockStatus: http.StatusInternalServerError,
			mockBody:   `{"error": "internal server error"}`,
			wantErr:    true,
		},
		{
			name: "model not found",
			request: &GenerateRequest{
				Model:  "nonexistent",
				Prompt: "Hello",
			},
			mockStatus: http.StatusNotFound,
			mockBody:   `{"error": "model not found"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/generate" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				if r.Method != http.MethodPost {
					t.Errorf("unexpected method: %s", r.Method)
				}

				w.WriteHeader(tt.mockStatus)
				w.Write([]byte(tt.mockBody))
			}))
			defer server.Close()

			client := NewClient(
				WithEndpoint(server.URL),
				WithMaxRetries(0), // disable retries for faster tests
			)

			ctx := context.Background()
			resp, err := client.Generate(ctx, tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkResp != nil {
				tt.checkResp(t, resp)
			}
		})
	}
}

func TestClient_GenerateWithVision(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req GenerateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if len(req.Images) == 0 {
			t.Error("expected images in request")
		}
		if req.Format != "json" {
			t.Errorf("format = %v, want json", req.Format)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"model": "llava",
			"response": "This is an image",
			"done": true,
			"created_at": "2025-12-31T12:00:00Z"
		}`))
	}))
	defer server.Close()

	client := NewClient(WithEndpoint(server.URL))
	ctx := context.Background()

	resp, err := client.GenerateWithVision(ctx, "llava", "Describe this image", []string{"base64data"})
	if err != nil {
		t.Fatalf("GenerateWithVision() error = %v", err)
	}
	if resp.Response != "This is an image" {
		t.Errorf("response = %v, want 'This is an image'", resp.Response)
	}
}

func TestClient_GenerateOCR(t *testing.T) {
	tests := []struct {
		name       string
		mockBody   string
		wantWords  int
		wantErr    bool
	}{
		{
			name: "successful OCR",
			mockBody: `{
				"model": "llava",
				"response": "[{\"text\":\"Hello\",\"bbox\":[10,20,50,30]},{\"text\":\"World\",\"bbox\":[70,20,50,30]}]",
				"done": true,
				"created_at": "2025-12-31T12:00:00Z"
			}`,
			wantWords: 2,
			wantErr:   false,
		},
		{
			name: "empty result",
			mockBody: `{
				"model": "llava",
				"response": "[]",
				"done": true,
				"created_at": "2025-12-31T12:00:00Z"
			}`,
			wantWords: 0,
			wantErr:   false,
		},
		{
			name: "invalid JSON response",
			mockBody: `{
				"model": "llava",
				"response": "not json",
				"done": true,
				"created_at": "2025-12-31T12:00:00Z"
			}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.mockBody))
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			ctx := context.Background()

			words, err := client.GenerateOCR(ctx, "llava", "base64data")
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateOCR() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(words) != tt.wantWords {
				t.Errorf("got %d words, want %d", len(words), tt.wantWords)
			}
		})
	}
}

func TestClient_ListModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"models": [
				{
					"name": "llama2",
					"modified_at": "2025-12-31T12:00:00Z",
					"size": 1234567890,
					"digest": "abc123"
				},
				{
					"name": "llava",
					"modified_at": "2025-12-31T12:00:00Z",
					"size": 9876543210,
					"digest": "def456"
				}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient(WithEndpoint(server.URL))
	ctx := context.Background()

	resp, err := client.ListModels(ctx)
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}

	if len(resp.Models) != 2 {
		t.Errorf("got %d models, want 2", len(resp.Models))
	}
	if resp.Models[0].Name != "llama2" {
		t.Errorf("first model name = %v, want llama2", resp.Models[0].Name)
	}
}

func TestClient_PullModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/pull" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}

		var req PullRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if req.Name != "llama2" {
			t.Errorf("model name = %v, want llama2", req.Name)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
	}))
	defer server.Close()

	client := NewClient(WithEndpoint(server.URL))
	ctx := context.Background()

	err := client.PullModel(ctx, "llama2")
	if err != nil {
		t.Fatalf("PullModel() error = %v", err)
	}
}

func TestClient_HealthCheck(t *testing.T) {
	tests := []struct {
		name       string
		mockStatus int
		wantErr    bool
	}{
		{
			name:       "healthy",
			mockStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "unhealthy",
			mockStatus: http.StatusServiceUnavailable,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.mockStatus)
			}))
			defer server.Close()

			client := NewClient(WithEndpoint(server.URL))
			ctx := context.Background()

			err := client.HealthCheck(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("HealthCheck() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_Retry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"model": "llama2",
			"response": "Success after retries",
			"done": true,
			"created_at": "2025-12-31T12:00:00Z"
		}`))
	}))
	defer server.Close()

	client := NewClient(
		WithEndpoint(server.URL),
		WithMaxRetries(3),
		WithRetryDelay(10*time.Millisecond),
	)
	ctx := context.Background()

	req := &GenerateRequest{
		Model:  "llama2",
		Prompt: "test",
	}
	resp, err := client.Generate(ctx, req)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if resp.Response != "Success after retries" {
		t.Errorf("response = %v, want 'Success after retries'", resp.Response)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(WithEndpoint(server.URL))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	req := &GenerateRequest{
		Model:  "llama2",
		Prompt: "test",
	}
	_, err := client.Generate(ctx, req)
	if err == nil {
		t.Error("expected context cancellation error")
	}
}

func TestEncodeImageToBase64(t *testing.T) {
	// Create a simple test image
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}

	tests := []struct {
		name    string
		format  string
		wantErr bool
	}{
		{
			name:    "PNG format",
			format:  "png",
			wantErr: false,
		},
		{
			name:    "JPEG format",
			format:  "jpeg",
			wantErr: false,
		},
		{
			name:    "JPG format",
			format:  "jpg",
			wantErr: false,
		},
		{
			name:    "unsupported format",
			format:  "bmp",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EncodeImageToBase64(img, tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("EncodeImageToBase64() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == "" {
				t.Error("expected non-empty base64 string")
			}
		})
	}
}

func TestEncodeBytesToBase64(t *testing.T) {
	tests := []struct {
		name  string
		data  []byte
		want  string
	}{
		{
			name: "simple data",
			data: []byte("hello"),
			want: "aGVsbG8=",
		},
		{
			name: "empty data",
			data: []byte{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncodeBytesToBase64(tt.data)
			if got != tt.want {
				t.Errorf("EncodeBytesToBase64() = %v, want %v", got, tt.want)
			}
		})
	}
}
