# Sync Orchestrator

Package sync coordinates the complete synchronization workflow for reMarkable documents.

## Overview

The Sync Orchestrator is the core component that ties together all the pieces of the legible application. It manages the end-to-end workflow from downloading documents from the reMarkable API to generating searchable PDFs with OCR text layers.

## Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                     Sync Orchestrator                             │
└────────┬──────────────┬──────────────┬──────────────┬────────────┘
         │              │              │              │
         ▼              ▼              ▼              ▼
   ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐
   │ RMClient │  │  State   │  │Converter │  │   OCR    │
   └──────────┘  └──────────┘  └──────────┘  └──────────┘
         │              │              │              │
         ▼              ▼              ▼              ▼
    reMarkable      Local DB      .rmdoc → PDF   Tesseract
       API                                          OCR
```

## Workflow

The orchestrator follows this workflow for each sync operation:

### 1. **List Documents**
   - Fetch document list from reMarkable API
   - Retrieve document metadata (ID, title, version, modified time, labels)

### 2. **Filter by Labels** (Optional)
   - If label filters are configured, filter documents to sync
   - Only documents matching configured labels are processed
   - If no filters, all documents are processed

### 3. **Load Sync State**
   - Load previous sync state from local storage
   - State contains document IDs, versions, and last sync times
   - Initialize fresh state if none exists

### 4. **Identify New/Changed Documents**
   - Compare API documents with local state
   - Documents to sync:
     - New documents (not in state)
     - Changed documents (version mismatch)
   - Skip unchanged documents (same version in state)

### 5. **Process Each Document**
   For each document to sync, the orchestrator runs this pipeline:

   **a. Download**
   - Download `.rmdoc` file from reMarkable API
   - Save to temporary directory

   **b. Convert**
   - Convert `.rmdoc` to PDF format
   - Extract pages and render content
   - Generate standard PDF file

   **c. OCR** (Optional, if enabled)
   - The converter handles OCR internally when configured with `EnableOCR: true`
   - Renders PDF pages to images at 300 DPI for optimal OCR accuracy
   - Performs OCR using Ollama vision models (llava, mistral-small3.1, etc.)
   - Extracts text with bounding boxes and confidence scores
   - Adds invisible searchable text layer to PDF automatically
   - Makes PDF searchable without changing visual appearance

   **e. Save**
   - Move final PDF to configured output directory
   - Sanitize filename (remove invalid characters)
   - Update sync state with document info

### 6. **Error Handling**
   - Continue processing even if individual documents fail
   - Collect errors for failed documents
   - Update state incrementally (don't lose progress)
   - Report all errors at completion

### 7. **Final Summary**
   - Calculate totals and statistics
   - Report: total, processed, successful, failed
   - Display duration and any failures

## Usage

### Basic Usage

```go
package main

import (
	"context"

	"github.com/platinummonkey/legible/internal/config"
	"github.com/platinummonkey/legible/internal/converter"
	"github.com/platinummonkey/legible/internal/pdfenhancer"
	"github.com/platinummonkey/legible/internal/rmclient"
	"github.com/platinummonkey/legible/internal/state"
	"github.com/platinummonkey/legible/internal/sync"
)

func main() {
	// Create configuration
	cfg := &config.Config{
		OutputDir:  "/path/to/output",
		Labels:     []string{"work", "personal"}, // Optional filter
		OCREnabled: true,
	}

	// Initialize components
	rmClient := rmclient.New(&rmclient.Config{})
	stateStore := state.New(&state.Config{})
	converter := converter.New(&converter.Config{})
	pdfEnhancer := pdfenhancer.New(&pdfenhancer.Config{})

	// Create orchestrator
	orch, err := sync.New(&sync.Config{
		Config:      cfg,
		RMClient:    rmClient,
		StateStore:  stateStore,
		Converter:   converter,
		PDFEnhancer: pdfEnhancer,
	})
	if err != nil {
		panic(err)
	}

	// Run sync
	ctx := context.Background()
	result, err := orch.Sync(ctx)
	if err != nil {
		panic(err)
	}

	// Display results
	fmt.Println(result.Summary())
}
```

### With OCR Processor

```go
// Add OCR processor if Tesseract is installed
ocrProc := ocr.New(&ocr.Config{
	Languages: []string{"eng"},
})

orch, err := sync.New(&sync.Config{
	Config:       cfg,
	RMClient:     rmClient,
	StateStore:   stateStore,
	Converter:    converter,
	OCRProcessor: ocrProc,      // Optional
	PDFEnhancer:  pdfEnhancer,
})
```

### Result Handling

```go
result, err := orch.Sync(ctx)
if err != nil {
	log.Fatal(err)
}

// Check for failures
if result.HasFailures() {
	fmt.Println("Some documents failed:")
	for _, failure := range result.Failures {
		fmt.Printf("  - %s: %v\n", failure.Title, failure.Error)
	}
}

// Display statistics
fmt.Printf("Processed %d/%d documents in %v\n",
	result.SuccessCount,
	result.ProcessedDocuments,
	result.Duration)
```

## Configuration

The orchestrator uses the `config.Config` struct for configuration:

```go
type Config struct {
	OutputDir  string   // Directory for output PDFs
	Labels     []string // Label filters (empty = all documents)
	OCREnabled bool     // Enable OCR processing
}
```

**Label Filtering:**
- When `Labels` is empty or nil, all documents are synced
- When `Labels` contains values, only documents with matching labels are synced
- A document matches if it has *any* of the configured labels (OR logic)

## Progress Tracking

The orchestrator logs progress at INFO level:

```
INFO: Listing documents from reMarkable API
INFO: Retrieved documents from API count=25
INFO: Filtered documents by labels original=25 filtered=15
INFO: Identified documents to sync count=5
INFO: Processing document document=1 total=5 id=abc-123 title="Meeting Notes"
INFO: Downloading document document=1 total=5
INFO: Converting to PDF document=1 total=5
INFO: Performing OCR document=1 total=5
INFO: Adding OCR text layer document=1 total=5
INFO: Document processing completed document=1 total=5 output=/path/to/output/Meeting Notes.pdf duration=15s
...
INFO: Sync workflow completed total=15 processed=5 successful=5 failed=0 duration=1m30s
```

## Error Handling

The orchestrator implements robust error handling:

### Graceful Degradation
- If state loading fails, starts with empty state
- If OCR is requested but processor is nil, logs warning and skips OCR
- If text layer addition fails, uses original PDF

### Continue on Failure
- Individual document failures don't stop the sync
- Errors are collected and reported at the end
- State is updated incrementally after each successful document

### Error Collection
```go
result, _ := orch.Sync(ctx)

for _, failure := range result.Failures {
	fmt.Printf("Failed: %s (%s)\n", failure.Title, failure.DocumentID)
	fmt.Printf("  Error: %v\n", failure.Error)
}
```

## Testing

The package includes comprehensive tests:

**Unit Tests:**
- Orchestrator initialization and validation
- Label filtering logic
- Document identification (new/changed detection)
- Filename sanitization
- File copying utilities

**Result Tests:**
- Result creation and modification
- Success and failure tracking
- Summary generation
- String formatting

**Integration Tests:**
- Full workflow tests require mocking of dependencies
- Individual components have their own test suites

Run tests:
```bash
go test ./internal/sync
go test -v ./internal/sync
```

## Implementation Details

### Filename Sanitization

The orchestrator sanitizes document titles for use as filenames:

```go
// Input: "Project: Design/Architecture (v2)"
// Output: "Project- Design-Architecture (v2).pdf"
```

**Replaced Characters:**
- `/` → `-` (slash)
- `\` → `-` (backslash)
- `:` → `-` (colon)
- `*` → `_` (asterisk)
- `?` → `_` (question mark)
- `"` → `'` (quote)
- `<` → `_` (less than)
- `>` → `_` (greater than)
- `|` → `-` (pipe)

### Incremental State Updates

State is saved after each successful document:

```go
for _, doc := range docsToSync {
	// Process document...

	// Update state immediately
	currentState.UpdateDocument(docState)
	stateStore.Save(currentState)
}
```

**Benefits:**
- Progress is not lost if sync is interrupted
- Failed documents don't prevent state updates for successful ones
- Next sync picks up where previous sync left off

### Temporary File Handling

Each document is processed in its own temporary directory:

```go
tmpDir := os.MkdirTemp("", fmt.Sprintf("rmsync-%s-*", doc.ID))
defer os.RemoveAll(tmpDir)
```

**Benefits:**
- No conflicts between concurrent processing (future)
- Clean up happens automatically via defer
- Isolated processing environment per document

## Future Enhancements

### Immediate Priorities

1. **PDF-to-Image Rendering**
   - Implement PDF page rendering for OCR input
   - Support multiple image formats (PNG, JPEG)
   - Configurable resolution/DPI

2. **Parallel Processing**
   - Process multiple documents concurrently
   - Configurable worker pool size
   - Maintain progress tracking across workers

### Long-term Enhancements

3. **Retry Logic**
   - Exponential backoff for transient failures
   - Configurable retry attempts and delays
   - Distinguish between retryable and permanent errors

4. **Webhooks/Notifications**
   - Callback hooks for sync events
   - Email/Slack notifications on completion
   - Custom notification plugins

5. **Incremental Sync Optimization**
   - Skip downloading if local file exists with same version
   - Hash-based change detection
   - Resume interrupted downloads

6. **Dry Run Mode**
   - Preview what would be synced without syncing
   - Estimate download sizes and durations
   - Validate configuration before actual sync

7. **Selective Sync**
   - Sync specific documents by ID
   - Sync documents modified after a date
   - Sync only new documents (skip updates)

8. **Progress Callbacks**
   - Programmatic progress reporting
   - Real-time UI updates
   - Progress bar integration

## Dependencies

The orchestrator depends on these internal packages:

- **config**: Application configuration
- **logger**: Structured logging
- **rmclient**: reMarkable API client
- **state**: Sync state management
- **converter**: .rmdoc to PDF conversion
- **ocr**: OCR processing (optional)
- **pdfenhancer**: PDF text layer addition
- **types**: Shared type definitions

## License

Part of legible project. See project LICENSE for details.
