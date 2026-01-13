//go:build darwin
// +build darwin

package menubar

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDaemonClient_GetStatus_Success(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/status" {
			t.Errorf("Expected path /status, got %s", r.URL.Path)
		}

		status := Status{
			State:         StateIdle,
			UptimeSeconds: 3600,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	}))
	defer server.Close()

	client := NewDaemonClient(server.URL)
	ctx := context.Background()

	status, err := client.GetStatus(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if status.State != StateIdle {
		t.Errorf("Expected idle state, got %s", status.State)
	}

	if status.UptimeSeconds != 3600 {
		t.Errorf("Expected uptime 3600, got %d", status.UptimeSeconds)
	}
}

func TestDaemonClient_GetStatus_Offline(t *testing.T) {
	// Use a URL that will fail to connect
	client := NewDaemonClient("http://localhost:9999")
	ctx := context.Background()

	status, err := client.GetStatus(ctx)
	if err != nil {
		t.Fatalf("Expected no error (returns offline status), got %v", err)
	}

	if status.State != StateOffline {
		t.Errorf("Expected offline state, got %s", status.State)
	}
}

func TestDaemonClient_GetStatus_WithSyncProgress(t *testing.T) {
	now := time.Now()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status := Status{
			State: StateSyncing,
			CurrentSync: &SyncProgress{
				StartTime:          now,
				DocumentsTotal:     100,
				DocumentsProcessed: 42,
				CurrentDocument:    "Test.rmdoc",
				Stage:              "ocr",
			},
			UptimeSeconds: 1200,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	}))
	defer server.Close()

	client := NewDaemonClient(server.URL)
	ctx := context.Background()

	status, err := client.GetStatus(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if status.State != StateSyncing {
		t.Errorf("Expected syncing state, got %s", status.State)
	}

	if status.CurrentSync == nil {
		t.Fatal("Expected CurrentSync to be set")
	}

	if status.CurrentSync.DocumentsProcessed != 42 {
		t.Errorf("Expected 42 processed, got %d", status.CurrentSync.DocumentsProcessed)
	}

	if status.CurrentSync.CurrentDocument != "Test.rmdoc" {
		t.Errorf("Expected Test.rmdoc, got %s", status.CurrentSync.CurrentDocument)
	}
}

func TestDaemonClient_TriggerSync_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		if r.URL.Path != "/api/sync/trigger" {
			t.Errorf("Expected /api/sync/trigger, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Sync triggered",
		})
	}))
	defer server.Close()

	client := NewDaemonClient(server.URL)
	ctx := context.Background()

	err := client.TriggerSync(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestDaemonClient_TriggerSync_Conflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	defer server.Close()

	client := NewDaemonClient(server.URL)
	ctx := context.Background()

	err := client.TriggerSync(ctx)
	if err == nil {
		t.Fatal("Expected error for conflict, got nil")
	}

	if err.Error() != "sync already in progress" {
		t.Errorf("Expected 'sync already in progress', got %s", err.Error())
	}
}

func TestDaemonClient_IsHealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("Expected /health, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK\n"))
	}))
	defer server.Close()

	client := NewDaemonClient(server.URL)
	ctx := context.Background()

	healthy := client.IsHealthy(ctx)
	if !healthy {
		t.Error("Expected daemon to be healthy")
	}
}

func TestDaemonClient_IsHealthy_Offline(t *testing.T) {
	client := NewDaemonClient("http://localhost:9999")
	ctx := context.Background()

	healthy := client.IsHealthy(ctx)
	if healthy {
		t.Error("Expected daemon to be unhealthy")
	}
}
