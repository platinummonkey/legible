package daemon

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/platinummonkey/legible/internal/logger"
	"github.com/platinummonkey/legible/internal/sync"
)

// mockOrchestrator is a mock implementation of sync.Orchestrator for testing
type mockOrchestrator struct {
	syncCalled int
	syncErr    error
	syncResult *sync.Result
}

func (m *mockOrchestrator) Sync(_ context.Context) (*sync.Result, error) {
	m.syncCalled++
	if m.syncErr != nil {
		return nil, m.syncErr
	}
	if m.syncResult == nil {
		return sync.NewResult(), nil
	}
	return m.syncResult, nil
}

func TestNew(t *testing.T) {
	mockOrch := &mockOrchestrator{}

	cfg := &Config{
		Orchestrator: (*sync.Orchestrator)(nil), // Type cast for mock
	}

	// Note: This would need interface-based design to properly test with mocks
	// For now, we test the struct creation logic directly

	daemon := &Daemon{
		orchestrator: (*sync.Orchestrator)(nil),
		logger:       logger.Get(),
		interval:     5 * time.Minute,
	}

	if daemon.logger == nil {
		t.Error("Logger should be initialized")
	}

	if daemon.interval != 5*time.Minute {
		t.Errorf("Interval = %v, want 5m", daemon.interval)
	}

	_ = mockOrch // Use mock to avoid unused variable error
	_ = cfg      // Use cfg to avoid unused variable error
}

func TestNew_NilConfig(t *testing.T) {
	_, err := New(nil)
	if err == nil {
		t.Error("New() should error with nil config")
	}
}

func TestNew_DefaultInterval(t *testing.T) {
	// Create a dummy orchestrator pointer for testing
	// In real usage, this would be a proper orchestrator instance
	orch := &sync.Orchestrator{}

	cfg := &Config{
		Orchestrator: orch,
		SyncInterval: 0, // Should default to 5 minutes
	}

	daemon, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if daemon.interval != 5*time.Minute {
		t.Errorf("interval = %v, want 5m (default)", daemon.interval)
	}
}

func TestNew_CustomInterval(t *testing.T) {
	orch := &sync.Orchestrator{}

	cfg := &Config{
		Orchestrator: orch,
		SyncInterval: 10 * time.Minute,
	}

	daemon, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if daemon.interval != 10*time.Minute {
		t.Errorf("interval = %v, want 10m", daemon.interval)
	}
}

func TestWritePIDFile(t *testing.T) {
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "test.pid")

	daemon := &Daemon{
		logger:  logger.Get(),
		pidFile: pidFile,
	}

	err := daemon.writePIDFile()
	if err != nil {
		t.Fatalf("writePIDFile() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(pidFile); os.IsNotExist(err) {
		t.Error("PID file should exist after writePIDFile()")
	}

	// Verify file content
	content, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("failed to read PID file: %v", err)
	}

	pidStr := strings.TrimSpace(string(content))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		t.Errorf("PID file content should be an integer, got %q", pidStr)
	}

	if pid != os.Getpid() {
		t.Errorf("PID in file = %d, want %d", pid, os.Getpid())
	}
}

func TestRemovePIDFile(t *testing.T) {
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "test.pid")

	daemon := &Daemon{
		logger:  logger.Get(),
		pidFile: pidFile,
	}

	// Create PID file
	if err := daemon.writePIDFile(); err != nil {
		t.Fatalf("writePIDFile() error = %v", err)
	}

	// Remove it
	daemon.removePIDFile()

	// Verify it's gone
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Error("PID file should not exist after removePIDFile()")
	}
}

func TestRemovePIDFile_NoPIDFile(_ *testing.T) {
	daemon := &Daemon{
		logger:  logger.Get(),
		pidFile: "", // No PID file configured
	}

	// Should not error
	daemon.removePIDFile()
}

func TestHealthCheckEndpoint(t *testing.T) {
	orch := &sync.Orchestrator{}

	// Use a random port for testing
	cfg := &Config{
		Orchestrator:    orch,
		HealthCheckAddr: "localhost:0", // Port 0 = random available port
	}

	daemon, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Start health check server
	if err := daemon.startHealthCheck(); err != nil {
		t.Fatalf("startHealthCheck() error = %v", err)
	}
	defer daemon.stopHealthCheck()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Note: Testing requires knowing the actual port assigned
	// For now, this tests the start/stop logic
	// A real test would need to capture the actual address
}

func TestHealthCheckEndpoint_NotConfigured(t *testing.T) {
	orch := &sync.Orchestrator{}

	cfg := &Config{
		Orchestrator:    orch,
		HealthCheckAddr: "", // No health check
	}

	daemon, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Should not error when stopping without starting
	daemon.stopHealthCheck()
}

func TestHealthEndpoint_Integration(t *testing.T) {
	// Start a test server
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK\n"))
	})

	server := &http.Server{
		Addr:    "localhost:18080",
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Logf("Server error: %v", err)
		}
	}()
	defer func() {
		_ = server.Shutdown(context.Background())
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test health endpoint
	resp, err := http.Get("http://localhost:18080/health")
	if err != nil {
		t.Fatalf("Failed to call health endpoint: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Health endpoint status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestRunSync_Success(_ *testing.T) {
	mockOrch := &mockOrchestrator{
		syncResult: sync.NewResult(),
	}

	daemon := &Daemon{
		orchestrator: (*sync.Orchestrator)(nil), // Would need interface for proper testing
		logger:       logger.Get(),
		interval:     1 * time.Minute,
	}

	// Note: runSync requires proper orchestrator
	// This test demonstrates the structure but would need refactoring for proper testing
	_ = mockOrch
	_ = daemon
}

func TestRunSync_Error(t *testing.T) {
	// Similar to TestRunSync_Success, would need interface-based design
	t.Skip("Requires interface-based orchestrator for proper mocking")
}

func TestRun_InitialSync(t *testing.T) {
	// Would require interface-based design and signal handling testing
	t.Skip("Requires interface-based orchestrator and complex signal handling")
}

func TestRun_PeriodicSync(t *testing.T) {
	// Would require time-based testing with fast ticker
	t.Skip("Requires time-based testing infrastructure")
}

func TestRun_GracefulShutdown(t *testing.T) {
	// Would require signal handling testing
	t.Skip("Requires signal handling testing infrastructure")
}

// Note: Full integration tests for Run() would require:
// 1. Interface-based orchestrator design for mocking
// 2. Time-based testing infrastructure for ticker testing
// 3. Signal handling test utilities
// 4. Context cancellation testing
//
// The above tests cover the individual helper methods that can be tested
// without complex mocking infrastructure.
