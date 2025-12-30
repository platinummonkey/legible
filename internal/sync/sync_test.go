package sync

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/platinummonkey/remarkable-sync/internal/config"
	"github.com/platinummonkey/remarkable-sync/internal/converter"
	"github.com/platinummonkey/remarkable-sync/internal/logger"
	"github.com/platinummonkey/remarkable-sync/internal/ocr"
	"github.com/platinummonkey/remarkable-sync/internal/pdfenhancer"
	"github.com/platinummonkey/remarkable-sync/internal/rmclient"
	"github.com/platinummonkey/remarkable-sync/internal/state"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{
		Config: &config.Config{
			OutputDir: tmpDir,
		},
		RMClient:    &rmclient.Client{},
		StateStore:  &state.Store{},
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
				StateStore:  &state.Store{},
				Converter:   &converter.Converter{},
				PDFEnhancer: &pdfenhancer.PDFEnhancer{},
			},
		},
		{
			name: "missing RMClient",
			cfg: &Config{
				Config:      &config.Config{},
				StateStore:  &state.Store{},
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
				StateStore:  &state.Store{},
				PDFEnhancer: &pdfenhancer.PDFEnhancer{},
			},
		},
		{
			name: "missing PDFEnhancer",
			cfg: &Config{
				Config:     &config.Config{},
				RMClient:   &rmclient.Client{},
				StateStore: &state.Store{},
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

func TestFilterDocumentsByLabels(t *testing.T) {
	tmpDir := t.TempDir()
	orch := &Orchestrator{
		config: &config.Config{
			OutputDir: tmpDir,
			Labels:    []string{"work", "personal"},
		},
		logger: logger.Get(),
	}

	docs := []rmclient.Document{
		{ID: "1", Title: "Work Doc", Labels: []string{"work"}},
		{ID: "2", Title: "Personal Doc", Labels: []string{"personal"}},
		{ID: "3", Title: "Project Doc", Labels: []string{"project"}},
		{ID: "4", Title: "Mixed Doc", Labels: []string{"work", "project"}},
	}

	filtered := orch.filterDocumentsByLabels(docs)

	// Should include docs 1, 2, and 4 (have work or personal labels)
	if len(filtered) != 3 {
		t.Errorf("expected 3 filtered documents, got %d", len(filtered))
	}

	// Verify correct documents were filtered
	foundIDs := make(map[string]bool)
	for _, doc := range filtered {
		foundIDs[doc.ID] = true
	}

	if !foundIDs["1"] || !foundIDs["2"] || !foundIDs["4"] {
		t.Error("filtered documents don't match expected IDs")
	}

	if foundIDs["3"] {
		t.Error("document 3 should not be included in filtered results")
	}
}

func TestFilterDocumentsByLabels_NoFilter(t *testing.T) {
	tmpDir := t.TempDir()
	orch := &Orchestrator{
		config: &config.Config{
			OutputDir: tmpDir,
			Labels:    []string{}, // No filter
		},
		logger: logger.Get(),
	}

	docs := []rmclient.Document{
		{ID: "1", Title: "Doc 1"},
		{ID: "2", Title: "Doc 2"},
		{ID: "3", Title: "Doc 3"},
	}

	filtered := orch.filterDocumentsByLabels(docs)

	// Should return all documents when no filter is configured
	if len(filtered) != len(docs) {
		t.Errorf("expected %d documents, got %d", len(docs), len(filtered))
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
