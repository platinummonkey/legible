//go:build darwin
// +build darwin

// Package menubar provides the menu bar application logic for macOS.
package menubar

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"fyne.io/systray"
	"github.com/platinummonkey/legible/internal/logger"
)

// App represents the menu bar application.
type App struct {
	// Menu items
	mStatus       *systray.MenuItem
	mStartSync    *systray.MenuItem
	mStopSync     *systray.MenuItem
	mOpenOutput   *systray.MenuItem
	mPreferences  *systray.MenuItem
	mQuit         *systray.MenuItem

	// Application state
	isRunning     bool
	outputDir     string
	statusText    string
	daemonAddr    string

	// Daemon communication
	daemonClient  *DaemonClient

	// Channels for menu actions
	quitChan      chan struct{}
}

// Config holds configuration for the menu bar app
type Config struct {
	OutputDir  string
	DaemonAddr string // HTTP address of daemon (e.g., "http://localhost:8080")
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

	return &App{
		outputDir:    cfg.OutputDir,
		daemonAddr:   cfg.DaemonAddr,
		daemonClient: NewDaemonClient(cfg.DaemonAddr),
		statusText:   "Starting...",
		quitChan:     make(chan struct{}),
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
	a.mStatus = systray.AddMenuItem("Status: Checking daemon...", "Current sync status")
	a.mStatus.Disable() // Status is informational only

	systray.AddSeparator()

	a.mStartSync = systray.AddMenuItem("Trigger Sync", "Trigger an immediate sync")
	a.mStopSync = systray.AddMenuItem("Cancel Sync", "Cancel the running sync")
	a.mStopSync.Disable() // Disabled until sync is running

	systray.AddSeparator()

	a.mOpenOutput = systray.AddMenuItem("Open Output Folder", "Open the output directory in Finder")
	a.mPreferences = systray.AddMenuItem("Preferences...", "Configure settings")

	systray.AddSeparator()

	a.mQuit = systray.AddMenuItem("Quit", "Exit the application")

	// Start event loop and status polling
	go a.handleMenuEvents()
	go a.pollDaemonStatus()
}

// onExit is called when the application exits.
func (a *App) onExit() {
	logger.Info("Menu bar application exiting")
	close(a.quitChan)
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

// handlePreferences opens the preferences dialog.
func (a *App) handlePreferences() {
	logger.Info("Preferences clicked")

	// TODO: Implement preferences dialog
	// For now, just log
	logger.Info("Preferences (not yet implemented)")
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
