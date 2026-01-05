package main

import (
	"context"
	"fmt"
	"time"

	"github.com/platinummonkey/legible/internal/converter"
	"github.com/platinummonkey/legible/internal/daemon"
	"github.com/platinummonkey/legible/internal/logger"
	"github.com/platinummonkey/legible/internal/ocr"
	"github.com/platinummonkey/legible/internal/pdfenhancer"
	"github.com/platinummonkey/legible/internal/rmclient"
	"github.com/platinummonkey/legible/internal/state"
	"github.com/platinummonkey/legible/internal/sync"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// daemonCmd represents the daemon command
var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run in daemon mode with periodic sync",
	Long: `Run legible as a long-running daemon process.

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
  legible daemon

  # Run with custom interval
  legible daemon --interval 10m

  # Run with health check endpoint
  legible daemon --health-addr :8080

  # Run with PID file
  legible daemon --pid-file /var/run/legible.pid

  # Full example with all options
  legible daemon \
    --interval 10m \
    --health-addr :8080 \
    --pid-file /var/run/legible.pid \
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

	_ = viper.BindPFlag("daemon.interval", daemonCmd.Flags().Lookup("interval"))
	_ = viper.BindPFlag("daemon.health_addr", daemonCmd.Flags().Lookup("health-addr"))
	_ = viper.BindPFlag("daemon.pid_file", daemonCmd.Flags().Lookup("pid-file"))
}

func runDaemon(_ *cobra.Command, _ []string) error {
	// Load configuration first
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Override sync interval from daemon flag if set
	if viper.IsSet("daemon.interval") {
		cfg.SyncInterval = viper.GetDuration("daemon.interval")
	}

	// Initialize logger (JSON format for daemon mode)
	log, err := logger.New(&logger.Config{
		Level:  cfg.LogLevel,
		Format: "json",
	})
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	log.WithFields(
		"output_dir", cfg.OutputDir,
		"interval", cfg.SyncInterval,
	).Info("Starting daemon")

	// Initialize components
	rmClient, err := rmclient.NewClient(&rmclient.Config{
		Logger: log,
	})
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}

	stateStore, err := state.LoadOrCreate(cfg.StateFile)
	if err != nil {
		log.Fatal("Failed to initialize state:", err)
	}

	// Parse OCR languages from config (comma or plus separated)
	ocrLangs := []string{"eng"}
	if cfg.OCRLanguages != "" {
		ocrLangs = []string{cfg.OCRLanguages}
	}

	conv, err := converter.New(&converter.Config{
		Logger:       log,
		EnableOCR:    cfg.OCREnabled,
		OCRLanguages: ocrLangs,
	})
	if err != nil {
		return fmt.Errorf("failed to create converter: %w", err)
	}

	pdfEnhancer := pdfenhancer.New(&pdfenhancer.Config{
		Logger: log,
	})

	// Initialize OCR processor if enabled
	var ocrProc *ocr.Processor
	if cfg.OCREnabled {
		ocrProc, err = ocr.New(&ocr.Config{
			Logger: log,
			// Ollama handles language detection automatically via vision models
		})
		if err != nil {
			return fmt.Errorf("failed to create OCR processor: %w", err)
		}
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
		SyncInterval:    cfg.SyncInterval,
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
