//go:build darwin
// +build darwin

package menubar

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"sync"

	"github.com/platinummonkey/legible/internal/logger"
)

//go:embed templates/preferences.html
var preferencesHTML embed.FS

// PreferencesServer serves the preferences UI via HTTP
type PreferencesServer struct {
	server   *http.Server
	mu       sync.Mutex
	app      *App
	tmpl     *template.Template
	listener string
}

// NewPreferencesServer creates a new preferences server
func NewPreferencesServer(app *App) *PreferencesServer {
	tmpl, err := template.ParseFS(preferencesHTML, "templates/preferences.html")
	if err != nil {
		logger.Error("Failed to parse preferences template", "error", err)
		return nil
	}

	ps := &PreferencesServer{
		app:      app,
		tmpl:     tmpl,
		listener: "127.0.0.1:0", // Random available port
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", ps.handlePreferences)
	mux.HandleFunc("/save", ps.handleSave)
	mux.HandleFunc("/close", ps.handleClose)

	ps.server = &http.Server{
		Addr:    ps.listener,
		Handler: mux,
	}

	return ps
}

// Start starts the preferences server and returns the URL
func (ps *PreferencesServer) Start() (string, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("failed to create listener: %w", err)
	}

	ps.listener = fmt.Sprintf("http://%s", listener.Addr().String())

	go func() {
		if err := ps.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			logger.Error("Preferences server error", "error", err)
		}
	}()

	return ps.listener, nil
}

// Stop stops the preferences server
func (ps *PreferencesServer) Stop() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.server != nil {
		return ps.server.Close()
	}
	return nil
}

// handlePreferences serves the preferences HTML page
func (ps *PreferencesServer) handlePreferences(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data := struct {
		DaemonAddr       string
		SyncInterval     string
		OCREnabled       bool
		DaemonConfigFile string
	}{
		DaemonAddr:       ps.app.menuBarConfig.DaemonAddr,
		SyncInterval:     ps.app.menuBarConfig.SyncInterval,
		OCREnabled:       ps.app.menuBarConfig.OCREnabled,
		DaemonConfigFile: ps.app.menuBarConfig.DaemonConfigFile,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ps.tmpl.Execute(w, data); err != nil {
		logger.Error("Failed to render preferences template", "error", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

// handleSave handles saving preferences
func (ps *PreferencesServer) handleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Update config
	ps.app.menuBarConfig.DaemonAddr = r.FormValue("daemon_addr")
	ps.app.menuBarConfig.SyncInterval = r.FormValue("sync_interval")
	ps.app.menuBarConfig.OCREnabled = r.FormValue("ocr_enabled") == "on"
	ps.app.menuBarConfig.DaemonConfigFile = r.FormValue("daemon_config_file")

	// Save to file
	if err := SaveMenuBarConfig(ps.app.menuBarConfig, ps.app.configPath); err != nil {
		logger.Error("Failed to save configuration", "error", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "error",
			"message": fmt.Sprintf("Failed to save settings: %v", err),
		})
		return
	}

	logger.Info("Configuration saved successfully")

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Settings saved successfully! Please restart the app for changes to take effect.",
	})
}

// handleClose handles closing the preferences window
func (ps *PreferencesServer) handleClose(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})

	// Close the window (browser will handle this via JavaScript)
	go ps.Stop()
}
