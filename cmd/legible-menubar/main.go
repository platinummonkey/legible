//go:build darwin
// +build darwin

// Package main implements the macOS menu bar application for Legible.
// This application provides a system tray interface for managing the Legible daemon
// and triggering document synchronization from reMarkable tablets.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"fyne.io/systray"
	"github.com/platinummonkey/legible/internal/config"
	"github.com/platinummonkey/legible/internal/logger"
	"github.com/platinummonkey/legible/internal/menubar"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	// Parse command line flags
	showVersion := flag.Bool("version", false, "Show version information")
	configFile := flag.String("config", "", "Path to configuration file")
	outputDir := flag.String("output", "", "Output directory for synced documents")
	daemonAddr := flag.String("daemon-addr", "http://localhost:8080", "Daemon HTTP address")
	noAutoLaunch := flag.Bool("no-auto-launch", false, "Disable automatic daemon launch")
	flag.Parse()

	if *showVersion {
		fmt.Printf("legible-menubar version %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	// Initialize logger
	log, err := logger.New(&logger.Config{
		Level:  "info",
		Format: "console",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	_ = log // Use the logger if needed later

	logger.Info("Starting legible menu bar application",
		"version", version,
		"commit", commit,
		"date", date,
	)

	// Load configuration
	cfg, err := loadConfig(*configFile)
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Determine output directory
	outDir := determineOutputDir(*outputDir, cfg)

	logger.Info("Configuration loaded", "output_dir", outDir, "daemon_addr", *daemonAddr)

	// Load menu bar configuration
	menuBarCfg, err := menubar.LoadMenuBarConfig("")
	if err != nil {
		logger.Warn("Failed to load menu bar config, using defaults", "error", err)
		menuBarCfg = menubar.DefaultMenuBarConfig()
	}

	// Build daemon args from menu bar config
	daemonArgs := []string{"daemon"}
	if menuBarCfg.DaemonConfigFile != "" {
		daemonArgs = append(daemonArgs, "--config", menuBarCfg.DaemonConfigFile)
	}
	if menuBarCfg.SyncInterval != "" {
		daemonArgs = append(daemonArgs, "--interval", menuBarCfg.SyncInterval)
	}
	if !menuBarCfg.OCREnabled {
		daemonArgs = append(daemonArgs, "--no-ocr")
	}

	logger.Info("Daemon arguments configured", "args", daemonArgs)

	// Create daemon manager (unless auto-launch is disabled)
	var daemonManager *menubar.DaemonManager
	if !*noAutoLaunch {
		dm, err := menubar.NewDaemonManager(&menubar.DaemonManagerConfig{
			DaemonAddr: *daemonAddr,
			DaemonArgs: daemonArgs,
			AutoLaunch: true,
		})
		if err != nil {
			logger.Error("Failed to create daemon manager", "error", err)
			fmt.Fprintf(os.Stderr, "Warning: Failed to create daemon manager: %v\n", err)
			fmt.Fprintf(os.Stderr, "Menu bar app will connect to existing daemon if available.\n")
			// Continue without daemon manager - menu bar app can still work if daemon is running
		} else {
			daemonManager = dm
			logger.Info("Daemon manager configured", "auto_launch", true)
		}
	} else {
		logger.Info("Daemon auto-launch disabled")
	}

	// Create the menu bar app
	app := menubar.New(&menubar.Config{
		OutputDir:     outDir,
		DaemonAddr:    *daemonAddr,
		DaemonManager: daemonManager,
	})

	// Set up signal handler to ensure clean shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		logger.Info("Received signal, stopping daemon and quitting", "signal", sig)

		// Stop daemon before quitting to ensure clean shutdown
		if daemonManager != nil {
			logger.Info("Stopping daemon from signal handler")
			if err := daemonManager.Stop(); err != nil {
				logger.Error("Error stopping daemon from signal handler", "error", err)
			} else {
				logger.Info("Daemon stopped successfully from signal handler")
			}
		}

		systray.Quit()
	}()

	// Run the menu bar app
	app.Run()

	logger.Info("Menu bar application exited")
}

// loadConfig loads the configuration from file or returns defaults.
func loadConfig(configPath string) (*config.Config, error) {
	if configPath != "" {
		// Load from specified file
		return config.Load(configPath)
	}

	// Try default locations
	homeDir, err := os.UserHomeDir()
	if err == nil {
		defaultPath := filepath.Join(homeDir, ".legible.yaml")
		if _, err := os.Stat(defaultPath); err == nil {
			return config.Load(defaultPath)
		}
	}

	// Return config with defaults (Load will return defaults if file doesn't exist)
	logger.Info("No config file found, using defaults")
	return config.Load("")
}

// determineOutputDir determines the output directory from flags or config.
func determineOutputDir(flagValue string, cfg *config.Config) string {
	if flagValue != "" {
		return flagValue
	}

	if cfg.OutputDir != "" {
		return cfg.OutputDir
	}

	// Default to ~/Documents/reMarkable
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./output"
	}

	return filepath.Join(homeDir, "Documents", "reMarkable")
}
