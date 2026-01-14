//go:build darwin
// +build darwin

package menubar

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

const launchAgentPlistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.github.platinummonkey.legible</string>
	<key>ProgramArguments</key>
	<array>
		<string>{{.AppPath}}</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<false/>
	<key>ProcessType</key>
	<string>Interactive</string>
	<key>StandardOutPath</key>
	<string>{{.LogPath}}</string>
	<key>StandardErrorPath</key>
	<string>{{.LogPath}}</string>
</dict>
</plist>
`

// LaunchAgentConfig holds the configuration for the launch agent plist
type LaunchAgentConfig struct {
	AppPath string
	LogPath string
}

// GetLaunchAgentPath returns the path to the Launch Agent plist file
func GetLaunchAgentPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, "Library", "LaunchAgents", "com.github.platinummonkey.legible.plist"), nil
}

// IsAutoStartEnabled checks if auto-start is currently enabled
func IsAutoStartEnabled() (bool, error) {
	plistPath, err := GetLaunchAgentPath()
	if err != nil {
		return false, err
	}

	// Check if the plist file exists
	_, err = os.Stat(plistPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check plist file: %w", err)
	}

	return true, nil
}

// EnableAutoStart creates and loads the Launch Agent plist
func EnableAutoStart() error {
	// Find the app bundle path
	appPath, err := getAppBundlePath()
	if err != nil {
		return fmt.Errorf("failed to find app bundle: %w", err)
	}

	// Get log directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	logPath := filepath.Join(homeDir, ".legible", "menubar.log")

	// Ensure log directory exists
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Get plist path
	plistPath, err := GetLaunchAgentPath()
	if err != nil {
		return err
	}

	// Ensure LaunchAgents directory exists
	launchAgentsDir := filepath.Dir(plistPath)
	if err := os.MkdirAll(launchAgentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create LaunchAgents directory: %w", err)
	}

	// Generate plist content from template
	tmpl, err := template.New("plist").Parse(launchAgentPlistTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse plist template: %w", err)
	}

	// Create plist file
	f, err := os.Create(plistPath)
	if err != nil {
		return fmt.Errorf("failed to create plist file: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close plist file: %w", closeErr)
		}
	}()

	// Execute template
	config := LaunchAgentConfig{
		AppPath: appPath,
		LogPath: logPath,
	}
	if err := tmpl.Execute(f, config); err != nil {
		return fmt.Errorf("failed to write plist file: %w", err)
	}

	// Load the launch agent
	cmd := exec.Command("launchctl", "load", plistPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to load launch agent: %w (output: %s)", err, string(output))
	}

	return nil
}

// DisableAutoStart unloads and removes the Launch Agent plist
func DisableAutoStart() error {
	plistPath, err := GetLaunchAgentPath()
	if err != nil {
		return err
	}

	// Check if plist exists
	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		// Already disabled
		return nil
	}

	// Unload the launch agent (ignore errors if not loaded)
	cmd := exec.Command("launchctl", "unload", plistPath)
	_ = cmd.Run() // Ignore error as it may not be loaded

	// Remove the plist file
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove plist file: %w", err)
	}

	return nil
}

// getAppBundlePath finds the path to the Legible.app bundle
func getAppBundlePath() (string, error) {
	// Try common installation locations
	possiblePaths := []string{
		"/Applications/Legible.app/Contents/MacOS/legible-menubar",
		filepath.Join(os.Getenv("HOME"), "Applications", "Legible.app", "Contents", "MacOS", "legible-menubar"),
	}

	// Also try to get the current executable path
	exePath, err := os.Executable()
	if err == nil {
		// If running from .app bundle, use that path
		if filepath.Base(filepath.Dir(exePath)) == "MacOS" &&
			filepath.Base(filepath.Dir(filepath.Dir(exePath))) == "Contents" {
			possiblePaths = append([]string{exePath}, possiblePaths...)
		}
	}

	// Find the first existing path
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// If not found in standard locations, use current executable
	if exePath != "" {
		return exePath, nil
	}

	return "", fmt.Errorf("could not find Legible.app bundle")
}
