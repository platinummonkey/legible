// Package rmclient provides a client for interacting with the reMarkable cloud API.
package rmclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/juruen/rmapi/api"
	"github.com/juruen/rmapi/auth"
	"github.com/juruen/rmapi/model"
	"github.com/juruen/rmapi/transport"
	"github.com/platinummonkey/remarkable-sync/internal/logger"
)

// urlFixingRoundTripper is a custom HTTP transport that fixes URLs containing doesnotexist.remarkable.com
// The reMarkable API sometimes returns URLs with this invalid hostname, so we replace it with my.remarkable.com
type urlFixingRoundTripper struct {
	base http.RoundTripper
}

// RoundTrip implements http.RoundTripper
func (u *urlFixingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Fix the URL if it contains doesnotexist.remarkable.com
	if strings.Contains(req.URL.Host, "doesnotexist.remarkable.com") {
		fixedURL := *req.URL
		fixedURL.Host = strings.Replace(fixedURL.Host, "doesnotexist.remarkable.com", "my.remarkable.com", 1)
		req.URL = &fixedURL
	}

	// Call the base transport
	if u.base == nil {
		return http.DefaultTransport.RoundTrip(req)
	}
	return u.base.RoundTrip(req)
}

// wrapHTTPClient wraps an HTTP client with URL fixing middleware
func wrapHTTPClient(client *http.Client) *http.Client {
	if client == nil {
		client = &http.Client{}
	}

	// Wrap the transport with our URL fixer
	client.Transport = &urlFixingRoundTripper{
		base: client.Transport,
	}

	return client
}

// Client wraps the reMarkable cloud API for document synchronization
type Client struct {
	tokenPath string
	logger    *logger.Logger
	token     string
	apiCtx    api.ApiCtx
}

// Config holds configuration for the reMarkable client
type Config struct {
	// TokenPath is the path to store the authentication token
	TokenPath string

	// Logger is the logger instance to use
	Logger *logger.Logger
}

// jsonTokenStore implements auth.TokenStore for our JSON token format
// It bridges our JSON token file format to rmapi's TokenStore interface
type jsonTokenStore struct {
	tokenPath string
}

// Save persists tokens to our JSON format
func (jts *jsonTokenStore) Save(t auth.TokenSet) error {
	// Create token directory if it doesn't exist
	tokenDir := filepath.Dir(jts.tokenPath)
	if err := os.MkdirAll(tokenDir, 0700); err != nil {
		return fmt.Errorf("failed to create token directory: %w", err)
	}

	tokenData := map[string]string{
		"device_token": t.DeviceToken,
	}

	// Also save user token if present
	if t.UserToken != "" {
		tokenData["user_token"] = t.UserToken
	}

	data, err := json.MarshalIndent(tokenData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	if err := os.WriteFile(jts.tokenPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

// Load loads tokens from our JSON format
func (jts *jsonTokenStore) Load() (auth.TokenSet, error) {
	// Return empty if file doesn't exist
	if _, err := os.Stat(jts.tokenPath); os.IsNotExist(err) {
		return auth.TokenSet{}, nil
	}

	data, err := os.ReadFile(jts.tokenPath)
	if err != nil {
		return auth.TokenSet{}, fmt.Errorf("failed to read token file: %w", err)
	}

	var tokenData map[string]string
	if err := json.Unmarshal(data, &tokenData); err != nil {
		return auth.TokenSet{}, fmt.Errorf("failed to parse token file: %w", err)
	}

	return auth.TokenSet{
		DeviceToken: tokenData["device_token"],
		UserToken:   tokenData["user_token"],
	}, nil
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

		// Initialize rmapi client
		if err := c.initializeAPIClient(); err != nil {
			c.logger.WithError(err).Error("Failed to initialize rmapi client")
			return fmt.Errorf("failed to initialize API client: %w", err)
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

// initializeAPIClient initializes the rmapi API context
func (c *Client) initializeAPIClient() error {
	// Create token store
	tokenStore := &jsonTokenStore{tokenPath: c.tokenPath}

	// Create auth instance
	authClient := auth.NewFromStore(tokenStore)

	// Get authenticated HTTP client and wrap it with URL fixer
	// This fixes the doesnotexist.remarkable.com issue
	httpClient := wrapHTTPClient(authClient.Client())

	// Get the user token to extract sync version
	userToken, err := authClient.Token()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}

	// Parse token to get user info and sync version
	userInfo, err := api.ParseToken(userToken)
	if err != nil {
		return fmt.Errorf("failed to parse user token: %w", err)
	}

	c.logger.WithFields("sync_version", userInfo.SyncVersion, "user", userInfo.User).Debug("Parsed user token")

	// Create HTTP context - we need to get tokens again from store
	tokens, err := tokenStore.Load()
	if err != nil {
		return fmt.Errorf("failed to load tokens: %w", err)
	}

	// Create API context
	// We need to create the transport manually since api.CreateApiCtx expects HttpClientCtx
	// But the auth package gives us an http.Client
	// Let's use the direct approach from rmapi

	// Actually, looking at rmapi source, we need to use api.AuthHttpCtx
	// But that does interactive auth. Since we have tokens, let's create HttpClientCtx directly
	httpCtx := &transport.HttpClientCtx{
		Client: httpClient,
		Tokens: model.AuthTokens{
			DeviceToken: tokens.DeviceToken,
			UserToken:   tokens.UserToken,
		},
	}

	// Create API context
	apiCtx, err := api.CreateApiCtx(httpCtx, userInfo.SyncVersion)
	if err != nil {
		return fmt.Errorf("failed to create API context: %w", err)
	}

	c.apiCtx = apiCtx
	return nil
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

	if c.apiCtx == nil {
		return nil, fmt.Errorf("API client not initialized, call Authenticate() first")
	}

	c.logger.WithFields("labels", labels).Debug("Listing documents")

	// Get the file tree
	tree := c.apiCtx.Filetree()
	if tree == nil {
		return nil, fmt.Errorf("failed to get file tree")
	}

	// Collect all documents from the tree
	var documents []Document
	c.collectDocuments(tree.Root(), labels, &documents)

	c.logger.WithFields("count", len(documents)).Info("Listed documents")
	return documents, nil
}

// parseTime parses a timestamp string from rmapi to time.Time
// rmapi returns timestamps as RFC3339 strings
func parseTime(timeStr string) time.Time {
	if timeStr == "" {
		return time.Time{}
	}

	// Try RFC3339 format
	t, err := time.Parse(time.RFC3339, timeStr)
	if err == nil {
		return t
	}

	// Try RFC3339Nano format
	t, err = time.Parse(time.RFC3339Nano, timeStr)
	if err == nil {
		return t
	}

	// Return zero time if parsing fails
	return time.Time{}
}

// collectDocuments recursively collects documents from the file tree
func (c *Client) collectDocuments(node *model.Node, labels []string, documents *[]Document) {
	if node == nil {
		return
	}

	// Process current node if it's a document (not a directory)
	if node.IsFile() && node.Document != nil {
		doc := node.Document

		// Note: Label filtering not yet implemented.
		// rmapi doesn't directly expose labels/tags in the Document model.
		// Labels would need to be checked via metadata or parent folder names.
		// TODO: Implement proper label filtering when metadata structure is clarified.
		_ = labels // Avoid unused parameter warning for now

		*documents = append(*documents, Document{
			ID:             doc.ID,
			Name:           doc.Name,
			Type:           doc.Type,
			Version:        doc.Version,
			ModifiedClient: parseTime(doc.ModifiedClient),
			Parent:         doc.Parent,
			CurrentPage:    doc.CurrentPage,
		})
	}

	// Recursively process children
	for _, child := range node.Children {
		c.collectDocuments(child, labels, documents)
	}
}

// GetDocumentMetadata retrieves metadata for a specific document from the API
// Returns a Document with full information from the reMarkable cloud
func (c *Client) GetDocumentMetadata(id string) (*Document, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("client not authenticated")
	}

	if c.apiCtx == nil {
		return nil, fmt.Errorf("API client not initialized, call Authenticate() first")
	}

	c.logger.WithDocumentID(id).Debug("Getting document metadata")

	// Get the file tree
	tree := c.apiCtx.Filetree()
	if tree == nil {
		return nil, fmt.Errorf("failed to get file tree")
	}

	// Find the node by ID
	node := tree.NodeById(id)
	if node == nil {
		return nil, fmt.Errorf("document not found: %s", id)
	}

	if node.Document == nil {
		return nil, fmt.Errorf("node is not a document: %s", id)
	}

	doc := node.Document

	return &Document{
		ID:             doc.ID,
		Name:           doc.Name,
		Type:           doc.Type,
		Version:        doc.Version,
		ModifiedClient: parseTime(doc.ModifiedClient),
		Parent:         doc.Parent,
		CurrentPage:    doc.CurrentPage,
	}, nil
}

// DownloadDocument downloads a document to the specified path
func (c *Client) DownloadDocument(id, outputPath string) error {
	if !c.IsAuthenticated() {
		return fmt.Errorf("client not authenticated")
	}

	if c.apiCtx == nil {
		return fmt.Errorf("API client not initialized, call Authenticate() first")
	}

	c.logger.WithDocumentID(id).WithFields("output_path", outputPath).Info("Downloading document")

	// Create output directory
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Use rmapi to fetch the document
	// FetchDocument downloads the document as a .zip file
	if err := c.apiCtx.FetchDocument(id, outputPath); err != nil {
		return fmt.Errorf("failed to download document: %w", err)
	}

	c.logger.WithDocumentID(id).Info("Successfully downloaded document")
	return nil
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
