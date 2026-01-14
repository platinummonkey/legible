package main

import (
	"context"
	"fmt"
	"time"

	"github.com/platinummonkey/legible/internal/config"
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
	daemonCmd.Flags().Bool("monitor-tokens", false, "enable token renewal monitoring and statistics")
	daemonCmd.Flags().String("token-stats-file", "", "file to save token statistics (requires --monitor-tokens)")

	_ = viper.BindPFlag("daemon.interval", daemonCmd.Flags().Lookup("interval"))
	_ = viper.BindPFlag("daemon.health_addr", daemonCmd.Flags().Lookup("health-addr"))
	_ = viper.BindPFlag("daemon.pid_file", daemonCmd.Flags().Lookup("pid-file"))
	_ = viper.BindPFlag("daemon.monitor_tokens", daemonCmd.Flags().Lookup("monitor-tokens"))
	_ = viper.BindPFlag("daemon.token_stats_file", daemonCmd.Flags().Lookup("token-stats-file"))
}

// configureRMClient creates and configures the reMarkable client with monitoring options
func configureRMClient(log *logger.Logger) (*rmclient.Client, error) {
	rmClientCfg := &rmclient.Config{
		Logger: log,
	}

	// Enable token monitoring if requested
	if viper.GetBool("daemon.monitor_tokens") {
		rmClientCfg.EnableTokenMonitoring = true

		// Set stats file if provided
		if viper.IsSet("daemon.token_stats_file") {
			rmClientCfg.TokenStatsFile = viper.GetString("daemon.token_stats_file")
		}

		log.Info("Token monitoring enabled for daemon")
	}

	return rmclient.NewClient(rmClientCfg)
}

// initializeOCR sets up OCR processor and PDF enhancer if enabled in config
func initializeOCR(cfg *config.Config, log *logger.Logger) (*ocr.Processor, *pdfenhancer.PDFEnhancer, error) {
	// Initialize OCR processor and PDF enhancer if enabled
	var ocrProc *ocr.Processor
	var pdfEnhancer *pdfenhancer.PDFEnhancer

	if cfg.OCREnabled {
		// Convert config.LLMConfig to ocr.VisionClientConfig
		visionConfig := &ocr.VisionClientConfig{
			Provider:    ocr.ProviderType(cfg.LLM.Provider),
			Model:       cfg.LLM.Model,
			Endpoint:    cfg.LLM.Endpoint,
			APIKey:      cfg.LLM.APIKey,
			MaxRetries:  cfg.LLM.MaxRetries,
			Temperature: cfg.LLM.Temperature,
		}

		var err error
		ocrProc, err = ocr.New(&ocr.Config{
			Logger:       log,
			VisionConfig: visionConfig,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create OCR processor: %w", err)
		}

		pdfEnhancer = pdfenhancer.New(&pdfenhancer.Config{
			Logger: log,
		})
	}

	return ocrProc, pdfEnhancer, nil
}

// createDaemonWithComponents initializes converter, orchestrator, and daemon
func createDaemonWithComponents(
	cfg *config.Config,
	log *logger.Logger,
	rmClient *rmclient.Client,
	stateStore *state.Manager,
	ocrProc *ocr.Processor,
	pdfEnhancer *pdfenhancer.PDFEnhancer,
) (*daemon.Daemon, error) {
	// Parse OCR languages
	ocrLangs := []string{"eng"}
	if cfg.OCRLanguages != "" {
		ocrLangs = []string{cfg.OCRLanguages}
	}

	// Initialize converter with pre-configured processors
	conv, err := converter.New(&converter.Config{
		Logger:       log,
		EnableOCR:    cfg.OCREnabled,
		OCRLanguages: ocrLangs,
		OCRProcessor: ocrProc,
		PDFEnhancer:  pdfEnhancer,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create converter: %w", err)
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
		return nil, fmt.Errorf("failed to create orchestrator: %w", err)
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
		return nil, fmt.Errorf("failed to create daemon: %w", err)
	}

	return d, nil
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

	// Initialize reMarkable client
	rmClient, err := configureRMClient(log)
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}

	// Authenticate with the reMarkable API
	if err := rmClient.Authenticate(); err != nil {
		return fmt.Errorf("authentication failed: %w. Please run 'legible auth' first", err)
	}

	// Ensure client cleanup on exit
	defer func() {
		// Print token monitoring statistics if enabled
		if tokenMonitor := rmClient.GetTokenMonitor(); tokenMonitor != nil {
			tokenMonitor.PrintSummary()
		}

		if err := rmClient.Close(); err != nil {
			log.WithError(err).Error("Failed to close client")
		}
	}()

	stateStore, err := state.LoadOrCreate(cfg.StateFile)
	if err != nil {
		log.Fatal("Failed to initialize state:", err)
	}

	// Initialize OCR components if enabled
	ocrProc, pdfEnhancer, err := initializeOCR(cfg, log)
	if err != nil {
		return err
	}

	// Create daemon with all components
	d, err := createDaemonWithComponents(cfg, log, rmClient, stateStore, ocrProc, pdfEnhancer)
	if err != nil {
		return err
	}

	// Run daemon (blocks until shutdown signal)
	ctx := context.Background()
	if err := d.Run(ctx); err != nil && err != context.Canceled {
		return fmt.Errorf("daemon error: %w", err)
	}

	log.Info("Daemon shutdown complete")
	return nil
}
