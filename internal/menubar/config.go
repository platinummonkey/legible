//go:build darwin
// +build darwin

package menubar

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// MenuBarConfig holds menu bar application-specific configuration.
//
//nolint:revive // MenuBarConfig is intentionally descriptive
type MenuBarConfig struct {
	// Daemon configuration file path to pass to the daemon
	DaemonConfigFile string `yaml:"daemon_config_file"`

	// Auto-start the menu bar app on login
	AutoStartEnabled bool `yaml:"auto_start_enabled"`

	// Daemon HTTP address
	DaemonAddr string `yaml:"daemon_addr"`

	// Sync interval for daemon (e.g., "30m", "1h")
	SyncInterval string `yaml:"sync_interval"`

	// Enable OCR
	OCREnabled bool `yaml:"ocr_enabled"`
}

// DefaultMenuBarConfig returns default configuration.
func DefaultMenuBarConfig() *MenuBarConfig {
	homeDir, _ := os.UserHomeDir()
	defaultDaemonConfig := filepath.Join(homeDir, ".legible.yaml")

	return &MenuBarConfig{
		DaemonConfigFile: defaultDaemonConfig,
		AutoStartEnabled: false,
		DaemonAddr:       "http://localhost:8080",
		SyncInterval:     "30m",
		OCREnabled:       true,
	}
}

// LoadMenuBarConfig loads the menu bar configuration from file.
// If the file doesn't exist, returns default configuration.
func LoadMenuBarConfig(path string) (*MenuBarConfig, error) {
	// If no path specified, use default
	if path == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(homeDir, ".legible", "menubar-config.yaml")
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Return defaults if file doesn't exist
		return DefaultMenuBarConfig(), nil
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var cfg MenuBarConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// SaveMenuBarConfig saves the menu bar configuration to file.
func SaveMenuBarConfig(cfg *MenuBarConfig, path string) error {
	// If no path specified, use default
	if path == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(homeDir, ".legible", "menubar-config.yaml")
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetConfigPath returns the default path for the menu bar config file.
func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".legible", "menubar-config.yaml"), nil
}
