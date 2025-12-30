package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	manager := NewManager("/tmp/test-state.json")

	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}

	if manager.filePath != "/tmp/test-state.json" {
		t.Errorf("expected filePath /tmp/test-state.json, got %s", manager.filePath)
	}

	if manager.state == nil {
		t.Error("state should be initialized")
	}
}

func TestManager_Load_FileDoesNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "state.json")

	manager := NewManager(filePath)
	err := manager.Load()

	if err != nil {
		t.Errorf("Load() should not error when file doesn't exist, got: %v", err)
	}

	if len(manager.state.Documents) != 0 {
		t.Error("state should be empty when file doesn't exist")
	}
}

func TestManager_Load_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "state.json")

	// Create a valid state file
	state := NewSyncState()
	doc := NewDocumentState("doc-123", "Test Doc", "DocumentType", "")
	state.AddDocument(doc)

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal test state: %v", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("failed to write test state file: %v", err)
	}

	// Load it
	manager := NewManager(filePath)
	if err := manager.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify
	if len(manager.state.Documents) != 1 {
		t.Errorf("expected 1 document, got %d", len(manager.state.Documents))
	}

	loadedDoc := manager.GetDocument("doc-123")
	if loadedDoc == nil {
		t.Fatal("document not found after load")
	}

	if loadedDoc.Name != "Test Doc" {
		t.Errorf("expected name 'Test Doc', got %s", loadedDoc.Name)
	}
}

func TestManager_Load_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "state.json")

	// Write invalid JSON
	if err := os.WriteFile(filePath, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	manager := NewManager(filePath)
	err := manager.Load()

	if err == nil {
		t.Error("Load() should error on invalid JSON")
	}
}

func TestManager_Load_UnsupportedVersion(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "state.json")

	// Create state with unsupported version
	state := NewSyncState()
	state.Version = 999

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal test state: %v", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("failed to write test state file: %v", err)
	}

	manager := NewManager(filePath)
	err = manager.Load()

	if err == nil {
		t.Error("Load() should error on unsupported version")
	}
}

func TestManager_Save(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "state.json")

	manager := NewManager(filePath)
	doc := NewDocumentState("doc-123", "Test Doc", "DocumentType", "")
	manager.AddDocument(doc)

	// Save
	if err := manager.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("state file should exist after Save()")
	}

	// Load it back
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}

	var loadedState SyncState
	if err := json.Unmarshal(data, &loadedState); err != nil {
		t.Fatalf("failed to unmarshal saved state: %v", err)
	}

	if len(loadedState.Documents) != 1 {
		t.Errorf("expected 1 document in saved state, got %d", len(loadedState.Documents))
	}
}

func TestManager_Save_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "subdir", "state.json")

	manager := NewManager(filePath)
	doc := NewDocumentState("doc-123", "Test Doc", "DocumentType", "")
	manager.AddDocument(doc)

	// Save (should create directory)
	if err := manager.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("state file should exist after Save()")
	}
}

func TestManager_Save_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "state.json")

	manager := NewManager(filePath)
	doc := NewDocumentState("doc-123", "Test Doc", "DocumentType", "")
	manager.AddDocument(doc)

	// Save
	if err := manager.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify temp file was cleaned up
	tmpFile := filePath + ".tmp"
	if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
		t.Error("temporary file should be cleaned up after Save()")
	}
}

func TestManager_GetDocument(t *testing.T) {
	manager := NewManager("/tmp/test.json")

	doc := NewDocumentState("doc-123", "Test Doc", "DocumentType", "")
	manager.AddDocument(doc)

	retrieved := manager.GetDocument("doc-123")
	if retrieved == nil {
		t.Fatal("GetDocument() returned nil")
	}

	if retrieved.ID != "doc-123" {
		t.Errorf("expected ID doc-123, got %s", retrieved.ID)
	}

	notFound := manager.GetDocument("nonexistent")
	if notFound != nil {
		t.Error("GetDocument() should return nil for nonexistent document")
	}
}

func TestManager_AddDocument(t *testing.T) {
	manager := NewManager("/tmp/test.json")

	doc := NewDocumentState("doc-123", "Test Doc", "DocumentType", "")
	manager.AddDocument(doc)

	if manager.Count() != 1 {
		t.Errorf("expected 1 document, got %d", manager.Count())
	}

	// Update existing document
	doc.Name = "Updated Name"
	manager.AddDocument(doc)

	if manager.Count() != 1 {
		t.Errorf("expected 1 document after update, got %d", manager.Count())
	}

	retrieved := manager.GetDocument("doc-123")
	if retrieved.Name != "Updated Name" {
		t.Errorf("expected updated name, got %s", retrieved.Name)
	}
}

func TestManager_RemoveDocument(t *testing.T) {
	manager := NewManager("/tmp/test.json")

	doc := NewDocumentState("doc-123", "Test Doc", "DocumentType", "")
	manager.AddDocument(doc)

	manager.RemoveDocument("doc-123")

	if manager.Count() != 0 {
		t.Errorf("expected 0 documents after removal, got %d", manager.Count())
	}

	retrieved := manager.GetDocument("doc-123")
	if retrieved != nil {
		t.Error("document should be nil after removal")
	}
}

func TestManager_UpdateLastSync(t *testing.T) {
	manager := NewManager("/tmp/test.json")

	before := manager.state.LastSync
	time.Sleep(10 * time.Millisecond)

	manager.UpdateLastSync()

	after := manager.state.LastSync
	if !after.After(before) {
		t.Error("LastSync should be updated to current time")
	}
}

func TestManager_GetDocumentsByLabel(t *testing.T) {
	manager := NewManager("/tmp/test.json")

	doc1 := NewDocumentState("doc-1", "Doc 1", "DocumentType", "")
	doc1.Labels = []string{"work"}
	manager.AddDocument(doc1)

	doc2 := NewDocumentState("doc-2", "Doc 2", "DocumentType", "")
	doc2.Labels = []string{"personal"}
	manager.AddDocument(doc2)

	workDocs := manager.GetDocumentsByLabel("work")
	if len(workDocs) != 1 {
		t.Errorf("expected 1 work document, got %d", len(workDocs))
	}

	personalDocs := manager.GetDocumentsByLabel("personal")
	if len(personalDocs) != 1 {
		t.Errorf("expected 1 personal document, got %d", len(personalDocs))
	}
}

func TestManager_GetDocumentsByStatus(t *testing.T) {
	manager := NewManager("/tmp/test.json")

	doc1 := NewDocumentState("doc-1", "Doc 1", "DocumentType", "")
	doc1.ConversionStatus = ConversionStatusCompleted
	manager.AddDocument(doc1)

	doc2 := NewDocumentState("doc-2", "Doc 2", "DocumentType", "")
	doc2.ConversionStatus = ConversionStatusPending
	manager.AddDocument(doc2)

	completed := manager.GetDocumentsByStatus(ConversionStatusCompleted)
	if len(completed) != 1 {
		t.Errorf("expected 1 completed document, got %d", len(completed))
	}

	pending := manager.GetDocumentsByStatus(ConversionStatusPending)
	if len(pending) != 1 {
		t.Errorf("expected 1 pending document, got %d", len(pending))
	}
}

func TestManager_GetDocumentsNeedingOCR(t *testing.T) {
	manager := NewManager("/tmp/test.json")

	doc1 := NewDocumentState("doc-1", "Doc 1", "DocumentType", "")
	doc1.ConversionStatus = ConversionStatusCompleted
	doc1.OCRProcessed = false
	manager.AddDocument(doc1)

	doc2 := NewDocumentState("doc-2", "Doc 2", "DocumentType", "")
	doc2.ConversionStatus = ConversionStatusPending
	manager.AddDocument(doc2)

	needingOCR := manager.GetDocumentsNeedingOCR()
	if len(needingOCR) != 1 {
		t.Errorf("expected 1 document needing OCR, got %d", len(needingOCR))
	}

	if len(needingOCR) > 0 && needingOCR[0].ID != "doc-1" {
		t.Errorf("expected doc-1 to need OCR, got %s", needingOCR[0].ID)
	}
}

func TestLoadOrCreate_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "state.json")

	manager, err := LoadOrCreate(filePath)
	if err != nil {
		t.Fatalf("LoadOrCreate() error = %v", err)
	}

	if manager == nil {
		t.Fatal("LoadOrCreate() returned nil manager")
	}

	// Verify file was created
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("state file should be created")
	}

	if manager.Count() != 0 {
		t.Errorf("expected 0 documents in new state, got %d", manager.Count())
	}
}

func TestLoadOrCreate_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "state.json")

	// Create existing state
	state := NewSyncState()
	doc := NewDocumentState("doc-123", "Test Doc", "DocumentType", "")
	state.AddDocument(doc)

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal test state: %v", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("failed to write test state file: %v", err)
	}

	// Load it
	manager, err := LoadOrCreate(filePath)
	if err != nil {
		t.Fatalf("LoadOrCreate() error = %v", err)
	}

	if manager.Count() != 1 {
		t.Errorf("expected 1 document, got %d", manager.Count())
	}

	retrieved := manager.GetDocument("doc-123")
	if retrieved == nil || retrieved.Name != "Test Doc" {
		t.Error("existing document should be loaded")
	}
}

func TestManager_Reset(t *testing.T) {
	manager := NewManager("/tmp/test.json")

	// Add some documents
	doc1 := NewDocumentState("doc-1", "Doc 1", "DocumentType", "")
	doc2 := NewDocumentState("doc-2", "Doc 2", "DocumentType", "")
	manager.AddDocument(doc1)
	manager.AddDocument(doc2)
	manager.UpdateLastSync()

	if manager.Count() != 2 {
		t.Errorf("expected 2 documents before reset, got %d", manager.Count())
	}

	// Reset
	manager.Reset()

	if manager.Count() != 0 {
		t.Errorf("expected 0 documents after reset, got %d", manager.Count())
	}

	if !manager.state.LastSync.IsZero() {
		t.Error("LastSync should be zero after reset")
	}
}

func TestManager_Count(t *testing.T) {
	manager := NewManager("/tmp/test.json")

	if manager.Count() != 0 {
		t.Errorf("expected 0 documents initially, got %d", manager.Count())
	}

	doc1 := NewDocumentState("doc-1", "Doc 1", "DocumentType", "")
	manager.AddDocument(doc1)

	if manager.Count() != 1 {
		t.Errorf("expected 1 document after add, got %d", manager.Count())
	}

	doc2 := NewDocumentState("doc-2", "Doc 2", "DocumentType", "")
	manager.AddDocument(doc2)

	if manager.Count() != 2 {
		t.Errorf("expected 2 documents after second add, got %d", manager.Count())
	}

	manager.RemoveDocument("doc-1")

	if manager.Count() != 1 {
		t.Errorf("expected 1 document after removal, got %d", manager.Count())
	}
}

func TestManager_ConcurrentAccess(t *testing.T) {
	manager := NewManager("/tmp/test.json")

	// Concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			doc := NewDocumentState(fmt.Sprintf("doc-%d", id), "Test", "DocumentType", "")
			manager.AddDocument(doc)
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < 10; i++ {
		<-done
	}

	if manager.Count() != 10 {
		t.Errorf("expected 10 documents after concurrent writes, got %d", manager.Count())
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func(id int) {
			_ = manager.GetDocument(fmt.Sprintf("doc-%d", id))
			done <- true
		}(i)
	}

	// Wait for all reads
	for i := 0; i < 10; i++ {
		<-done
	}
}
