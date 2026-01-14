package menubar

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultMenuBarConfig(t *testing.T) {
	cfg := DefaultMenuBarConfig()

	if cfg.DaemonAddr != "http://localhost:8080" {
		t.Errorf("Expected default DaemonAddr to be http://localhost:8080, got %s", cfg.DaemonAddr)
	}

	if cfg.SyncInterval != "30m" {
		t.Errorf("Expected default SyncInterval to be 30m, got %s", cfg.SyncInterval)
	}

	if cfg.OCREnabled != true {
		t.Error("Expected default OCREnabled to be true")
	}

	if cfg.DaemonConfigFile == "" {
		t.Error("Expected default DaemonConfigFile to be set")
	}

	if cfg.AutoStartEnabled != false {
		t.Error("Expected default AutoStartEnabled to be false")
	}
}

func TestSaveAndLoadMenuBarConfig(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "legible-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testPath := filepath.Join(tmpDir, "test-config.yaml")

	// Create test config
	testCfg := &MenuBarConfig{
		DaemonAddr:       "http://localhost:9090",
		SyncInterval:     "1h",
		OCREnabled:       false,
		DaemonConfigFile: "/tmp/test.yaml",
		AutoStartEnabled: true,
	}

	// Save config
	err = SaveMenuBarConfig(testCfg, testPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load config
	loadedCfg, err := LoadMenuBarConfig(testPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify all fields match
	if loadedCfg.DaemonAddr != testCfg.DaemonAddr {
		t.Errorf("DaemonAddr mismatch: expected %s, got %s", testCfg.DaemonAddr, loadedCfg.DaemonAddr)
	}

	if loadedCfg.SyncInterval != testCfg.SyncInterval {
		t.Errorf("SyncInterval mismatch: expected %s, got %s", testCfg.SyncInterval, loadedCfg.SyncInterval)
	}

	if loadedCfg.OCREnabled != testCfg.OCREnabled {
		t.Errorf("OCREnabled mismatch: expected %v, got %v", testCfg.OCREnabled, loadedCfg.OCREnabled)
	}

	if loadedCfg.DaemonConfigFile != testCfg.DaemonConfigFile {
		t.Errorf("DaemonConfigFile mismatch: expected %s, got %s", testCfg.DaemonConfigFile, loadedCfg.DaemonConfigFile)
	}

	if loadedCfg.AutoStartEnabled != testCfg.AutoStartEnabled {
		t.Errorf("AutoStartEnabled mismatch: expected %v, got %v", testCfg.AutoStartEnabled, loadedCfg.AutoStartEnabled)
	}
}

func TestLoadMenuBarConfigNonExistent(t *testing.T) {
	// Try to load config from non-existent path
	// Should return default config, not an error
	cfg, err := LoadMenuBarConfig("/tmp/non-existent-legible-config-test.yaml")

	if err != nil {
		t.Errorf("Expected no error when loading non-existent config, got: %v", err)
	}

	if cfg == nil {
		t.Fatal("Expected default config when loading non-existent file, got nil")
	}

	// Verify it's the default config
	defaultCfg := DefaultMenuBarConfig()
	if cfg.DaemonAddr != defaultCfg.DaemonAddr {
		t.Errorf("Expected default DaemonAddr %s, got %s", defaultCfg.DaemonAddr, cfg.DaemonAddr)
	}
}

func TestSaveMenuBarConfigInvalidPath(t *testing.T) {
	cfg := DefaultMenuBarConfig()

	// Try to save to invalid path (directory that doesn't exist)
	err := SaveMenuBarConfig(cfg, "/non-existent-directory/test-config.yaml")

	if err == nil {
		t.Error("Expected error when saving to invalid path, got nil")
	}
}

func TestGetConfigPath(t *testing.T) {
	path, err := GetConfigPath()

	if err != nil {
		t.Fatalf("GetConfigPath failed: %v", err)
	}

	if path == "" {
		t.Error("GetConfigPath returned empty string")
	}

	// Verify it ends with the expected filename
	if filepath.Base(path) != "menubar-config.yaml" {
		t.Errorf("Expected config filename to be menubar-config.yaml, got %s", filepath.Base(path))
	}

	// Verify directory is .legible
	if filepath.Base(filepath.Dir(path)) != ".legible" {
		t.Errorf("Expected config directory to be .legible, got %s", filepath.Base(filepath.Dir(path)))
	}
}

func TestSaveMenuBarConfigCreatesDirectory(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "legible-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Path with non-existent subdirectory
	testPath := filepath.Join(tmpDir, "subdir", "config.yaml")

	cfg := DefaultMenuBarConfig()

	// Save config - should create the directory
	err = SaveMenuBarConfig(cfg, testPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Verify directory was created
	if _, err := os.Stat(filepath.Dir(testPath)); os.IsNotExist(err) {
		t.Error("Config directory was not created")
	}
}
