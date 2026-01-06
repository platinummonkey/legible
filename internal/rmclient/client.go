// Package rmclient provides a client for interacting with the reMarkable cloud API.
package rmclient

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/juruen/rmapi/api"
	"github.com/juruen/rmapi/config"
	"github.com/juruen/rmapi/model"
	"github.com/juruen/rmapi/transport"
	"github.com/platinummonkey/legible/internal/logger"
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

// maskToken masks a token for safe logging (shows first/last 4 chars)
func maskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

// tokenExpirationBuffer is the time before expiration to consider a token expired
// This prevents using tokens that are about to expire
const tokenExpirationBuffer = 5 * time.Minute

// isTokenExpired checks if a JWT token is expired or about to expire
// Returns true if the token is expired, invalid, or expires within the buffer period
func isTokenExpired(token string) bool {
	if token == "" {
		return true
	}

	// Parse token without verification (we just need to check expiration)
	parser := jwt.Parser{}
	claims := jwt.MapClaims{}
	_, _, err := parser.ParseUnverified(token, claims)
	if err != nil {
		// If we can't parse the token, consider it expired
		return true
	}

	// Check if exp claim exists
	exp, ok := claims["exp"]
	if !ok {
		// No expiration claim, consider it expired for safety
		return true
	}

	// Convert exp to time.Time
	var expTime time.Time
	switch v := exp.(type) {
	case float64:
		expTime = time.Unix(int64(v), 0)
	case int64:
		expTime = time.Unix(v, 0)
	default:
		// Unknown format, consider expired
		return true
	}

	// Check if token is expired or about to expire
	now := time.Now()
	return now.Add(tokenExpirationBuffer).After(expTime)
}

// getTokenExpiration returns the expiration time of a JWT token
// Returns zero time if the token is invalid or has no expiration
func getTokenExpiration(token string) time.Time {
	if token == "" {
		return time.Time{}
	}

	parser := jwt.Parser{}
	claims := jwt.MapClaims{}
	_, _, err := parser.ParseUnverified(token, claims)
	if err != nil {
		return time.Time{}
	}

	exp, ok := claims["exp"]
	if !ok {
		return time.Time{}
	}

	switch v := exp.(type) {
	case float64:
		return time.Unix(int64(v), 0)
	case int64:
		return time.Unix(v, 0)
	default:
		return time.Time{}
	}
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

// jsonTokenStore stores tokens in JSON format
// It works with model.AuthTokens from rmapi
type jsonTokenStore struct {
	tokenPath string
}

// Save persists tokens to our JSON format
func (jts *jsonTokenStore) Save(t model.AuthTokens) error {
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
func (jts *jsonTokenStore) Load() (*model.AuthTokens, error) {
	// Return empty if file doesn't exist
	if _, err := os.Stat(jts.tokenPath); os.IsNotExist(err) {
		return &model.AuthTokens{}, nil
	}

	data, err := os.ReadFile(jts.tokenPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	var tokenData map[string]string
	if err := json.Unmarshal(data, &tokenData); err != nil {
		return nil, fmt.Errorf("failed to parse token file: %w", err)
	}

	return &model.AuthTokens{
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
		tokenPath = filepath.Join(home, ".legible", "token.json")
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

	// No token found, need to register device
	c.logger.Info("=== Starting device registration flow ===")
	c.logger.Info("No device token found. Starting device registration...")
	c.logger.Info("Visit https://my.remarkable.com/device/remarkable?showOtp=true to get a one-time code")

	// Prompt for one-time code
	fmt.Print("Enter one-time code: ")
	reader := bufio.NewReader(os.Stdin)
	code, err := reader.ReadString('\n')
	if err != nil {
		c.logger.WithError(err).Error("Failed to read one-time code from stdin")
		return fmt.Errorf("failed to read one-time code: %w", err)
	}

	code = strings.TrimSpace(code)
	c.logger.WithFields("code_length", len(code)).Debug("Received one-time code")

	if len(code) != 8 {
		c.logger.WithFields("expected", 8, "actual", len(code)).Error("Invalid code length")
		return fmt.Errorf("invalid code length: expected 8 characters, got %d", len(code))
	}

	// Register device with the code
	c.logger.Info("Registering device with one-time code...")
	deviceToken, err := c.registerDevice(code)
	if err != nil {
		return fmt.Errorf("failed to register device: %w", err)
	}

	c.logger.Info("✓ Device registered successfully")

	// Save the device token
	c.logger.Debug("Saving device token to file...")
	if err := c.saveToken(deviceToken); err != nil {
		c.logger.WithError(err).Error("Failed to save device token")
		return fmt.Errorf("failed to save device token: %w", err)
	}
	c.logger.Info("✓ Device token saved")

	// Initialize API client with the new device token
	c.logger.Info("Initializing API client with new device token...")
	if err := c.initializeAPIClient(); err != nil {
		c.logger.WithError(err).Error("Failed to initialize rmapi client")
		return fmt.Errorf("failed to initialize API client: %w", err)
	}

	c.logger.Info("=== Successfully authenticated with new device token ===")
	return nil
}

// registerDevice registers a new device with the reMarkable API using a one-time code
func (c *Client) registerDevice(code string) (string, error) {
	c.logger.WithFields("code_length", len(code)).Debug("Registering device with reMarkable API")

	// Use tablet device ID
	deviceID := "remarkable"
	c.logger.WithFields("device_id", deviceID).Debug("Using tablet device ID")

	// Create device registration request
	req := model.DeviceTokenRequest{
		Code:       code,
		DeviceDesc: "remarkable",
		DeviceId:   deviceID,
	}

	c.logger.WithFields(
		"endpoint", config.NewTokenDevice,
		"device_desc", "remarkable",
	).Info("Calling device registration API")

	// Create HTTP context for device registration (no auth required)
	httpCtx := &transport.HttpClientCtx{
		Client: wrapHTTPClient(&http.Client{
			Timeout: 60 * time.Second,
		}),
		Tokens: model.AuthTokens{},
	}

	// Call device registration API
	resp := transport.BodyString{}
	err := httpCtx.Post(transport.EmptyBearer, config.NewTokenDevice, req, &resp)
	if err != nil {
		c.logger.WithError(err).WithFields("endpoint", config.NewTokenDevice).Error("Device registration API call failed")
		return "", fmt.Errorf("failed to register device: %w", err)
	}

	// Mask the token for logging (show first/last 4 chars only)
	maskedToken := maskToken(resp.Content)
	c.logger.WithFields(
		"device_id", deviceID,
		"token_length", len(resp.Content),
		"token_preview", maskedToken,
	).Info("Device registered successfully, received device token")

	return resp.Content, nil
}

// renewUserToken renews the user token using the device token
// This implementation uses config.NewUserDevice which points to the correct API endpoint
func (c *Client) renewUserToken(deviceToken string) (string, error) {
	maskedDeviceToken := maskToken(deviceToken)
	c.logger.WithFields(
		"device_token_preview", maskedDeviceToken,
		"device_token_length", len(deviceToken),
	).Debug("Starting user token renewal")

	// Create HTTP context with device token
	httpCtx := &transport.HttpClientCtx{
		Client: wrapHTTPClient(&http.Client{
			Timeout: 60 * time.Second,
		}),
		Tokens: model.AuthTokens{
			DeviceToken: deviceToken,
		},
	}

	c.logger.WithFields(
		"endpoint", config.NewUserDevice,
		"auth_type", "DeviceBearer",
	).Info("Calling user token renewal API")

	// Use config.NewUserDevice which uses webapp-prod.cloud.remarkable.engineering
	// instead of the hardcoded my.remarkable.com that causes redirects
	resp := transport.BodyString{}
	err := httpCtx.Post(transport.DeviceBearer, config.NewUserDevice, nil, &resp)
	if err != nil {
		c.logger.WithError(err).WithFields(
			"endpoint", config.NewUserDevice,
			"device_token_preview", maskedDeviceToken,
		).Error("User token renewal API call failed")
		return "", fmt.Errorf("failed to renew user token: %w", err)
	}

	maskedUserToken := maskToken(resp.Content)
	c.logger.WithFields(
		"user_token_length", len(resp.Content),
		"user_token_preview", maskedUserToken,
	).Info("User token renewed successfully")

	return resp.Content, nil
}

// initializeAPIClient initializes the rmapi API context
func (c *Client) initializeAPIClient() error {
	c.logger.Info("Initializing reMarkable API client")

	// Create token store
	tokenStore := &jsonTokenStore{tokenPath: c.tokenPath}

	// Load tokens
	c.logger.WithFields("token_path", c.tokenPath).Debug("Loading tokens from file")
	tokens, err := tokenStore.Load()
	if err != nil {
		c.logger.WithError(err).Error("Failed to load tokens from file")
		return fmt.Errorf("failed to load tokens: %w", err)
	}

	maskedDeviceToken := maskToken(tokens.DeviceToken)
	c.logger.WithFields(
		"has_device_token", tokens.DeviceToken != "",
		"device_token_preview", maskedDeviceToken,
		"has_user_token", tokens.UserToken != "",
	).Debug("Loaded tokens")

	// Check if we need to renew the user token (empty or expired)
	userToken := tokens.UserToken
	needsRenewal := false
	if userToken == "" {
		c.logger.Info("No user token found, need to renew from device token")
		needsRenewal = true
	} else if isTokenExpired(userToken) {
		expTime := getTokenExpiration(userToken)
		c.logger.WithFields(
			"expiration", expTime,
			"time_until_expiry", time.Until(expTime),
		).Info("User token is expired or about to expire, need to renew")
		needsRenewal = true
	} else {
		expTime := getTokenExpiration(userToken)
		c.logger.WithFields(
			"expiration", expTime,
			"time_until_expiry", time.Until(expTime),
		).Debug("Using existing valid user token from file")
	}

	if needsRenewal {
		userToken, err = c.renewUserToken(tokens.DeviceToken)
		if err != nil {
			return fmt.Errorf("failed to renew user token: %w", err)
		}

		// Save the new user token
		c.logger.Debug("Saving renewed user token to file")
		tokens.UserToken = userToken
		if err := tokenStore.Save(*tokens); err != nil {
			c.logger.WithError(err).Warn("Failed to save user token to file")
		} else {
			expTime := getTokenExpiration(userToken)
			c.logger.WithFields(
				"expiration", expTime,
				"valid_for", time.Until(expTime),
			).Info("User token renewed and saved successfully")
		}
	}

	// Parse token to get user info and sync version
	c.logger.Debug("Parsing user token to extract user info")
	userInfo, err := api.ParseToken(userToken)
	if err != nil {
		c.logger.WithError(err).Error("Failed to parse user token")
		return fmt.Errorf("failed to parse user token: %w", err)
	}

	c.logger.WithFields(
		"sync_version", userInfo.SyncVersion,
		"user", userInfo.User,
	).Info("Successfully parsed user token")

	// Create HTTP client with URL fixing
	c.logger.Debug("Creating HTTP client with URL fixing middleware")
	httpClient := wrapHTTPClient(&http.Client{
		Timeout: 60 * time.Second,
	})

	// Create HTTP context
	c.logger.Debug("Creating HTTP context with tokens")
	httpCtx := &transport.HttpClientCtx{
		Client: httpClient,
		Tokens: model.AuthTokens{
			DeviceToken: tokens.DeviceToken,
			UserToken:   userToken,
		},
	}

	// Create API context
	c.logger.WithFields("sync_version", userInfo.SyncVersion).Debug("Creating API context")
	apiCtx, err := api.CreateApiCtx(httpCtx, userInfo.SyncVersion)
	if err != nil {
		c.logger.WithError(err).Error("Failed to create API context")
		return fmt.Errorf("failed to create API context: %w", err)
	}

	c.apiCtx = apiCtx
	c.logger.Info("API client initialized successfully")
	return nil
}

// loadToken loads an existing authentication token from disk
func (c *Client) loadToken() error {
	c.logger.WithFields("path", c.tokenPath).Debug("Loading token from file")

	data, err := os.ReadFile(c.tokenPath)
	if err != nil {
		c.logger.WithError(err).Error("Failed to read token file")
		return fmt.Errorf("failed to read token file: %w", err)
	}

	c.logger.WithFields("file_size", len(data)).Debug("Token file read successfully")

	var tokenData map[string]string
	if err := json.Unmarshal(data, &tokenData); err != nil {
		c.logger.WithError(err).Error("Failed to parse token JSON")
		return fmt.Errorf("failed to parse token file: %w", err)
	}

	deviceToken, ok := tokenData["device_token"]
	if !ok || deviceToken == "" {
		c.logger.Error("Token file missing device_token field")
		return fmt.Errorf("token file missing device_token field")
	}

	maskedToken := maskToken(deviceToken)
	c.logger.WithFields(
		"device_token_preview", maskedToken,
		"device_token_length", len(deviceToken),
	).Debug("Device token loaded from file")

	c.token = deviceToken
	return nil
}

// saveToken saves the authentication token to disk
func (c *Client) saveToken(deviceToken string) error {
	maskedToken := maskToken(deviceToken)
	c.logger.WithFields(
		"path", c.tokenPath,
		"device_token_preview", maskedToken,
		"device_token_length", len(deviceToken),
	).Debug("Saving device token to file")

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

// ensureValidToken checks if the current user token is valid and renews it if necessary
// This should be called before making API requests to prevent mid-operation token expiration
func (c *Client) ensureValidToken() error {
	if c.apiCtx == nil {
		return fmt.Errorf("API client not initialized")
	}

	// Get current tokens
	tokenStore := &jsonTokenStore{tokenPath: c.tokenPath}
	tokens, err := tokenStore.Load()
	if err != nil {
		return fmt.Errorf("failed to load tokens: %w", err)
	}

	// Check if user token needs renewal
	if tokens.UserToken == "" || isTokenExpired(tokens.UserToken) {
		c.logger.Info("User token expired or missing, renewing before API call")

		userToken, err := c.renewUserToken(tokens.DeviceToken)
		if err != nil {
			return fmt.Errorf("failed to renew user token: %w", err)
		}

		// Update tokens
		tokens.UserToken = userToken
		if err := tokenStore.Save(*tokens); err != nil {
			c.logger.WithError(err).Warn("Failed to save renewed token")
		}

		// Update API context with new token
		// We need to recreate the HTTP context with the new token
		httpClient := wrapHTTPClient(&http.Client{
			Timeout: 60 * time.Second,
		})

		httpCtx := &transport.HttpClientCtx{
			Client: httpClient,
			Tokens: model.AuthTokens{
				DeviceToken: tokens.DeviceToken,
				UserToken:   userToken,
			},
		}

		// Parse token to get sync version
		userInfo, err := api.ParseToken(userToken)
		if err != nil {
			return fmt.Errorf("failed to parse renewed token: %w", err)
		}

		// Recreate API context
		apiCtx, err := api.CreateApiCtx(httpCtx, userInfo.SyncVersion)
		if err != nil {
			return fmt.Errorf("failed to recreate API context: %w", err)
		}

		c.apiCtx = apiCtx
		c.logger.Info("User token renewed successfully")
	}

	return nil
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

	// Ensure token is valid before making API call
	if err := c.ensureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
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

		// If label filters are specified, check if document has matching tags
		if len(labels) > 0 {
			hasMatchingTag := false
			for _, label := range labels {
				for _, tag := range doc.Tags {
					if tag == label {
						hasMatchingTag = true
						break
					}
				}
				if hasMatchingTag {
					break
				}
			}

			// Skip this document if it doesn't have any matching tags
			if !hasMatchingTag {
				c.logger.WithFields("id", doc.ID, "name", doc.Name, "tags", doc.Tags).
					Debug("Skipping document without matching tags")
				// Still process children (for collections)
				for _, child := range node.Children {
					c.collectDocuments(child, labels, documents)
				}
				return
			}
		}

		c.logger.WithFields("id", doc.ID, "name", doc.Name, "tags", doc.Tags).
			Debug("Including document")

		*documents = append(*documents, Document{
			ID:             doc.ID,
			Name:           doc.Name,
			Type:           doc.Type,
			Version:        doc.Version,
			ModifiedClient: parseTime(doc.ModifiedClient),
			Parent:         doc.Parent,
			CurrentPage:    doc.CurrentPage,
			Tags:           doc.Tags,
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

	// Ensure token is valid before making API call
	if err := c.ensureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
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
		Tags:           doc.Tags,
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

	// Ensure token is valid before making API call
	if err := c.ensureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
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

// GetFolderPath returns the full folder path for a document by traversing its parent chain
// Returns an empty string if the document is in the root folder
// Returns an error if the document is not found or if there's a circular reference
func (c *Client) GetFolderPath(documentID string) (string, error) {
	if !c.IsAuthenticated() {
		return "", fmt.Errorf("client not authenticated")
	}

	if c.apiCtx == nil {
		return "", fmt.Errorf("API client not initialized, call Authenticate() first")
	}

	// Ensure token is valid before making API call
	if err := c.ensureValidToken(); err != nil {
		return "", fmt.Errorf("failed to ensure valid token: %w", err)
	}

	// Get the file tree
	tree := c.apiCtx.Filetree()
	if tree == nil {
		return "", fmt.Errorf("failed to get file tree")
	}

	// Find the node by ID
	node := tree.NodeById(documentID)
	if node == nil {
		return "", fmt.Errorf("document not found: %s", documentID)
	}

	// Build the folder path by traversing parents
	var pathParts []string
	visited := make(map[string]bool)
	currentNode := node

	// Traverse up to the root, collecting folder names
	for currentNode != nil && currentNode.Parent != nil {
		parentID := ""
		if currentNode.Document != nil {
			parentID = currentNode.Document.Parent
		}

		// Check for circular references
		if visited[parentID] {
			return "", fmt.Errorf("circular reference detected in folder hierarchy")
		}
		visited[parentID] = true

		// Get parent node
		parentNode := tree.NodeById(parentID)
		if parentNode == nil {
			// Parent not found, assume we've reached root
			break
		}

		// Add parent folder name to path (if it's a collection)
		if parentNode.Document != nil && parentNode.Document.Type == CollectionType {
			sanitized := sanitizeFolderName(parentNode.Document.Name)
			// Only add if the folder name is valid (not empty and not just special chars)
			if isValidFolderName(sanitized) {
				// Prepend to build path from root to document
				pathParts = append([]string{sanitized}, pathParts...)
			}
		}

		currentNode = parentNode
	}

	// Join path parts with forward slash
	return filepath.Join(pathParts...), nil
}

// sanitizeFolderName removes or replaces characters that are invalid in folder names
func sanitizeFolderName(name string) string {
	// Trim whitespace first
	name = strings.TrimSpace(name)

	// Replace common problematic characters
	replacements := map[rune]string{
		'/':  "-",
		'\\': "-",
		':':  "-",
		'*':  "_",
		'?':  "_",
		'"':  "'",
		'<':  "_",
		'>':  "_",
		'|':  "-",
	}

	result := ""
	for _, ch := range name {
		if replacement, found := replacements[ch]; found {
			result += replacement
		} else {
			result += string(ch)
		}
	}

	return strings.TrimSpace(result)
}

// isValidFolderName checks if a sanitized folder name is valid for use in paths
func isValidFolderName(name string) bool {
	// Empty names are invalid
	if name == "" {
		return false
	}

	// Names consisting only of special characters are invalid
	// Common cases after sanitization: "-", "_", ".", etc.
	invalidNames := map[string]bool{
		"-":  true,
		"_":  true,
		".":  true,
		"..": true,
	}

	return !invalidNames[name]
}

// Close closes the client and cleans up resources
func (c *Client) Close() error {
	// No cleanup required for now
	return nil
}
