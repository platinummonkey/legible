//go:build darwin
// +build darwin

package menubar

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// DaemonClient handles communication with the legible daemon HTTP API
type DaemonClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewDaemonClient creates a new daemon client
func NewDaemonClient(baseURL string) *DaemonClient {
	return &DaemonClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// SyncState represents the daemon's sync state
type SyncState string

const (
	StateIdle     SyncState = "idle"
	StateSyncing  SyncState = "syncing"
	StateError    SyncState = "error"
	StateOffline  SyncState = "offline" // Daemon not reachable
)

// Status represents the daemon status response
type Status struct {
	State          SyncState      `json:"state"`
	LastSyncTime   *time.Time     `json:"last_sync_time,omitempty"`
	NextSyncTime   *time.Time     `json:"next_sync_time,omitempty"`
	SyncDuration   *time.Duration `json:"sync_duration,omitempty"`
	ErrorMessage   string         `json:"error_message,omitempty"`
	CurrentSync    *SyncProgress  `json:"current_sync,omitempty"`
	LastSyncResult *SyncSummary   `json:"last_sync_result,omitempty"`
	UptimeSeconds  int64          `json:"uptime_seconds"`
}

// SyncProgress tracks the progress of an in-progress sync
type SyncProgress struct {
	StartTime          time.Time `json:"start_time"`
	DocumentsTotal     int       `json:"documents_total"`
	DocumentsProcessed int       `json:"documents_processed"`
	CurrentDocument    string    `json:"current_document,omitempty"`
	Stage              string    `json:"stage,omitempty"`
}

// SyncSummary contains a summary of a completed sync
type SyncSummary struct {
	StartTime          time.Time     `json:"start_time"`
	EndTime            time.Time     `json:"end_time"`
	Duration           time.Duration `json:"duration"`
	TotalDocuments     int           `json:"total_documents"`
	ProcessedDocuments int           `json:"processed_documents"`
	SuccessCount       int           `json:"success_count"`
	FailureCount       int           `json:"failure_count"`
	SkippedCount       int           `json:"skipped_count"`
}

// GetStatus retrieves the current daemon status
func (c *DaemonClient) GetStatus(ctx context.Context) (*Status, error) {
	url := fmt.Sprintf("%s/status", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Return offline status if daemon is not reachable
		return &Status{State: StateOffline}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var status Status
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &status, nil
}

// TriggerSync triggers a manual sync operation
func (c *DaemonClient) TriggerSync(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/sync/trigger", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to trigger sync: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		return fmt.Errorf("sync already in progress")
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// CancelSync attempts to cancel an in-progress sync
func (c *DaemonClient) CancelSync(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/sync/cancel", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to cancel sync: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		return fmt.Errorf("no sync in progress")
	}

	if resp.StatusCode == http.StatusNotImplemented {
		return fmt.Errorf("cancel not yet implemented")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// IsHealthy checks if the daemon is responding
func (c *DaemonClient) IsHealthy(ctx context.Context) bool {
	url := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
