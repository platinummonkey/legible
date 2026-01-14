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
	stopping      bool          // Set to true when Stop() is called
	restartCount  int
	maxRestarts   int
	restartDelay  time.Duration
	processDied   chan struct{} // Closed when daemon process exits

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

	// Apply defaults for zero-value fields
	if cfg.MaxRestarts == 0 {
		cfg.MaxRestarts = 5
	}
	if cfg.RestartDelay == 0 {
		cfg.RestartDelay = 5 * time.Second
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

	// Don't start if already running
	if dm.isRunning {
		logger.Info("Daemon already running")
		return nil
	}

	// Reset stopping flag in case we're restarting after a stop
	dm.stopping = false

	// Recreate context if it was cancelled
	if dm.ctx.Err() != nil {
		dm.ctx, dm.cancel = context.WithCancel(context.Background())
		dm.stopChan = make(chan struct{})
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

	// Set stopping flag to prevent monitor from restarting
	dm.mu.Lock()
	dm.stopping = true
	dm.mu.Unlock()

	// Cancel context (stops monitor loop)
	dm.cancel()

	dm.mu.Lock()
	processDied := dm.processDied // Get channel before unlocking
	pid := 0
	if dm.cmd != nil && dm.cmd.Process != nil {
		pid = dm.cmd.Process.Pid
		logger.Info("Sending SIGTERM to daemon", "pid", pid)

		// Send SIGTERM for graceful shutdown
		if err := dm.cmd.Process.Signal(os.Interrupt); err != nil {
			logger.Warn("Failed to send SIGTERM", "error", err)
			// Force kill if graceful shutdown fails
			dm.cmd.Process.Kill()
		}
	}
	dm.mu.Unlock()

	// Wait for process to exit (with timeout)
	// The processDied channel will be closed when cmd.Wait() completes
	if processDied != nil {
		select {
		case <-time.After(10 * time.Second):
			dm.mu.Lock()
			if dm.cmd != nil && dm.cmd.Process != nil {
				logger.Warn("Daemon did not exit gracefully, force killing", "pid", dm.cmd.Process.Pid)
				dm.cmd.Process.Kill()
				dm.mu.Unlock()
				// Wait a bit more for the kill to take effect
				select {
				case <-processDied:
					logger.Info("Daemon killed")
				case <-time.After(2 * time.Second):
					logger.Error("Daemon did not exit after force kill")
				}
			} else {
				dm.mu.Unlock()
			}
		case <-processDied:
			logger.Info("Daemon exited gracefully", "pid", pid)
		}
	}

	// Close stopChan to stop monitor
	close(dm.stopChan)

	dm.mu.Lock()
	dm.isRunning = false
	dm.cmd = nil
	dm.mu.Unlock()

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
	dm.processDied = make(chan struct{})

	logger.Info("Daemon started", "pid", cmd.Process.Pid)

	// Start goroutine to wait for process exit
	go func() {
		pid := cmd.Process.Pid
		err := cmd.Wait() // This blocks until process exits
		logger.Info("Daemon process exited", "pid", pid, "error", err)
		close(dm.processDied) // Signal that process died
	}()

	return nil
}

// monitor monitors the daemon health and restarts if needed
func (dm *DaemonManager) monitor() {
	for {
		select {
		case <-dm.stopChan:
			return
		case <-dm.ctx.Done():
			return
		case <-dm.processDied:
			// Daemon process died
			dm.mu.Lock()

			// Don't restart if we're stopping
			if dm.stopping {
				logger.Info("Daemon process exited during shutdown")
				dm.isRunning = false
				dm.cmd = nil
				dm.mu.Unlock()
				return
			}

			// Attempt restart
			logger.Warn("Daemon process died, attempting restart", "pid", dm.cmd.Process.Pid)
			dm.isRunning = false
			dm.cmd = nil
			dm.attemptRestart()
			dm.mu.Unlock()

			// If restart succeeded, processDied channel was recreated
			// If restart failed, we'll exit the loop
			if dm.cmd == nil {
				logger.Info("Monitor exiting - daemon not restarted")
				return
			}
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
