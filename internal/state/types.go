package state

import "time"

// SyncState represents the overall synchronization state for the application
type SyncState struct {
	// LastSync is the timestamp of the last successful sync operation
	LastSync time.Time `json:"last_sync"`

	// Documents is a map of document ID to their sync state
	Documents map[string]*DocumentState `json:"documents"`

	// Version is the state file format version
	Version int `json:"version"`
}

// DocumentState represents the sync state for a single document
type DocumentState struct {
	// ID is the document's unique identifier
	ID string `json:"id"`

	// Name is the document's visible name
	Name string `json:"name"`

	// Version is the document version number from reMarkable API
	Version int `json:"version"`

	// ModifiedClient is when the document was last modified on the client
	ModifiedClient time.Time `json:"modified_client"`

	// LastSynced is when this document was last synced
	LastSynced time.Time `json:"last_synced"`

	// LocalPath is the path to the local copy of the document
	LocalPath string `json:"local_path"`

	// Hash is the SHA256 hash of the downloaded content for change detection
	Hash string `json:"hash"`

	// Type is the document type (DocumentType or CollectionType)
	Type string `json:"type"`

	// Parent is the ID of the parent collection
	Parent string `json:"parent"`

	// OCRProcessed indicates if OCR has been performed on this document
	OCRProcessed bool `json:"ocr_processed"`

	// OCRTimestamp is when OCR was last performed
	OCRTimestamp time.Time `json:"ocr_timestamp,omitempty"`

	// ConversionStatus is the status of PDF conversion
	ConversionStatus ConversionStatus `json:"conversion_status"`

	// Error contains any error message from the last sync attempt
	Error string `json:"error,omitempty"`

	// RetryCount is the number of sync retry attempts
	RetryCount int `json:"retry_count"`

	// Labels are the reMarkable labels/tags associated with this document
	Labels []string `json:"labels,omitempty"`
}

// ConversionStatus represents the status of document conversion
type ConversionStatus string

const (
	// ConversionStatusPending indicates conversion has not been attempted
	ConversionStatusPending ConversionStatus = "pending"

	// ConversionStatusInProgress indicates conversion is currently running
	ConversionStatusInProgress ConversionStatus = "in_progress"

	// ConversionStatusCompleted indicates conversion completed successfully
	ConversionStatusCompleted ConversionStatus = "completed"

	// ConversionStatusFailed indicates conversion failed
	ConversionStatusFailed ConversionStatus = "failed"

	// ConversionStatusSkipped indicates conversion was skipped (e.g., already a PDF)
	ConversionStatusSkipped ConversionStatus = "skipped"
)

// StateFileVersion is the current version of the state file format
const StateFileVersion = 1

// NewSyncState creates a new empty SyncState
func NewSyncState() *SyncState {
	return &SyncState{
		Documents: make(map[string]*DocumentState),
		Version:   StateFileVersion,
	}
}

// NewDocumentState creates a new DocumentState for a document
func NewDocumentState(id, name, docType, parent string) *DocumentState {
	return &DocumentState{
		ID:               id,
		Name:             name,
		Type:             docType,
		Parent:           parent,
		ConversionStatus: ConversionStatusPending,
		Labels:           []string{},
	}
}

// NeedsSync returns true if the document needs to be synced based on version or modification time
func (ds *DocumentState) NeedsSync(remoteVersion int, remoteModified time.Time) bool {
	// If we've never synced, we need to sync
	if ds.LastSynced.IsZero() {
		return true
	}

	// If remote version is newer, we need to sync
	if remoteVersion > ds.Version {
		return true
	}

	// If remote modification time is newer (with 1 second tolerance for rounding), we need to sync
	if remoteModified.Truncate(time.Second).After(ds.ModifiedClient.Truncate(time.Second)) {
		return true
	}

	return false
}

// NeedsOCR returns true if the document needs OCR processing
func (ds *DocumentState) NeedsOCR() bool {
	// OCR not needed if conversion isn't complete
	if ds.ConversionStatus != ConversionStatusCompleted {
		return false
	}

	// Need OCR if never processed
	if !ds.OCRProcessed {
		return true
	}

	// Need OCR if document was synced after last OCR processing
	if ds.LastSynced.After(ds.OCRTimestamp) {
		return true
	}

	return false
}

// MarkSynced updates the document state after a successful sync
func (ds *DocumentState) MarkSynced(version int, modifiedClient time.Time, localPath, hash string) {
	ds.Version = version
	ds.ModifiedClient = modifiedClient
	ds.LastSynced = time.Now()
	ds.LocalPath = localPath
	ds.Hash = hash
	ds.Error = ""
	ds.RetryCount = 0
}

// MarkError records an error during sync
func (ds *DocumentState) MarkError(err error) {
	ds.Error = err.Error()
	ds.RetryCount++
}

// MarkOCRComplete updates the document state after successful OCR
func (ds *DocumentState) MarkOCRComplete() {
	ds.OCRProcessed = true
	ds.OCRTimestamp = time.Now()
}

// SetConversionStatus updates the conversion status
func (ds *DocumentState) SetConversionStatus(status ConversionStatus) {
	ds.ConversionStatus = status
}

// GetDocument returns the DocumentState for a document ID, or nil if not found
func (ss *SyncState) GetDocument(id string) *DocumentState {
	return ss.Documents[id]
}

// AddDocument adds or updates a document in the sync state
func (ss *SyncState) AddDocument(doc *DocumentState) {
	ss.Documents[doc.ID] = doc
}

// RemoveDocument removes a document from the sync state
func (ss *SyncState) RemoveDocument(id string) {
	delete(ss.Documents, id)
}

// UpdateLastSync updates the last sync timestamp
func (ss *SyncState) UpdateLastSync() {
	ss.LastSync = time.Now()
}

// GetDocumentsByLabel returns all documents with the specified label
func (ss *SyncState) GetDocumentsByLabel(label string) []*DocumentState {
	var docs []*DocumentState
	for _, doc := range ss.Documents {
		for _, l := range doc.Labels {
			if l == label {
				docs = append(docs, doc)
				break
			}
		}
	}
	return docs
}

// GetDocumentsByStatus returns all documents with the specified conversion status
func (ss *SyncState) GetDocumentsByStatus(status ConversionStatus) []*DocumentState {
	var docs []*DocumentState
	for _, doc := range ss.Documents {
		if doc.ConversionStatus == status {
			docs = append(docs, doc)
		}
	}
	return docs
}

// GetDocumentsNeedingOCR returns all documents that need OCR processing
func (ss *SyncState) GetDocumentsNeedingOCR() []*DocumentState {
	var docs []*DocumentState
	for _, doc := range ss.Documents {
		if doc.NeedsOCR() {
			docs = append(docs, doc)
		}
	}
	return docs
}
