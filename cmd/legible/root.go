package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile  string
	logLevel string
	version  = "dev" // Set via build flags
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "legible",
	Short: "Sync and enhance reMarkable documents with OCR",
	Long: `legible synchronizes documents from your reMarkable tablet
to your local machine, converting them to searchable PDFs with optional OCR.

Features:
  - Download .rmdoc files from reMarkable cloud
  - Convert to standard PDF format
  - Add invisible OCR text layer (optional)
  - Filter by labels
  - Daemon mode for continuous sync
  - Maintain sync state to avoid redundant downloads`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.legible.yaml)")
	rootCmd.PersistentFlags().String("output", "", "output directory for PDFs")
	rootCmd.PersistentFlags().StringSlice("labels", []string{}, "filter documents by labels (comma-separated)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().Bool("no-ocr", false, "disable OCR processing")

	// Bind flags to viper
	_ = viper.BindPFlag("output_dir", rootCmd.PersistentFlags().Lookup("output"))
	_ = viper.BindPFlag("labels", rootCmd.PersistentFlags().Lookup("labels"))
	_ = viper.BindPFlag("log_level", rootCmd.PersistentFlags().Lookup("log-level"))
	_ = viper.BindPFlag("ocr_enabled", rootCmd.PersistentFlags().Lookup("no-ocr"))
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding home directory: %v\n", err)
			os.Exit(1)
		}

		// Search config in home directory with name ".legible" (without extension)
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".legible")
	}

	// Read environment variables with RMSYNC prefix
	viper.SetEnvPrefix("RMSYNC")
	viper.AutomaticEnv()

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintf(os.Stderr, "Using config file: %s\n", viper.ConfigFileUsed())
	}
}
