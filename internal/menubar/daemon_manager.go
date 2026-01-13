//go:build darwin
// +build darwin

package menubar

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/platinummonkey/legible/internal/logger"
)

// DaemonManager manages the daemon process lifecycle
type DaemonManager struct {
	mu sync.Mutex

	// Configuration
	daemonPath    string   // Path to legible binary
	daemonAddr    string   // Daemon HTTP address
	daemonArgs    []string // Additional daemon arguments
	autoLaunch    bool     // Whether to auto-launch daemon

	// Process management
	cmd           *exec.Cmd
	isRunning     bool
	restartCount  int
	maxRestarts   int
	restartDelay  time.Duration

	// Control
	ctx           context.Context
	cancel        context.CancelFunc
	stopChan      chan struct{}
}

// DaemonManagerConfig holds configuration for the daemon manager
type DaemonManagerConfig struct {
	DaemonPath   string   // Path to legible binary (default: find in PATH)
	DaemonAddr   string   // HTTP address for daemon (e.g., "localhost:8080")
	DaemonArgs   []string // Additional arguments to pass to daemon
	AutoLaunch   bool     // Whether to auto-launch daemon (default: true)
	MaxRestarts  int      // Max restart attempts (default: 5)
	RestartDelay time.Duration // Delay between restarts (default: 5s)
}

// NewDaemonManager creates a new daemon manager
func NewDaemonManager(cfg *DaemonManagerConfig) (*DaemonManager, error) {
	if cfg == nil {
		cfg = &DaemonManagerConfig{
			AutoLaunch:   true,
			MaxRestarts:  5,
			RestartDelay: 5 * time.Second,
		}
	}

	// Find daemon binary if not specified
	daemonPath := cfg.DaemonPath
	if daemonPath == "" {
		// Try to find legible in PATH
		path, err := exec.LookPath("legible")
		if err != nil {
			// Try relative to menu bar binary
			menubarPath, _ := os.Executable()
			if menubarPath != "" {
				// Check in same directory
				daemonPath = menubarPath[:len(menubarPath)-len("legible-menubar")] + "legible"
				if _, err := os.Stat(daemonPath); os.IsNotExist(err) {
					return nil, fmt.Errorf("daemon binary not found: tried PATH and %s", daemonPath)
				}
			} else {
				return nil, fmt.Errorf("daemon binary not found in PATH")
			}
		} else {
			daemonPath = path
		}
	}

	// Verify daemon binary exists
	if _, err := os.Stat(daemonPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("daemon binary not found at %s", daemonPath)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &DaemonManager{
		daemonPath:   daemonPath,
		daemonAddr:   cfg.DaemonAddr,
		daemonArgs:   cfg.DaemonArgs,
		autoLaunch:   cfg.AutoLaunch,
		maxRestarts:  cfg.MaxRestarts,
		restartDelay: cfg.RestartDelay,
		ctx:          ctx,
		cancel:       cancel,
		stopChan:     make(chan struct{}),
	}, nil
}

// Start starts managing the daemon
func (dm *DaemonManager) Start() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if !dm.autoLaunch {
		logger.Info("Daemon auto-launch disabled")
		return nil
	}

	logger.Info("Starting daemon manager", "daemon_path", dm.daemonPath)

	// Start daemon process
	if err := dm.startDaemon(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// Start monitoring in background
	go dm.monitor()

	return nil
}

// Stop stops the daemon gracefully
func (dm *DaemonManager) Stop() error {
	logger.Info("Stopping daemon manager")

	// Cancel context
	dm.cancel()
	close(dm.stopChan)

	dm.mu.Lock()
	defer dm.mu.Unlock()

	if dm.cmd != nil && dm.cmd.Process != nil {
		logger.Info("Sending SIGTERM to daemon", "pid", dm.cmd.Process.Pid)

		// Send SIGTERM for graceful shutdown
		if err := dm.cmd.Process.Signal(os.Interrupt); err != nil {
			logger.Warn("Failed to send SIGTERM", "error", err)
			// Force kill if graceful shutdown fails
			dm.cmd.Process.Kill()
		}

		// Wait for process to exit (with timeout)
		done := make(chan error, 1)
		go func() {
			done <- dm.cmd.Wait()
		}()

		select {
		case <-time.After(10 * time.Second):
			logger.Warn("Daemon did not exit gracefully, killing")
			dm.cmd.Process.Kill()
		case err := <-done:
			if err != nil {
				logger.Info("Daemon exited", "error", err)
			} else {
				logger.Info("Daemon exited gracefully")
			}
		}

		dm.isRunning = false
		dm.cmd = nil
	}

	return nil
}

// IsRunning returns whether the daemon is running
func (dm *DaemonManager) IsRunning() bool {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	return dm.isRunning
}

// startDaemon starts the daemon process (must be called with lock held)
func (dm *DaemonManager) startDaemon() error {
	if dm.isRunning {
		return nil
	}

	// Build daemon command
	args := []string{"daemon"}

	// Add health check address (strip http:// or https:// prefix if present)
	if dm.daemonAddr != "" {
		healthAddr := dm.daemonAddr
		healthAddr = strings.TrimPrefix(healthAddr, "http://")
		healthAddr = strings.TrimPrefix(healthAddr, "https://")
		args = append(args, "--health-addr", healthAddr)
	}

	// Add additional arguments
	args = append(args, dm.daemonArgs...)

	logger.Info("Starting daemon process", "args", args)

	// Create command
	cmd := exec.CommandContext(dm.ctx, dm.daemonPath, args...)

	// Capture output
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	dm.cmd = cmd
	dm.isRunning = true

	logger.Info("Daemon started", "pid", cmd.Process.Pid)

	return nil
}

// monitor monitors the daemon health and restarts if needed
func (dm *DaemonManager) monitor() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-dm.stopChan:
			return
		case <-dm.ctx.Done():
			return
		case <-ticker.C:
			dm.checkHealth()
		}
	}
}

// checkHealth checks if daemon is healthy and restarts if needed
func (dm *DaemonManager) checkHealth() {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// Check if process is still running
	if dm.cmd != nil && dm.cmd.Process != nil {
		// Try to signal process (signal 0 checks existence)
		err := dm.cmd.Process.Signal(os.Signal(nil))
		if err != nil {
			// Process died
			logger.Warn("Daemon process died", "error", err)
			dm.isRunning = false
			dm.cmd = nil

			// Attempt restart
			dm.attemptRestart()
		}
	}
}

// attemptRestart attempts to restart the daemon (must be called with lock held)
func (dm *DaemonManager) attemptRestart() {
	if dm.restartCount >= dm.maxRestarts {
		logger.Error("Max restart attempts reached, giving up", "count", dm.restartCount)
		return
	}

	dm.restartCount++
	logger.Info("Attempting to restart daemon",
		"attempt", dm.restartCount,
		"max", dm.maxRestarts,
	)

	// Wait before restarting
	time.Sleep(dm.restartDelay)

	if err := dm.startDaemon(); err != nil {
		logger.Error("Failed to restart daemon", "error", err)
	} else {
		logger.Info("Daemon restarted successfully")
		// Reset restart count on successful start
		dm.restartCount = 0
	}
}
