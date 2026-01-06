package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/platinummonkey/legible/internal/config"
	"github.com/platinummonkey/legible/internal/converter"
	"github.com/platinummonkey/legible/internal/logger"
	"github.com/platinummonkey/legible/internal/ocr"
	"github.com/platinummonkey/legible/internal/pdfenhancer"
	"github.com/platinummonkey/legible/internal/rmclient"
	"github.com/platinummonkey/legible/internal/state"
	"github.com/platinummonkey/legible/internal/sync"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Perform a one-time synchronization",
	Long: `Synchronize documents from reMarkable cloud to local directory.

This command:
1. Lists documents from reMarkable API
2. Filters by labels (if specified)
3. Identifies new or changed documents
4. Downloads and converts to PDF
5. Optionally adds OCR text layer
6. Saves to output directory

The sync state is maintained to avoid re-downloading unchanged documents.

Examples:
  # Sync all documents
  legible sync

  # Sync only documents with "work" label
  legible sync --labels work

  # Sync to specific directory without OCR
  legible sync --output ~/Documents/ReMarkable --no-ocr

  # Force re-sync all documents (ignores state)
  legible sync --force`,
	RunE: runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)

	// Sync-specific flags
	syncCmd.Flags().Bool("force", false, "force re-sync all documents (ignore state)")
	_ = viper.BindPFlag("force", syncCmd.Flags().Lookup("force"))
}

func runSync(_ *cobra.Command, _ []string) error {
	// Load configuration and initialize logger
	cfg, log, err := initConfigAndLogger()
	if err != nil {
		return err
	}

	log.WithFields("output_dir", cfg.OutputDir).Info("Starting sync")

	// Initialize all components
	orch, err := initSyncComponents(cfg, log)
	if err != nil {
		return err
	}

	// Run sync
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	result, err := orch.Sync(ctx)
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	// Display results
	return displaySyncResults(result)
}

func initConfigAndLogger() (*config.Config, *logger.Logger, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	log, err := logger.New(&logger.Config{
		Level:  cfg.LogLevel,
		Format: "console",
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	return cfg, log, nil
}

func initSyncComponents(cfg *config.Config, log *logger.Logger) (*sync.Orchestrator, error) {
	// Initialize reMarkable client
	rmClient, err := rmclient.NewClient(&rmclient.Config{
		Logger: log,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Authenticate with the reMarkable API
	if err := rmClient.Authenticate(); err != nil {
		return nil, fmt.Errorf("authentication failed: %w. Please run 'legible auth' first", err)
	}

	// Load state store
	stateStore, err := state.LoadOrCreate(cfg.StateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize state: %w", err)
	}

	// Handle force flag
	if viper.GetBool("force") {
		log.Info("Force flag set, clearing sync state")
		if err := os.Remove(cfg.StateFile); err != nil && !os.IsNotExist(err) {
			log.WithFields("error", err).Warn("Failed to remove state file")
		}
	}

	// Parse OCR languages from config
	ocrLangs := []string{"eng"}
	if cfg.OCRLanguages != "" {
		ocrLangs = []string{cfg.OCRLanguages}
	}

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

		ocrProc, err = ocr.New(&ocr.Config{
			Logger:       log,
			VisionConfig: visionConfig,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create OCR processor: %w", err)
		}

		pdfEnhancer = pdfenhancer.New(&pdfenhancer.Config{
			Logger: log,
		})
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

	// Create and return sync orchestrator
	return sync.New(&sync.Config{
		Config:       cfg,
		Logger:       log,
		RMClient:     rmClient,
		StateStore:   stateStore,
		Converter:    conv,
		OCRProcessor: ocrProc,
		PDFEnhancer:  pdfEnhancer,
	})
}

func displaySyncResults(result *sync.Result) error {
	fmt.Println()
	fmt.Println("=== Sync Complete ===")
	fmt.Printf("Total documents: %d\n", result.TotalDocuments)
	fmt.Printf("Processed: %d\n", result.ProcessedDocuments)
	fmt.Printf("Successful: %d\n", result.SuccessCount)
	fmt.Printf("Failed: %d\n", result.FailureCount)
	fmt.Printf("Duration: %v\n", result.Duration)

	if result.HasFailures() {
		fmt.Println("\nFailures:")
		for _, failure := range result.Failures {
			fmt.Printf("  - %s: %v\n", failure.Title, failure.Error)
		}
		return fmt.Errorf("sync completed with %d failures", result.FailureCount)
	}

	return nil
}

func loadConfig() (*config.Config, error) {
	// Get the config file path if viper has read one
	// Pass empty string if no config file was found (will use defaults)
	configFile := viper.ConfigFileUsed()

	// Use the proper config loading which handles all fields
	cfg, err := config.Load(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Override with command-line flags if provided
	if viper.IsSet("output-dir") {
		cfg.OutputDir = viper.GetString("output-dir")
	}
	if viper.IsSet("labels") {
		cfg.Labels = viper.GetStringSlice("labels")
	}
	if viper.IsSet("no-ocr") {
		cfg.OCREnabled = !viper.GetBool("no-ocr")
	}
	if viper.IsSet("log-level") {
		cfg.LogLevel = viper.GetString("log-level")
	}

	return cfg, nil
}
