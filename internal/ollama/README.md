# Ollama HTTP Client

This package provides an HTTP client for communicating with the Ollama local API, with a focus on vision model inference for OCR tasks.

## Overview

The `ollama` package implements a robust HTTP client for the Ollama API with:

- Configurable endpoint and timeout
- Automatic retry with exponential backoff
- Connection pooling
- Comprehensive error handling
- Image encoding utilities for vision models
- Dedicated OCR functionality

## Quick Start

```go
import (
    "context"
    "github.com/platinummonkey/legible/internal/ollama"
)

// Create a client with default settings
client := ollama.NewClient()

// Or with custom options
client := ollama.NewClient(
    ollama.WithEndpoint("http://localhost:11434"),
    ollama.WithTimeout(5 * time.Minute),
    ollama.WithLogger(myLogger),
)

// Check if Ollama is running
ctx := context.Background()
if err := client.HealthCheck(ctx); err != nil {
    log.Fatal(err)
}
```

## Core Features

### Text Generation

```go
req := &ollama.GenerateRequest{
    Model:  "llama2",
    Prompt: "Explain quantum computing",
    Stream: false,
}

resp, err := client.Generate(ctx, req)
if err != nil {
    log.Fatal(err)
}
fmt.Println(resp.Response)
```

### Vision Model Inference

```go
// Encode an image to base64
imageData, err := ollama.EncodeImageToBase64(img, "png")
if err != nil {
    log.Fatal(err)
}

// Use vision model
resp, err := client.GenerateWithVision(
    ctx,
    "llava",
    "Describe this image",
    []string{imageData},
)
```

### OCR with Bounding Boxes

```go
// Perform OCR on an image
words, err := client.GenerateOCR(ctx, "llava", imageData)
if err != nil {
    log.Fatal(err)
}

for _, word := range words {
    fmt.Printf("Text: %s, BBox: %v\n", word.Text, word.BBox)
}
```

### Model Management

```go
// List available models
models, err := client.ListModels(ctx)
if err != nil {
    log.Fatal(err)
}

for _, model := range models.Models {
    fmt.Printf("Model: %s, Size: %d bytes\n", model.Name, model.Size)
}

// Pull a model if not available
if err := client.PullModel(ctx, "llava"); err != nil {
    log.Fatal(err)
}
```

## Configuration Options

### WithEndpoint

Set a custom Ollama API endpoint:

```go
client := ollama.NewClient(
    ollama.WithEndpoint("http://192.168.1.100:11434"),
)
```

### WithTimeout

Configure the HTTP client timeout:

```go
client := ollama.NewClient(
    ollama.WithTimeout(10 * time.Minute),
)
```

### WithLogger

Provide a custom logger:

```go
client := ollama.NewClient(
    ollama.WithLogger(myLogger),
)
```

### WithMaxRetries and WithRetryDelay

Configure retry behavior:

```go
client := ollama.NewClient(
    ollama.WithMaxRetries(5),
    ollama.WithRetryDelay(2 * time.Second),
)
```

## OCR Prompt

The package includes a carefully designed prompt for OCR with bounding boxes. The prompt instructs the model to:

1. Extract all handwritten text from reMarkable tablet notes
2. Return results as a JSON array
3. Include text and bounding box coordinates for each word
4. Use pixel coordinates from top-left origin
5. Return an empty array if no text is found

Example output format:

```json
[
  {"text": "Hello", "bbox": [120, 45, 85, 32]},
  {"text": "World", "bbox": [210, 45, 78, 32]}
]
```

## Error Handling

The client handles several error conditions:

- **Connection refused**: Ollama is not running
- **Model not found**: The specified model is not available
- **Timeout errors**: Request took too long to complete
- **HTTP errors**: Non-2xx status codes from the API
- **JSON parsing errors**: Invalid response format

All errors are wrapped with context for easier debugging.

## Image Encoding

The package provides two utility functions for encoding images:

### EncodeImageToBase64

Encodes an `image.Image` to base64:

```go
base64Str, err := ollama.EncodeImageToBase64(img, "png")
```

Supported formats: `png`, `PNG`, `jpeg`, `jpg`, `JPEG`, `JPG`

### EncodeBytesToBase64

Encodes raw bytes to base64:

```go
base64Str := ollama.EncodeBytesToBase64(imageBytes)
```

## Connection Pooling

The client uses HTTP connection pooling with:

- 10 max idle connections
- 10 max idle connections per host
- 90 second idle connection timeout

This ensures efficient resource usage when making multiple requests.

## Retry Logic

Failed requests are automatically retried with exponential backoff:

1. First retry: wait 1 second (configurable)
2. Second retry: wait 2 seconds
3. Third retry: wait 4 seconds
4. And so on...

The maximum number of retries is configurable (default: 3).

## Testing

The package includes comprehensive unit tests covering:

- Client creation with various options
- All API methods (Generate, ListModels, PullModel, etc.)
- Error handling (server errors, timeouts, invalid responses)
- Retry logic with exponential backoff
- Context cancellation
- Image encoding utilities

Run tests with:

```bash
go test ./internal/ollama/...
```

Run tests with coverage:

```bash
go test -cover ./internal/ollama/...
```

## Dependencies

The package depends on:

- `github.com/platinummonkey/legible/internal/logger` - Logging utilities
- Standard library packages: `net/http`, `encoding/json`, `image`, etc.

## Thread Safety

The HTTP client is safe for concurrent use. Multiple goroutines can share a single `Client` instance.
