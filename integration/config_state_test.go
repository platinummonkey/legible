package integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/platinummonkey/remarkable-sync/internal/config"
	"github.com/platinummonkey/remarkable-sync/internal/logger"
	"github.com/platinummonkey/remarkable-sync/internal/rmclient"
	"github.com/platinummonkey/remarkable-sync/internal/state"
)

// TestConfigStateIntegration tests the integration between config and state management
func TestConfigStateIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test config file
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
output-dir: ` + filepath.Join(tmpDir, "output") + `
labels:
  - test
  - integration
ocr-enabled: false
log-level: info
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Set environment variable to use test config
	if err := os.Setenv("REMARKABLE_SYNC_CONFIG", configPath); err != nil {
		t.Fatalf("Failed to set environment variable: %v", err)
	}
	defer func() {
		_ = os.Unsetenv("REMARKABLE_SYNC_CONFIG")
	}()

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Config validation failed: %v", err)
	}

	// Verify config values
	if cfg.OutputDir != filepath.Join(tmpDir, "output") {
		t.Errorf("Expected output_dir %s, got %s", filepath.Join(tmpDir, "output"), cfg.OutputDir)
	}

	if len(cfg.Labels) != 2 {
		t.Errorf("Expected 2 labels, got %d", len(cfg.Labels))
	}

	if cfg.OCREnabled {
		t.Error("Expected OCR to be disabled")
	}

	// Create state manager using config
	stateFile := filepath.Join(cfg.OutputDir, ".sync-state.json")
	mgr := state.NewManager(stateFile)

	// Test state persistence workflow
	doc := &state.DocumentState{
		ID:               "test-doc-123",
		Name:             "Test Document",
		Type:             "DocumentType",
		Version:          1,
		ModifiedClient:   time.Now(),
		LastSynced:       time.Now(),
		LocalPath:        filepath.Join(cfg.OutputDir, "test-doc.pdf"),
		ConversionStatus: state.ConversionStatusCompleted,
		OCRProcessed:     true,
		OCRTimestamp:     time.Now(),
		Labels:           cfg.Labels,
	}

	mgr.AddDocument(doc)

	// Save state
	if err := mgr.Save(); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Verify state file was created
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Error("State file should exist after Save()")
	}

	// Load state in new manager
	mgr2 := state.NewManager(stateFile)
	if err := mgr2.Load(); err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// Verify document was persisted
	loadedDoc := mgr2.GetDocument("test-doc-123")
	if loadedDoc == nil {
		t.Fatal("Document should be loaded from state file")
	}

	if loadedDoc.Name != "Test Document" {
		t.Errorf("Expected document name 'Test Document', got %s", loadedDoc.Name)
	}

	if len(loadedDoc.Labels) != 2 {
		t.Errorf("Expected 2 labels, got %d", len(loadedDoc.Labels))
	}
}

// TestLoggerIntegration tests logger initialization and usage across components
func TestLoggerIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	// Create logger with file output
	log, err := logger.New(&logger.Config{
		Level:      "debug",
		Format:     "json",
		OutputPath: logFile,
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test logging with different components
	log.Info("Starting integration test")
	log.WithFields("component", "config").Debug("Loading configuration")
	log.WithFields("component", "state").Info("Initializing state manager")
	log.WithDocumentID("doc-123").WithOperation("sync").Info("Processing document")

	// Sync logger to flush buffers
	if err := log.Sync(); err != nil && err.Error() != "sync /dev/stdout: bad file descriptor" {
		t.Logf("Logger sync warning: %v", err)
	}

	// Verify log file was created and has content
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Error("Log file should exist")
	}

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Log file should not be empty")
	}
}

// TestRMClientTokenPersistence tests authentication token persistence
func TestRMClientTokenPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "token.json")

	log, err := logger.New(&logger.Config{
		Level:  "info",
		Format: "console",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Create first client and set token
	client1, err := rmclient.NewClient(&rmclient.Config{
		TokenPath: tokenPath,
		Logger:    log,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	testToken := "test-token-12345-integration"
	if err := client1.SetToken(testToken); err != nil {
		t.Fatalf("Failed to set token: %v", err)
	}

	// Verify client is authenticated
	if !client1.IsAuthenticated() {
		t.Error("Client should be authenticated after SetToken")
	}

	// Create second client with same token path
	client2, err := rmclient.NewClient(&rmclient.Config{
		TokenPath: tokenPath,
		Logger:    log,
	})
	if err != nil {
		t.Fatalf("Failed to create second client: %v", err)
	}

	// Authenticate should load existing token
	if err := client2.Authenticate(); err != nil {
		t.Fatalf("Failed to authenticate with existing token: %v", err)
	}

	// Verify second client is authenticated
	if !client2.IsAuthenticated() {
		t.Error("Second client should be authenticated with persisted token")
	}

	// Clean up
	_ = client1.Close()
	_ = client2.Close()
}

// TestMultipleDocumentWorkflow tests managing multiple documents through state
func TestMultipleDocumentWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, ".sync-state.json")

	mgr := state.NewManager(stateFile)

	// Add multiple documents with different states
	documents := []*state.DocumentState{
		{
			ID:               "doc-1",
			Name:             "Completed Document",
			Version:          1,
			ModifiedClient:   time.Now(),
			LastSynced:       time.Now(),
			ConversionStatus: state.ConversionStatusCompleted,
			OCRProcessed:     true,
			OCRTimestamp:     time.Now(),
			Labels:           []string{"work", "important"},
		},
		{
			ID:               "doc-2",
			Name:             "Pending Document",
			Version:          1,
			ModifiedClient:   time.Now(),
			ConversionStatus: state.ConversionStatusPending,
			OCRProcessed:     false,
			Labels:           []string{"personal"},
		},
		{
			ID:               "doc-3",
			Name:             "Failed Document",
			Version:          1,
			ModifiedClient:   time.Now(),
			ConversionStatus: state.ConversionStatusFailed,
			OCRProcessed:     false,
			Labels:           []string{"work"},
		},
	}

	for _, doc := range documents {
		mgr.AddDocument(doc)
	}

	// Test filtering by status
	completed := mgr.GetDocumentsByStatus(state.ConversionStatusCompleted)
	if len(completed) != 1 {
		t.Errorf("Expected 1 completed document, got %d", len(completed))
	}

	pending := mgr.GetDocumentsByStatus(state.ConversionStatusPending)
	if len(pending) != 1 {
		t.Errorf("Expected 1 pending document, got %d", len(pending))
	}

	failed := mgr.GetDocumentsByStatus(state.ConversionStatusFailed)
	if len(failed) != 1 {
		t.Errorf("Expected 1 failed document, got %d", len(failed))
	}

	// Test filtering by labels
	workDocs := mgr.GetDocumentsByLabel("work")
	if len(workDocs) != 2 {
		t.Errorf("Expected 2 work documents, got %d", len(workDocs))
	}

	personalDocs := mgr.GetDocumentsByLabel("personal")
	if len(personalDocs) != 1 {
		t.Errorf("Expected 1 personal document, got %d", len(personalDocs))
	}

	// Test documents needing OCR
	needsOCR := mgr.GetDocumentsNeedingOCR()
	if len(needsOCR) != 0 { // All have OCR complete, pending, or not required
		t.Errorf("Expected 0 documents needing OCR, got %d", len(needsOCR))
	}

	// Save and reload
	if err := mgr.Save(); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	mgr2 := state.NewManager(stateFile)
	if err := mgr2.Load(); err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// Verify all documents persisted
	if mgr2.Count() != 3 {
		t.Errorf("Expected 3 documents after reload, got %d", mgr2.Count())
	}
}

// TestStateUpdateWorkflow tests updating document states through sync lifecycle
func TestStateUpdateWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, ".sync-state.json")

	mgr := state.NewManager(stateFile)

	// Create document in pending state
	doc := &state.DocumentState{
		ID:               "doc-workflow",
		Name:             "Workflow Test Document",
		Version:          1,
		ModifiedClient:   time.Now(),
		ConversionStatus: state.ConversionStatusPending,
		OCRProcessed:     false,
	}

	mgr.AddDocument(doc)

	// Simulate conversion completion
	doc = mgr.GetDocument("doc-workflow")
	doc.ConversionStatus = state.ConversionStatusCompleted
	doc.LocalPath = filepath.Join(tmpDir, "doc-workflow.pdf")
	mgr.AddDocument(doc) // Update

	if err := mgr.Save(); err != nil {
		t.Fatalf("Failed to save after conversion: %v", err)
	}

	// Reload and verify
	_ = mgr.Load()
	doc = mgr.GetDocument("doc-workflow")
	if doc.ConversionStatus != state.ConversionStatusCompleted {
		t.Error("Document should have ConversionStatusCompleted status")
	}

	// Simulate OCR completion
	doc.OCRProcessed = true
	doc.OCRTimestamp = time.Now()
	mgr.AddDocument(doc)

	if err := mgr.Save(); err != nil {
		t.Fatalf("Failed to save after OCR: %v", err)
	}

	// Final verification
	_ = mgr.Load()
	doc = mgr.GetDocument("doc-workflow")
	if !doc.OCRProcessed {
		t.Error("Document should have OCRProcessed set to true")
	}

	if doc.OCRTimestamp.IsZero() {
		t.Error("OCRTimestamp should be set")
	}
}
