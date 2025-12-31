package main

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/platinummonkey/remarkable-sync/internal/converter"
	"github.com/platinummonkey/remarkable-sync/internal/daemon"
	"github.com/platinummonkey/remarkable-sync/internal/logger"
	"github.com/platinummonkey/remarkable-sync/internal/ocr"
	"github.com/platinummonkey/remarkable-sync/internal/pdfenhancer"
	"github.com/platinummonkey/remarkable-sync/internal/rmclient"
	"github.com/platinummonkey/remarkable-sync/internal/state"
	"github.com/platinummonkey/remarkable-sync/internal/sync"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// daemonCmd represents the daemon command
var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run in daemon mode with periodic sync",
	Long: `Run remarkable-sync as a long-running daemon process.

The daemon performs periodic synchronization at the specified interval.
It handles signals gracefully and can be monitored via health check endpoints.

Features:
- Periodic sync at configurable interval
- Graceful shutdown on SIGTERM/SIGINT
- Optional health check HTTP endpoints
- Optional PID file for process management
- Continues running even if individual syncs fail

Examples:
  # Run daemon with default 5 minute interval
  remarkable-sync daemon

  # Run with custom interval
  remarkable-sync daemon --interval 10m

  # Run with health check endpoint
  remarkable-sync daemon --health-addr :8080

  # Run with PID file
  remarkable-sync daemon --pid-file /var/run/remarkable-sync.pid

  # Full example with all options
  remarkable-sync daemon \
    --interval 10m \
    --health-addr :8080 \
    --pid-file /var/run/remarkable-sync.pid \
    --output ~/Documents/ReMarkable \
    --labels work,personal`,
	RunE: runDaemon,
}

func init() {
	rootCmd.AddCommand(daemonCmd)

	// Daemon-specific flags
	daemonCmd.Flags().Duration("interval", 5*time.Minute, "sync interval (e.g., 5m, 1h)")
	daemonCmd.Flags().String("health-addr", "", "health check HTTP address (e.g., :8080)")
	daemonCmd.Flags().String("pid-file", "", "PID file path")

	viper.BindPFlag("daemon.interval", daemonCmd.Flags().Lookup("interval"))
	viper.BindPFlag("daemon.health_addr", daemonCmd.Flags().Lookup("health-addr"))
	viper.BindPFlag("daemon.pid_file", daemonCmd.Flags().Lookup("pid-file"))
}

func runDaemon(cmd *cobra.Command, args []string) error {
	// Initialize logger (JSON format for daemon mode)
	log, err := logger.New(&logger.Config{
		Level:  viper.GetString("log_level"),
		Format: "json",
	})
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	log.WithFields(
		"output_dir", cfg.OutputDir,
		"interval", viper.GetDuration("daemon.interval"),
	).Info("Starting daemon")

	// Initialize components
	rmClient, err := rmclient.NewClient(&rmclient.Config{
		Logger: log,
	})
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}

	stateFile := filepath.Join(cfg.OutputDir, ".remarkable-sync-state.json")
	stateStore, err := state.LoadOrCreate(stateFile)
	if err != nil {
		log.Fatal("Failed to initialize state:", err)
	}

	conv := converter.New(&converter.Config{
		Logger: log,
	})

	pdfEnhancer := pdfenhancer.New(&pdfenhancer.Config{
		Logger: log,
	})

	// Initialize OCR processor if enabled
	var ocrProc *ocr.Processor
	if cfg.OCREnabled {
		ocrProc = ocr.New(&ocr.Config{
			Logger:    log,
			Languages: []string{"eng"},
		})
	}

	// Create sync orchestrator
	orch, err := sync.New(&sync.Config{
		Config:       cfg,
		Logger:       log,
		RMClient:     rmClient,
		StateStore:   stateStore,
		Converter:    conv,
		OCRProcessor: ocrProc,
		PDFEnhancer:  pdfEnhancer,
	})
	if err != nil {
		return fmt.Errorf("failed to create orchestrator: %w", err)
	}

	// Create daemon
	d, err := daemon.New(&daemon.Config{
		Orchestrator:    orch,
		Logger:          log,
		SyncInterval:    viper.GetDuration("daemon.interval"),
		HealthCheckAddr: viper.GetString("daemon.health_addr"),
		PIDFile:         viper.GetString("daemon.pid_file"),
	})
	if err != nil {
		return fmt.Errorf("failed to create daemon: %w", err)
	}

	// Run daemon (blocks until shutdown signal)
	ctx := context.Background()
	if err := d.Run(ctx); err != nil && err != context.Canceled {
		return fmt.Errorf("daemon error: %w", err)
	}

	log.Info("Daemon shutdown complete")
	return nil
}
