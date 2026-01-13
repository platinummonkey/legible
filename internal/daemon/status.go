package daemon

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// SyncState represents the current state of the daemon sync process
type SyncState string

const (
	// StateIdle indicates the daemon is running but not actively syncing
	StateIdle SyncState = "idle"

	// StateSyncing indicates an active sync operation is in progress
	StateSyncing SyncState = "syncing"

	// StateError indicates the last sync operation failed
	StateError SyncState = "error"
)

// Status represents the current daemon status
type Status struct {
	// State is the current sync state (idle, syncing, error)
	State SyncState `json:"state"`

	// LastSyncTime is the timestamp of the last sync attempt
	LastSyncTime *time.Time `json:"last_sync_time,omitempty"`

	// NextSyncTime is the estimated time of the next sync
	NextSyncTime *time.Time `json:"next_sync_time,omitempty"`

	// SyncDuration is how long the last sync took
	SyncDuration *time.Duration `json:"sync_duration,omitempty"`

	// ErrorMessage contains the error from the last failed sync
	ErrorMessage string `json:"error_message,omitempty"`

	// CurrentSync contains information about an in-progress sync
	CurrentSync *SyncProgress `json:"current_sync,omitempty"`

	// LastSyncResult contains the result of the last completed sync
	LastSyncResult *SyncSummary `json:"last_sync_result,omitempty"`

	// UptimeSeconds is how long the daemon has been running
	UptimeSeconds int64 `json:"uptime_seconds"`
}

// SyncProgress tracks the progress of an in-progress sync operation
type SyncProgress struct {
	// StartTime is when the current sync started
	StartTime time.Time `json:"start_time"`

	// DocumentsTotal is the total number of documents to process
	DocumentsTotal int `json:"documents_total"`

	// DocumentsProcessed is how many documents have been processed so far
	DocumentsProcessed int `json:"documents_processed"`

	// CurrentDocument is the document currently being processed
	CurrentDocument string `json:"current_document,omitempty"`

	// Stage is the current stage of processing (downloading, converting, ocr, enhancing)
	Stage string `json:"stage,omitempty"`
}

// SyncSummary contains a summary of a completed sync operation
type SyncSummary struct {
	// StartTime is when the sync started
	StartTime time.Time `json:"start_time"`

	// EndTime is when the sync completed
	EndTime time.Time `json:"end_time"`

	// Duration is how long the sync took
	Duration time.Duration `json:"duration"`

	// TotalDocuments is the total number of documents checked
	TotalDocuments int `json:"total_documents"`

	// ProcessedDocuments is how many documents were processed
	ProcessedDocuments int `json:"processed_documents"`

	// SuccessCount is how many documents succeeded
	SuccessCount int `json:"success_count"`

	// FailureCount is how many documents failed
	FailureCount int `json:"failure_count"`

	// SkippedCount is how many documents were skipped (no changes)
	SkippedCount int `json:"skipped_count"`
}

// StatusTracker tracks the daemon's current status in a thread-safe manner
type StatusTracker struct {
	mu         sync.RWMutex
	state      SyncState
	startTime  time.Time
	lastSync   *time.Time
	nextSync   *time.Time
	lastDur    *time.Duration
	errMsg     string
	curSync    *SyncProgress
	lastResult *SyncSummary
}

// NewStatusTracker creates a new status tracker
func NewStatusTracker() *StatusTracker {
	return &StatusTracker{
		state:     StateIdle,
		startTime: time.Now(),
	}
}

// GetStatus returns the current status
func (st *StatusTracker) GetStatus() Status {
	st.mu.RLock()
	defer st.mu.RUnlock()

	uptime := time.Since(st.startTime)

	return Status{
		State:          st.state,
		LastSyncTime:   st.lastSync,
		NextSyncTime:   st.nextSync,
		SyncDuration:   st.lastDur,
		ErrorMessage:   st.errMsg,
		CurrentSync:    st.curSync,
		LastSyncResult: st.lastResult,
		UptimeSeconds:  int64(uptime.Seconds()),
	}
}

// SetState updates the current state
func (st *StatusTracker) SetState(state SyncState) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.state = state
}

// SetError records an error state
func (st *StatusTracker) SetError(err error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.state = StateError
	if err != nil {
		st.errMsg = err.Error()
	}
}

// SyncStarted records the start of a sync operation
func (st *StatusTracker) SyncStarted(totalDocs int) {
	st.mu.Lock()
	defer st.mu.Unlock()

	now := time.Now()
	st.state = StateSyncing
	st.lastSync = &now
	st.errMsg = ""
	st.curSync = &SyncProgress{
		StartTime:      now,
		DocumentsTotal: totalDocs,
	}
}

// UpdateProgress updates the current sync progress
func (st *StatusTracker) UpdateProgress(processed int, currentDoc, stage string) {
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.curSync != nil {
		st.curSync.DocumentsProcessed = processed
		st.curSync.CurrentDocument = currentDoc
		st.curSync.Stage = stage
	}
}

// SyncCompleted records a successful sync completion
func (st *StatusTracker) SyncCompleted(summary SyncSummary) {
	st.mu.Lock()
	defer st.mu.Unlock()

	st.state = StateIdle
	st.curSync = nil
	st.lastResult = &summary
	st.errMsg = ""

	dur := summary.Duration
	st.lastDur = &dur
}

// SyncFailed records a failed sync
func (st *StatusTracker) SyncFailed(err error, duration time.Duration) {
	st.mu.Lock()
	defer st.mu.Unlock()

	st.state = StateError
	st.curSync = nil
	st.lastDur = &duration

	if err != nil {
		st.errMsg = err.Error()
	}
}

// SetNextSyncTime updates when the next sync is scheduled
func (st *StatusTracker) SetNextSyncTime(t time.Time) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.nextSync = &t
}

// handleStatus serves the current status as JSON
func (d *Daemon) handleStatus(w http.ResponseWriter, _ *http.Request) {
	status := d.statusTracker.GetStatus()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		d.logger.WithError(err).Error("Failed to encode status")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
