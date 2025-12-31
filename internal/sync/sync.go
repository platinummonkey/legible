package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/platinummonkey/remarkable-sync/internal/config"
	"github.com/platinummonkey/remarkable-sync/internal/converter"
	"github.com/platinummonkey/remarkable-sync/internal/logger"
	"github.com/platinummonkey/remarkable-sync/internal/ocr"
	"github.com/platinummonkey/remarkable-sync/internal/pdfenhancer"
	"github.com/platinummonkey/remarkable-sync/internal/rmclient"
	"github.com/platinummonkey/remarkable-sync/internal/state"
)

// Orchestrator coordinates the complete sync workflow
type Orchestrator struct {
	config      *config.Config
	logger      *logger.Logger
	rmClient    *rmclient.Client
	stateStore  *state.Manager
	converter   *converter.Converter
	ocrProc     *ocr.Processor
	pdfEnhancer *pdfenhancer.PDFEnhancer
}

// Config holds configuration for the sync orchestrator
type Config struct {
	Config       *config.Config
	Logger       *logger.Logger
	RMClient     *rmclient.Client
	StateStore   *state.Manager
	Converter    *converter.Converter
	OCRProcessor *ocr.Processor
	PDFEnhancer  *pdfenhancer.PDFEnhancer
}

// New creates a new sync orchestrator
func New(cfg *Config) (*Orchestrator, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Use provided logger or get default
	log := cfg.Logger
	if log == nil {
		log = logger.Get()
	}

	// Validate required dependencies
	if cfg.Config == nil {
		return nil, fmt.Errorf("config.Config is required")
	}
	if cfg.RMClient == nil {
		return nil, fmt.Errorf("rmClient is required")
	}
	if cfg.StateStore == nil {
		return nil, fmt.Errorf("stateStore is required")
	}
	if cfg.Converter == nil {
		return nil, fmt.Errorf("converter is required")
	}
	if cfg.PDFEnhancer == nil {
		return nil, fmt.Errorf("pdfEnhancer is required")
	}

	return &Orchestrator{
		config:      cfg.Config,
		logger:      log,
		rmClient:    cfg.RMClient,
		stateStore:  cfg.StateStore,
		converter:   cfg.Converter,
		ocrProc:     cfg.OCRProcessor,
		pdfEnhancer: cfg.PDFEnhancer,
	}, nil
}

// Sync performs a complete synchronization workflow
func (o *Orchestrator) Sync(ctx context.Context) (*Result, error) {
	o.logger.Info("Starting sync workflow")
	startTime := time.Now()

	result := NewResult()

	// Step 1: List documents from API
	o.logger.Info("Listing documents from reMarkable API")
	docs, err := o.rmClient.ListDocuments(o.config.Labels)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}
	o.logger.WithFields("count", len(docs)).Info("Retrieved documents from API")

	// Step 2: Filter by labels if configured
	filteredDocs := o.filterDocumentsByLabels(docs)
	o.logger.WithFields("original", len(docs), "filtered", len(filteredDocs)).
		Info("Filtered documents by labels")

	// Step 3: Load current sync state
	if err := o.stateStore.Load(); err != nil {
		o.logger.WithFields("error", err).Warn("Failed to load state, starting fresh")
	}
	currentState := o.stateStore.GetState()

	// Step 4: Identify new/changed documents
	docsToSync := o.identifyDocumentsToSync(filteredDocs, currentState)
	o.logger.WithFields("count", len(docsToSync)).Info("Identified documents to sync")

	// Step 5: Process each document
	for i, doc := range docsToSync {
		docNum := i + 1
		totalDocs := len(docsToSync)

		o.logger.WithFields(
			"document", docNum,
			"total", totalDocs,
			"id", doc.ID,
			"title", doc.Name,
		).Info("Processing document")

		// Process document through pipeline
		docResult, err := o.processDocument(ctx, doc, docNum, totalDocs)
		if err != nil {
			o.logger.WithFields("id", doc.ID, "error", err).Error("Document processing failed")
			result.AddError(doc.ID, doc.Name, err)
			continue
		}

		// Update result
		result.AddSuccess(docResult)

		// Update state incrementally (don't lose progress on failures)
		docState := &state.DocumentState{
			ID:             doc.ID,
			Version:        doc.Version,
			ModifiedClient: doc.ModifiedClient,
			LastSynced:     time.Now(),
			LocalPath:      docResult.OutputPath,
		}
		docState.SetConversionStatus(state.ConversionStatusCompleted)
		currentState.AddDocument(docState)

		// Save state after each document
		if err := o.stateStore.Save(); err != nil {
			o.logger.WithFields("error", err).Warn("Failed to save state")
		}
	}

	// Step 6: Finalize result
	result.Duration = time.Since(startTime)
	result.TotalDocuments = len(filteredDocs)
	result.ProcessedDocuments = len(docsToSync)

	o.logger.WithFields(
		"total", result.TotalDocuments,
		"processed", result.ProcessedDocuments,
		"successful", result.SuccessCount,
		"failed", result.FailureCount,
		"duration", result.Duration,
	).Info("Sync workflow completed")

	return result, nil
}

// filterDocumentsByLabels filters documents by configured labels
func (o *Orchestrator) filterDocumentsByLabels(docs []rmclient.Document) []rmclient.Document {
	// If no label filters configured, return all documents
	if len(o.config.Labels) == 0 {
		return docs
	}

	// Build label filter map for fast lookup
	labelFilter := make(map[string]bool)
	for _, label := range o.config.Labels {
		labelFilter[label] = true
	}

	// Filter documents
	var filtered []rmclient.Document
	for _, doc := range docs {
		// TODO: Document labels are not available in the current API
		// The API now filters by labels at the ListDocuments call level
		// For now, accept all documents returned by the API
		hasMatchingLabel := true

		if hasMatchingLabel {
			filtered = append(filtered, doc)
		}
	}

	return filtered
}

// identifyDocumentsToSync compares API documents with state to find new/changed documents
func (o *Orchestrator) identifyDocumentsToSync(docs []rmclient.Document, currentState *state.SyncState) []rmclient.Document {
	var toSync []rmclient.Document

	for _, doc := range docs {
		// Check if document exists in state
		docState := currentState.GetDocument(doc.ID)

		// Sync if document is new or version changed
		if docState == nil || docState.Version != doc.Version {
			toSync = append(toSync, doc)
		}
	}

	return toSync
}

// processDocument processes a single document through the complete pipeline
func (o *Orchestrator) processDocument(_ context.Context, doc rmclient.Document, docNum, totalDocs int) (*DocumentResult, error) {
	result := &DocumentResult{
		DocumentID: doc.ID,
		Title:      doc.Name,
		StartTime:  time.Now(),
	}

	// Create temporary directory for processing
	tmpDir, err := os.MkdirTemp("", fmt.Sprintf("rmsync-%s-*", doc.ID))
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Stage 1: Download .rmdoc file
	o.logger.WithFields("document", docNum, "total", totalDocs).
		Info("Downloading document")

	rmdocPath := filepath.Join(tmpDir, fmt.Sprintf("%s.rmdoc", doc.ID))
	if err := o.rmClient.DownloadDocument(doc.ID, rmdocPath); err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}

	// Stage 2: Convert .rmdoc to PDF
	o.logger.WithFields("document", docNum, "total", totalDocs).
		Info("Converting to PDF")

	pdfPath := filepath.Join(tmpDir, fmt.Sprintf("%s.pdf", doc.ID))
	convResult, err := o.converter.ConvertRmdoc(rmdocPath, pdfPath)
	if err != nil {
		return nil, fmt.Errorf("conversion failed: %w", err)
	}
	result.PageCount = convResult.PageCount

	// Stage 3: OCR processing (if enabled and OCR processor available)
	var ocrResults *ocr.DocumentOCR
	if o.config.OCREnabled && o.ocrProc != nil {
		o.logger.WithFields("document", docNum, "total", totalDocs).
			Info("Performing OCR")

		// TODO: Implement PDF-to-image rendering for OCR
		// For now, we skip OCR and log a warning
		o.logger.Warn("OCR requested but PDF-to-image rendering not yet implemented")
	}

	// Stage 4: Add text layer (if OCR was performed)
	finalPath := pdfPath
	if ocrResults != nil && len(ocrResults.Pages) > 0 {
		o.logger.WithFields("document", docNum, "total", totalDocs).
			Info("Adding OCR text layer")

		enhancedPath := filepath.Join(tmpDir, fmt.Sprintf("%s_ocr.pdf", doc.ID))
		if err := o.pdfEnhancer.AddTextLayer(pdfPath, enhancedPath, ocrResults); err != nil {
			o.logger.WithFields("error", err).Warn("Failed to add text layer, using original PDF")
		} else {
			finalPath = enhancedPath
		}
	}

	// Stage 5: Determine output path with folder structure
	// Get the folder path for this document from reMarkable
	folderPath, err := o.rmClient.GetFolderPath(doc.ID)
	if err != nil {
		o.logger.WithFields("id", doc.ID, "error", err).
			Warn("Failed to get folder path, saving to root output directory")
		folderPath = "" // Fall back to root if path lookup fails
	}

	// Build output directory path (OutputDir + folder path)
	outputDir := o.config.OutputDir
	if folderPath != "" {
		outputDir = filepath.Join(o.config.OutputDir, folderPath)
		o.logger.WithFields("document", docNum, "folder_path", folderPath).
			Debug("Preserving folder structure")
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	outputFilename := fmt.Sprintf("%s.pdf", sanitizeFilename(doc.Name))
	outputPath := filepath.Join(outputDir, outputFilename)

	// Copy final PDF to output location
	if err := copyFile(finalPath, outputPath); err != nil {
		return nil, fmt.Errorf("failed to copy to output directory: %w", err)
	}

	result.OutputPath = outputPath
	result.Duration = time.Since(result.StartTime)

	o.logger.WithFields(
		"document", docNum,
		"total", totalDocs,
		"output", outputPath,
		"duration", result.Duration,
	).Info("Document processing completed")

	return result, nil
}

// sanitizeFilename removes or replaces characters that are invalid in filenames
func sanitizeFilename(name string) string {
	// Replace common problematic characters
	replacements := map[rune]string{
		'/':  "-",
		'\\': "-",
		':':  "-",
		'*':  "_",
		'?':  "_",
		'"':  "'",
		'<':  "_",
		'>':  "_",
		'|':  "-",
	}

	result := ""
	for _, ch := range name {
		if replacement, found := replacements[ch]; found {
			result += replacement
		} else {
			result += string(ch)
		}
	}

	return result
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	if err := os.WriteFile(dst, data, 0644); err != nil {
		return fmt.Errorf("failed to write destination file: %w", err)
	}

	return nil
}
