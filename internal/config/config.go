// Package config provides configuration management for the legible application.
package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration settings for the legible application.
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

	// LogLevel controls logging verbosity (debug, info, warn, error)
	LogLevel string

	// RemarkableToken is the authentication token for the reMarkable API
	RemarkableToken string

	// DaemonMode enables continuous sync operation
	DaemonMode bool

	// LLM configuration for OCR processing
	LLM LLMConfig
}

// LLMConfig holds configuration for LLM-based OCR providers
type LLMConfig struct {
	// Provider is the LLM provider to use (ollama, openai, anthropic, google)
	Provider string

	// Model is the specific model to use for OCR
	Model string

	// Endpoint is the API endpoint (primarily for Ollama)
	Endpoint string

	// APIKey is the API key for cloud providers (typically from env vars or keychain)
	// This will be populated from:
	// 1. macOS Keychain (if UseKeychain is true)
	// 2. Environment variables:
	//    - OPENAI_API_KEY for OpenAI
	//    - ANTHROPIC_API_KEY for Anthropic
	//    - GOOGLE_API_KEY or GOOGLE_APPLICATION_CREDENTIALS for Google
	APIKey string

	// MaxRetries is the maximum number of retry attempts for API calls
	MaxRetries int

	// Temperature controls randomness (0.0 = deterministic, recommended for OCR)
	Temperature float64

	// UseKeychain enables macOS Keychain lookup for API keys (macOS only)
	UseKeychain bool

	// KeychainServicePrefix is the prefix for keychain service names
	// Service names will be: {prefix}-{provider} (e.g., "legible-openai")
	KeychainServicePrefix string
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
			v.SetConfigName(".legible")
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
	v.SetEnvPrefix("LEGIBLE")
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
		LogLevel:        v.GetString("log-level"),
		RemarkableToken: v.GetString("api-token"),
		DaemonMode:      v.GetBool("daemon-mode"),
		LLM: LLMConfig{
			Provider:              v.GetString("llm-provider"),
			Model:                 v.GetString("llm-model"),
			Endpoint:              v.GetString("llm-endpoint"),
			MaxRetries:            v.GetInt("llm-max-retries"),
			Temperature:           v.GetFloat64("llm-temperature"),
			UseKeychain:           v.GetBool("llm-use-keychain"),
			KeychainServicePrefix: v.GetString("llm-keychain-service-prefix"),
		},
	}

	// Load API keys from keychain or environment variables based on provider
	config.LLM.APIKey = loadAPIKeyForProvider(config.LLM.Provider, config.LLM.UseKeychain, config.LLM.KeychainServicePrefix)

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

	defaultOutputDir := filepath.Join(home, "legible")
	defaultStateFile := filepath.Join(home, ".legible-state.json")

	v.SetDefault("output-dir", defaultOutputDir)
	v.SetDefault("labels", []string{})
	v.SetDefault("ocr-enabled", true)
	v.SetDefault("ocr-languages", "eng")
	v.SetDefault("sync-interval", 0*time.Second) // 0 = run once
	v.SetDefault("state-file", defaultStateFile)
	v.SetDefault("log-level", "info")
	v.SetDefault("api-token", "")
	v.SetDefault("daemon-mode", false)

	// LLM defaults (Ollama by default for backward compatibility)
	v.SetDefault("llm-provider", "ollama")
	v.SetDefault("llm-model", "llava")
	v.SetDefault("llm-endpoint", "http://localhost:11434")
	v.SetDefault("llm-max-retries", 3)
	v.SetDefault("llm-temperature", 0.0)
	v.SetDefault("llm-use-keychain", false)
	v.SetDefault("llm-keychain-service-prefix", "legible")
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
	}

	// Validate sync interval for daemon mode
	if c.DaemonMode && c.SyncInterval <= 0 {
		return fmt.Errorf("sync-interval must be positive when daemon-mode is enabled")
	}

	// Validate LLM configuration
	if c.OCREnabled {
		if err := c.validateLLMConfig(); err != nil {
			return fmt.Errorf("invalid LLM configuration: %w", err)
		}
	}

	return nil
}

// validateLLMConfig validates the LLM provider configuration
func (c *Config) validateLLMConfig() error {
	// Validate provider
	validProviders := map[string]bool{
		"ollama":    true,
		"openai":    true,
		"anthropic": true,
		"google":    true,
	}
	if !validProviders[strings.ToLower(c.LLM.Provider)] {
		return fmt.Errorf("invalid llm-provider %q, must be one of: ollama, openai, anthropic, google", c.LLM.Provider)
	}
	c.LLM.Provider = strings.ToLower(c.LLM.Provider)

	// Validate model is set
	if c.LLM.Model == "" {
		return fmt.Errorf("llm-model cannot be empty when OCR is enabled")
	}

	// For Ollama, validate endpoint
	if c.LLM.Provider == "ollama" && c.LLM.Endpoint == "" {
		return fmt.Errorf("llm-endpoint cannot be empty for Ollama provider")
	}

	// For cloud providers, validate API key is available
	if c.LLM.Provider != "ollama" && c.LLM.APIKey == "" {
		return fmt.Errorf("API key not found for provider %s, check environment variables", c.LLM.Provider)
	}

	// Validate temperature range
	if c.LLM.Temperature < 0.0 || c.LLM.Temperature > 2.0 {
		return fmt.Errorf("llm-temperature must be between 0.0 and 2.0, got %f", c.LLM.Temperature)
	}

	// Validate max retries
	if c.LLM.MaxRetries < 0 {
		return fmt.Errorf("llm-max-retries must be non-negative, got %d", c.LLM.MaxRetries)
	}

	return nil
}

// loadAPIKeyForProvider loads the appropriate API key from keychain or environment variables
func loadAPIKeyForProvider(provider string, useKeychain bool, keychainPrefix string) string {
	// Try keychain first if enabled (macOS only)
	if useKeychain {
		if key := loadFromKeychain(provider, keychainPrefix); key != "" {
			return key
		}
	}

	// Fall back to environment variables
	switch strings.ToLower(provider) {
	case "openai":
		return os.Getenv("OPENAI_API_KEY")
	case "anthropic":
		return os.Getenv("ANTHROPIC_API_KEY")
	case "google":
		// Try GOOGLE_API_KEY first, then GOOGLE_APPLICATION_CREDENTIALS
		if key := os.Getenv("GOOGLE_API_KEY"); key != "" {
			return key
		}
		return os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	default:
		// Ollama and others don't need API keys
		return ""
	}
}

// loadFromKeychain attempts to retrieve an API key from macOS Keychain
// Service name format: {prefix}-{provider} (e.g., "legible-openai")
// Returns empty string if not found or on non-macOS platforms
func loadFromKeychain(provider, prefix string) string {
	// Only attempt on macOS
	if !isMacOS() {
		return ""
	}

	serviceName := fmt.Sprintf("%s-%s", prefix, strings.ToLower(provider))

	// Use security command to retrieve password
	// security find-generic-password -s "service-name" -w
	cmd := exec.Command("security", "find-generic-password", "-s", serviceName, "-w")
	output, err := cmd.Output()
	if err != nil {
		// Key not found or other error - silently fail and fall back to env vars
		return ""
	}

	// Trim whitespace and newlines
	key := strings.TrimSpace(string(output))
	return key
}

// isMacOS checks if the current platform is macOS
func isMacOS() bool {
	return runtime.GOOS == "darwin"
}

// String returns a string representation of the configuration (with sensitive data redacted)
func (c *Config) String() string {
	token := "not set"
	if c.RemarkableToken != "" {
		token = "***" + c.RemarkableToken[len(c.RemarkableToken)-4:]
	}

	apiKey := "not set"
	if c.LLM.APIKey != "" {
		if len(c.LLM.APIKey) > 8 {
			apiKey = "***" + c.LLM.APIKey[len(c.LLM.APIKey)-4:]
		} else {
			apiKey = "***"
		}
	}

	return fmt.Sprintf(`Configuration:
  OutputDir: %s
  Labels: %v
  OCREnabled: %t
  OCRLanguages: %s
  SyncInterval: %s
  StateFile: %s
  LogLevel: %s
  RemarkableToken: %s
  DaemonMode: %t
  LLM:
    Provider: %s
    Model: %s
    Endpoint: %s
    APIKey: %s
    MaxRetries: %d
    Temperature: %.2f
    UseKeychain: %t
    KeychainServicePrefix: %s`,
		c.OutputDir,
		c.Labels,
		c.OCREnabled,
		c.OCRLanguages,
		c.SyncInterval,
		c.StateFile,
		c.LogLevel,
		token,
		c.DaemonMode,
		c.LLM.Provider,
		c.LLM.Model,
		c.LLM.Endpoint,
		apiKey,
		c.LLM.MaxRetries,
		c.LLM.Temperature,
		c.LLM.UseKeychain,
		c.LLM.KeychainServicePrefix,
	)
}
