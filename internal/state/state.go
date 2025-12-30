package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Manager handles sync state persistence and operations
type Manager struct {
	state    *SyncState
	filePath string
	mu       sync.RWMutex
}

// NewManager creates a new state manager
func NewManager(filePath string) *Manager {
	return &Manager{
		state:    NewSyncState(),
		filePath: filePath,
	}
}

// Load reads the sync state from the JSON file
// If the file doesn't exist, returns a new empty state (not an error)
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if file exists
	if _, err := os.Stat(m.filePath); os.IsNotExist(err) {
		// File doesn't exist, use new empty state
		m.state = NewSyncState()
		return nil
	}

	// Read file
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return fmt.Errorf("failed to read state file: %w", err)
	}

	// Unmarshal JSON
	var state SyncState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to parse state file: %w", err)
	}

	// Validate version
	if state.Version != StateFileVersion {
		return fmt.Errorf("unsupported state file version %d (expected %d)", state.Version, StateFileVersion)
	}

	m.state = &state
	return nil
}

// Save writes the sync state to the JSON file atomically
func (m *Manager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(m.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Write atomically: write to temp file, then rename
	tmpFile := m.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp state file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpFile, m.filePath); err != nil {
		os.Remove(tmpFile) // Clean up temp file on error
		return fmt.Errorf("failed to rename temp state file: %w", err)
	}

	return nil
}

// GetState returns a copy of the current sync state
func (m *Manager) GetState() *SyncState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

// GetDocument returns the document state for a specific document ID
func (m *Manager) GetDocument(id string) *DocumentState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state.GetDocument(id)
}

// AddDocument adds or updates a document in the state
func (m *Manager) AddDocument(doc *DocumentState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.AddDocument(doc)
}

// RemoveDocument removes a document from the state
func (m *Manager) RemoveDocument(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.RemoveDocument(id)
}

// UpdateLastSync updates the last sync timestamp
func (m *Manager) UpdateLastSync() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.UpdateLastSync()
}

// GetDocumentsByLabel returns all documents with a specific label
func (m *Manager) GetDocumentsByLabel(label string) []*DocumentState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state.GetDocumentsByLabel(label)
}

// GetDocumentsByStatus returns all documents with a specific conversion status
func (m *Manager) GetDocumentsByStatus(status ConversionStatus) []*DocumentState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state.GetDocumentsByStatus(status)
}

// GetDocumentsNeedingOCR returns all documents that need OCR processing
func (m *Manager) GetDocumentsNeedingOCR() []*DocumentState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state.GetDocumentsNeedingOCR()
}

// LoadOrCreate loads an existing state file or creates a new one if it doesn't exist
// This is a convenience function that combines Load with automatic Save on new state
func LoadOrCreate(filePath string) (*Manager, error) {
	manager := NewManager(filePath)

	if err := manager.Load(); err != nil {
		return nil, err
	}

	// If this is a new state (no documents), save it to create the file
	if len(manager.state.Documents) == 0 && manager.state.LastSync.IsZero() {
		if err := manager.Save(); err != nil {
			return nil, fmt.Errorf("failed to save initial state: %w", err)
		}
	}

	return manager, nil
}

// Reset clears all state and creates a fresh empty state
func (m *Manager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = NewSyncState()
}

// Count returns the total number of documents in the state
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.state.Documents)
}
