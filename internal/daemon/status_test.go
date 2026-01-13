package daemon

import (
	"encoding/json"
	"testing"
	"time"
)

func TestStatusTracker(t *testing.T) {
	st := NewStatusTracker()

	// Test initial state
	status := st.GetStatus()
	if status.State != StateIdle {
		t.Errorf("Expected initial state to be idle, got %s", status.State)
	}

	// Test sync started
	st.SyncStarted(100)
	status = st.GetStatus()
	if status.State != StateSyncing {
		t.Errorf("Expected state to be syncing, got %s", status.State)
	}
	if status.CurrentSync == nil {
		t.Fatal("Expected CurrentSync to be set")
	}
	if status.CurrentSync.DocumentsTotal != 100 {
		t.Errorf("Expected 100 total docs, got %d", status.CurrentSync.DocumentsTotal)
	}

	// Test progress update
	st.UpdateProgress(42, "Test.rmdoc", "ocr")
	status = st.GetStatus()
	if status.CurrentSync.DocumentsProcessed != 42 {
		t.Errorf("Expected 42 processed, got %d", status.CurrentSync.DocumentsProcessed)
	}
	if status.CurrentSync.CurrentDocument != "Test.rmdoc" {
		t.Errorf("Expected current doc to be Test.rmdoc, got %s", status.CurrentSync.CurrentDocument)
	}

	// Test sync completed
	summary := SyncSummary{
		StartTime:          time.Now().Add(-1 * time.Minute),
		EndTime:            time.Now(),
		Duration:           1 * time.Minute,
		TotalDocuments:     100,
		ProcessedDocuments: 50,
		SuccessCount:       48,
		FailureCount:       2,
		SkippedCount:       50,
	}
	st.SyncCompleted(summary)
	status = st.GetStatus()
	if status.State != StateIdle {
		t.Errorf("Expected state to be idle after completion, got %s", status.State)
	}
	if status.CurrentSync != nil {
		t.Error("Expected CurrentSync to be nil after completion")
	}
	if status.LastSyncResult == nil {
		t.Fatal("Expected LastSyncResult to be set")
	}
	if status.LastSyncResult.SuccessCount != 48 {
		t.Errorf("Expected 48 successes, got %d", status.LastSyncResult.SuccessCount)
	}
}

func TestStatusTrackerError(t *testing.T) {
	st := NewStatusTracker()

	// Start sync
	st.SyncStarted(10)

	// Fail sync
	st.SyncFailed(
		&testError{msg: "connection timeout"},
		30*time.Second,
	)

	status := st.GetStatus()
	if status.State != StateError {
		t.Errorf("Expected state to be error, got %s", status.State)
	}
	if status.ErrorMessage != "connection timeout" {
		t.Errorf("Expected error message, got %s", status.ErrorMessage)
	}
	if status.CurrentSync != nil {
		t.Error("Expected CurrentSync to be nil after failure")
	}
}

func TestStatusJSON(t *testing.T) {
	st := NewStatusTracker()

	now := time.Now()
	summary := SyncSummary{
		StartTime:          now.Add(-1 * time.Minute),
		EndTime:            now,
		Duration:           1 * time.Minute,
		TotalDocuments:     100,
		ProcessedDocuments: 50,
		SuccessCount:       50,
		FailureCount:       0,
		SkippedCount:       50,
	}
	st.SyncCompleted(summary)
	st.SetNextSyncTime(now.Add(5 * time.Minute))

	status := st.GetStatus()

	// Test JSON marshaling
	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal status: %v", err)
	}

	// Test JSON unmarshaling
	var decoded Status
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal status: %v", err)
	}

	if decoded.State != status.State {
		t.Errorf("State mismatch after JSON round-trip")
	}
}

func TestStatusTrackerConcurrency(t *testing.T) {
	st := NewStatusTracker()

	// Start sync
	st.SyncStarted(100)

	// Simulate concurrent access
	done := make(chan bool)
	go func() {
		for i := 0; i < 100; i++ {
			st.UpdateProgress(i, "doc.rmdoc", "ocr")
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = st.GetStatus()
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for goroutines
	<-done
	<-done

	// Should not panic and should have final state
	status := st.GetStatus()
	if status.State != StateSyncing {
		t.Errorf("Expected syncing state, got %s", status.State)
	}
}

// testError is a simple error implementation for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
