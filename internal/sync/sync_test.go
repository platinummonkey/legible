package sync

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/platinummonkey/legible/internal/config"
	"github.com/platinummonkey/legible/internal/converter"
	"github.com/platinummonkey/legible/internal/logger"
	"github.com/platinummonkey/legible/internal/pdfenhancer"
	"github.com/platinummonkey/legible/internal/rmclient"
	"github.com/platinummonkey/legible/internal/state"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{
		Config: &config.Config{
			OutputDir: tmpDir,
		},
		RMClient:    &rmclient.Client{},
		StateStore:  &state.Manager{},
		Converter:   &converter.Converter{},
		PDFEnhancer: &pdfenhancer.PDFEnhancer{},
	}

	orch, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if orch == nil {
		t.Fatal("New() returned nil")
	}

	if orch.logger == nil {
		t.Error("logger should be initialized")
	}
}

func TestNew_NilConfig(t *testing.T) {
	_, err := New(nil)
	if err == nil {
		t.Error("New() should error with nil config")
	}
}

func TestNew_MissingDependencies(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
	}{
		{
			name: "missing Config",
			cfg: &Config{
				RMClient:    &rmclient.Client{},
				StateStore:  &state.Manager{},
				Converter:   &converter.Converter{},
				PDFEnhancer: &pdfenhancer.PDFEnhancer{},
			},
		},
		{
			name: "missing RMClient",
			cfg: &Config{
				Config:      &config.Config{},
				StateStore:  &state.Manager{},
				Converter:   &converter.Converter{},
				PDFEnhancer: &pdfenhancer.PDFEnhancer{},
			},
		},
		{
			name: "missing StateStore",
			cfg: &Config{
				Config:      &config.Config{},
				RMClient:    &rmclient.Client{},
				Converter:   &converter.Converter{},
				PDFEnhancer: &pdfenhancer.PDFEnhancer{},
			},
		},
		{
			name: "missing Converter",
			cfg: &Config{
				Config:      &config.Config{},
				RMClient:    &rmclient.Client{},
				StateStore:  &state.Manager{},
				PDFEnhancer: &pdfenhancer.PDFEnhancer{},
			},
		},
		{
			name: "missing PDFEnhancer",
			cfg: &Config{
				Config:     &config.Config{},
				RMClient:   &rmclient.Client{},
				StateStore: &state.Manager{},
				Converter:  &converter.Converter{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.cfg)
			if err == nil {
				t.Errorf("New() should error for %s", tt.name)
			}
		})
	}
}

func TestIdentifyDocumentsToSync(t *testing.T) {
	tmpDir := t.TempDir()
	orch := &Orchestrator{
		config: &config.Config{
			OutputDir: tmpDir,
		},
		logger: logger.Get(),
	}

	// Create state with one existing document
	currentState := state.NewSyncState()
	currentState.AddDocument(&state.DocumentState{
		ID:      "1",
		Version: 1,
	})

	docs := []rmclient.Document{
		{ID: "1", Version: 1}, // Unchanged
		{ID: "2", Version: 1}, // New
		{ID: "3", Version: 2}, // New
		{ID: "1", Version: 2}, // Changed (same ID, different version)
	}

	// Note: In real usage, docs wouldn't have duplicate IDs, but this tests the logic
	toSync := orch.identifyDocumentsToSync(docs, currentState)

	// Should sync docs: 2, 3, and the updated version of 1
	// Depends on deduplication logic, but at minimum should identify new and changed
	if len(toSync) < 2 {
		t.Errorf("expected at least 2 documents to sync, got %d", len(toSync))
	}
}

func TestIdentifyDocumentsToSync_MissingLocalFile(t *testing.T) {
	tmpDir := t.TempDir()
	orch := &Orchestrator{
		config: &config.Config{
			OutputDir: tmpDir,
		},
		logger: logger.Get(),
	}

	// Create a test file that exists
	existingFilePath := filepath.Join(tmpDir, "existing.pdf")
	if err := os.WriteFile(existingFilePath, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create state with documents in various states
	currentState := state.NewSyncState()

	// Doc 1: Exists in state with valid local file (should NOT sync)
	currentState.AddDocument(&state.DocumentState{
		ID:        "doc1",
		Version:   1,
		LocalPath: existingFilePath,
	})

	// Doc 2: Exists in state with missing local file (should sync)
	missingFilePath := filepath.Join(tmpDir, "missing.pdf")
	currentState.AddDocument(&state.DocumentState{
		ID:        "doc2",
		Version:   1,
		LocalPath: missingFilePath,
	})

	// Doc 3: Exists in state with empty LocalPath (should NOT sync)
	currentState.AddDocument(&state.DocumentState{
		ID:        "doc3",
		Version:   1,
		LocalPath: "",
	})

	// API returns these documents
	docs := []rmclient.Document{
		{ID: "doc1", Version: 1}, // File exists
		{ID: "doc2", Version: 1}, // File missing
		{ID: "doc3", Version: 1}, // No local path
	}

	toSync := orch.identifyDocumentsToSync(docs, currentState)

	// Should only sync doc2 (missing file)
	if len(toSync) != 1 {
		t.Errorf("expected 1 document to sync, got %d", len(toSync))
	}

	if len(toSync) > 0 && toSync[0].ID != "doc2" {
		t.Errorf("expected doc2 to be synced, got %s", toSync[0].ID)
	}
}

func TestIdentifyDocumentsToSync_MissingFileAndVersionChange(t *testing.T) {
	tmpDir := t.TempDir()
	orch := &Orchestrator{
		config: &config.Config{
			OutputDir: tmpDir,
		},
		logger: logger.Get(),
	}

	// Create state with a document that has both missing file AND version change
	currentState := state.NewSyncState()
	missingFilePath := filepath.Join(tmpDir, "missing.pdf")
	currentState.AddDocument(&state.DocumentState{
		ID:        "doc1",
		Version:   1,
		LocalPath: missingFilePath,
	})

	// API returns document with newer version
	docs := []rmclient.Document{
		{ID: "doc1", Version: 2}, // Both missing file and version changed
	}

	toSync := orch.identifyDocumentsToSync(docs, currentState)

	// Should sync because of version change (checked first)
	if len(toSync) != 1 {
		t.Errorf("expected 1 document to sync, got %d", len(toSync))
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "normal-filename",
			expected: "normal-filename",
		},
		{
			input:    "file/with/slashes",
			expected: "file-with-slashes",
		},
		{
			input:    "file\\with\\backslashes",
			expected: "file-with-backslashes",
		},
		{
			input:    "file:with:colons",
			expected: "file-with-colons",
		},
		{
			input:    "file*with?special\"chars",
			expected: "file_with_special'chars",
		},
		{
			input:    "file<with>pipes|and<brackets>",
			expected: "file_with_pipes-and_brackets_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	srcPath := filepath.Join(tmpDir, "source.txt")
	content := []byte("test content")
	if err := os.WriteFile(srcPath, content, 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Copy file
	dstPath := filepath.Join(tmpDir, "dest.txt")
	if err := copyFile(srcPath, dstPath); err != nil {
		t.Fatalf("copyFile() error = %v", err)
	}

	// Verify destination file exists and has same content
	dstContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("failed to read destination file: %v", err)
	}

	if string(dstContent) != string(content) {
		t.Errorf("destination content = %q, want %q", string(dstContent), string(content))
	}
}

func TestCopyFile_SourceNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	srcPath := filepath.Join(tmpDir, "nonexistent.txt")
	dstPath := filepath.Join(tmpDir, "dest.txt")

	err := copyFile(srcPath, dstPath)
	if err == nil {
		t.Error("copyFile() should error when source doesn't exist")
	}
}

func TestCopyFile_InvalidDestination(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	srcPath := filepath.Join(tmpDir, "source.txt")
	if err := os.WriteFile(srcPath, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Try to copy to invalid destination
	dstPath := filepath.Join(tmpDir, "nonexistent-dir", "dest.txt")

	err := copyFile(srcPath, dstPath)
	if err == nil {
		t.Error("copyFile() should error with invalid destination directory")
	}
}

// Note: Full integration tests for Sync() would require mocking or test implementations
// of rmclient, state, converter, and pdfenhancer, which is complex. The above tests
// cover the individual components and helper functions of the orchestrator.
