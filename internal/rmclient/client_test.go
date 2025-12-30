package rmclient

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/platinummonkey/remarkable-sync/internal/logger"
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
	expectedPath := filepath.Join(home, ".remarkable-sync", "token.json")

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
