//go:build darwin
// +build darwin

// Package menubar provides the menu bar application logic for macOS.
package menubar

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"fyne.io/systray"
	"github.com/platinummonkey/legible/internal/logger"
)

// App represents the menu bar application.
type App struct {
	// Menu items
	mStatus      *systray.MenuItem
	mStartSync   *systray.MenuItem
	mStopSync    *systray.MenuItem
	mOpenOutput  *systray.MenuItem
	mAutoStart   *systray.MenuItem
	mPreferences *systray.MenuItem
	mQuit        *systray.MenuItem

	// Application state
	isRunning  bool
	outputDir  string
	statusText string
	daemonAddr string

	// Configuration
	menuBarConfig *MenuBarConfig
	configPath    string

	// Daemon management
	daemonManager *DaemonManager
	daemonClient  *DaemonClient

	// Channels for menu actions
	quitChan chan struct{}
}

// Config holds configuration for the menu bar app
type Config struct {
	OutputDir     string
	DaemonAddr    string         // HTTP address of daemon (e.g., "http://localhost:8080")
	DaemonManager *DaemonManager // Optional daemon manager (if nil, no auto-launch)
}

// New creates a new menu bar application.
func New(cfg *Config) *App {
	if cfg == nil {
		cfg = &Config{
			OutputDir:  "./output",
			DaemonAddr: "http://localhost:8080",
		}
	}

	if cfg.DaemonAddr == "" {
		cfg.DaemonAddr = "http://localhost:8080"
	}

	// Load menu bar configuration
	configPath, _ := GetConfigPath()
	menuBarCfg, err := LoadMenuBarConfig("")
	if err != nil {
		logger.Warn("Failed to load menu bar config, using defaults", "error", err)
		menuBarCfg = DefaultMenuBarConfig()
	}

	return &App{
		outputDir:     cfg.OutputDir,
		daemonAddr:    cfg.DaemonAddr,
		menuBarConfig: menuBarCfg,
		configPath:    configPath,
		daemonManager: cfg.DaemonManager,
		daemonClient:  NewDaemonClient(cfg.DaemonAddr),
		statusText:    "Starting...",
		quitChan:      make(chan struct{}),
	}
}

// Run starts the menu bar application.
func (a *App) Run() {
	systray.Run(a.onReady, a.onExit)
}

// onReady is called when the systray is ready.
func (a *App) onReady() {
	logger.Info("Menu bar application starting")

	// Set initial icon (gray/starting state)
	systray.SetIcon(iconGreen())
	systray.SetTitle("Legible")
	systray.SetTooltip("reMarkable Sync - Starting...")

	// Create menu items
	a.mStatus = systray.AddMenuItem("Status: Starting daemon...", "Current sync status")
	a.mStatus.Disable() // Status is informational only

	systray.AddSeparator()

	a.mStartSync = systray.AddMenuItem("Trigger Sync", "Trigger an immediate sync")
	a.mStopSync = systray.AddMenuItem("Cancel Sync", "Cancel the running sync")
	a.mStopSync.Disable() // Disabled until sync is running

	systray.AddSeparator()

	a.mOpenOutput = systray.AddMenuItem("Open Output Folder", "Open the output directory in Finder")

	systray.AddSeparator()

	// Auto-start menu item with checkbox
	a.mAutoStart = systray.AddMenuItemCheckbox("Start at Login", "Launch automatically when you log in", false)
	// Set initial checkbox state
	if enabled, err := IsAutoStartEnabled(); err == nil && enabled {
		a.mAutoStart.Check()
		a.menuBarConfig.AutoStartEnabled = true
	}

	a.mPreferences = systray.AddMenuItem("Preferences...", "Configure settings")

	systray.AddSeparator()

	a.mQuit = systray.AddMenuItem("Quit", "Exit the application")

	// Start daemon manager if configured
	if a.daemonManager != nil {
		logger.Info("Starting daemon manager")
		if err := a.daemonManager.Start(); err != nil {
			logger.Error("Failed to start daemon manager", "error", err)
			a.setStatus("Error: Failed to start daemon", iconRed())
		} else {
			logger.Info("Daemon manager started successfully")
		}
	}

	// Start event loop and status polling
	go a.handleMenuEvents()
	go a.pollDaemonStatus()
}

// onExit is called when the application exits.
func (a *App) onExit() {
	logger.Info("Menu bar application exiting")
	close(a.quitChan)

	// Stop daemon manager if configured
	if a.daemonManager != nil {
		logger.Info("Stopping daemon manager")
		if err := a.daemonManager.Stop(); err != nil {
			logger.Error("Error stopping daemon manager", "error", err)
		} else {
			logger.Info("Daemon manager stopped successfully")
		}
	}
}

// handleMenuEvents processes menu item clicks.
func (a *App) handleMenuEvents() {
	for {
		select {
		case <-a.mStartSync.ClickedCh:
			a.handleStartSync()
		case <-a.mStopSync.ClickedCh:
			a.handleStopSync()
		case <-a.mOpenOutput.ClickedCh:
			a.handleOpenOutput()
		case <-a.mAutoStart.ClickedCh:
			a.handleAutoStartToggle()
		case <-a.mPreferences.ClickedCh:
			a.handlePreferences()
		case <-a.mQuit.ClickedCh:
			systray.Quit()
			return
		case <-a.quitChan:
			return
		}
	}
}

// handleStartSync handles the start sync action.
func (a *App) handleStartSync() {
	logger.Info("Trigger sync clicked")

	ctx := context.Background()
	if err := a.daemonClient.TriggerSync(ctx); err != nil {
		logger.Error("Failed to trigger sync", "error", err)
		// Show error to user (could enhance with notification)
		a.setStatus(fmt.Sprintf("Error: %s", err.Error()), iconRed())
		return
	}

	logger.Info("Sync triggered successfully")
}

// handleStopSync handles the stop sync action.
func (a *App) handleStopSync() {
	logger.Info("Cancel sync clicked")

	ctx := context.Background()
	if err := a.daemonClient.CancelSync(ctx); err != nil {
		logger.Error("Failed to cancel sync", "error", err)
		a.setStatus(fmt.Sprintf("Error: %s", err.Error()), iconRed())
		return
	}

	logger.Info("Sync cancellation requested")
}

// handleOpenOutput opens the output directory in Finder.
func (a *App) handleOpenOutput() {
	logger.Info("Open output folder clicked", "path", a.outputDir)

	if runtime.GOOS != "darwin" {
		logger.Warn("Open folder only supported on macOS")
		return
	}

	cmd := exec.Command("open", a.outputDir)
	if err := cmd.Run(); err != nil {
		logger.Error("Failed to open output folder", "error", err, "path", a.outputDir)
	}
}

// handleAutoStartToggle handles toggling the auto-start feature.
func (a *App) handleAutoStartToggle() {
	logger.Info("Auto-start toggle clicked")

	// Check current state
	enabled, err := IsAutoStartEnabled()
	if err != nil {
		logger.Error("Failed to check auto-start status", "error", err)
		return
	}

	// Toggle the state
	if enabled {
		// Disable auto-start
		if err := DisableAutoStart(); err != nil {
			logger.Error("Failed to disable auto-start", "error", err)
			return
		}
		a.mAutoStart.Uncheck()
		a.menuBarConfig.AutoStartEnabled = false
		logger.Info("Auto-start disabled")
	} else {
		// Enable auto-start
		if err := EnableAutoStart(); err != nil {
			logger.Error("Failed to enable auto-start", "error", err)
			return
		}
		a.mAutoStart.Check()
		a.menuBarConfig.AutoStartEnabled = true
		logger.Info("Auto-start enabled")
	}

	// Save configuration
	if err := SaveMenuBarConfig(a.menuBarConfig, a.configPath); err != nil {
		logger.Error("Failed to save configuration", "error", err)
	}
}

// handlePreferences opens the preferences dialog.
func (a *App) handlePreferences() {
	logger.Info("Preferences clicked")

	// Build preferences message
	ocrStatus := "Enabled"
	if !a.menuBarConfig.OCREnabled {
		ocrStatus = "Disabled"
	}
	autoStartStatus := "Disabled"
	if a.menuBarConfig.AutoStartEnabled {
		autoStartStatus = "Enabled"
	}

	message := fmt.Sprintf(`Current Settings:

Daemon Address: %s
Sync Interval: %s
OCR: %s
Auto-start at Login: %s
Daemon Config File: %s

To change these settings, would you like to open the configuration file?

Config Location: %s`,
		a.menuBarConfig.DaemonAddr,
		a.menuBarConfig.SyncInterval,
		ocrStatus,
		autoStartStatus,
		a.menuBarConfig.DaemonConfigFile,
		a.configPath)

	// Show dialog and ask if user wants to open config file
	if runtime.GOOS == "darwin" {
		go a.showPreferencesDialog(message)
	} else {
		logger.Info("Preferences", "config_path", a.configPath)
	}
}

// showPreferencesDialog displays a native macOS dialog with current preferences
func (a *App) showPreferencesDialog(message string) {
	// Use osascript to show a native macOS dialog
	// The dialog has "Open Config File" and "Cancel" buttons
	script := fmt.Sprintf(`display dialog %q buttons {"Cancel", "Open Config File"} default button "Open Config File" with title "Legible Preferences" with icon note`,
		message)

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()

	if err != nil {
		// User clicked Cancel or closed dialog
		logger.Info("Preferences dialog dismissed")
		return
	}

	// Check if user clicked "Open Config File"
	outputStr := string(output)
	if len(outputStr) > 0 && (len(outputStr) < 20 || outputStr[:20] != "button returned:Open") {
		// User clicked Cancel
		logger.Info("User canceled preferences")
		return
	}

	// User wants to open config file - create it if it doesn't exist
	if _, err := os.Stat(a.configPath); os.IsNotExist(err) {
		// Create config directory
		if err := os.MkdirAll(filepath.Dir(a.configPath), 0755); err != nil {
			logger.Error("Failed to create config directory", "error", err)
			a.showErrorDialog(fmt.Sprintf("Failed to create config directory: %v", err))
			return
		}

		// Save default config
		if err := SaveMenuBarConfig(a.menuBarConfig, a.configPath); err != nil {
			logger.Error("Failed to save config file", "error", err)
			a.showErrorDialog(fmt.Sprintf("Failed to create config file: %v", err))
			return
		}
	}

	// Open config file in default editor
	cmd = exec.Command("open", a.configPath)
	if err := cmd.Run(); err != nil {
		logger.Error("Failed to open config file", "error", err, "path", a.configPath)
		a.showErrorDialog(fmt.Sprintf("Failed to open config file: %v", err))
	} else {
		logger.Info("Opened config file in default editor", "path", a.configPath)
	}
}

// showErrorDialog displays an error message in a native macOS dialog
func (a *App) showErrorDialog(message string) {
	script := fmt.Sprintf(`display dialog %q buttons {"OK"} default button "OK" with title "Legible Error" with icon stop`,
		message)
	cmd := exec.Command("osascript", "-e", script)
	_ = cmd.Run() // Ignore error
}

// setStatus updates the status display and icon.
func (a *App) setStatus(status string, icon []byte) {
	a.statusText = status
	a.mStatus.SetTitle(fmt.Sprintf("Status: %s", status))
	systray.SetIcon(icon)
	systray.SetTooltip(fmt.Sprintf("reMarkable Sync - %s", status))
}

// SetStatusIdle sets the status to idle (green).
func (a *App) SetStatusIdle() {
	a.setStatus("Idle", iconGreen())
	a.mStartSync.Enable()
	a.mStopSync.Disable()
}

// SetStatusSyncing sets the status to syncing (yellow).
func (a *App) SetStatusSyncing() {
	a.setStatus("Syncing", iconYellow())
	a.mStartSync.Disable()
	a.mStopSync.Enable()
}

// SetStatusError sets the status to error (red).
func (a *App) SetStatusError(errMsg string) {
	status := fmt.Sprintf("Error: %s", errMsg)
	a.setStatus(status, iconRed())
	a.mStartSync.Enable()
	a.mStopSync.Disable()
}

// pollDaemonStatus polls the daemon for status updates
func (a *App) pollDaemonStatus() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	// Do an immediate check
	a.updateStatusFromDaemon()

	for {
		select {
		case <-ticker.C:
			a.updateStatusFromDaemon()
		case <-a.quitChan:
			return
		}
	}
}

// updateStatusFromDaemon fetches status from daemon and updates UI
func (a *App) updateStatusFromDaemon() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	status, err := a.daemonClient.GetStatus(ctx)
	if err != nil {
		logger.Error("Failed to get daemon status", "error", err)
		a.setStatus("Error: Cannot connect to daemon", iconRed())
		a.mStartSync.Disable()
		a.mStopSync.Disable()
		return
	}

	// Update UI based on daemon state
	switch status.State {
	case StateOffline:
		a.setStatus("Daemon offline", iconRed())
		a.mStartSync.Disable()
		a.mStopSync.Disable()

	case StateIdle:
		// Show last sync info if available
		statusText := "Idle"
		if status.LastSyncResult != nil {
			statusText = fmt.Sprintf("Idle - Last sync: %d docs (%d success, %d failed)",
				status.LastSyncResult.ProcessedDocuments,
				status.LastSyncResult.SuccessCount,
				status.LastSyncResult.FailureCount)
		}
		a.setStatus(statusText, iconGreen())
		a.mStartSync.Enable()
		a.mStopSync.Disable()

	case StateSyncing:
		// Show sync progress
		statusText := "Syncing..."
		if status.CurrentSync != nil {
			if status.CurrentSync.DocumentsTotal > 0 {
				statusText = fmt.Sprintf("Syncing: %d/%d docs",
					status.CurrentSync.DocumentsProcessed,
					status.CurrentSync.DocumentsTotal)
			}
			if status.CurrentSync.CurrentDocument != "" {
				statusText += fmt.Sprintf(" (%s)", status.CurrentSync.CurrentDocument)
			}
		}
		a.setStatus(statusText, iconYellow())
		a.mStartSync.Disable()
		a.mStopSync.Enable()

	case StateError:
		errMsg := "Sync failed"
		if status.ErrorMessage != "" {
			errMsg = fmt.Sprintf("Error: %s", status.ErrorMessage)
		}
		a.setStatus(errMsg, iconRed())
		a.mStartSync.Enable()
		a.mStopSync.Disable()
	}
}
