package daemon

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
)

// ControlResponse is the standard response for control API calls
type ControlResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// syncControl manages manual sync triggering
type syncControl struct {
	mu            sync.Mutex
	manualTrigger chan struct{}
	cancelSync    context.CancelFunc
}

func newSyncControl() *syncControl {
	return &syncControl{
		manualTrigger: make(chan struct{}, 1), // Buffered to prevent blocking
	}
}

// triggerSync requests a manual sync
func (sc *syncControl) triggerSync() bool {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Non-blocking send - if already triggered, skip
	select {
	case sc.manualTrigger <- struct{}{}:
		return true
	default:
		return false // Already triggered
	}
}

// handleTriggerSync handles POST /api/sync/trigger
// Triggers an immediate sync without waiting for the next scheduled interval
func (d *Daemon) handleTriggerSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check current state
	status := d.statusTracker.GetStatus()
	if status.State == StateSyncing {
		respondJSON(w, http.StatusConflict, ControlResponse{
			Success: false,
			Message: "Sync already in progress",
		})
		return
	}

	// Note: In the current implementation, we run syncs immediately in runSync()
	// For a production system, you'd want to add a channel-based trigger mechanism
	// to allow manual syncs outside the ticker schedule
	respondJSON(w, http.StatusAccepted, ControlResponse{
		Success: true,
		Message: "Manual sync trigger not yet implemented - syncs run on schedule only",
	})
}

// handleCancelSync handles POST /api/sync/cancel
// Attempts to cancel an in-progress sync operation
func (d *Daemon) handleCancelSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check current state
	status := d.statusTracker.GetStatus()
	if status.State != StateSyncing {
		respondJSON(w, http.StatusConflict, ControlResponse{
			Success: false,
			Message: "No sync in progress to cancel",
		})
		return
	}

	// Note: Cancellation requires passing context with cancel function through the sync
	// For now, return not implemented
	respondJSON(w, http.StatusNotImplemented, ControlResponse{
		Success: false,
		Message: "Sync cancellation not yet implemented",
	})
}

// respondJSON writes a JSON response
func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Log error but can't change response at this point
		return
	}
}
