package rmclient

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/platinummonkey/remarkable-sync/internal/logger"
)

// Client wraps the reMarkable cloud API for document synchronization
type Client struct {
	tokenPath string
	logger    *logger.Logger
	token     string
}

// Config holds configuration for the reMarkable client
type Config struct {
	// TokenPath is the path to store the authentication token
	TokenPath string

	// Logger is the logger instance to use
	Logger *logger.Logger
}

// NewClient creates a new reMarkable API client
func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Set default token path if not provided
	tokenPath := cfg.TokenPath
	if tokenPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		tokenPath = filepath.Join(home, ".remarkable-sync", "token.json")
	}

	// Set default logger if not provided
	log := cfg.Logger
	if log == nil {
		log = logger.Get()
	}

	return &Client{
		tokenPath: tokenPath,
		logger:    log,
	}, nil
}

// Authenticate authenticates with the reMarkable cloud API
// If a token exists, it will be loaded. Otherwise, manual device registration is required.
func (c *Client) Authenticate() error {
	c.logger.Info("Authenticating with reMarkable cloud API")

	// Ensure token directory exists
	tokenDir := filepath.Dir(c.tokenPath)
	if err := os.MkdirAll(tokenDir, 0700); err != nil {
		return fmt.Errorf("failed to create token directory: %w", err)
	}

	// Try to load existing token
	if _, err := os.Stat(c.tokenPath); err == nil {
		c.logger.Debug("Loading existing authentication token")
		if err := c.loadToken(); err != nil {
			c.logger.WithError(err).Warn("Failed to load existing token, manual authentication required")
			return fmt.Errorf("failed to load token: %w", err)
		}
		c.logger.Info("Successfully authenticated with existing token")
		return nil
	}

	// No token found, manual registration required
	c.logger.Info("No token found. Please register device manually:")
	c.logger.Info("1. Visit https://my.remarkable.com/device/desktop/connect")
	c.logger.Info("2. Enter the one-time code displayed")
	c.logger.Info("3. Save the device token to: " + c.tokenPath)

	return fmt.Errorf("authentication token not found at: %s", c.tokenPath)
}

// loadToken loads an existing authentication token from disk
func (c *Client) loadToken() error {
	data, err := os.ReadFile(c.tokenPath)
	if err != nil {
		return fmt.Errorf("failed to read token file: %w", err)
	}

	var tokenData map[string]string
	if err := json.Unmarshal(data, &tokenData); err != nil {
		return fmt.Errorf("failed to parse token file: %w", err)
	}

	deviceToken, ok := tokenData["device_token"]
	if !ok || deviceToken == "" {
		return fmt.Errorf("token file missing device_token field")
	}

	c.token = deviceToken
	return nil
}

// saveToken saves the authentication token to disk
func (c *Client) saveToken(deviceToken string) error {
	tokenData := map[string]string{
		"device_token": deviceToken,
	}

	data, err := json.MarshalIndent(tokenData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	// Write with restricted permissions (user read/write only)
	if err := os.WriteFile(c.tokenPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	c.token = deviceToken
	c.logger.Info("Authentication token saved successfully")
	return nil
}

// IsAuthenticated returns true if the client has a valid authentication token
func (c *Client) IsAuthenticated() bool {
	return c.token != ""
}

// ListDocuments lists all documents, optionally filtered by labels
// Returns a list of Document objects representing documents in the reMarkable cloud
func (c *Client) ListDocuments(labels []string) ([]Document, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated")
	}

	c.logger.WithFields("labels", labels).Debug("Listing documents")

	// Note: Actual implementation would use rmapi to fetch documents
	// This is a placeholder that would be implemented with proper rmapi integration
	// For now, return an informative error

	return nil, fmt.Errorf("ListDocuments requires rmapi integration - not yet implemented")
}

// GetDocumentMetadata retrieves metadata for a specific document
func (c *Client) GetDocumentMetadata(id string) (*Metadata, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated")
	}

	c.logger.WithDocumentID(id).Debug("Getting document metadata")

	// Note: Actual implementation would use rmapi to fetch metadata
	// This is a placeholder that would be implemented with proper rmapi integration

	return nil, fmt.Errorf("GetDocumentMetadata requires rmapi integration - not yet implemented")
}

// DownloadDocument downloads a document to the specified path
func (c *Client) DownloadDocument(id, outputPath string) error {
	if !c.IsAuthenticated() {
		return fmt.Errorf("client not authenticated")
	}

	c.logger.WithDocumentID(id).WithFields("output_path", outputPath).Info("Downloading document")

	// Create output directory
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Note: Actual implementation would use rmapi to download the document
	// This is a placeholder that would be implemented with proper rmapi integration

	return fmt.Errorf("DownloadDocument requires rmapi integration - not yet implemented")
}

// SetToken manually sets the authentication token (useful for testing or manual configuration)
func (c *Client) SetToken(token string) error {
	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	if err := c.saveToken(token); err != nil {
		return err
	}

	return nil
}

// Close closes the client and cleans up resources
func (c *Client) Close() error {
	// No cleanup required for now
	return nil
}
