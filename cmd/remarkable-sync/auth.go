package main

import (
	"context"
	"fmt"
	"time"

	"github.com/platinummonkey/remarkable-sync/internal/logger"
	"github.com/platinummonkey/remarkable-sync/internal/rmclient"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
  remarkable-sync auth`,
	RunE: runAuth,
}

func init() {
	rootCmd.AddCommand(authCmd)
}

func runAuth(cmd *cobra.Command, args []string) error {
	// Initialize logger
	log, err := logger.New(&logger.Config{
		Level:  viper.GetString("log_level"),
		Format: "console",
	})
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	log.Info("Starting reMarkable authentication")

	// Create rmclient
	client := rmclient.New(&rmclient.Config{
		Logger: log,
	})

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Start authentication flow
	log.Info("Opening browser for authentication...")
	log.Info("Please follow the instructions in your browser")

	if err := client.Authenticate(ctx); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	log.Info("Authentication successful!")
	log.Info("Credentials saved for future use")

	return nil
}
