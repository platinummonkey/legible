package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/platinummonkey/remarkable-sync/internal/config"
	"github.com/platinummonkey/remarkable-sync/internal/converter"
	"github.com/platinummonkey/remarkable-sync/internal/logger"
	"github.com/platinummonkey/remarkable-sync/internal/ocr"
	"github.com/platinummonkey/remarkable-sync/internal/pdfenhancer"
	"github.com/platinummonkey/remarkable-sync/internal/rmclient"
	"github.com/platinummonkey/remarkable-sync/internal/state"
	"github.com/platinummonkey/remarkable-sync/internal/sync"
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
  remarkable-sync sync

  # Sync only documents with "work" label
  remarkable-sync sync --labels work

  # Sync to specific directory without OCR
  remarkable-sync sync --output ~/Documents/ReMarkable --no-ocr

  # Force re-sync all documents (ignores state)
  remarkable-sync sync --force`,
	RunE: runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)

	// Sync-specific flags
	syncCmd.Flags().Bool("force", false, "force re-sync all documents (ignore state)")
	_ = viper.BindPFlag("force", syncCmd.Flags().Lookup("force"))
}

func runSync(_ *cobra.Command, _ []string) error {
	// Initialize logger
	log, err := logger.New(&logger.Config{
		Level:  viper.GetString("log_level"),
		Format: "console",
	})
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	log.WithFields("output_dir", cfg.OutputDir).Info("Starting sync")

	// Initialize components
	rmClient, err := rmclient.NewClient(&rmclient.Config{
		Logger: log,
	})
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}

	// Determine state file path
	stateFile := filepath.Join(cfg.OutputDir, ".remarkable-sync-state.json")
	stateStore, err := state.LoadOrCreate(stateFile)
	if err != nil {
		log.Fatal("Failed to initialize state:", err)
	}

	// If force flag is set, clear state
	if viper.GetBool("force") {
		log.Info("Force flag set, clearing sync state")
		if err := os.Remove(stateFile); err != nil && !os.IsNotExist(err) {
			log.WithFields("error", err).Warn("Failed to remove state file")
		}
	}

	// Initialize converter
	conv := converter.New(&converter.Config{
		Logger: log,
	})

	// Initialize PDF enhancer
	pdfEnhancer := pdfenhancer.New(&pdfenhancer.Config{
		Logger: log,
	})

	// Initialize OCR processor if enabled
	var ocrProc *ocr.Processor
	if cfg.OCREnabled {
		ocrProc = ocr.New(&ocr.Config{
			Logger:    log,
			Languages: []string{"eng"}, // TODO: Make configurable
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

	// Run sync
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	result, err := orch.Sync(ctx)
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	// Display results
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
	// Get output directory
	outputDir := viper.GetString("output_dir")
	if outputDir == "" {
		// Default to ~/ReMarkable
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		outputDir = filepath.Join(home, "ReMarkable")
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Get labels
	labels := viper.GetStringSlice("labels")

	// Get OCR enabled (note: --no-ocr flag inverts the logic)
	ocrEnabled := !viper.GetBool("no-ocr")

	return &config.Config{
		OutputDir:  outputDir,
		Labels:     labels,
		OCREnabled: ocrEnabled,
	}, nil
}
