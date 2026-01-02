// Package daemon provides long-running background sync functionality.
package daemon

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/platinummonkey/legible/internal/logger"
	"github.com/platinummonkey/legible/internal/sync"
)

// Daemon manages periodic synchronization in the background
type Daemon struct {
	orchestrator *sync.Orchestrator
	logger       *logger.Logger
	interval     time.Duration
	healthAddr   string
	pidFile      string
	httpServer   *http.Server
}

// Config holds configuration for the daemon
type Config struct {
	Orchestrator    *sync.Orchestrator
	Logger          *logger.Logger
	SyncInterval    time.Duration // How often to sync (default: 5 minutes)
	HealthCheckAddr string        // Optional health check address (e.g. ":8080")
	PIDFile         string        // Optional PID file path
}

// New creates a new daemon instance
func New(cfg *Config) (*Daemon, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if cfg.Orchestrator == nil {
		return nil, fmt.Errorf("orchestrator is required")
	}

	// Use provided logger or get default
	log := cfg.Logger
	if log == nil {
		log = logger.Get()
	}

	// Default sync interval
	interval := cfg.SyncInterval
	if interval == 0 {
		interval = 5 * time.Minute
	}

	return &Daemon{
		orchestrator: cfg.Orchestrator,
		logger:       log,
		interval:     interval,
		healthAddr:   cfg.HealthCheckAddr,
		pidFile:      cfg.PIDFile,
	}, nil
}

// Run starts the daemon and blocks until shutdown signal received
func (d *Daemon) Run(ctx context.Context) error {
	d.logger.WithFields("interval", d.interval).Info("Starting daemon")

	// Write PID file if configured
	if d.pidFile != "" {
		if err := d.writePIDFile(); err != nil {
			return fmt.Errorf("failed to write PID file: %w", err)
		}
		defer d.removePIDFile()
	}

	// Start health check server if configured
	if d.healthAddr != "" {
		if err := d.startHealthCheck(); err != nil {
			return fmt.Errorf("failed to start health check: %w", err)
		}
		defer d.stopHealthCheck()
	}

	// Create context that can be canceled
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	// Create ticker for periodic syncs
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	// Run initial sync immediately
	d.logger.Info("Running initial sync")
	d.runSync(ctx)

	// Main loop
	for {
		select {
		case <-ctx.Done():
			d.logger.Info("Context canceled, shutting down")
			return ctx.Err()

		case sig := <-sigChan:
			d.logger.WithFields("signal", sig.String()).Info("Received shutdown signal")
			// Allow current sync to complete by returning gracefully
			return nil

		case <-ticker.C:
			d.logger.Info("Sync interval elapsed, triggering sync")
			d.runSync(ctx)
		}
	}
}

// runSync executes a single sync operation with error recovery
func (d *Daemon) runSync(ctx context.Context) {
	d.logger.Info("Starting sync")
	startTime := time.Now()

	// Run sync with timeout context
	syncCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	result, err := d.orchestrator.Sync(syncCtx)
	duration := time.Since(startTime)

	if err != nil {
		d.logger.WithFields("error", err, "duration", duration).
			Error("Sync failed")
		return
	}

	// Log summary
	d.logger.WithFields(
		"total", result.TotalDocuments,
		"processed", result.ProcessedDocuments,
		"successful", result.SuccessCount,
		"failed", result.FailureCount,
		"duration", duration,
	).Info("Sync completed")

	// Log failures if any
	if result.HasFailures() {
		d.logger.WithFields("count", result.FailureCount).
			Warn("Sync completed with failures")
		for _, failure := range result.Failures {
			d.logger.WithFields(
				"document_id", failure.DocumentID,
				"title", failure.Title,
				"error", failure.Error,
			).Warn("Document sync failed")
		}
	}
}

// writePIDFile writes the current process ID to the configured PID file
func (d *Daemon) writePIDFile() error {
	pid := os.Getpid()
	content := fmt.Sprintf("%d\n", pid)

	if err := os.WriteFile(d.pidFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	d.logger.WithFields("pid", pid, "file", d.pidFile).Info("Wrote PID file")
	return nil
}

// removePIDFile removes the PID file
func (d *Daemon) removePIDFile() {
	if d.pidFile == "" {
		return
	}

	if err := os.Remove(d.pidFile); err != nil {
		d.logger.WithFields("file", d.pidFile, "error", err).
			Warn("Failed to remove PID file")
	} else {
		d.logger.WithFields("file", d.pidFile).Info("Removed PID file")
	}
}

// startHealthCheck starts the health check HTTP server
func (d *Daemon) startHealthCheck() error {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK\n"))
	})

	// Ready check endpoint (same as health for now)
	mux.HandleFunc("/ready", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK\n"))
	})

	d.httpServer = &http.Server{
		Addr:    d.healthAddr,
		Handler: mux,
	}

	// Start server in background
	go func() {
		d.logger.WithFields("addr", d.healthAddr).Info("Starting health check server")
		if err := d.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			d.logger.WithFields("error", err).Error("Health check server failed")
		}
	}()

	return nil
}

// stopHealthCheck stops the health check HTTP server
func (d *Daemon) stopHealthCheck() {
	if d.httpServer == nil {
		return
	}

	d.logger.Info("Stopping health check server")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := d.httpServer.Shutdown(ctx); err != nil {
		d.logger.WithFields("error", err).Warn("Failed to shutdown health check server gracefully")
	} else {
		d.logger.Info("Health check server stopped")
	}
}
