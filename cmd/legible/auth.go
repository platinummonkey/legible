// Package main provides the reMarkable Sync CLI application.
package main

import (
	"fmt"

	"github.com/platinummonkey/legible/internal/logger"
	"github.com/platinummonkey/legible/internal/rmclient"
	"github.com/spf13/cobra"
)

// authCmd represents the auth command
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with reMarkable API",
	Long: `Authenticate with the reMarkable cloud API using a one-time code.

This command guides you through the authentication process:
1. Opens your browser to get a one-time code from reMarkable
2. You enter the code
3. Credentials are stored for future use

Authentication is required before syncing documents.

Example:
  legible auth`,
	RunE: runAuth,
}

func init() {
	rootCmd.AddCommand(authCmd)
}

func runAuth(_ *cobra.Command, _ []string) error {
	// Load configuration first
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize logger
	log, err := logger.New(&logger.Config{
		Level:  cfg.LogLevel,
		Format: "console",
	})
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	log.Info("Starting reMarkable authentication")

	// Create rmclient
	client, err := rmclient.NewClient(&rmclient.Config{
		Logger: log,
	})
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}

	// Start authentication flow
	log.Info("Opening browser for authentication...")
	log.Info("Please follow the instructions in your browser")

	if err := client.Authenticate(); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	log.Info("Authentication successful!")
	log.Info("Credentials saved for future use")

	return nil
}
