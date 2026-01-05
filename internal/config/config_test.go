package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	// Create a temporary directory for test output
	tmpDir := t.TempDir()

	// Set environment variable for output dir to avoid creating in actual home
	t.Setenv("LEGIBLE_OUTPUT_DIR", tmpDir)
	t.Setenv("LEGIBLE_STATE_FILE", filepath.Join(tmpDir, "state.json"))
	// Set HOME to temp dir to avoid loading user's ~/.legible.yaml
	t.Setenv("HOME", tmpDir)

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.OutputDir != tmpDir {
		t.Errorf("expected OutputDir = %s, got %s", tmpDir, cfg.OutputDir)
	}

	if cfg.OCREnabled != true {
		t.Errorf("expected OCREnabled = true, got %t", cfg.OCREnabled)
	}

	if cfg.OCRLanguages != "eng" {
		t.Errorf("expected OCRLanguages = eng, got %s", cfg.OCRLanguages)
	}

	if cfg.LogLevel != "info" {
		t.Errorf("expected LogLevel = info, got %s", cfg.LogLevel)
	}

	if cfg.SyncInterval != 0 {
		t.Errorf("expected SyncInterval = 0, got %s", cfg.SyncInterval)
	}
}

func TestLoad_EnvironmentVariables(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("LEGIBLE_OUTPUT_DIR", tmpDir)
	t.Setenv("LEGIBLE_STATE_FILE", filepath.Join(tmpDir, "state.json"))
	t.Setenv("LEGIBLE_OCR_ENABLED", "false")
	t.Setenv("LEGIBLE_OCR_LANGUAGES", "fra")
	t.Setenv("LEGIBLE_LOG_LEVEL", "debug")
	t.Setenv("LEGIBLE_LABELS", "work,personal")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.OCREnabled != false {
		t.Errorf("expected OCREnabled = false, got %t", cfg.OCREnabled)
	}

	if cfg.OCRLanguages != "fra" {
		t.Errorf("expected OCRLanguages = fra, got %s", cfg.OCRLanguages)
	}

	if cfg.LogLevel != "debug" {
		t.Errorf("expected LogLevel = debug, got %s", cfg.LogLevel)
	}

	// Viper parses comma-separated env vars as slices
	expectedLabels := []string{"work,personal"}
	if len(cfg.Labels) != len(expectedLabels) {
		t.Errorf("expected Labels length = %d, got %d", len(expectedLabels), len(cfg.Labels))
	}
}

func TestLoad_ConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-config.yaml")

	configContent := `
output-dir: ` + tmpDir + `
labels:
  - work
  - personal
ocr-enabled: false
ocr-languages: "deu"
log-level: warn
state-file: ` + filepath.Join(tmpDir, "state.json") + `
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := Load(configFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.OutputDir != tmpDir {
		t.Errorf("expected OutputDir = %s, got %s", tmpDir, cfg.OutputDir)
	}

	expectedLabels := []string{"work", "personal"}
	if len(cfg.Labels) != len(expectedLabels) {
		t.Errorf("expected %d labels, got %d", len(expectedLabels), len(cfg.Labels))
	}

	if cfg.OCREnabled != false {
		t.Errorf("expected OCREnabled = false, got %t", cfg.OCREnabled)
	}

	if cfg.OCRLanguages != "deu" {
		t.Errorf("expected OCRLanguages = deu, got %s", cfg.OCRLanguages)
	}

	if cfg.LogLevel != "warn" {
		t.Errorf("expected LogLevel = warn, got %s", cfg.LogLevel)
	}
}

func TestValidate_InvalidLogLevel(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		OutputDir:    tmpDir,
		StateFile:    filepath.Join(tmpDir, "state.json"),
		LogLevel:     "invalid",
		OCREnabled:   false,
		OCRLanguages: "eng",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for invalid log level")
	}

	if !strings.Contains(err.Error(), "invalid log-level") {
		t.Errorf("expected error about invalid log-level, got: %v", err)
	}
}

func TestValidate_EmptyOutputDir(t *testing.T) {
	cfg := &Config{
		OutputDir:  "",
		StateFile:  "/tmp/state.json",
		LogLevel:   "info",
		OCREnabled: false,
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for empty output-dir")
	}

	if !strings.Contains(err.Error(), "output-dir cannot be empty") {
		t.Errorf("expected error about empty output-dir, got: %v", err)
	}
}

func TestValidate_OCRLanguagesRequired(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		OutputDir:    tmpDir,
		StateFile:    filepath.Join(tmpDir, "state.json"),
		LogLevel:     "info",
		OCREnabled:   true,
		OCRLanguages: "",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for empty OCR languages when OCR enabled")
	}

	if !strings.Contains(err.Error(), "ocr-languages cannot be empty") {
		t.Errorf("expected error about empty ocr-languages, got: %v", err)
	}
}

func TestValidate_DaemonModeRequiresSyncInterval(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		OutputDir:    tmpDir,
		StateFile:    filepath.Join(tmpDir, "state.json"),
		LogLevel:     "info",
		OCREnabled:   false,
		OCRLanguages: "eng",
		DaemonMode:   true,
		SyncInterval: 0,
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for daemon mode with zero sync interval")
	}

	if !strings.Contains(err.Error(), "sync-interval must be positive") {
		t.Errorf("expected error about sync-interval, got: %v", err)
	}
}

func TestValidate_ValidConfiguration(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		OutputDir:       tmpDir,
		StateFile:       filepath.Join(tmpDir, "state.json"),
		LogLevel:        "info",
		OCREnabled:      true,
		OCRLanguages:    "eng",
		Labels:          []string{"work"},
		SyncInterval:    5 * time.Minute,
		DaemonMode:      true,
		RemarkableToken: "test-token-12345",
		LLM: LLMConfig{
			Provider:    "ollama",
			Model:       "llava",
			Endpoint:    "http://localhost:11434",
			MaxRetries:  3,
			Temperature: 0.0,
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("expected valid configuration, got error: %v", err)
	}
}

func TestValidate_HomeDirectoryExpansion(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot determine home directory")
	}

	cfg := &Config{
		OutputDir:    "~/test-legible",
		StateFile:    "~/.test-legible-state.json",
		LogLevel:     "info",
		OCREnabled:   false,
		OCRLanguages: "eng",
	}

	err = cfg.Validate()
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	expectedOutputDir := filepath.Join(home, "test-legible")
	if cfg.OutputDir != expectedOutputDir {
		t.Errorf("expected OutputDir = %s, got %s", expectedOutputDir, cfg.OutputDir)
	}

	expectedStateFile := filepath.Join(home, ".test-legible-state.json")
	if cfg.StateFile != expectedStateFile {
		t.Errorf("expected StateFile = %s, got %s", expectedStateFile, cfg.StateFile)
	}

	// Clean up created test directories
	_ = os.RemoveAll(cfg.OutputDir)
	_ = os.Remove(cfg.StateFile)
}

func TestString_RedactsToken(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		OutputDir:       tmpDir,
		StateFile:       filepath.Join(tmpDir, "state.json"),
		LogLevel:        "info",
		OCREnabled:      true,
		OCRLanguages:    "eng",
		RemarkableToken: "secret-token-12345",
	}

	str := cfg.String()

	if strings.Contains(str, "secret-token-12345") {
		t.Error("String() should redact the full token")
	}

	if !strings.Contains(str, "***2345") {
		t.Error("String() should show last 4 characters of token")
	}
}

func TestString_NoToken(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		OutputDir:       tmpDir,
		StateFile:       filepath.Join(tmpDir, "state.json"),
		LogLevel:        "info",
		OCREnabled:      false,
		OCRLanguages:    "eng",
		RemarkableToken: "",
	}

	str := cfg.String()

	if !strings.Contains(str, "not set") {
		t.Error("String() should indicate token is not set")
	}
}

func TestLoad_KeychainConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-keychain-config.yaml")

	// Set a dummy API key for the test
	t.Setenv("OPENAI_API_KEY", "sk-test-dummy-key")

	configContent := `
output-dir: ` + tmpDir + `
state-file: ` + filepath.Join(tmpDir, "state.json") + `
ocr-enabled: true
ocr-languages: eng
llm-provider: openai
llm-model: gpt-4o-mini
llm-use-keychain: true
llm-keychain-service-prefix: myapp
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := Load(configFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.LLM.Provider != "openai" {
		t.Errorf("expected LLM.Provider = openai, got %s", cfg.LLM.Provider)
	}

	if cfg.LLM.Model != "gpt-4o-mini" {
		t.Errorf("expected LLM.Model = gpt-4o-mini, got %s", cfg.LLM.Model)
	}

	if cfg.LLM.UseKeychain != true {
		t.Errorf("expected LLM.UseKeychain = true, got %t", cfg.LLM.UseKeychain)
	}

	if cfg.LLM.KeychainServicePrefix != "myapp" {
		t.Errorf("expected LLM.KeychainServicePrefix = myapp, got %s", cfg.LLM.KeychainServicePrefix)
	}
}

func TestLoad_KeychainDefaults(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("LEGIBLE_OUTPUT_DIR", tmpDir)
	t.Setenv("LEGIBLE_STATE_FILE", filepath.Join(tmpDir, "state.json"))

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Keychain should be disabled by default
	if cfg.LLM.UseKeychain != false {
		t.Errorf("expected LLM.UseKeychain = false by default, got %t", cfg.LLM.UseKeychain)
	}

	// Default prefix should be "legible"
	if cfg.LLM.KeychainServicePrefix != "legible" {
		t.Errorf("expected LLM.KeychainServicePrefix = legible by default, got %s", cfg.LLM.KeychainServicePrefix)
	}
}

func TestLoad_KeychainEnvironmentVariables(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("LEGIBLE_OUTPUT_DIR", tmpDir)
	t.Setenv("LEGIBLE_STATE_FILE", filepath.Join(tmpDir, "state.json"))
	t.Setenv("LEGIBLE_LLM_USE_KEYCHAIN", "true")
	t.Setenv("LEGIBLE_LLM_KEYCHAIN_SERVICE_PREFIX", "customprefix")
	t.Setenv("LEGIBLE_LLM_PROVIDER", "ollama") // Use ollama to avoid API key requirement

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.LLM.UseKeychain != true {
		t.Errorf("expected LLM.UseKeychain = true, got %t", cfg.LLM.UseKeychain)
	}

	if cfg.LLM.KeychainServicePrefix != "customprefix" {
		t.Errorf("expected LLM.KeychainServicePrefix = customprefix, got %s", cfg.LLM.KeychainServicePrefix)
	}
}

func TestLoadAPIKeyForProvider_EnvironmentVariables(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		envKey   string
		envValue string
		expected string
	}{
		{
			name:     "OpenAI from env",
			provider: "openai",
			envKey:   "OPENAI_API_KEY",
			envValue: "sk-test-key",
			expected: "sk-test-key",
		},
		{
			name:     "Anthropic from env",
			provider: "anthropic",
			envKey:   "ANTHROPIC_API_KEY",
			envValue: "sk-ant-test",
			expected: "sk-ant-test",
		},
		{
			name:     "Google from GOOGLE_API_KEY",
			provider: "google",
			envKey:   "GOOGLE_API_KEY",
			envValue: "google-key",
			expected: "google-key",
		},
		{
			name:     "Ollama no key needed",
			provider: "ollama",
			envKey:   "",
			envValue: "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all API key env vars
			_ = os.Unsetenv("OPENAI_API_KEY")
			_ = os.Unsetenv("ANTHROPIC_API_KEY")
			_ = os.Unsetenv("GOOGLE_API_KEY")
			_ = os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")

			// Set the test env var
			if tt.envKey != "" {
				t.Setenv(tt.envKey, tt.envValue)
			}

			// Call with keychain disabled
			result := loadAPIKeyForProvider(tt.provider, false, "legible")

			if result != tt.expected {
				t.Errorf("expected key %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestLoadFromKeychain_NonMacOS(t *testing.T) {
	// This test verifies that loadFromKeychain returns empty string on non-macOS
	// We can't actually test macOS keychain functionality without running on macOS
	// with actual keychain entries, so we just verify the platform check works

	if isMacOS() {
		t.Skip("Skipping non-macOS test on macOS platform")
	}

	// Should return empty string on non-macOS platforms
	result := loadFromKeychain("openai", "legible")
	if result != "" {
		t.Errorf("expected empty string on non-macOS platform, got %q", result)
	}
}

func TestString_IncludesKeychainSettings(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		OutputDir:    tmpDir,
		StateFile:    filepath.Join(tmpDir, "state.json"),
		LogLevel:     "info",
		OCREnabled:   true,
		OCRLanguages: "eng",
		LLM: LLMConfig{
			Provider:              "openai",
			Model:                 "gpt-4o",
			UseKeychain:           true,
			KeychainServicePrefix: "myapp",
		},
	}

	str := cfg.String()

	if !strings.Contains(str, "UseKeychain: true") {
		t.Error("String() should include UseKeychain setting")
	}

	if !strings.Contains(str, "KeychainServicePrefix: myapp") {
		t.Error("String() should include KeychainServicePrefix setting")
	}
}
