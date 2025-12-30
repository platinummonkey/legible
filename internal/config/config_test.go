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
	t.Setenv("REMARKABLE_SYNC_OUTPUT_DIR", tmpDir)
	t.Setenv("REMARKABLE_SYNC_STATE_FILE", filepath.Join(tmpDir, "state.json"))

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

	t.Setenv("REMARKABLE_SYNC_OUTPUT_DIR", tmpDir)
	t.Setenv("REMARKABLE_SYNC_STATE_FILE", filepath.Join(tmpDir, "state.json"))
	t.Setenv("REMARKABLE_SYNC_OCR_ENABLED", "false")
	t.Setenv("REMARKABLE_SYNC_OCR_LANGUAGES", "fra")
	t.Setenv("REMARKABLE_SYNC_LOG_LEVEL", "debug")
	t.Setenv("REMARKABLE_SYNC_LABELS", "work,personal")

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
		OutputDir:    "~/test-remarkable-sync",
		StateFile:    "~/.test-remarkable-sync-state.json",
		LogLevel:     "info",
		OCREnabled:   false,
		OCRLanguages: "eng",
	}

	err = cfg.Validate()
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	expectedOutputDir := filepath.Join(home, "test-remarkable-sync")
	if cfg.OutputDir != expectedOutputDir {
		t.Errorf("expected OutputDir = %s, got %s", expectedOutputDir, cfg.OutputDir)
	}

	expectedStateFile := filepath.Join(home, ".test-remarkable-sync-state.json")
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
