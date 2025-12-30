package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCLIBuild tests that the CLI binary can be built
func TestCLIBuild(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI build test in short mode")
	}

	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "remarkable-sync-test")

	cmd := exec.Command("go", "build", "-o", binaryPath, "../cmd/remarkable-sync")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build CLI: %v\nOutput: %s", err, output)
	}

	// Verify binary was created
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Error("Binary should exist after build")
	}

	// Verify binary is executable
	info, err := os.Stat(binaryPath)
	if err != nil {
		t.Fatalf("Failed to stat binary: %v", err)
	}

	mode := info.Mode()
	if mode&0111 == 0 {
		t.Error("Binary should be executable")
	}
}

// TestCLIVersion tests the version command
func TestCLIVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI test in short mode")
	}

	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "remarkable-sync-test")

	// Build binary first
	cmd := exec.Command("go", "build", "-o", binaryPath, "../cmd/remarkable-sync")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build CLI: %v\nOutput: %s", err, output)
	}

	// Run version command
	cmd = exec.Command(binaryPath, "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Version command failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Version:") {
		t.Error("Version output should contain 'Version:'")
	}

	t.Logf("Version output:\n%s", outputStr)
}

// TestCLIHelp tests the help command and flag
func TestCLIHelp(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI test in short mode")
	}

	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "remarkable-sync-test")

	// Build binary
	cmd := exec.Command("go", "build", "-o", binaryPath, "../cmd/remarkable-sync")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build CLI: %v\nOutput: %s", err, output)
	}

	tests := []struct {
		name string
		args []string
	}{
		{"help command", []string{"help"}},
		{"help flag", []string{"--help"}},
		{"short help flag", []string{"-h"}},
		{"sync help", []string{"sync", "--help"}},
		{"auth help", []string{"auth", "--help"}},
		{"daemon help", []string{"daemon", "--help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binaryPath, tt.args...)
			output, _ := cmd.CombinedOutput()

			// Help commands may exit with code 0 or 1 depending on implementation
			outputStr := string(output)

			if !strings.Contains(outputStr, "Usage:") && !strings.Contains(outputStr, "Available Commands") {
				t.Errorf("Help output should contain usage information\nOutput: %s", outputStr)
			}
		})
	}
}

// TestCLISyncCommandFlags tests sync command flag parsing
func TestCLISyncCommandFlags(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI test in short mode")
	}

	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "remarkable-sync-test")

	// Build binary
	cmd := exec.Command("go", "build", "-o", binaryPath, "../cmd/remarkable-sync")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build CLI: %v\nOutput: %s", err, output)
	}

	// Test various flag combinations with --help to see if they're recognized
	tests := []struct {
		name  string
		flags []string
	}{
		{"output flag", []string{"sync", "--output", tmpDir, "--help"}},
		{"labels flag", []string{"sync", "--labels", "work,personal", "--help"}},
		{"no-ocr flag", []string{"sync", "--no-ocr", "--help"}},
		{"log-level flag", []string{"sync", "--log-level", "debug", "--help"}},
		{"force flag", []string{"sync", "--force", "--help"}},
		{"config flag", []string{"sync", "--config", "/tmp/config.yaml", "--help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binaryPath, tt.flags...)
			output, _ := cmd.CombinedOutput()

			outputStr := string(output)

			// Should show help text, not flag parsing errors
			if strings.Contains(outputStr, "unknown flag") {
				t.Errorf("Flag should be recognized\nOutput: %s", outputStr)
			}
		})
	}
}

// TestCLIDaemonCommandFlags tests daemon command flag parsing
func TestCLIDaemonCommandFlags(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI test in short mode")
	}

	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "remarkable-sync-test")

	// Build binary
	cmd := exec.Command("go", "build", "-o", binaryPath, "../cmd/remarkable-sync")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build CLI: %v\nOutput: %s", err, output)
	}

	// Test daemon-specific flags
	tests := []struct {
		name  string
		flags []string
	}{
		{"interval flag", []string{"daemon", "--interval", "10m", "--help"}},
		{"health-addr flag", []string{"daemon", "--health-addr", ":8080", "--help"}},
		{"pid-file flag", []string{"daemon", "--pid-file", "/tmp/daemon.pid", "--help"}},
		{"combined flags", []string{"daemon", "--interval", "5m", "--health-addr", ":9090", "--help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binaryPath, tt.flags...)
			output, _ := cmd.CombinedOutput()

			outputStr := string(output)

			if strings.Contains(outputStr, "unknown flag") {
				t.Errorf("Flag should be recognized\nOutput: %s", outputStr)
			}
		})
	}
}

// TestCLIAuthCommand tests auth command structure
func TestCLIAuthCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI test in short mode")
	}

	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "remarkable-sync-test")

	// Build binary
	cmd := exec.Command("go", "build", "-o", binaryPath, "../cmd/remarkable-sync")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build CLI: %v\nOutput: %s", err, output)
	}

	// Run auth command (will fail without token, but tests command exists)
	cmd = exec.Command(binaryPath, "auth", "--config", filepath.Join(tmpDir, "config.yaml"))
	output, _ := cmd.CombinedOutput()

	outputStr := string(output)

	// Should fail due to missing token, but command should be recognized
	if strings.Contains(outputStr, "unknown command") {
		t.Error("Auth command should be recognized")
	}

	// Should mention authentication or token
	if !strings.Contains(outputStr, "auth") && !strings.Contains(outputStr, "token") && !strings.Contains(outputStr, "Available Commands") {
		t.Logf("Auth command output: %s", outputStr)
	}

	t.Logf("Auth command executed (expected to fail without token)")
}

// TestCLIConfigFile tests config file usage
func TestCLIConfigFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI test in short mode")
	}

	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "remarkable-sync-test")
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create test config file
	configContent := `
output-dir: ` + filepath.Join(tmpDir, "output") + `
log-level: debug
labels:
  - test
ocr-enabled: false
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Build binary
	cmd := exec.Command("go", "build", "-o", binaryPath, "../cmd/remarkable-sync")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build CLI: %v\nOutput: %s", err, output)
	}

	// Test that config file is accepted (use help to avoid execution)
	cmd = exec.Command(binaryPath, "--config", configPath, "sync", "--help")
	output, _ := cmd.CombinedOutput()

	outputStr := string(output)

	// Should not have config parsing errors
	if strings.Contains(outputStr, "invalid configuration") || strings.Contains(outputStr, "failed to load config") {
		t.Errorf("Config file should be valid\nOutput: %s", outputStr)
	}
}

// TestCLIInvalidCommand tests error handling for invalid commands
func TestCLIInvalidCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI test in short mode")
	}

	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "remarkable-sync-test")

	// Build binary
	cmd := exec.Command("go", "build", "-o", binaryPath, "../cmd/remarkable-sync")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build CLI: %v\nOutput: %s", err, output)
	}

	// Test invalid command
	cmd = exec.Command(binaryPath, "invalid-command")
	output, _ := cmd.CombinedOutput()

	outputStr := string(output)
	if !strings.Contains(outputStr, "unknown command") && !strings.Contains(outputStr, "Error") {
		t.Errorf("Should show error for invalid command\nOutput: %s", outputStr)
	}
}
