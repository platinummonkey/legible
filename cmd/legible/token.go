package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/spf13/cobra"
)

// tokenCmd represents the token command
var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Manage authentication tokens",
	Long: `Display information about authentication tokens and manage token lifecycle.

Subcommands allow you to view current token status, decode token contents,
and monitor token refresh patterns.`,
}

// tokenInfoCmd shows information about the current tokens
var tokenInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Display current token information",
	Long: `Display detailed information about device and user authentication tokens.

Shows:
- Token type and device information
- Token expiration and validity period
- Time remaining until expiration
- Token status (valid/expired)

This command reads tokens from disk without making any API calls.`,
	RunE: runTokenInfo,
}

func init() {
	rootCmd.AddCommand(tokenCmd)
	tokenCmd.AddCommand(tokenInfoCmd)

	// Flags for token info command
	tokenInfoCmd.Flags().Bool("json", false, "output in JSON format")
	tokenInfoCmd.Flags().Bool("verbose", false, "show full token details")
}

// tokenInfo holds decoded token information
type tokenInfo struct {
	Type           string    `json:"type"`            // "device" or "user"
	DeviceDesc     string    `json:"device_desc"`     // e.g., "mobile-ios"
	DeviceID       string    `json:"device_id"`       // UUID
	IssuedAt       time.Time `json:"issued_at"`       // iat claim
	ExpiresAt      time.Time `json:"expires_at"`      // exp claim (zero if not present)
	ValidFor       string    `json:"valid_for"`       // human-readable validity period
	TimeRemaining  string    `json:"time_remaining"`  // human-readable time until expiration
	Status         string    `json:"status"`          // "valid", "expired", "no_expiration"
	Scopes         string    `json:"scopes,omitempty"` // user token scopes
	TokenLength    int       `json:"token_length"`    // length of token string
	HasExpiration  bool      `json:"has_expiration"`  // whether token has exp claim
}

func runTokenInfo(_ *cobra.Command, _ []string) error {
	// Get token file path
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	tokenPath := filepath.Join(home, ".legible", "token.json")

	// Check if token file exists
	if _, err := os.Stat(tokenPath); os.IsNotExist(err) {
		return fmt.Errorf("no authentication token found\nRun 'legible auth register' to authenticate first")
	}

	// Read token file
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return fmt.Errorf("failed to read token file: %w", err)
	}

	var tokenData map[string]string
	if err := json.Unmarshal(data, &tokenData); err != nil {
		return fmt.Errorf("failed to parse token file: %w", err)
	}

	// Get file modification time
	fileInfo, _ := os.Stat(tokenPath)
	lastModified := fileInfo.ModTime()

	// Parse device token
	deviceToken := tokenData["device_token"]
	var deviceInfo *tokenInfo
	if deviceToken != "" {
		deviceInfo = parseToken(deviceToken, "device")
	}

	// Parse user token
	userToken := tokenData["user_token"]
	var userInfo *tokenInfo
	if userToken != "" {
		userInfo = parseToken(userToken, "user")
	}

	// Display information
	fmt.Println("=== Authentication Token Information ===")
	fmt.Println()
	fmt.Printf("Token file: %s\n", tokenPath)
	fmt.Printf("Last modified: %s\n", lastModified.Format("2006-01-02 15:04:05"))
	fmt.Println()

	if deviceInfo != nil {
		fmt.Println("## Device Token")
		fmt.Println()
		fmt.Printf("Device type: %s\n", deviceInfo.DeviceDesc)
		fmt.Printf("Device ID: %s\n", deviceInfo.DeviceID)
		fmt.Printf("Issued at: %s\n", deviceInfo.IssuedAt.Format(time.RFC3339))
		if deviceInfo.HasExpiration {
			fmt.Printf("Expires at: %s\n", deviceInfo.ExpiresAt.Format(time.RFC3339))
			fmt.Printf("Valid for: %s\n", deviceInfo.ValidFor)
			fmt.Printf("Time remaining: %s\n", deviceInfo.TimeRemaining)
			fmt.Printf("Status: %s\n", deviceInfo.Status)
		} else {
			fmt.Println("Expiration: No exp claim (does not expire)")
			fmt.Println("Status: valid (indefinite)")
		}
		fmt.Printf("Token length: %d characters\n", deviceInfo.TokenLength)
		fmt.Println()
	} else {
		fmt.Println("## Device Token")
		fmt.Println()
		fmt.Println("No device token found")
		fmt.Println()
	}

	if userInfo != nil {
		fmt.Println("## User Token")
		fmt.Println()
		fmt.Printf("Device type: %s\n", userInfo.DeviceDesc)
		if userInfo.Scopes != "" {
			fmt.Printf("Scopes: %s\n", userInfo.Scopes)
		}
		fmt.Printf("Issued at: %s\n", userInfo.IssuedAt.Format(time.RFC3339))
		if userInfo.HasExpiration {
			fmt.Printf("Expires at: %s\n", userInfo.ExpiresAt.Format(time.RFC3339))
			fmt.Printf("Validity period: %s\n", userInfo.ValidFor)
			fmt.Printf("Time remaining: %s\n", userInfo.TimeRemaining)
			fmt.Printf("Status: %s\n", userInfo.Status)
		}
		fmt.Printf("Token length: %d characters\n", userInfo.TokenLength)
		fmt.Println()
	} else {
		fmt.Println("## User Token")
		fmt.Println()
		fmt.Println("No user token found (will be obtained on first API call)")
		fmt.Println()
	}

	// Summary
	fmt.Println("=== Summary ===")
	fmt.Println()

	if deviceInfo != nil {
		fmt.Println("✓ Device token present")

		if userInfo != nil {
			if userInfo.Status == "expired" {
				fmt.Println("⚠ User token expired (will be renewed on next API call)")
			} else {
				fmt.Println("✓ User token valid")
			}
		} else {
			fmt.Println("⚠ No user token (will be obtained on first API call)")
		}

		fmt.Println()
		fmt.Println("Ready to use. Run 'legible sync' or 'legible daemon' to start syncing.")
	} else {
		fmt.Println("✗ No device token found")
		fmt.Println()
		fmt.Println("Run 'legible auth register' to authenticate first.")
	}

	return nil
}

// parseToken decodes a JWT token and extracts relevant information
func parseToken(token string, tokenType string) *tokenInfo {
	if token == "" {
		return nil
	}

	parser := jwt.Parser{}
	claims := jwt.MapClaims{}
	_, _, err := parser.ParseUnverified(token, claims)
	if err != nil {
		return &tokenInfo{
			Type:        tokenType,
			TokenLength: len(token),
			Status:      "invalid",
		}
	}

	info := &tokenInfo{
		Type:        tokenType,
		TokenLength: len(token),
	}

	// Extract device description
	if desc, ok := claims["device-desc"].(string); ok {
		info.DeviceDesc = desc
	}

	// Extract device ID
	if id, ok := claims["device-id"].(string); ok {
		info.DeviceID = id
	}

	// Extract scopes (user tokens only)
	if scopes, ok := claims["scopes"].(string); ok {
		info.Scopes = scopes
	}

	// Extract issued at time
	if iat, ok := claims["iat"]; ok {
		switch v := iat.(type) {
		case float64:
			info.IssuedAt = time.Unix(int64(v), 0)
		case int64:
			info.IssuedAt = time.Unix(v, 0)
		}
	}

	// Extract expiration time
	if exp, ok := claims["exp"]; ok {
		info.HasExpiration = true

		var expTime time.Time
		switch v := exp.(type) {
		case float64:
			expTime = time.Unix(int64(v), 0)
		case int64:
			expTime = time.Unix(v, 0)
		}

		info.ExpiresAt = expTime

		// Calculate validity period
		if !info.IssuedAt.IsZero() {
			validFor := expTime.Sub(info.IssuedAt)
			info.ValidFor = formatDuration(validFor)
		}

		// Calculate time remaining
		now := time.Now()
		remaining := expTime.Sub(now)

		if remaining > 0 {
			info.TimeRemaining = formatDuration(remaining)
			info.Status = "valid"
		} else {
			info.TimeRemaining = "0s (expired)"
			info.Status = "expired"
		}
	} else {
		info.HasExpiration = false
		info.Status = "valid"
		info.TimeRemaining = "indefinite"
	}

	return info
}

// formatDuration formats a duration in human-readable format
func formatDuration(d time.Duration) string {
	if d < 0 {
		return "0s"
	}

	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	result := ""
	if days > 0 {
		result += fmt.Sprintf("%dd ", days)
	}
	if hours > 0 || days > 0 {
		result += fmt.Sprintf("%dh ", hours)
	}
	if minutes > 0 || hours > 0 || days > 0 {
		result += fmt.Sprintf("%dm ", minutes)
	}
	if seconds > 0 || result == "" {
		result += fmt.Sprintf("%ds", seconds)
	}

	return result
}
