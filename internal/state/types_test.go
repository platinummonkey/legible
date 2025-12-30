package state

import (
	"testing"
	"time"
)

func TestNewSyncState(t *testing.T) {
	state := NewSyncState()

	if state == nil {
		t.Fatal("NewSyncState() returned nil")
	}

	if state.Documents == nil {
		t.Error("Documents map should be initialized")
	}

	if state.Version != StateFileVersion {
		t.Errorf("expected version %d, got %d", StateFileVersion, state.Version)
	}
}

func TestNewDocumentState(t *testing.T) {
	doc := NewDocumentState("doc-123", "Test Doc", "DocumentType", "parent-456")

	if doc.ID != "doc-123" {
		t.Errorf("expected ID doc-123, got %s", doc.ID)
	}

	if doc.Name != "Test Doc" {
		t.Errorf("expected name 'Test Doc', got %s", doc.Name)
	}

	if doc.ConversionStatus != ConversionStatusPending {
		t.Errorf("expected status pending, got %s", doc.ConversionStatus)
	}

	if doc.Labels == nil {
		t.Error("Labels should be initialized")
	}
}

func TestDocumentState_NeedsSync(t *testing.T) {
	tests := []struct {
		name           string
		doc            *DocumentState
		remoteVersion  int
		remoteModified time.Time
		want           bool
	}{
		{
			name:           "never synced",
			doc:            &DocumentState{},
			remoteVersion:  1,
			remoteModified: time.Now(),
			want:           true,
		},
		{
			name: "remote version newer",
			doc: &DocumentState{
				Version:        1,
				LastSynced:     time.Now().Add(-1 * time.Hour),
				ModifiedClient: time.Now().Add(-2 * time.Hour),
			},
			remoteVersion:  2,
			remoteModified: time.Now().Add(-2 * time.Hour),
			want:           true,
		},
		{
			name: "remote modification newer",
			doc: &DocumentState{
				Version:        1,
				LastSynced:     time.Now().Add(-1 * time.Hour),
				ModifiedClient: time.Now().Add(-2 * time.Hour),
			},
			remoteVersion:  1,
			remoteModified: time.Now().Add(-30 * time.Minute),
			want:           true,
		},
		{
			name: "up to date",
			doc: &DocumentState{
				Version:        2,
				LastSynced:     time.Now().Add(-1 * time.Hour),
				ModifiedClient: time.Now().Add(-2 * time.Hour),
			},
			remoteVersion:  2,
			remoteModified: time.Now().Add(-2 * time.Hour),
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.doc.NeedsSync(tt.remoteVersion, tt.remoteModified)
			if got != tt.want {
				t.Errorf("NeedsSync() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDocumentState_NeedsOCR(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name string
		doc  *DocumentState
		want bool
	}{
		{
			name: "not processed and conversion complete",
			doc: &DocumentState{
				OCRProcessed:     false,
				ConversionStatus: ConversionStatusCompleted,
			},
			want: true,
		},
		{
			name: "already processed recently",
			doc: &DocumentState{
				OCRProcessed:     true,
				ConversionStatus: ConversionStatusCompleted,
				OCRTimestamp:     now,
				LastSynced:       now.Add(-1 * time.Hour),
			},
			want: false,
		},
		{
			name: "processed but document changed",
			doc: &DocumentState{
				OCRProcessed:     true,
				ConversionStatus: ConversionStatusCompleted,
				OCRTimestamp:     now.Add(-2 * time.Hour),
				LastSynced:       now.Add(-1 * time.Hour),
			},
			want: true,
		},
		{
			name: "conversion not complete",
			doc: &DocumentState{
				OCRProcessed:     false,
				ConversionStatus: ConversionStatusPending,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.doc.NeedsOCR()
			if got != tt.want {
				t.Errorf("NeedsOCR() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDocumentState_MarkSynced(t *testing.T) {
	doc := NewDocumentState("doc-123", "Test", "DocumentType", "")
	doc.Error = "previous error"
	doc.RetryCount = 3

	modifiedTime := time.Now().Add(-1 * time.Hour)
	doc.MarkSynced(5, modifiedTime, "/path/to/doc", "hash123")

	if doc.Version != 5 {
		t.Errorf("expected version 5, got %d", doc.Version)
	}

	if doc.ModifiedClient != modifiedTime {
		t.Error("ModifiedClient not set correctly")
	}

	if doc.LocalPath != "/path/to/doc" {
		t.Errorf("expected LocalPath /path/to/doc, got %s", doc.LocalPath)
	}

	if doc.Hash != "hash123" {
		t.Errorf("expected hash hash123, got %s", doc.Hash)
	}

	if doc.Error != "" {
		t.Errorf("expected error cleared, got %s", doc.Error)
	}

	if doc.RetryCount != 0 {
		t.Errorf("expected retry count 0, got %d", doc.RetryCount)
	}
}

func TestSyncState_AddDocument(t *testing.T) {
	state := NewSyncState()
	doc := NewDocumentState("doc-123", "Test", "DocumentType", "")

	state.AddDocument(doc)

	retrieved := state.GetDocument("doc-123")
	if retrieved == nil {
		t.Fatal("document not found after adding")
	}

	if retrieved.ID != "doc-123" {
		t.Errorf("expected ID doc-123, got %s", retrieved.ID)
	}
}

func TestSyncState_RemoveDocument(t *testing.T) {
	state := NewSyncState()
	doc := NewDocumentState("doc-123", "Test", "DocumentType", "")

	state.AddDocument(doc)
	state.RemoveDocument("doc-123")

	retrieved := state.GetDocument("doc-123")
	if retrieved != nil {
		t.Error("document should be removed")
	}
}

func TestSyncState_GetDocumentsByLabel(t *testing.T) {
	state := NewSyncState()

	doc1 := NewDocumentState("doc-1", "Doc 1", "DocumentType", "")
	doc1.Labels = []string{"work", "important"}

	doc2 := NewDocumentState("doc-2", "Doc 2", "DocumentType", "")
	doc2.Labels = []string{"personal"}

	doc3 := NewDocumentState("doc-3", "Doc 3", "DocumentType", "")
	doc3.Labels = []string{"work"}

	state.AddDocument(doc1)
	state.AddDocument(doc2)
	state.AddDocument(doc3)

	workDocs := state.GetDocumentsByLabel("work")
	if len(workDocs) != 2 {
		t.Errorf("expected 2 work documents, got %d", len(workDocs))
	}

	personalDocs := state.GetDocumentsByLabel("personal")
	if len(personalDocs) != 1 {
		t.Errorf("expected 1 personal document, got %d", len(personalDocs))
	}
}

func TestSyncState_GetDocumentsByStatus(t *testing.T) {
	state := NewSyncState()

	doc1 := NewDocumentState("doc-1", "Doc 1", "DocumentType", "")
	doc1.ConversionStatus = ConversionStatusCompleted

	doc2 := NewDocumentState("doc-2", "Doc 2", "DocumentType", "")
	doc2.ConversionStatus = ConversionStatusPending

	doc3 := NewDocumentState("doc-3", "Doc 3", "DocumentType", "")
	doc3.ConversionStatus = ConversionStatusCompleted

	state.AddDocument(doc1)
	state.AddDocument(doc2)
	state.AddDocument(doc3)

	completedDocs := state.GetDocumentsByStatus(ConversionStatusCompleted)
	if len(completedDocs) != 2 {
		t.Errorf("expected 2 completed documents, got %d", len(completedDocs))
	}

	pendingDocs := state.GetDocumentsByStatus(ConversionStatusPending)
	if len(pendingDocs) != 1 {
		t.Errorf("expected 1 pending document, got %d", len(pendingDocs))
	}
}

func TestSyncState_GetDocumentsNeedingOCR(t *testing.T) {
	state := NewSyncState()

	doc1 := NewDocumentState("doc-1", "Doc 1", "DocumentType", "")
	doc1.ConversionStatus = ConversionStatusCompleted
	doc1.OCRProcessed = false

	doc2 := NewDocumentState("doc-2", "Doc 2", "DocumentType", "")
	doc2.ConversionStatus = ConversionStatusPending
	doc2.OCRProcessed = false

	doc3 := NewDocumentState("doc-3", "Doc 3", "DocumentType", "")
	doc3.ConversionStatus = ConversionStatusCompleted
	doc3.OCRProcessed = true
	doc3.OCRTimestamp = time.Now()
	doc3.LastSynced = time.Now().Add(-1 * time.Hour)

	state.AddDocument(doc1)
	state.AddDocument(doc2)
	state.AddDocument(doc3)

	needingOCR := state.GetDocumentsNeedingOCR()
	if len(needingOCR) != 1 {
		t.Errorf("expected 1 document needing OCR, got %d", len(needingOCR))
	}

	if len(needingOCR) > 0 && needingOCR[0].ID != "doc-1" {
		t.Errorf("expected doc-1 to need OCR, got %s", needingOCR[0].ID)
	}
}
