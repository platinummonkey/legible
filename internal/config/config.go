// Package config provides configuration management for the remarkable-sync application.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration settings for the remarkable-sync application.
// Configuration precedence: CLI flags > Environment variables > Config file > Defaults
type Config struct {
	// OutputDir is the directory where downloaded and processed files will be saved
	OutputDir string

	// Labels filters documents by reMarkable labels (empty means sync all documents)
	Labels []string

	// OCREnabled determines whether OCR processing should be performed
	OCREnabled bool

	// OCRLanguages specifies the languages to use for OCR (e.g., "eng", "eng+fra")
	OCRLanguages string

	// SyncInterval is the duration between sync operations in daemon mode (0 = run once)
	SyncInterval time.Duration

	// StateFile is the path to the sync state persistence file
	StateFile string

	// TesseractPath is the path to the Tesseract executable (empty = use system PATH)
	TesseractPath string

	// LogLevel controls logging verbosity (debug, info, warn, error)
	LogLevel string

	// RemarkableToken is the authentication token for the reMarkable API
	RemarkableToken string

	// DaemonMode enables continuous sync operation
	DaemonMode bool
}

// Load reads configuration from multiple sources and returns a Config instance.
// Sources are checked in this order: CLI flags > env vars > config file > defaults
func Load(configFile string) (*Config, error) {
	v := viper.New()

	// Set default values
	setDefaults(v)

	// Set up config file
	if configFile != "" {
		v.SetConfigFile(configFile)
	} else {
		// Look for config in home directory
		home, err := os.UserHomeDir()
		if err == nil {
			v.AddConfigPath(home)
			v.SetConfigName(".remarkable-sync")
			v.SetConfigType("yaml")
		}
	}

	// Read config file if it exists (optional)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found is OK - we'll use env vars and defaults
	}

	// Enable environment variable support
	v.SetEnvPrefix("REMARKABLE_SYNC")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	// Build config struct
	config := &Config{
		OutputDir:       v.GetString("output-dir"),
		Labels:          v.GetStringSlice("labels"),
		OCREnabled:      v.GetBool("ocr-enabled"),
		OCRLanguages:    v.GetString("ocr-languages"),
		SyncInterval:    v.GetDuration("sync-interval"),
		StateFile:       v.GetString("state-file"),
		TesseractPath:   v.GetString("tesseract-path"),
		LogLevel:        v.GetString("log-level"),
		RemarkableToken: v.GetString("remarkable-token"),
		DaemonMode:      v.GetBool("daemon-mode"),
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Get user home directory for default paths
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}

	defaultOutputDir := filepath.Join(home, "remarkable-sync")
	defaultStateFile := filepath.Join(home, ".remarkable-sync-state.json")

	v.SetDefault("output-dir", defaultOutputDir)
	v.SetDefault("labels", []string{})
	v.SetDefault("ocr-enabled", true)
	v.SetDefault("ocr-languages", "eng")
	v.SetDefault("sync-interval", 0*time.Second) // 0 = run once
	v.SetDefault("state-file", defaultStateFile)
	v.SetDefault("tesseract-path", "")
	v.SetDefault("log-level", "info")
	v.SetDefault("remarkable-token", "")
	v.SetDefault("daemon-mode", false)
}

// Validate checks that the configuration is valid and internally consistent
func (c *Config) Validate() error {
	// Validate output directory
	if c.OutputDir == "" {
		return fmt.Errorf("output-dir cannot be empty")
	}

	// Expand home directory if present
	if strings.HasPrefix(c.OutputDir, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to expand home directory in output-dir: %w", err)
		}
		c.OutputDir = filepath.Join(home, c.OutputDir[2:])
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(c.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", c.OutputDir, err)
	}

	// Validate state file path
	if c.StateFile == "" {
		return fmt.Errorf("state-file cannot be empty")
	}

	// Expand home directory in state file path
	if strings.HasPrefix(c.StateFile, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to expand home directory in state-file: %w", err)
		}
		c.StateFile = filepath.Join(home, c.StateFile[2:])
	}

	// Create state file directory if it doesn't exist
	stateDir := filepath.Dir(c.StateFile)
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state file directory %s: %w", stateDir, err)
	}

	// Validate log level
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[strings.ToLower(c.LogLevel)] {
		return fmt.Errorf("invalid log-level %q, must be one of: debug, info, warn, error", c.LogLevel)
	}
	c.LogLevel = strings.ToLower(c.LogLevel)

	// Validate OCR settings
	if c.OCREnabled {
		if c.OCRLanguages == "" {
			return fmt.Errorf("ocr-languages cannot be empty when OCR is enabled")
		}

		// If tesseract path is specified, verify it exists
		if c.TesseractPath != "" {
			if _, err := os.Stat(c.TesseractPath); err != nil {
				return fmt.Errorf("tesseract-path %q does not exist: %w", c.TesseractPath, err)
			}
		}
	}

	// Validate sync interval for daemon mode
	if c.DaemonMode && c.SyncInterval <= 0 {
		return fmt.Errorf("sync-interval must be positive when daemon-mode is enabled")
	}

	return nil
}

// String returns a string representation of the configuration (with sensitive data redacted)
func (c *Config) String() string {
	token := "not set"
	if c.RemarkableToken != "" {
		token = "***" + c.RemarkableToken[len(c.RemarkableToken)-4:]
	}

	return fmt.Sprintf(`Configuration:
  OutputDir: %s
  Labels: %v
  OCREnabled: %t
  OCRLanguages: %s
  SyncInterval: %s
  StateFile: %s
  TesseractPath: %s
  LogLevel: %s
  RemarkableToken: %s
  DaemonMode: %t`,
		c.OutputDir,
		c.Labels,
		c.OCREnabled,
		c.OCRLanguages,
		c.SyncInterval,
		c.StateFile,
		c.TesseractPath,
		c.LogLevel,
		token,
		c.DaemonMode,
	)
}
