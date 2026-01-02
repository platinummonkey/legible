package rmclient

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/platinummonkey/legible/internal/logger"
)

func TestNewClient(t *testing.T) {
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "token.json")

	cfg := &Config{
		TokenPath: tokenPath,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if client == nil {
		t.Fatal("NewClient() returned nil client")
	}

	if client.tokenPath != tokenPath {
		t.Errorf("expected tokenPath %s, got %s", tokenPath, client.tokenPath)
	}

	if client.logger == nil {
		t.Error("logger should be initialized")
	}
}

func TestNewClient_DefaultTokenPath(t *testing.T) {
	cfg := &Config{}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	home, _ := os.UserHomeDir()
	expectedPath := filepath.Join(home, ".legible", "token.json")

	if client.tokenPath != expectedPath {
		t.Errorf("expected default tokenPath %s, got %s", expectedPath, client.tokenPath)
	}
}

func TestNewClient_CustomLogger(t *testing.T) {
	tmpDir := t.TempDir()

	customLogger, err := logger.New(&logger.Config{
		Level:  "debug",
		Format: "console",
	})
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	cfg := &Config{
		TokenPath: filepath.Join(tmpDir, "token.json"),
		Logger:    customLogger,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if client.logger != customLogger {
		t.Error("custom logger should be used")
	}
}

func TestNewClient_NilConfig(t *testing.T) {
	_, err := NewClient(nil)
	if err == nil {
		t.Error("NewClient() should error with nil config")
	}
}

func TestClient_SaveToken(t *testing.T) {
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "token.json")

	cfg := &Config{
		TokenPath: tokenPath,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	deviceToken := "test-device-token-12345"
	if err := client.saveToken(deviceToken); err != nil {
		t.Fatalf("saveToken() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(tokenPath); os.IsNotExist(err) {
		t.Error("token file should exist after saveToken()")
	}

	// Verify file permissions (should be 0600)
	info, err := os.Stat(tokenPath)
	if err != nil {
		t.Fatalf("failed to stat token file: %v", err)
	}

	mode := info.Mode().Perm()
	if mode != 0600 {
		t.Errorf("expected file permissions 0600, got %o", mode)
	}

	// Verify content
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatalf("failed to read token file: %v", err)
	}

	var token map[string]string
	if err := json.Unmarshal(data, &token); err != nil {
		t.Fatalf("failed to unmarshal token: %v", err)
	}

	if token["device_token"] != deviceToken {
		t.Errorf("expected device_token %s, got %s", deviceToken, token["device_token"])
	}
}

func TestClient_LoadToken_ValidToken(t *testing.T) {
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "token.json")

	// Create a valid token file
	token := map[string]string{
		"device_token": "test-token-12345",
	}

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal token: %v", err)
	}

	if err := os.WriteFile(tokenPath, data, 0600); err != nil {
		t.Fatalf("failed to write token file: %v", err)
	}

	cfg := &Config{
		TokenPath: tokenPath,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Load token
	if err := client.loadToken(); err != nil {
		t.Fatalf("loadToken() error = %v", err)
	}

	// Verify token was loaded
	if !client.IsAuthenticated() {
		t.Error("client should be authenticated after loadToken()")
	}

	if client.token != "test-token-12345" {
		t.Errorf("expected token test-token-12345, got %s", client.token)
	}
}

func TestClient_LoadToken_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "token.json")

	// Write invalid JSON
	if err := os.WriteFile(tokenPath, []byte("invalid json"), 0600); err != nil {
		t.Fatalf("failed to write token file: %v", err)
	}

	cfg := &Config{
		TokenPath: tokenPath,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	err = client.loadToken()
	if err == nil {
		t.Error("loadToken() should error on invalid JSON")
	}
}

func TestClient_LoadToken_MissingDeviceToken(t *testing.T) {
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "token.json")

	// Create token file without device_token field
	token := map[string]string{
		"other_field": "value",
	}

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal token: %v", err)
	}

	if err := os.WriteFile(tokenPath, data, 0600); err != nil {
		t.Fatalf("failed to write token file: %v", err)
	}

	cfg := &Config{
		TokenPath: tokenPath,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	err = client.loadToken()
	if err == nil {
		t.Error("loadToken() should error when device_token is missing")
	}
}

func TestClient_LoadToken_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "nonexistent.json")

	cfg := &Config{
		TokenPath: tokenPath,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	err = client.loadToken()
	if err == nil {
		t.Error("loadToken() should error when file doesn't exist")
	}
}

func TestClient_Close(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		TokenPath: filepath.Join(tmpDir, "token.json"),
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Close should not error
	if err := client.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

// Note: Testing ListDocuments, GetDocumentMetadata, and DownloadDocument
// requires mocking the rmapi library or integration tests with a real
// reMarkable account, which is beyond the scope of unit tests.
// These would typically be tested with:
// 1. Integration tests with a test reMarkable account
// 2. Mock interfaces for the rmapi client
// 3. Contract tests to verify the API behavior

func TestClient_SetToken(t *testing.T) {
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "token.json")

	cfg := &Config{
		TokenPath: tokenPath,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	deviceToken := "test-device-token-67890"
	if err := client.SetToken(deviceToken); err != nil {
		t.Fatalf("SetToken() error = %v", err)
	}

	// Verify token was set
	if !client.IsAuthenticated() {
		t.Error("client should be authenticated after SetToken()")
	}

	if client.token != deviceToken {
		t.Errorf("expected token %s, got %s", deviceToken, client.token)
	}

	// Verify file was created
	if _, err := os.Stat(tokenPath); os.IsNotExist(err) {
		t.Error("token file should exist after SetToken()")
	}
}

func TestClient_SetToken_EmptyToken(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		TokenPath: filepath.Join(tmpDir, "token.json"),
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	err = client.SetToken("")
	if err == nil {
		t.Error("SetToken() should error with empty token")
	}
}

func TestClient_IsAuthenticated(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		TokenPath: filepath.Join(tmpDir, "token.json"),
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Should not be authenticated initially
	if client.IsAuthenticated() {
		t.Error("client should not be authenticated initially")
	}

	// Set token
	if err := client.SetToken("test-token"); err != nil {
		t.Fatalf("SetToken() error = %v", err)
	}

	// Should be authenticated now
	if !client.IsAuthenticated() {
		t.Error("client should be authenticated after SetToken()")
	}
}

func TestClient_ListDocuments_NotAuthenticated(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		TokenPath: filepath.Join(tmpDir, "token.json"),
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Should error when not authenticated
	_, err = client.ListDocuments(nil)
	if err == nil {
		t.Error("ListDocuments() should error when not authenticated")
	}
}

func TestClient_GetDocumentMetadata_NotAuthenticated(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		TokenPath: filepath.Join(tmpDir, "token.json"),
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Should error when not authenticated
	_, err = client.GetDocumentMetadata("doc-123")
	if err == nil {
		t.Error("GetDocumentMetadata() should error when not authenticated")
	}
}

func TestClient_DownloadDocument_NotAuthenticated(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		TokenPath: filepath.Join(tmpDir, "token.json"),
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Should error when not authenticated
	err = client.DownloadDocument("doc-123", "/tmp/output.zip")
	if err == nil {
		t.Error("DownloadDocument() should error when not authenticated")
	}
}

func TestClient_DownloadDocument_Authenticated(t *testing.T) {
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "token.json")
	outputPath := filepath.Join(tmpDir, "output", "subdir", "doc.zip")

	cfg := &Config{
		TokenPath: tokenPath,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Set authentication token
	if err := client.SetToken("test-token"); err != nil {
		t.Fatalf("SetToken() error = %v", err)
	}

	// Should return error that API client is not initialized
	// (since we haven't called Authenticate() which initializes rmapi)
	err = client.DownloadDocument("doc-123", outputPath)
	if err == nil {
		t.Error("DownloadDocument() should return error when API client not initialized")
	}

	if !strings.Contains(err.Error(), "API client not initialized") {
		t.Errorf("Expected 'API client not initialized' error, got: %v", err)
	}
}

func TestClient_Authenticate_WithExistingToken(t *testing.T) {
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "token.json")

	// Create a valid token file
	token := map[string]string{
		"device_token": "existing-token-12345",
	}
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal token: %v", err)
	}
	if err := os.WriteFile(tokenPath, data, 0600); err != nil {
		t.Fatalf("failed to write token file: %v", err)
	}

	cfg := &Config{
		TokenPath: tokenPath,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Note: Authenticate() now actually tries to connect to the reMarkable API
	// With a test token, this will fail with a network/auth error
	// This tests that the token is loaded and the API initialization is attempted
	err = client.Authenticate()

	// Verify token was loaded from file (even if API init failed)
	if client.token != "existing-token-12345" {
		t.Errorf("expected token existing-token-12345, got %s", client.token)
	}

	// API initialization should fail in tests (no real credentials)
	// We expect an error about getting user token or API initialization
	if err == nil {
		t.Error("Authenticate() should fail with test token (network/auth error expected)")
	}

	if !strings.Contains(err.Error(), "failed to") {
		t.Logf("Got expected error: %v", err)
	}
}

func TestClient_Authenticate_NoToken(t *testing.T) {
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "token.json")

	cfg := &Config{
		TokenPath: tokenPath,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Authenticate should fail when no token exists
	err = client.Authenticate()
	if err == nil {
		t.Error("Authenticate() should error when no token exists")
	}

	// Verify client is not authenticated
	if client.IsAuthenticated() {
		t.Error("client should not be authenticated when Authenticate() fails")
	}
}

func TestClient_Authenticate_InvalidToken(t *testing.T) {
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "token.json")

	// Create an invalid token file (invalid JSON)
	if err := os.WriteFile(tokenPath, []byte("not valid json"), 0600); err != nil {
		t.Fatalf("failed to write token file: %v", err)
	}

	cfg := &Config{
		TokenPath: tokenPath,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Authenticate should fail with invalid token
	err = client.Authenticate()
	if err == nil {
		t.Error("Authenticate() should error with invalid token")
	}

	// Verify client is not authenticated
	if client.IsAuthenticated() {
		t.Error("client should not be authenticated when token is invalid")
	}
}

func TestClient_Authenticate_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	tokenDir := filepath.Join(tmpDir, "subdir", "auth")
	tokenPath := filepath.Join(tokenDir, "token.json")

	cfg := &Config{
		TokenPath: tokenPath,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Authenticate should create the directory
	_ = client.Authenticate() // Will fail due to no token, but should create dir

	// Verify directory was created
	if _, err := os.Stat(tokenDir); os.IsNotExist(err) {
		t.Error("token directory should be created by Authenticate()")
	}
}

func TestClient_ListDocuments_Authenticated(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		TokenPath: filepath.Join(tmpDir, "token.json"),
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Set authentication token
	if err := client.SetToken("test-token"); err != nil {
		t.Fatalf("SetToken() error = %v", err)
	}

	// Should return not-implemented error when authenticated
	_, err = client.ListDocuments([]string{"label1", "label2"})
	if err == nil {
		t.Error("ListDocuments() should return not-implemented error")
	}
}

func TestClient_GetDocumentMetadata_Authenticated(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		TokenPath: filepath.Join(tmpDir, "token.json"),
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Set authentication token
	if err := client.SetToken("test-token"); err != nil {
		t.Fatalf("SetToken() error = %v", err)
	}

	// Should return not-implemented error when authenticated
	_, err = client.GetDocumentMetadata("doc-456")
	if err == nil {
		t.Error("GetDocumentMetadata() should return not-implemented error")
	}
}

func TestClient_LoadToken_EmptyDeviceToken(t *testing.T) {
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "token.json")

	// Create token file with empty device_token
	token := map[string]string{
		"device_token": "",
	}
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal token: %v", err)
	}
	if err := os.WriteFile(tokenPath, data, 0600); err != nil {
		t.Fatalf("failed to write token file: %v", err)
	}

	cfg := &Config{
		TokenPath: tokenPath,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	err = client.loadToken()
	if err == nil {
		t.Error("loadToken() should error when device_token is empty")
	}
}

func TestClient_GetFolderPath_NotAuthenticated(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		TokenPath: filepath.Join(tmpDir, "token.json"),
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Should error when not authenticated
	_, err = client.GetFolderPath("doc-123")
	if err == nil {
		t.Error("GetFolderPath() should error when not authenticated")
	}

	if !strings.Contains(err.Error(), "not authenticated") {
		t.Errorf("Expected 'not authenticated' error, got: %v", err)
	}
}

func TestClient_GetFolderPath_APINotInitialized(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		TokenPath: filepath.Join(tmpDir, "token.json"),
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Set token but don't initialize API
	if err := client.SetToken("test-token"); err != nil {
		t.Fatalf("SetToken() error = %v", err)
	}

	// Should error when API client not initialized
	_, err = client.GetFolderPath("doc-123")
	if err == nil {
		t.Error("GetFolderPath() should error when API client not initialized")
	}

	if !strings.Contains(err.Error(), "API client not initialized") {
		t.Errorf("Expected 'API client not initialized' error, got: %v", err)
	}
}

func TestSanitizeFolderName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean name",
			input:    "MyFolder",
			expected: "MyFolder",
		},
		{
			name:     "forward slash",
			input:    "Work/Projects",
			expected: "Work-Projects",
		},
		{
			name:     "backslash",
			input:    "Work\\Projects",
			expected: "Work-Projects",
		},
		{
			name:     "colon",
			input:    "Project: Important",
			expected: "Project- Important",
		},
		{
			name:     "asterisk",
			input:    "Files*Backup",
			expected: "Files_Backup",
		},
		{
			name:     "question mark",
			input:    "What?",
			expected: "What_",
		},
		{
			name:     "quotes",
			input:    "My \"Notes\"",
			expected: "My 'Notes'",
		},
		{
			name:     "angle brackets",
			input:    "<Important>",
			expected: "_Important_",
		},
		{
			name:     "pipe",
			input:    "Option A | Option B",
			expected: "Option A - Option B",
		},
		{
			name:     "multiple special chars",
			input:    "Work/Projects: Notes*2024",
			expected: "Work-Projects- Notes_2024",
		},
		{
			name:     "unicode and spaces",
			input:    "Notes 📝",
			expected: "Notes 📝",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "leading and trailing whitespace",
			input:    "  MyFolder  ",
			expected: "MyFolder",
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: "",
		},
		{
			name:     "single slash becomes dash",
			input:    "/",
			expected: "-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFolderName(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFolderName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsValidFolderName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "valid folder name",
			input: "MyFolder",
			want:  true,
		},
		{
			name:  "valid with spaces",
			input: "My Folder",
			want:  true,
		},
		{
			name:  "valid with numbers",
			input: "Folder123",
			want:  true,
		},
		{
			name:  "empty string is invalid",
			input: "",
			want:  false,
		},
		{
			name:  "single dash is invalid",
			input: "-",
			want:  false,
		},
		{
			name:  "single underscore is invalid",
			input: "_",
			want:  false,
		},
		{
			name:  "single dot is invalid",
			input: ".",
			want:  false,
		},
		{
			name:  "double dot is invalid",
			input: "..",
			want:  false,
		},
		{
			name:  "dash with text is valid",
			input: "My-Folder",
			want:  true,
		},
		{
			name:  "underscore with text is valid",
			input: "My_Folder",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidFolderName(tt.input)
			if result != tt.want {
				t.Errorf("isValidFolderName(%q) = %v, want %v", tt.input, result, tt.want)
			}
		})
	}
}

// Note: Testing GetFolderPath with actual folder hierarchy requires mocking the rmapi library
// or integration tests with a real reMarkable account. The following scenarios would be tested
// with proper mocks:
// 1. Document in root folder (empty parent) -> returns ""
// 2. Document in single folder -> returns "FolderName"
// 3. Document in nested folders -> returns "Parent/Child/Grandchild"
// 4. Circular reference detection -> returns error
// 5. Document not found -> returns error
// These would be covered by integration tests or with a proper mock of api.ApiCtx and model.Filetree

func TestIsTokenExpired(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected bool
	}{
		{
			name:     "empty token",
			token:    "",
			expected: true,
		},
		{
			name:     "invalid JWT format",
			token:    "not-a-jwt-token",
			expected: true,
		},
		{
			name:     "valid token with future expiration",
			token:    createTestJWT(t, time.Now().Add(1*time.Hour)),
			expected: false,
		},
		{
			name:     "expired token",
			token:    createTestJWT(t, time.Now().Add(-1*time.Hour)),
			expected: true,
		},
		{
			name:     "token expiring within buffer period",
			token:    createTestJWT(t, time.Now().Add(2*time.Minute)),
			expected: true,
		},
		{
			name:     "token expiring just outside buffer period",
			token:    createTestJWT(t, time.Now().Add(10*time.Minute)),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTokenExpired(tt.token)
			if result != tt.expected {
				t.Errorf("isTokenExpired() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetTokenExpiration(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		hasTime  bool
	}{
		{
			name:    "empty token",
			token:   "",
			hasTime: false,
		},
		{
			name:    "invalid JWT format",
			token:   "not-a-jwt-token",
			hasTime: false,
		},
		{
			name:    "valid token with expiration",
			token:   createTestJWT(t, time.Now().Add(1*time.Hour)),
			hasTime: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTokenExpiration(tt.token)
			if tt.hasTime && result.IsZero() {
				t.Error("expected non-zero time, got zero time")
			}
			if !tt.hasTime && !result.IsZero() {
				t.Error("expected zero time, got non-zero time")
			}
		})
	}
}

// createTestJWT creates a simple JWT token for testing
// Note: This is a minimal JWT for testing purposes only
func createTestJWT(t *testing.T, expiration time.Time) string {
	t.Helper()

	// Create a simple JWT header and payload
	header := map[string]interface{}{
		"alg": "none",
		"typ": "JWT",
	}
	payload := map[string]interface{}{
		"exp": expiration.Unix(),
		"sub": "test-user",
	}

	// Marshal to JSON
	headerJSON, err := json.Marshal(header)
	if err != nil {
		t.Fatalf("failed to marshal header: %v", err)
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	// Base64 encode (URL safe, no padding)
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

	// Create JWT (header.payload.signature)
	// For testing, we don't need a real signature
	return headerB64 + "." + payloadB64 + "."
}
