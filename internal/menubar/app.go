//go:build darwin
// +build darwin

// Package menubar provides the menu bar application logic for macOS.
package menubar

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
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

	if runtime.GOOS == "darwin" {
		go a.showPreferencesEditor()
	} else {
		logger.Info("Preferences", "config_path", a.configPath)
	}
}

// showPreferencesEditor displays an interactive preferences editor
func (a *App) showPreferencesEditor() {
	for {
		// Build current settings display
		ocrStatus := "✓ Enabled"
		if !a.menuBarConfig.OCREnabled {
			ocrStatus = "✗ Disabled"
		}

		message := fmt.Sprintf(`Choose a setting to edit:

1. Daemon Address: %s
2. Sync Interval: %s
3. OCR Processing: %s
4. Daemon Config File: %s

Select a number (1-4) to edit that setting, or click Done to close.`,
			a.menuBarConfig.DaemonAddr,
			a.menuBarConfig.SyncInterval,
			ocrStatus,
			a.menuBarConfig.DaemonConfigFile)

		// Show selection dialog
		script := fmt.Sprintf(`display dialog %q buttons {"Done", "Edit"} default button "Edit" with title "Legible Preferences" default answer "" with icon note`,
			message)

		cmd := exec.Command("osascript", "-e", script)
		output, err := cmd.CombinedOutput()

		if err != nil || len(output) == 0 {
			// User clicked Done or closed dialog
			logger.Info("Preferences editor closed")
			return
		}

		outputStr := string(output)

		// Check if user clicked Done
		if len(outputStr) > 15 && outputStr[:15] == "button returned" && outputStr[16:20] == "Done" {
			logger.Info("User finished editing preferences")
			return
		}

		// Extract the user's choice from "text returned:X"
		choice := ""
		if idx := strings.Index(outputStr, "text returned:"); idx >= 0 {
			textPart := outputStr[idx+14:]
			if len(textPart) > 0 {
				choice = strings.TrimSpace(strings.Split(textPart, ",")[0])
			}
		}

		// Handle the user's choice
		configChanged := false
		switch choice {
		case "1":
			configChanged = a.editDaemonAddress()
		case "2":
			configChanged = a.editSyncInterval()
		case "3":
			configChanged = a.editOCRSetting()
		case "4":
			configChanged = a.editDaemonConfigFile()
		default:
			// Invalid choice, show error and continue
			a.showInfoDialog("Please enter a number between 1 and 4")
			continue
		}

		// If config changed, save it
		if configChanged {
			if err := SaveMenuBarConfig(a.menuBarConfig, a.configPath); err != nil {
				logger.Error("Failed to save configuration", "error", err)
				a.showErrorDialog(fmt.Sprintf("Failed to save settings: %v", err))
			} else {
				logger.Info("Configuration saved successfully")
				// Ask if user wants to restart daemon
				a.promptDaemonRestart()
			}
		}
	}
}

// editDaemonAddress allows user to edit the daemon address
func (a *App) editDaemonAddress() bool {
	script := fmt.Sprintf(`display dialog "Enter daemon address (e.g., http://localhost:8080):" default answer %q buttons {"Cancel", "Save"} default button "Save" with title "Edit Daemon Address"`,
		a.menuBarConfig.DaemonAddr)

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	// Extract new value
	newValue := a.extractTextInput(string(output))
	if newValue != "" && newValue != a.menuBarConfig.DaemonAddr {
		a.menuBarConfig.DaemonAddr = newValue
		a.daemonAddr = newValue
		a.daemonClient = NewDaemonClient(newValue)
		logger.Info("Daemon address updated", "new_addr", newValue)
		return true
	}
	return false
}

// editSyncInterval allows user to edit the sync interval
func (a *App) editSyncInterval() bool {
	script := fmt.Sprintf(`display dialog "Enter sync interval (e.g., 30m, 1h, 2h):" default answer %q buttons {"Cancel", "Save"} default button "Save" with title "Edit Sync Interval"`,
		a.menuBarConfig.SyncInterval)

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	// Extract new value
	newValue := a.extractTextInput(string(output))
	if newValue != "" && newValue != a.menuBarConfig.SyncInterval {
		a.menuBarConfig.SyncInterval = newValue
		logger.Info("Sync interval updated", "new_interval", newValue)
		return true
	}
	return false
}

// editOCRSetting allows user to toggle OCR on/off
func (a *App) editOCRSetting() bool {
	currentStatus := "Enabled"
	if !a.menuBarConfig.OCREnabled {
		currentStatus = "Disabled"
	}

	script := fmt.Sprintf(`display dialog "OCR Processing is currently: %s\n\nDo you want to enable or disable OCR?" buttons {"Cancel", "Disable", "Enable"} default button "Enable" with title "Edit OCR Setting"`,
		currentStatus)

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	outputStr := string(output)
	if strings.Contains(outputStr, "button returned:Enable") {
		if !a.menuBarConfig.OCREnabled {
			a.menuBarConfig.OCREnabled = true
			logger.Info("OCR enabled")
			return true
		}
	} else if strings.Contains(outputStr, "button returned:Disable") {
		if a.menuBarConfig.OCREnabled {
			a.menuBarConfig.OCREnabled = false
			logger.Info("OCR disabled")
			return true
		}
	}
	return false
}

// editDaemonConfigFile allows user to edit the daemon config file path
func (a *App) editDaemonConfigFile() bool {
	script := fmt.Sprintf(`display dialog "Enter path to daemon config file:" default answer %q buttons {"Cancel", "Save"} default button "Save" with title "Edit Daemon Config File"`,
		a.menuBarConfig.DaemonConfigFile)

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	// Extract new value
	newValue := a.extractTextInput(string(output))
	if newValue != "" && newValue != a.menuBarConfig.DaemonConfigFile {
		a.menuBarConfig.DaemonConfigFile = newValue
		logger.Info("Daemon config file updated", "new_path", newValue)
		return true
	}
	return false
}

// promptDaemonRestart asks user if they want to restart the app
func (a *App) promptDaemonRestart() {
	script := `display dialog "Settings saved successfully.\n\nTo apply changes, please restart the Legible menu bar app. Would you like to quit now?" buttons {"Later", "Quit App"} default button "Quit App" with title "Restart Required"`

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return
	}

	if strings.Contains(string(output), "button returned:Quit App") {
		logger.Info("User requested app restart, quitting...")
		systray.Quit()
	}
}

// extractTextInput extracts text input from osascript output
func (a *App) extractTextInput(output string) string {
	if idx := strings.Index(output, "text returned:"); idx >= 0 {
		textPart := output[idx+14:]
		// Extract until comma or end
		if commaIdx := strings.Index(textPart, ","); commaIdx >= 0 {
			return strings.TrimSpace(textPart[:commaIdx])
		}
		return strings.TrimSpace(textPart)
	}
	return ""
}

// showInfoDialog displays an informational message
func (a *App) showInfoDialog(message string) {
	script := fmt.Sprintf(`display dialog %q buttons {"OK"} default button "OK" with title "Legible" with icon note`,
		message)
	cmd := exec.Command("osascript", "-e", script)
	_ = cmd.Run() // Ignore error
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
